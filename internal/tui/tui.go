// Package tui implements the terminal user interface for the todo list.
package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"

	"todo-list/internal/storage"
	"todo-list/internal/todo"
)

// Tabs defines the order of the status columns on the kanban board.
var Tabs = []todo.Status{todo.StatusNotStarted, todo.StatusInProgress, todo.StatusDone}

// Add-flow phases for the two-step "title → description" input.
const (
	addPhaseTitle = iota
	addPhaseDescription
)

// Color palette used across the whole UI so the look stays cohesive.
var (
	accentColor = lipgloss.Color("39") // bright cyan

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(accentColor)

	activeColStyle = lipgloss.NewStyle().
			Background(accentColor).
			Foreground(lipgloss.Color("0")).
			Bold(true).
			Padding(0, 2)

	inactiveColStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("238")).
				Foreground(lipgloss.Color("250")).
				Padding(0, 2)

	// 行内无横向 padding:行宽 = 前缀(2 格)+ 文字(fit 上限 = 列宽-前缀),
	// 保证每行恰好占满分配列宽、三栏并排不重叠溢出。
	selectedRowStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("252")).
				Foreground(lipgloss.Color("0")).
				Bold(true)

	rowStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))

	emptyStyle = lipgloss.NewStyle().
			Italic(true).
			Foreground(lipgloss.Color("240"))

	descriptionStyle = lipgloss.NewStyle().
				Italic(true).
				Foreground(lipgloss.Color("240"))

	footerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("245"))

	errorStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("196"))

	promptStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(accentColor)
)

// Model is the root Bubble Tea model of the todo-list application.
type Model struct {
	store        storage.Storage
	todos        []todo.Todo
	activeCol    int // focused kanban column (index into Tabs)
	selected     int // row index within the focused column
	inputMode    bool
	input        textinput.Model
	addPhase     int    // which field of the add/edit flow is being edited
	pendingTitle string // title captured when advancing to the description phase
	editMode     bool   // true while editing an existing todo
	editingID    string // ID of the todo being edited
	width        int
	height       int
	err          string
}

// NewModel builds the application model. todos is the initial list loaded at
// startup (may be empty).
func NewModel(store storage.Storage, todos []todo.Todo) Model {
	input := textinput.New()
	input.Placeholder = "输入任务标题…"
	input.Prompt = "▸ "
	input.CharLimit = 200

	m := Model{
		store: store,
		todos: todos,
		input: input,
	}
	m.clampSelection()
	return m
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return textinput.Blink
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case tea.KeyMsg:
		return m.handleKey(msg)
	default:
		return m, nil
	}
}

// handleKey dispatches keyboard input. While input mode is active, all keys
// are forwarded to the text input except Enter (save), Esc (cancel), and
// Ctrl+C (quit).
func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.inputMode {
		switch msg.String() {
		case "enter":
			m.advanceAdd()
			return m, nil
		case "esc":
			m.cancelAdd()
			return m, nil
		case "ctrl+c":
			return m, tea.Quit
		}
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	}

	// Bubble Tea may deliver several printable runes read in one chunk as a
	// single KeyRunes message (fast typing, scripts, or terminal-side
	// coalescing). Handle each rune so no key is dropped; when n is among
	// them, the runes after it become input text for the new todo.
	if msg.Type == tea.KeyRunes {
		for i, r := range msg.Runes {
			switch r {
			case 'n':
				m.startAdd()
				if i+1 < len(msg.Runes) {
					m.input, _ = m.input.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: msg.Runes[i+1:]})
				}
				return m, nil
			case 'e':
				m.startEdit()
				if i+1 < len(msg.Runes) {
					m.input, _ = m.input.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: msg.Runes[i+1:]})
				}
				return m, nil
			case 'i':
				m.markInProgress()
			case ' ':
				m.toggleDone()
			case 'q':
				return m, tea.Quit
			}
		}
		return m, nil
	}

	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "up", "k":
		m.moveSelection(-1)
	case "down", "j":
		m.moveSelection(1)
	case "left", "h":
		m.switchCol(-1)
	case "right", "l":
		m.switchCol(1)
	case " ": // a lone space is delivered as KeySpace, not KeyRunes
		m.toggleDone()
	}
	return m, nil
}

