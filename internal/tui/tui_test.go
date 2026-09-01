package tui

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
	"github.com/muesli/termenv"

	"todo-list/internal/storage"
	"todo-list/internal/todo"
)

func TestMain(m *testing.M) {
	// 单元测试环境不是 TTY,lipgloss 默认按 Ascii profile 渲染、不输出 ANSI
	// 转义序列;固定为 ANSI256 以便渲染契约断言能验证斜体/背景色等样式是否
	// 真正生效(该 API 即为 lipgloss 官方为测试提供的入口)。
	lipgloss.SetColorProfile(termenv.ANSI256)
	os.Exit(m.Run())
}

// ansiSeq matches SGR escape sequences (colors, attributes, resets).
var ansiSeq = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// stripANSI removes SGR sequences so assertions can reason about the visible
// text and its display width instead of raw bytes.
func stripANSI(s string) string {
	return ansiSeq.ReplaceAllString(s, "")
}

// nonEmptyLines returns the visible, non-blank lines of a rendered view.
func nonEmptyLines(v string) []string {
	var out []string
	for _, l := range strings.Split(v, "\n") {
		if s := strings.TrimSpace(stripANSI(l)); s != "" {
			out = append(out, s)
		}
	}
	return out
}

// newTestModel returns a model backed by a temp-file JSON store.
func newTestModel(t *testing.T) (Model, *storage.JSONStorage) {
	t.Helper()
	store, err := storage.NewJSONStorage(filepath.Join(t.TempDir(), "todos.json"))
	if err != nil {
		t.Fatalf("NewJSONStorage: %v", err)
	}
	return NewModel(store, nil), store
}

// keyMsg builds a tea.KeyMsg for a symbolic name or a literal rune sequence.
func keyMsg(s string) tea.KeyMsg {
	switch s {
	case "up":
		return tea.KeyMsg{Type: tea.KeyUp}
	case "down":
		return tea.KeyMsg{Type: tea.KeyDown}
	case "left":
		return tea.KeyMsg{Type: tea.KeyLeft}
	case "right":
		return tea.KeyMsg{Type: tea.KeyRight}
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEsc}
	case "space":
		return tea.KeyMsg{Type: tea.KeySpace}
	case "backspace":
		return tea.KeyMsg{Type: tea.KeyBackspace}
	case "ctrl+c":
		return tea.KeyMsg{Type: tea.KeyCtrlC}
	default:
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
	}
}

// press feeds the given keys through Update and returns the updated model.
func press(m Model, keys ...string) Model {
	var cur tea.Model = m
	for _, k := range keys {
		next, _ := cur.Update(keyMsg(k))
		cur = next
	}
	return cur.(Model)
}

func TestAddTodoViaInput(t *testing.T) {
	m, store := newTestModel(t)

	// 提供真实终端宽度,避免描述被截断。
	w, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	m = w.(Model)

	m = press(m, "n")
	if !m.inputMode {
		t.Fatal("expected input mode after pressing n")
	}
	m = press(m, "买", "牛", "奶")
	m = press(m, "enter") // 标题阶段:Enter 进入描述阶段
	if m.addPhase != addPhaseDescription {
		t.Fatal("expected description phase after title Enter")
	}
	m = press(m, "家", "门", "口", "取")
	m = press(m, "enter") // 描述阶段:Enter 保存

	if m.inputMode {
		t.Fatal("expected to leave input mode after enter")
	}
	items, err := store.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(items))
	}
	if items[0].Title != "买牛奶" {
		t.Errorf("title = %q, want 买牛奶", items[0].Title)
	}
	if items[0].Description != "家门口取" {
		t.Errorf("description = %q, want 家门口取", items[0].Description)
	}
	if items[0].Status != todo.StatusNotStarted {
		t.Errorf("status = %q, want %q", items[0].Status, todo.StatusNotStarted)
	}
	if m.activeCol != colIndex(todo.StatusNotStarted) {
		t.Errorf("activeCol = %d, want 未开始 column", m.activeCol)
	}
	if !strings.Contains(m.View(), "买牛奶") {
		t.Errorf("view should render the new todo:\n%s", m.View())
	}
	if !strings.Contains(m.View(), "家门口取") {
		t.Errorf("view should render the description:\n%s", m.View())
	}
}

