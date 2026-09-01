package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/creack/pty"
)

// testBin is the path of the CLI binary built once in TestMain.
// moduleRoot is the module root directory (go test runs with cwd = package dir).
var (
	testBin    string
	moduleRoot string
)

func TestMain(m *testing.M) {
	out, err := exec.Command("go", "env", "GOMOD").Output()
	if err != nil {
		fmt.Fprintln(os.Stderr, "go env GOMOD:", err)
		os.Exit(1)
	}
	moduleRoot = filepath.Dir(strings.TrimSpace(string(out)))

	dir, err := os.MkdirTemp("", "todo-list-bin-*")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	testBin = filepath.Join(dir, "todo-list")
	cmd := exec.Command("go", "build", "-o", testBin, "./cmd/todo-list")
	cmd.Dir = moduleRoot
	if out, err := cmd.CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "go build: %v\n%s", err, out)
		os.Exit(1)
	}
	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}

// ptyReader drains the pty master in the background into a thread-safe buffer
// so expects can poll without blocking the test goroutine. It also answers the
// startup terminal queries (OSC 11 background color, DA cursor position) that
// bubbletea/termenv send; without answers the app's profile-detection reader
// keeps consuming input until it times out, swallowing the keys we send.
type ptyReader struct {
	mu      sync.Mutex
	buf     bytes.Buffer
	queries bool // both startup queries have been answered
}

func startReader(f *os.File) *ptyReader {
	r := &ptyReader{}
	go func() {
		var pending []byte
		chunk := make([]byte, 4096)
		for {
			n, err := f.Read(chunk)
			if n > 0 {
				data := chunk[:n]
				r.mu.Lock()
				r.buf.Write(data)
				r.mu.Unlock()

				pending = append(pending, data...)
				if r.answerQueries(f, &pending) {
					r.mu.Lock()
					r.queries = true
					r.mu.Unlock()
				}
			}
			if err != nil {
				return // EIO once the child exits or the master is closed
			}
		}
	}()
	return r
}

// answerQueries writes the responses a real terminal would, scanning across
// chunk boundaries. It runs at most once: after answering it clears the
// pending window, and sawQueries() prevents any re-answer (re-sending the
// response would leak the bytes into the app's input stream).
func (r *ptyReader) answerQueries(master *os.File, pending *[]byte) bool {
	if r.sawQueries() {
		return true
	}
	const (
		oscQuery   = "\x1b]11;?"
		oscTermST  = "\x1b\\"
		oscTermBEL = "\x07"
		daQuery    = "\x1b[6n"
	)
	all := string(*pending)
	answered := false
	if strings.Contains(all, oscQuery) && (strings.Contains(all, oscTermST) || strings.Contains(all, oscTermBEL)) {
		master.WriteString("\x1b]11;rgb:1010/1010/1010\x1b\\")
		answered = true
	}
	if strings.Contains(all, daQuery) {
		master.WriteString("\x1b[1;1R")
		answered = true
	}
	if answered {
		*pending = nil
	}
	return answered
}

func (r *ptyReader) sawQueries() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.queries
}

func (r *ptyReader) String() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.buf.String()
}

// Len returns the number of bytes read so far (used as a marker).
func (r *ptyReader) Len() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.buf.Len()
}

// After returns the accumulated output starting at byte offset n.
func (r *ptyReader) After(n int) string {
	r.mu.Lock()
	defer r.mu.Unlock()
	if n >= r.buf.Len() {
		return ""
	}
	return r.buf.String()[n:]
}

// startApp launches the binary in a fresh PTY with an isolated XDG config
// home and returns the PTY master, an output reader, the config home, and a
// wait func that returns the process exit code.
func startApp(t *testing.T) (master *os.File, reader *ptyReader, configHome string, wait func() int) {
	t.Helper()
	configHome = t.TempDir()

	cmd := exec.Command(testBin)
	cmd.Env = append(os.Environ(),
		"TERM=xterm-256color",
		"XDG_CONFIG_HOME="+configHome,
	)

	// Start with a 100x30 terminal so the layout has room.
	master, err := pty.StartWithSize(cmd, &pty.Winsize{Rows: 30, Cols: 100})
	if err != nil {
		t.Fatalf("pty start: %v", err)
	}
	reader = startReader(master)
	// Wait until the startup terminal queries have been answered, then give
	// the app's profile-detection reader a moment to consume the responses,
	// so keys we send afterwards are never mistaken for query responses.
	deadline := time.Now().Add(5 * time.Second)
	for !reader.sawQueries() && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	time.Sleep(300 * time.Millisecond)
	t.Cleanup(func() {
		cmd.Process.Kill()
		cmd.Wait()
	})

	wait = func() int {
		err := cmd.Wait()
		if err == nil {
			return 0
		}
		if ee, ok := err.(*exec.ExitError); ok {
			return ee.ExitCode()
		}
		t.Fatalf("wait: %v", err)
		return -1
	}
	return master, reader, configHome, wait
}