// View implements tea.Model.
func (m Model) View() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("待办清单 Todo List"))
	b.WriteString("\n\n")

	// 看板:三个状态栏并排渲染,任何时刻都同时可见。
	cols := make([]string, 0, len(Tabs))
	for i, t := range Tabs {
		cols = append(cols, m.renderColumn(i, t))
	}
	b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, cols...))

	// Input prompt or help footer. 看板后仅留 1 个空行,竖向更紧凑。
	if m.inputMode {
		if m.addPhase == addPhaseDescription {
			if m.editMode {
				b.WriteString("\n" + promptStyle.Render("编辑描述："))
			} else {
				b.WriteString("\n" + promptStyle.Render("新任务描述："))
			}
			b.WriteString(m.input.View())
			b.WriteString("\n" + footerStyle.Render("Enter 保存 · Esc 取消"))
		} else {
			if m.editMode {
				b.WriteString("\n" + promptStyle.Render("编辑标题："))
			} else {
				b.WriteString("\n" + promptStyle.Render("新任务标题："))
			}
			b.WriteString(m.input.View())
			b.WriteString("\n" + footerStyle.Render("Enter 下一步 · Esc 取消"))
		}
	} else {
		b.WriteString("\n" + footerStyle.Render("↑/↓ 选择 · ←/→ 切换栏 · n 新建 · e 编辑 · Space 完成/重开 · i 进行中 · q 退出"))
	}

	if m.err != "" {
		b.WriteString("\n" + errorStyle.Render("⚠ "+m.err))
	}

	return b.String()
}

// renderColumn renders one kanban column: a status header followed by its
// items. Only the focused column highlights its selected row; every column is
// always rendered so all three statuses stay visible.
func (m Model) renderColumn(idx int, status todo.Status) string {
	var b strings.Builder

	// 列头占满整列宽(焦点列头青色底、其余 238 底),使三栏宽度一致。
	colWidth := m.columnWidth()
	if idx == m.activeCol {
		b.WriteString(activeColStyle.Width(colWidth).Render(string(status)))
	} else {
		b.WriteString(inactiveColStyle.Width(colWidth).Render(string(status)))
	}

	items := m.column(idx)
	if len(items) == 0 {
		b.WriteString("\n" + emptyStyle.Render("暂无任务"))
		return b.String()
	}

	for i, t := range items {
		b.WriteString("\n")
		// 文字 fit 上限 = 列宽 - 前缀(2 格):选中行前缀 ▸、普通行前缀为两空格,
		// 加上前缀后行宽恰好等于分配列宽,不会溢出或与相邻列重叠。
		line := t.Title
		if t.Status == todo.StatusDone {
			line = "✓ " + line
		}
		line = fit(line, colWidth-2)
		if idx == m.activeCol && i == m.selected {
			b.WriteString(selectedRowStyle.Render("▸ " + line))
		} else {
			b.WriteString(rowStyle.Render("  " + line))
		}
		// 描述行以斜体缩进显示(比标题多缩进 2 格),同样不超过列宽。
		if t.Description != "" {
			b.WriteString("\n" + descriptionStyle.Render("    "+fit(t.Description, colWidth-4)))
		}
	}
	return b.String()
}

// columnWidth returns the total width of each kanban column. The three
// columns together span width-2 cells, so each column is (width-2)/3
// (100 列终端下每列 32 格)。
func (m Model) columnWidth() int {
	w := m.width
	if w <= 0 {
		w = 100
	}
	col := (w - 2) / 3
	if col < 10 {
		col = 10
	}
	return col
}

// column returns the todos whose status belongs to the given kanban column.
func (m Model) column(idx int) []todo.Todo {
	status := Tabs[idx]
	out := make([]todo.Todo, 0, len(m.todos))
	for _, t := range m.todos {
		if t.Status == status {
			out = append(out, t)
		}
	}
	return out
}

// visible returns the todos that belong to the focused column.
func (m Model) visible() []todo.Todo {
	return m.column(m.activeCol)
}

// startAdd enters the first phase of the add flow (title input).
func (m *Model) startAdd() {
	m.inputMode = true
	m.addPhase = addPhaseTitle
	m.pendingTitle = ""
	m.input.SetValue("")
	m.input.Placeholder = "输入任务标题…"
	m.input.CharLimit = 200
	m.input.Focus()
}

// advanceAdd moves the add/edit flow forward: title → description → save. An
// empty title cancels the flow; an empty description is allowed and simply
// saves without one.
func (m *Model) advanceAdd() {
	if m.addPhase == addPhaseTitle {
		title := strings.TrimSpace(m.input.Value())
		if title == "" {
			m.cancelAdd()
			return
		}
		m.pendingTitle = title
		m.addPhase = addPhaseDescription
		if m.editMode {
			// 编辑:预填当前描述,光标落在末尾。
			if t, ok := m.todoByID(m.editingID); ok {
				m.input.SetValue(t.Description)
			}
			m.input.Placeholder = "编辑任务描述(可选)…"
		} else {
			m.input.SetValue("")
			m.input.Placeholder = "输入任务描述(可选)…"
		}
		m.input.CharLimit = 500
		return
	}
	m.commitAdd()
}

// cancelAdd exits the add/edit flow without saving anything.
func (m *Model) cancelAdd() {
	m.inputMode = false
	m.addPhase = addPhaseTitle
	m.pendingTitle = ""
	m.editMode = false
	m.editingID = ""
	m.input.SetValue("")
	m.input.Placeholder = "输入任务标题…"
	m.input.CharLimit = 200
	m.input.Blur()
}