func TestAddDescriptionSkippable(t *testing.T) {
	m, store := newTestModel(t)

	// 描述可留空:标题阶段 Enter 后再按一次 Enter 直接保存。
	m = press(m, "n", "任务B", "enter", "enter")
	if m.inputMode {
		t.Fatal("expected to leave input mode after second enter")
	}
	items, err := store.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(items))
	}
	if items[0].Title != "任务B" {
		t.Errorf("title = %q, want 任务B", items[0].Title)
	}
	if items[0].Description != "" {
		t.Errorf("description = %q, want empty (skipped)", items[0].Description)
	}
}

func TestAddCancelWithEsc(t *testing.T) {
	m, store := newTestModel(t)

	m = press(m, "n", "测试")
	if !m.inputMode {
		t.Fatal("expected input mode")
	}
	m = press(m, "esc")

	if m.inputMode {
		t.Fatal("expected input mode off after esc")
	}
	items, err := store.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("len(items) = %d, want 0 (esc must not save)", len(items))
	}

	// Esc 在描述阶段也必须丢弃整个新增流程。
	m = press(m, "n", "标题", "enter", "描述")
	if m.addPhase != addPhaseDescription {
		t.Fatal("expected description phase")
	}
	m = press(m, "esc")
	if m.inputMode {
		t.Fatal("expected input mode off after esc in description phase")
	}
	items, _ = store.List()
	if len(items) != 0 {
		t.Fatalf("len(items) = %d, want 0 (esc in description phase must not save)", len(items))
	}
}

func TestSpaceTogglesDone(t *testing.T) {
	m, store := newTestModel(t)
	m = press(m, "n", "任务A", "enter", "enter")

	// 在空栏上按 space/i 不能崩溃。
	m = press(m, "right") // 进行中 is empty
	m = press(m, "space", "i")
	m = press(m, "left") // back to 未开始

	// Space 将 任务A 标记为已完成,任务跨栏移到 已完成 栏。
	m = press(m, "space")
	items, _ := store.List()
	if items[0].Status != todo.StatusDone {
		t.Errorf("status = %q, want %q", items[0].Status, todo.StatusDone)
	}

	// 看板下三栏始终可见:已完成 栏渲染 任务A(带 ✓),光标可移动到该栏。
	m = press(m, "right", "right") // 进行中 → 已完成
	if !strings.Contains(m.View(), "任务A") {
		t.Errorf("已完成 column should show 任务A:\n%s", m.View())
	}

	// Space 再次切换回 未开始。
	m = press(m, "space")
	items, _ = store.List()
	if items[0].Status != todo.StatusNotStarted {
		t.Errorf("status = %q, want %q", items[0].Status, todo.StatusNotStarted)
	}
}

func TestMarkInProgress(t *testing.T) {
	m, store := newTestModel(t)
	m = press(m, "n", "任务A", "enter", "enter")

	m = press(m, "i")
	items, _ := store.List()
	if items[0].Status != todo.StatusInProgress {
		t.Errorf("status = %q, want %q", items[0].Status, todo.StatusInProgress)
	}

	// The item moved to the 进行中 column (still visible, kanban keeps all columns).
	m = press(m, "right")
	if !strings.Contains(m.View(), "任务A") {
		t.Errorf("进行中 column should show 任务A:\n%s", m.View())
	}
}