// expect polls the pty output until want appears.
func expect(t *testing.T, r *ptyReader, want string) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if strings.Contains(r.String(), want) {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("timeout waiting for %q in pty output; got %q", want, r.String())
}

// expectAfter is expect restricted to output produced after a marker.
func expectAfter(t *testing.T, r *ptyReader, mark int, want string) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if strings.Contains(r.After(mark), want) {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("timeout waiting for %q after marker; got %q", want, r.After(mark))
}

// expectSameLine polls until a single output line contains all of the given
// strings. Because the kanban columns are joined side by side, their headers
// must appear on the same line — this proves the three columns are rendered
// simultaneously on one screen instead of being switched between.
func expectSameLine(t *testing.T, r *ptyReader, wants ...string) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		for _, line := range strings.Split(strings.ReplaceAll(r.String(), "\r\n", "\n"), "\n") {
			ok := true
			for _, w := range wants {
				if !strings.Contains(line, w) {
					ok = false
					break
				}
			}
			if ok {
				return
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("timeout waiting for one line containing all of %v; got %q", wants, r.String())
}

func send(t *testing.T, f *os.File, s string) {
	t.Helper()
	if _, err := f.WriteString(s); err != nil {
		t.Fatalf("write to pty: %v", err)
	}
}

// readTodos reads and parses the todos.json written under configHome.
func readTodos(t *testing.T, configHome string) ([]map[string]any, string) {
	t.Helper()
	path := filepath.Join(configHome, "todo-list", "todos.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var items []map[string]any
	if err := json.Unmarshal(data, &items); err != nil {
		t.Fatalf("parse %s: %v\n%s", path, err, data)
	}
	return items, string(data)
}

func TestBuildAll(t *testing.T) {
	cmd := exec.Command("go", "build", "./...")
	cmd.Dir = moduleRoot
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go build ./... failed: %v\n%s", err, out)
	}
}

// TestBinaryLaunchesTUIAndQuits drives the real binary in a PTY and asserts
// the three kanban columns render side by side on one screen (before any key
// is pressed — no tab switching) and q exits cleanly.
func TestBinaryLaunchesTUIAndQuits(t *testing.T) {
	master, reader, _, wait := startApp(t)
	defer master.Close()

	// 三栏同屏:同一行同时含三个列头,无需任何按键。
	expectSameLine(t, reader, "未开始", "进行中", "已完成")
	expect(t, reader, "暂无任务") // 三栏初始均为空
	expect(t, reader, "q 退出")

	send(t, master, "q")
	if code := wait(); code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
}

// TestAddTodoAndPersist types a new todo through the UI and verifies the JSON
// file contains a human-readable record with all required fields.
func TestAddTodoAndPersist(t *testing.T) {
	master, reader, configHome, wait := startApp(t)
	defer master.Close()

	expect(t, reader, "未开始")
	expect(t, reader, "暂无任务") // empty state before any todo

	send(t, master, "n")
	expect(t, reader, "新任务标题") // input mode rendered — app is caught up

	mark := reader.Len()
	for _, r := range "买牛奶" {
		send(t, master, string(r))
	}
	expectAfter(t, reader, mark, "买牛奶") // echoed inside the input field

	// After commit the input mode must end (list-mode footer reappears) and
	// the new todo must render as the selected row under 未开始. The echo
	// (prompt "▸ " + text) already matched above, so take a fresh marker
	// after the echo but before Enter to prove the row render is new.
	mark = reader.Len()
	send(t, master, "\r")      // Enter: 标题 → 描述阶段
	expect(t, reader, "新任务描述") // description phase rendered
	send(t, master, "\r")      // Enter(空描述):保存
	expectAfter(t, reader, mark, "q 退出")
	expectAfter(t, reader, mark, "▸ 买牛奶")

	send(t, master, "q")
	if code := wait(); code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}

	items, raw := readTodos(t, configHome)
	if len(items) != 1 {
		t.Fatalf("len(todos) = %d, want 1", len(items))
	}
	it := items[0]
	if it["title"] != "买牛奶" {
		t.Errorf("title = %v, want 买牛奶", it["title"])
	}
	if it["status"] != "未开始" {
		t.Errorf("status = %v, want 未开始", it["status"])
	}
	for _, key := range []string{"id", "title", "description", "status", "created_at"} {
		if _, ok := it[key]; !ok {
			t.Errorf("missing key %q in persisted record: %v", key, it)
		}
	}
	if !strings.Contains(raw, "\n") {
		t.Error("todos.json is not human-readable (no newlines)")
	}
}