// commitAdd saves the typed title and description. In edit mode the existing
// todo (matched by editingID) is updated in place, preserving its id, status
// and created_at; otherwise a new 未开始 todo is added and selected.
func (m *Model) commitAdd() {
	title := strings.TrimSpace(m.pendingTitle)
	description := strings.TrimSpace(m.input.Value())
	editMode := m.editMode
	editingID := m.editingID
	m.inputMode = false
	m.addPhase = addPhaseTitle
	m.pendingTitle = ""
	m.editMode = false
	m.editingID = ""
	m.input.SetValue("")
	m.input.Placeholder = "输入任务标题…"
	m.input.CharLimit = 200
	m.input.Blur()
	if title == "" {
		return
	}

	if editMode {
		t, ok := m.todoByID(editingID)
		if !ok {
			m.err = "保存失败：任务不存在"
			return
		}
		t.Title = title
		t.Description = description
		if err := m.store.Update(t); err != nil {
			m.err = "保存失败：" + err.Error()
			return
		}
		m.err = ""
		m.reload()
		return
	}

	if err := m.store.Add(todo.Todo{Title: title, Description: description, Status: todo.StatusNotStarted}); err != nil {
		m.err = "保存失败：" + err.Error()
		return
	}
	m.err = ""
	m.reload()
	// 新任务落在 未开始 栏并选中它。
	m.activeCol = colIndex(todo.StatusNotStarted)
	m.selected = len(m.visible()) - 1
}

// startEdit enters the edit flow for the selected todo (title phase, prefilled
// with the current title). Does nothing when the focused column is empty.
func (m *Model) startEdit() {
	items := m.visible()
	if m.selected < 0 || m.selected >= len(items) {
		return
	}
	t := items[m.selected]
	m.editMode = true
	m.editingID = t.ID
	m.inputMode = true
	m.addPhase = addPhaseTitle
	m.pendingTitle = ""
	m.input.SetValue(t.Title)
	m.input.Placeholder = "编辑任务标题…"
	m.input.CharLimit = 200
	m.input.Focus()
}

// todoByID returns the todo with the given ID from the loaded list.
func (m Model) todoByID(id string) (todo.Todo, bool) {
	for _, t := range m.todos {
		if t.ID == id {
			return t, true
		}
	}
	return todo.Todo{}, false
}

// toggleDone flips the selected todo between 已完成 and 未开始.
func (m *Model) toggleDone() {
	m.updateSelected(func(t *todo.Todo) {
		if t.Status == todo.StatusDone {
			t.Status = todo.StatusNotStarted
		} else {
			t.Status = todo.StatusDone
		}
	})
}

// markInProgress sets the selected todo to 进行中.
func (m *Model) markInProgress() {
	m.updateSelected(func(t *todo.Todo) {
		t.Status = todo.StatusInProgress
	})
}

// updateSelected applies mutate to the selected visible todo and persists it.
func (m *Model) updateSelected(mutate func(*todo.Todo)) {
	items := m.visible()
	if m.selected < 0 || m.selected >= len(items) {
		return
	}
	t := items[m.selected]
	mutate(&t)
	if err := m.store.Update(t); err != nil {
		m.err = "保存失败：" + err.Error()
		return
	}
	m.err = ""
	m.reload()
	m.clampSelection()
}

// moveSelection moves the selection within the active tab, clamped to bounds.
func (m *Model) moveSelection(delta int) {
	if len(m.visible()) == 0 {
		return
	}
	m.selected += delta
	m.clampSelection()
}

// switchCol moves the focus between kanban columns, clamped to the edges, and
// resets the row selection.
func (m *Model) switchCol(delta int) {
	m.activeCol += delta
	if m.activeCol < 0 {
		m.activeCol = 0
	}
	if m.activeCol >= len(Tabs) {
		m.activeCol = len(Tabs) - 1
	}
	m.selected = 0
	m.clampSelection()
}

// clampSelection keeps the selection index inside the visible list.
func (m *Model) clampSelection() {
	n := len(m.visible())
	if n == 0 {
		m.selected = 0
		return
	}
	if m.selected < 0 {
		m.selected = 0
	}
	if m.selected >= n {
		m.selected = n - 1
	}
}

// reload refreshes m.todos from the storage backend.
func (m *Model) reload() {
	items, err := m.store.List()
	if err != nil {
		m.err = "读取失败：" + err.Error()
		return
	}
	m.todos = items
}

// colIndex returns the kanban column index for a status.
func colIndex(s todo.Status) int {
	for i, t := range Tabs {
		if t == s {
			return i
		}
	}
	return 0
}

// fit truncates s with a trailing ellipsis so its display width stays within
// max terminal cells. Terminal widths are counted in display cells so CJK
// titles truncate correctly.
func fit(s string, max int) string {
	if max <= 1 {
		return ""
	}
	width := 0
	for i, r := range s {
		w := runewidth.RuneWidth(r)
		if width+w > max-1 {
			return s[:i] + "…"
		}
		width += w
	}
	return s
}