func TestNavigationSelection(t *testing.T) {
	m, _ := newTestModel(t)
	m = press(m, "n", "A", "enter", "enter")
	m = press(m, "n", "B", "enter", "enter")
	m = press(m, "n", "C", "enter", "enter")

	if m.selected != 2 {
		t.Fatalf("selected = %d, want 2 (last added)", m.selected)
	}

	m = press(m, "up")
	if m.selected != 1 {
		t.Errorf("after up selected = %d, want 1", m.selected)
	}

	m = press(m, "up", "up", "up")
	if m.selected != 0 {
		t.Errorf("up at top should clamp: selected = %d, want 0", m.selected)
	}

	m = press(m, "down", "down", "down")
	if m.selected != 2 {
		t.Errorf("down at bottom should clamp: selected = %d, want 2", m.selected)
	}

	// ←/→ 在栏间移动,边界处钳制(不循环);换栏时选中行重置为 0。
	m = press(m, "left") // 已在 未开始 栏,左边无栏,钳制不动
	if m.activeCol != 0 {
		t.Errorf("left at first column should clamp: activeCol = %d, want 0", m.activeCol)
	}
	m = press(m, "right", "right") // → 进行中 → 已完成(空栏)
	if m.activeCol != 2 {
		t.Errorf("after two rights activeCol = %d, want 2", m.activeCol)
	}
	if m.selected != 0 {
		t.Errorf("selected on empty column = %d, want 0", m.selected)
	}
	m = press(m, "right") // 已在最后一栏,右边无栏,钳制不动
	if m.activeCol != 2 {
		t.Errorf("right at last column should clamp: activeCol = %d, want 2", m.activeCol)
	}
	m = press(m, "left", "left") // → 进行中 → 未开始
	if m.activeCol != 0 {
		t.Errorf("after two lefts activeCol = %d, want 0", m.activeCol)
	}
	if m.selected != 0 {
		t.Errorf("after column switch selected = %d, want 0", m.selected)
	}
}

func TestEmptyTabRendersGracefully(t *testing.T) {
	m, _ := newTestModel(t)
	m = press(m, "n", "A", "enter", "enter")

	m = press(m, "right") // 进行中 is empty
	if !strings.Contains(m.View(), "暂无任务") {
		t.Errorf("empty column should render an empty-state hint:\n%s", m.View())
	}
	m = press(m, "down", "up", "space", "i") // navigation must not crash

	m = press(m, "left")
	if !strings.Contains(m.View(), "A") {
		t.Errorf("未开始 column should still show A:\n%s", m.View())
	}
}

func TestQuitKeys(t *testing.T) {
	m, _ := newTestModel(t)

	_, cmd := m.Update(keyMsg("q"))
	if cmd == nil {
		t.Error("q should produce a quit command")
	}
	_, cmd = m.Update(keyMsg("ctrl+c"))
	if cmd == nil {
		t.Error("ctrl+c should produce a quit command")
	}
}

func TestModelPersistsAcrossRelaunch(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "todos.json")

	store1, err := storage.NewJSONStorage(path)
	if err != nil {
		t.Fatalf("NewJSONStorage: %v", err)
	}
	m1 := NewModel(store1, nil)
	m1 = press(m1, "n", "重启测试", "enter", "enter")

	// Simulate quitting and relaunching: a fresh store reads the same file.
	store2, err := storage.NewJSONStorage(path)
	if err != nil {
		t.Fatalf("NewJSONStorage: %v", err)
	}
	items, err := store2.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(items) != 1 || items[0].Title != "重启测试" {
		t.Fatalf("after relaunch got %+v", items)
	}

	m2 := NewModel(store2, items)
	if !strings.Contains(m2.View(), "重启测试") {
		t.Errorf("relaunched model should render the persisted todo:\n%s", m2.View())
	}
}

// TestKanbanShowsAllThreeColumns:不按任何键,三个列头就同时可见;三个
// 不同状态的任务也同时渲染在同一画面(旧 tab UI 下只有当前 tab 的内容可见)。
func TestKanbanShowsAllThreeColumns(t *testing.T) {
	m, _ := newTestModel(t)

	// 空看板:初始 View 即同时含三个列头,无需任何按键。
	v := m.View()
	for _, h := range []string{"未开始", "进行中", "已完成"} {
		if !strings.Contains(v, h) {
			t.Errorf("initial kanban view should contain header %q:\n%s", h, v)
		}
	}

	// 建三个任务并分散到三个状态:任务C 未开始、任务A 进行中、任务B 已完成。
	m = press(m, "n", "任务A", "enter", "enter")
	m = press(m, "n", "任务B", "enter", "enter")
	m = press(m, "n", "任务C", "enter", "enter")
	m = press(m, "up", "up", "i") // 选中 任务A → 进行中
	m = press(m, "space")         // 选中 任务B → 已完成

	v = m.View()
	for _, h := range []string{"未开始", "进行中", "已完成"} {
		if !strings.Contains(v, h) {
			t.Errorf("kanban view should contain column header %q:\n%s", h, v)
		}
	}
	for _, title := range []string{"任务A", "任务B", "任务C"} {
		if !strings.Contains(v, title) {
			t.Errorf("kanban view should render %q simultaneously:\n%s", title, v)
		}
	}
}