// TestAddTodoWithDescriptionAndPersist drives the two-step add flow through
// the real TUI — title then description — and verifies the JSON file contains
// the description key with the typed value, and the list view renders it.
func TestAddTodoWithDescriptionAndPersist(t *testing.T) {
	master, reader, configHome, wait := startApp(t)
	defer master.Close()

	expect(t, reader, "未开始")

	// n → 标题阶段:输入标题。
	send(t, master, "n")
	expect(t, reader, "新任务标题")
	mark := reader.Len()
	for _, r := range "写周报" {
		send(t, master, string(r))
	}
	expectAfter(t, reader, mark, "写周报")

	// Enter:标题 → 描述阶段,输入描述。
	send(t, master, "\r")
	expect(t, reader, "新任务描述")
	mark = reader.Len()
	for _, r := range "总结本周进展与下周计划" {
		send(t, master, string(r))
	}
	expectAfter(t, reader, mark, "总结本周进展与下周计划")

	// Enter:保存。列表行渲染标题,次要行渲染描述。
	mark = reader.Len()
	send(t, master, "\r")
	expectAfter(t, reader, mark, "q 退出")
	expectAfter(t, reader, mark, "▸ 写周报")
	expectAfter(t, reader, mark, "总结本周进展与下周计划")

	send(t, master, "q")
	if code := wait(); code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}

	items, raw := readTodos(t, configHome)
	if len(items) != 1 {
		t.Fatalf("len(todos) = %d, want 1", len(items))
	}
	it := items[0]
	if it["title"] != "写周报" {
		t.Errorf("title = %v, want 写周报", it["title"])
	}
	if it["description"] != "总结本周进展与下周计划" {
		t.Errorf("description = %v, want 总结本周进展与下周计划", it["description"])
	}
	if it["status"] != "未开始" {
		t.Errorf("status = %v, want 未开始", it["status"])
	}
	for _, key := range []string{"id", "title", "description", "status", "created_at"} {
		if _, ok := it[key]; !ok {
			t.Errorf("missing key %q in persisted record: %v", key, it)
		}
	}
	if !strings.Contains(raw, "\n") {
		t.Error("todos.json is not human-readable (no newlines)")
	}
}