// TestCrossColumnCursorClamp:状态变更导致任务跨栏后,selected 在所在栏内
// clamp;commitAdd 后新任务落 未开始 栏并选中。
func TestCrossColumnCursorClamp(t *testing.T) {
	m, store := newTestModel(t)
	m = press(m, "n", "任务A", "enter", "enter")
	m = press(m, "n", "任务B", "enter", "enter")
	m = press(m, "n", "任务C", "enter", "enter")

	// 未开始 栏:[任务A,任务B,任务C],selected=2(任务C,最后新增)。
	if m.activeCol != colIndex(todo.StatusNotStarted) {
		t.Fatalf("after add activeCol = %d, want 未开始 column", m.activeCol)
	}
	if m.selected != 2 {
		t.Fatalf("after add selected = %d, want 2 (newest item)", m.selected)
	}

	// 上移到 任务A(selected=0)并按 i:任务A 移入 进行中,
	// 未开始 栏剩 [任务B,任务C],selected 应 clamp 到 0(任务B)。
	m = press(m, "up", "up", "i")
	items, _ := store.List()
	if items[0].Status != todo.StatusInProgress {
		t.Errorf("任务A status = %q, want 进行中", items[0].Status)
	}
	if m.activeCol != colIndex(todo.StatusNotStarted) {
		t.Errorf("activeCol = %d, want 未开始 column", m.activeCol)
	}
	if m.selected != 0 {
		t.Errorf("after moving A out, selected = %d, want 0 (clamped)", m.selected)
	}

	// 光标切到 进行中 栏(任务A),space 后任务A 移入 已完成,
	// 进行中 栏变空,selected 应 clamp 到 0。
	m = press(m, "right")
	if m.selected != 0 {
		t.Errorf("进行中 column selected = %d, want 0", m.selected)
	}
	m = press(m, "space")
	items, _ = store.List()
	if items[0].Status != todo.StatusDone {
		t.Errorf("任务A status = %q, want 已完成", items[0].Status)
	}
	if m.activeCol != 1 {
		t.Errorf("activeCol = %d, want 1 (进行中 column)", m.activeCol)
	}
	if m.selected != 0 {
		t.Errorf("after moving A out of 进行中, selected = %d, want 0", m.selected)
	}
	// 空栏上的操作不能崩溃。
	m = press(m, "space", "i", "down", "up")
}

// TestEditTitleAndDescription:按 e 编辑选中任务,标题与描述两段预填,
// Enter 保存后 title/description 更新,而 id/status/created_at 保持不变。
func TestEditTitleAndDescription(t *testing.T) {
	m, store := newTestModel(t)
	m = press(m, "n", "旧标题", "enter", "旧描述", "enter")
	items, _ := store.List()
	orig := items[0]

	m = press(m, "e")
	if !m.inputMode || !m.editMode {
		t.Fatal("expected edit mode after e")
	}
	if m.input.Value() != "旧标题" {
		t.Errorf("title input prefilled = %q, want 旧标题", m.input.Value())
	}
	for range "旧标题" {
		m = press(m, "backspace")
	}
	m = press(m, "新", "标", "题")
	m = press(m, "enter") // → 描述阶段
	if m.addPhase != addPhaseDescription {
		t.Fatal("expected description phase")
	}
	if m.input.Value() != "旧描述" {
		t.Errorf("description input prefilled = %q, want 旧描述", m.input.Value())
	}
	for range "旧描述" {
		m = press(m, "backspace")
	}
	m = press(m, "新", "描", "述")
	m = press(m, "enter") // 保存

	if m.inputMode || m.editMode {
		t.Fatal("expected edit mode off after save")
	}
	items, _ = store.List()
	got := items[0]
	if got.ID != orig.ID {
		t.Errorf("ID changed: %q → %q", orig.ID, got.ID)
	}
	if got.Title != "新标题" {
		t.Errorf("title = %q, want 新标题", got.Title)
	}
	if got.Description != "新描述" {
		t.Errorf("description = %q, want 新描述", got.Description)
	}
	if got.Status != orig.Status {
		t.Errorf("status changed: %q → %q", orig.Status, got.Status)
	}
	if !got.CreatedAt.Equal(orig.CreatedAt) {
		t.Errorf("created_at changed: %v → %v", orig.CreatedAt, got.CreatedAt)
	}
}