// TestStatusFlowAndPersistence drives the full kanban happy path — add two
// todos (one with a description), mark one in-progress, complete it, reopen
// it — then checks the final JSON statuses and description keys.
func TestStatusFlowAndPersistence(t *testing.T) {
	master, reader, configHome, wait := startApp(t)
	defer master.Close()

	// 启动即三栏同屏(非切换)。
	expectSameLine(t, reader, "未开始", "进行中", "已完成")

	for _, item := range []struct{ title, desc string }{{"买牛奶", ""}, {"写周报", "整理本周数据"}} {
		send(t, master, "n")
		for _, r := range item.title {
			send(t, master, string(r))
		}
		send(t, master, "\r") // 标题 → 描述阶段
		if item.desc != "" {
			for _, r := range item.desc {
				send(t, master, string(r))
			}
		}
		send(t, master, "\r") // 描述(可空):保存
	}

	// Barrier: the last added item (写周报) is the selected row in 未开始.
	expect(t, reader, "▸ 写周报")

	// i moves 写周报 to 进行中; right focuses that column; space completes it;
	// right focuses 已完成 where the row shows the ✓ prefix and its description.
	mark := reader.Len()
	send(t, master, "i")
	send(t, master, "\x1b[C") // right arrow → 进行中 column
	send(t, master, " ")
	send(t, master, "\x1b[C") // right arrow → 已完成 column
	expectAfter(t, reader, mark, "▸ ✓ 写周报")
	expectAfter(t, reader, mark, "整理本周数据")

	// Space reopens it (已完成 → 未开始); left twice returns to 未开始. The
	// selection resets to the first item there (▸ 买牛奶) and 写周报 is back.
	mark = reader.Len()
	send(t, master, " ")
	send(t, master, "\x1b[D") // left arrow
	send(t, master, "\x1b[D")
	expectAfter(t, reader, mark, "▸ 买牛奶")
	expectAfter(t, reader, mark, "写周报")

	send(t, master, "q")
	if code := wait(); code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}

	items, _ := readTodos(t, configHome)
	byTitle := map[string]string{}
	descByTitle := map[string]string{}
	for _, it := range items {
		title, _ := it["title"].(string)
		status, _ := it["status"].(string)
		desc, _ := it["description"].(string)
		byTitle[title] = status
		descByTitle[title] = desc
		if _, ok := it["description"]; !ok {
			t.Errorf("missing description key in persisted record: %v", it)
		}
	}
	if len(byTitle) != 2 {
		t.Fatalf("expected 2 todos, got %v", byTitle)
	}
	if byTitle["写周报"] != "未开始" {
		t.Errorf("写周报 status = %q, want 未开始 (after reopen)", byTitle["写周报"])
	}
	if byTitle["买牛奶"] != "未开始" {
		t.Errorf("买牛奶 status = %q, want 未开始", byTitle["买牛奶"])
	}
	if descByTitle["写周报"] != "整理本周数据" {
		t.Errorf("写周报 description = %q, want 整理本周数据", descByTitle["写周报"])
	}
	if descByTitle["买牛奶"] != "" {
		t.Errorf("买牛奶 description = %q, want empty", descByTitle["买牛奶"])
	}
}

// TestEditFlowE2E drives the edit flow through the real TUI: add a todo,
// press e, rewrite the prefilled title and description, save with Enter, and
// verify the JSON shows the new values while id/status/created_at are
// preserved.
func TestEditFlowE2E(t *testing.T) {
	master, reader, configHome, wait := startApp(t)
	defer master.Close()

	// 新建任务(两段式:标题 + 描述)。
	send(t, master, "n")
	expect(t, reader, "新任务标题")
	for _, r := range "旧标题" {
		send(t, master, string(r))
	}
	expect(t, reader, "旧标题")
	send(t, master, "\r")
	expect(t, reader, "新任务描述")
	for _, r := range "旧描述" {
		send(t, master, string(r))
	}
	expect(t, reader, "旧描述")
	mark := reader.Len()
	send(t, master, "\r")
	expectAfter(t, reader, mark, "q 退出")
	expectAfter(t, reader, mark, "▸ 旧标题")

	before, _ := readTodos(t, configHome)
	if len(before) != 1 {
		t.Fatalf("len(before) = %d, want 1", len(before))
	}
	orig := before[0]

	// e 进入编辑:标题输入预填 旧标题,清空后输入 新标题。
	send(t, master, "e")
	expect(t, reader, "编辑标题")
	for i := 0; i < len([]rune("旧标题")); i++ {
		send(t, master, "\x7f")
	}
	for _, r := range "新标题" {
		send(t, master, string(r))
	}
	expect(t, reader, "新标题")

	// Enter:标题 → 描述阶段,预填 旧描述,清空后输入 新描述。
	send(t, master, "\r")
	expect(t, reader, "编辑描述")
	for i := 0; i < len([]rune("旧描述")); i++ {
		send(t, master, "\x7f")
	}
	for _, r := range "新描述" {
		send(t, master, string(r))
	}
	expect(t, reader, "新描述")

	// Enter:保存并退出输入模式,列表渲染新标题与新描述。
	mark = reader.Len()
	send(t, master, "\r")
	expectAfter(t, reader, mark, "q 退出")
	expectAfter(t, reader, mark, "▸ 新标题")
	expectAfter(t, reader, mark, "新描述")

	send(t, master, "q")
	if code := wait(); code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}

	after, _ := readTodos(t, configHome)
	if len(after) != 1 {
		t.Fatalf("len(after) = %d, want 1", len(after))
	}
	got := after[0]
	if got["title"] != "新标题" {
		t.Errorf("title = %v, want 新标题", got["title"])
	}
	if got["description"] != "新描述" {
		t.Errorf("description = %v, want 新描述", got["description"])
	}
	if got["id"] != orig["id"] {
		t.Errorf("id changed: %v → %v", orig["id"], got["id"])
	}
	if got["status"] != orig["status"] {
		t.Errorf("status changed: %v → %v", orig["status"], got["status"])
	}
	if got["created_at"] != orig["created_at"] {
		t.Errorf("created_at changed: %v → %v", orig["created_at"], got["created_at"])
	}
}

// TestEditCancelEscE2E verifies that pressing Esc during either phase of the
// edit flow abandons the edit without touching the persisted data.
func TestEditCancelEscE2E(t *testing.T) {
	master, reader, configHome, wait := startApp(t)
	defer master.Close()

	send(t, master, "n")
	expect(t, reader, "新任务标题")
	for _, r := range "旧标题" {
		send(t, master, string(r))
	}
	send(t, master, "\r")
	expect(t, reader, "新任务描述")
	for _, r := range "旧描述" {
		send(t, master, string(r))
	}
	send(t, master, "\r")
	expect(t, reader, "▸ 旧标题")

	// 标题阶段 Esc:输入被修改但取消后不落盘。
	send(t, master, "e")
	expect(t, reader, "编辑标题")
	for _, r := range "改动" {
		send(t, master, string(r))
	}
	send(t, master, "\x1b") // Esc
	expect(t, reader, "q 退出")

	items, _ := readTodos(t, configHome)
	if items[0]["title"] != "旧标题" {
		t.Errorf("title after esc = %v, want 旧标题", items[0]["title"])
	}
	if items[0]["description"] != "旧描述" {
		t.Errorf("description after esc = %v, want 旧描述", items[0]["description"])
	}

	// 描述阶段 Esc:同样不落盘。
	send(t, master, "e")
	expect(t, reader, "编辑标题")
	send(t, master, "\r") // 标题保持原样 → 描述阶段
	expect(t, reader, "编辑描述")
	for _, r := range "改描述" {
		send(t, master, string(r))
	}
	send(t, master, "\x1b") // Esc
	expect(t, reader, "q 退出")

	items, _ = readTodos(t, configHome)
	if items[0]["title"] != "旧标题" {
		t.Errorf("title after esc in description phase = %v, want 旧标题", items[0]["title"])
	}
	if items[0]["description"] != "旧描述" {
		t.Errorf("description after esc in description phase = %v, want 旧描述", items[0]["description"])
	}

	send(t, master, "q")
	if code := wait(); code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
}

// TestLongTitleFullyShownE2E verifies that a 14-character (28 display cells)
// title is fully visible in the list row on a 100-column terminal: the full
// string appears contiguously after saving, and the persisted JSON keeps it.
func TestLongTitleFullyShownE2E(t *testing.T) {
	master, reader, configHome, wait := startApp(t)
	defer master.Close()

	title := "一二三四五六七八九十甲乙丙丁" // 14 汉字 = 28 格

	send(t, master, "n")
	expect(t, reader, "新任务标题")
	for _, r := range title {
		send(t, master, string(r))
	}
	expect(t, reader, title) // 输入框回显完整标题

	// 保存后,列表选中行必须渲染完整的 14 汉字标题(整串连续出现,即未截断)。
	mark := reader.Len()
	send(t, master, "\r")
	send(t, master, "\r")
	expectAfter(t, reader, mark, "q 退出")
	expectAfter(t, reader, mark, "▸ "+title)

	send(t, master, "q")
	if code := wait(); code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}

	items, _ := readTodos(t, configHome)
	if len(items) != 1 {
		t.Fatalf("len(todos) = %d, want 1", len(items))
	}
	if items[0]["title"] != title {
		t.Errorf("persisted title = %v, want full 14-char title", items[0]["title"])
	}
}