// TestEditCancelWithEsc:Esc 在标题或描述阶段取消编辑,均不得落盘。
func TestEditCancelWithEsc(t *testing.T) {
	m, store := newTestModel(t)
	m = press(m, "n", "旧标题", "enter", "enter")

	// 标题阶段 Esc 取消。
	m = press(m, "e", "改", "esc")
	if m.inputMode || m.editMode {
		t.Fatal("expected edit mode off after esc")
	}
	items, _ := store.List()
	if items[0].Title != "旧标题" {
		t.Errorf("title changed after esc: %q", items[0].Title)
	}

	// 描述阶段 Esc 取消。
	m = press(m, "e", "enter", "改描述", "esc")
	if m.inputMode || m.editMode {
		t.Fatal("expected edit mode off after esc in description phase")
	}
	items, _ = store.List()
	if items[0].Title != "旧标题" {
		t.Errorf("title changed after esc in description phase: %q", items[0].Title)
	}
	if items[0].Description != "" {
		t.Errorf("description changed after esc: %q", items[0].Description)
	}
}

// TestEditDescriptionOnly:标题阶段直接 Enter 保持原样,只修改描述。
func TestEditDescriptionOnly(t *testing.T) {
	m, store := newTestModel(t)
	m = press(m, "n", "标题", "enter", "enter")

	m = press(m, "e", "enter") // 标题不动,进描述阶段
	if m.input.Value() != "" {
		t.Errorf("description prefilled = %q, want empty", m.input.Value())
	}
	m = press(m, "补", "充")
	m = press(m, "enter")

	items, _ := store.List()
	if items[0].Title != "标题" {
		t.Errorf("title changed: %q", items[0].Title)
	}
	if items[0].Description != "补充" {
		t.Errorf("description = %q, want 补充", items[0].Description)
	}
}

// TestEditKeepsColumn:编辑不改变任务状态与所在栏。
func TestEditKeepsColumn(t *testing.T) {
	m, store := newTestModel(t)
	m = press(m, "n", "任务A", "enter", "enter")
	m = press(m, "i")     // → 进行中
	m = press(m, "right") // 光标移到 进行中 栏
	m = press(m, "e", "enter", "enter")

	items, _ := store.List()
	if items[0].Status != todo.StatusInProgress {
		t.Errorf("status = %q, want 进行中", items[0].Status)
	}
	if m.activeCol != colIndex(todo.StatusInProgress) {
		t.Errorf("activeCol = %d, want 进行中 column", m.activeCol)
	}
	if !strings.Contains(m.View(), "任务A") {
		t.Errorf("view should still render 任务A:\n%s", m.View())
	}
}

// TestEditEmptyTitleIgnored:编辑时清空标题并 Enter,不得保存。
func TestEditEmptyTitleIgnored(t *testing.T) {
	m, store := newTestModel(t)
	m = press(m, "n", "旧标题", "enter", "enter")

	m = press(m, "e")
	for range "旧标题" {
		m = press(m, "backspace")
	}
	m = press(m, "enter") // 空标题 → 取消不保存
	if m.inputMode || m.editMode {
		t.Fatal("expected input mode off after empty title enter")
	}
	items, _ := store.List()
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(items))
	}
	if items[0].Title != "旧标题" {
		t.Errorf("title = %q, want 旧标题 (empty title must not save)", items[0].Title)
	}
}

// TestColumnWidth:columnWidth 按 (width-2)/3 计算,100 列终端下每列 32 格;
// 未收到窗口尺寸时按默认宽度 100 处理;过窄时钳制到最小 10 格。
func TestColumnWidth(t *testing.T) {
	m, _ := newTestModel(t)
	if got := m.columnWidth(); got != 32 {
		t.Errorf("columnWidth(default) = %d, want 32", got)
	}
	for _, tc := range []struct{ width, want int }{
		{100, 32}, // (100-2)/3
		{80, 26},  // (80-2)/3
		{64, 20},  // (64-2)/3 = 20.67 → 整除 20
		{20, 10},  // (20-2)/3 = 6 → 钳制到最小 10
	} {
		m, _ := newTestModel(t)
		w, _ := m.Update(tea.WindowSizeMsg{Width: tc.width, Height: 30})
		m = w.(Model)
		if got := m.columnWidth(); got != tc.want {
			t.Errorf("columnWidth(%d) = %d, want %d", tc.width, got, tc.want)
		}
	}
}

// TestViewStructure:View 渲染结构 —— 第 0 行标题、第 1 行空行(标题与看板
// 之间仅 1 个空行)、第 2 行是含三状态名的列头行;非空行顺序为
// 标题 → 列头行 → 内容 → footer。
func TestViewStructure(t *testing.T) {
	m, _ := newTestModel(t)
	w, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	m = w.(Model)
	m = press(m, "n", "任务A", "enter", "任务描述", "enter")

	v := m.View()
	raw := strings.Split(v, "\n")
	if len(raw) < 3 {
		t.Fatalf("view has %d lines, want >= 3:\n%s", len(raw), v)
	}

	// 行结构:0=标题,1=空行,2=列头行。
	if !strings.Contains(stripANSI(raw[0]), "待办清单 Todo List") {
		t.Errorf("line 0 should be the title: %q", stripANSI(raw[0]))
	}
	if got := stripANSI(raw[1]); got != "" {
		t.Errorf("line 1 should be the single blank line between title and board, got %q", got)
	}
	header := stripANSI(raw[2])
	for _, s := range []string{"未开始", "进行中", "已完成"} {
		if !strings.Contains(header, s) {
			t.Errorf("header row (line 2) should contain %q: %q", s, header)
		}
	}

	// 非空行顺序:标题 → 含三状态名的列头行 → 内容 → footer。
	lines := nonEmptyLines(v)
	if len(lines) < 4 {
		t.Fatalf("expected title/header/content/footer, got %d lines: %v", len(lines), lines)
	}
	if !strings.Contains(lines[0], "待办清单 Todo List") {
		t.Errorf("first non-empty line should be the title: %q", lines[0])
	}
	for _, s := range []string{"未开始", "进行中", "已完成"} {
		if !strings.Contains(lines[1], s) {
			t.Errorf("second non-empty line should be the header row with %q: %q", s, lines[1])
		}
	}
	content := strings.Join(lines[2:len(lines)-1], "\n")
	if !strings.Contains(content, "任务A") || !strings.Contains(content, "任务描述") {
		t.Errorf("content lines should render the todo title and description:\n%s", content)
	}
	if !strings.Contains(lines[len(lines)-1], "↑/↓ 选择") {
		t.Errorf("last non-empty line should be the footer: %q", lines[len(lines)-1])
	}
}

// TestLongTitleFullyRendered:100 列终端下,14 个汉字的标题完整渲染不截断,
// 且标题行宽度不超过分配列宽(不溢出到相邻列)。
func TestLongTitleFullyRendered(t *testing.T) {
	m, _ := newTestModel(t)
	w, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	m = w.(Model)

	title := "一二三四五六七八九十甲乙丙丁" // 14 汉字 = 28 格
	m = press(m, "n")
	for _, r := range title {
		m = press(m, string(r))
	}
	m = press(m, "enter", "enter")

	v := m.View()
	if !strings.Contains(v, title) {
		t.Errorf("14 汉字标题未完整渲染:\n%s", v)
	}
	if strings.Contains(v, "…") {
		t.Errorf("14 汉字标题不应被截断(出现省略号):\n%s", v)
	}

	// 新任务落在 未开始 栏并选中:标题行 = "▸ " + 完整标题。
	col := m.renderColumn(colIndex(todo.StatusNotStarted), todo.StatusNotStarted)
	found := false
	for _, line := range strings.Split(col, "\n") {
		s := stripANSI(line)
		if !strings.Contains(s, title) {
			continue
		}
		found = true
		if w := runewidth.StringWidth(s); w > m.columnWidth() {
			t.Errorf("标题行宽 %d 超过分配列宽 %d: %q", w, m.columnWidth(), s)
		}
	}
	if !found {
		t.Errorf("标题行未在 未开始 栏渲染:\n%s", col)
	}
}

// TestRowWidthWithinColumn:三栏并排渲染且列宽一致;每个栏内每一渲染行
// (列头/标题行/描述行)宽度均不超过分配列宽(无重叠溢出),三栏并排后的
// 列头行恰好占满 3×列宽。
func TestRowWidthWithinColumn(t *testing.T) {
	m, _ := newTestModel(t)
	w, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	m = w.(Model)

	// 三个状态栏各放一个任务;长描述用来验证 fit 截断后行宽仍不超列宽。
	m = press(m, "n", "未开始任务", "enter", "这是一段很长很长的描述用来验证描述行不会超出列宽上限", "enter")
	m = press(m, "n", "进行中任务", "enter", "进行中任务的描述文字", "enter")
	m = press(m, "n", "已完成任务", "enter", "已完成任务的描述文字", "enter")
	m = press(m, "up", "up", "i") // 未开始任务 → 进行中
	m = press(m, "space")         // 进行中任务 → 已完成

	colWidth := m.columnWidth()
	if colWidth != 32 {
		t.Fatalf("columnWidth = %d, want 32 at width 100", colWidth)
	}

	// 每个栏内每一行 ≤ 分配列宽。
	for i, s := range Tabs {
		col := m.renderColumn(i, s)
		for _, line := range strings.Split(col, "\n") {
			if line == "" {
				continue
			}
			if w := runewidth.StringWidth(stripANSI(line)); w > colWidth {
				t.Errorf("column %q line width %d > allocated %d: %q", s, w, colWidth, stripANSI(line))
			}
		}
	}

	// 三栏并排后,列头行宽 = 3×列宽(列宽一致、无重叠)。
	v := m.View()
	raw := strings.Split(v, "\n")
	if w := runewidth.StringWidth(stripANSI(raw[2])); w != 3*colWidth {
		t.Errorf("joined header row width = %d, want %d", w, 3*colWidth)
	}
}

// TestDescriptionRowStyle:描述行以斜体、4 格缩进显示,与选中行(252 底)
// 样式可区分,宽度不超过分配列宽。
func TestDescriptionRowStyle(t *testing.T) {
	m, _ := newTestModel(t)
	w, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	m = w.(Model)
	m = press(m, "n", "标题A", "enter", "描述B", "enter")

	col := m.renderColumn(colIndex(todo.StatusNotStarted), todo.StatusNotStarted)
	var descLine, titleLine string
	for _, line := range strings.Split(col, "\n") {
		switch {
		case strings.Contains(stripANSI(line), "描述B"):
			descLine = line
		case strings.Contains(stripANSI(line), "标题A"):
			titleLine = line
		}
	}
	if descLine == "" {
		t.Fatalf("description line not rendered:\n%s", col)
	}
	if !strings.HasPrefix(stripANSI(descLine), "    描述B") {
		t.Errorf("description should be indented by 4 spaces: %q", stripANSI(descLine))
	}
	if !strings.Contains(descLine, "\x1b[3") {
		t.Errorf("description should be rendered italic: %q", descLine)
	}
	if strings.Contains(descLine, "48;5;252") {
		t.Errorf("description should not use the selected-row background (252): %q", descLine)
	}
	if titleLine == "" || !strings.Contains(titleLine, "48;5;252") {
		t.Errorf("selected title row should carry the 252 background for contrast: %q", titleLine)
	}
	if w := runewidth.StringWidth(stripANSI(descLine)); w > m.columnWidth() {
		t.Errorf("description line width %d > allocated %d", w, m.columnWidth())
	}
}
