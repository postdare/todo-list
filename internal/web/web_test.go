package web

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"todo-list/internal/storage"
	"todo-list/internal/todo"
)

// newTestServer starts an httptest server backed by a temp JSON store.
func newTestServer(t *testing.T) (*httptest.Server, *storage.JSONStorage) {
	t.Helper()
	store, err := storage.NewJSONStorage(filepath.Join(t.TempDir(), "todos.json"))
	if err != nil {
		t.Fatalf("NewJSONStorage: %v", err)
	}
	srv := httptest.NewServer(NewServer(store))
	t.Cleanup(srv.Close)
	return srv, store
}

// request performs an HTTP request and returns the response and its body.
func request(t *testing.T, method, url, body string) (*http.Response, []byte) {
	t.Helper()
	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}
	req, err := http.NewRequest(method, url, reader)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, url, err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	return resp, data
}

func decodeTodo(t *testing.T, data []byte) todo.Todo {
	t.Helper()
	var item todo.Todo
	if err := json.Unmarshal(data, &item); err != nil {
		t.Fatalf("decode todo: %v\n%s", err, data)
	}
	return item
}

func TestIndexServesHTML(t *testing.T) {
	srv, _ := newTestServer(t)

	resp, body := request(t, http.MethodGet, srv.URL+"/", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Errorf("Content-Type = %q, want text/html", ct)
	}
	html := string(body)
	for _, want := range []string{"待办清单", "未开始", "进行中", "已完成", "/api/todos"} {
		if !strings.Contains(html, want) {
			t.Errorf("index.html missing %q", want)
		}
	}
}

func TestListEmptyReturnsJSONArray(t *testing.T) {
	srv, _ := newTestServer(t)

	resp, body := request(t, http.MethodGet, srv.URL+"/api/todos", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if got := strings.TrimSpace(string(body)); got != "[]" {
		t.Errorf("body = %q, want []", got)
	}
}

func TestCreateUpdateDeleteFlow(t *testing.T) {
	srv, store := newTestServer(t)

	// Create: fields are trimmed, status defaults to 未开始.
	resp, body := request(t, http.MethodPost, srv.URL+"/api/todos",
		`{"title":"  写周报  ","description":"  总结本周进展  "}`)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create status = %d, want 201\n%s", resp.StatusCode, body)
	}
	created := decodeTodo(t, body)
	if created.ID == "" {
		t.Error("created ID is empty")
	}
	if created.Title != "写周报" {
		t.Errorf("title = %q, want 写周报", created.Title)
	}
	if created.Description != "总结本周进展" {
		t.Errorf("description = %q, want 总结本周进展", created.Description)
	}
	if created.Status != todo.StatusNotStarted {
		t.Errorf("status = %q, want 未开始", created.Status)
	}
	if created.CreatedAt.IsZero() {
		t.Error("created_at is zero")
	}

	// List sees the new todo.
	resp, body = request(t, http.MethodGet, srv.URL+"/api/todos", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list status = %d, want 200", resp.StatusCode)
	}
	var items []todo.Todo
	if err := json.Unmarshal(body, &items); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if len(items) != 1 || items[0].ID != created.ID {
		t.Fatalf("list = %v, want the created todo", items)
	}

	// Partial update: only status changes; other fields are preserved.
	resp, body = request(t, http.MethodPut, srv.URL+"/api/todos/"+created.ID,
		`{"status":"进行中"}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("update status = %d, want 200\n%s", resp.StatusCode, body)
	}
	updated := decodeTodo(t, body)
	if updated.Status != todo.StatusInProgress {
		t.Errorf("updated status = %q, want 进行中", updated.Status)
	}
	if updated.Title != created.Title || updated.Description != created.Description {
		t.Errorf("partial update changed other fields: %+v", updated)
	}
	if !updated.CreatedAt.Equal(created.CreatedAt) {
		t.Errorf("created_at changed: %v → %v", created.CreatedAt, updated.CreatedAt)
	}

	// Partial update: title and description.
	resp, body = request(t, http.MethodPut, srv.URL+"/api/todos/"+created.ID,
		`{"title":"写月报","description":"整理数据"}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("update title status = %d, want 200\n%s", resp.StatusCode, body)
	}
	updated = decodeTodo(t, body)
	if updated.Title != "写月报" || updated.Description != "整理数据" {
		t.Errorf("updated = %+v, want 写月报 / 整理数据", updated)
	}
	if updated.Status != todo.StatusInProgress {
		t.Errorf("status changed by title update: %q", updated.Status)
	}

	// The same data is visible through the storage backed by the JSON file.
	persisted, err := store.List()
	if err != nil {
		t.Fatalf("store.List: %v", err)
	}
	if len(persisted) != 1 || persisted[0].Title != "写月报" || persisted[0].Description != "整理数据" {
		t.Fatalf("persisted = %+v, want updated todo", persisted)
	}

	// Delete removes it.
	resp, body = request(t, http.MethodDelete, srv.URL+"/api/todos/"+created.ID, "")
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("delete status = %d, want 204\n%s", resp.StatusCode, body)
	}
	persisted, err = store.List()
	if err != nil {
		t.Fatalf("store.List after delete: %v", err)
	}
	if len(persisted) != 0 {
		t.Fatalf("persisted = %+v, want empty", persisted)
	}
}

func TestCreateRejectsEmptyTitle(t *testing.T) {
	srv, _ := newTestServer(t)

	resp, body := request(t, http.MethodPost, srv.URL+"/api/todos", `{"title":"   "}`)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400\n%s", resp.StatusCode, body)
	}
	if !strings.Contains(string(body), "标题不能为空") {
		t.Errorf("body = %s, want 标题不能为空", body)
	}
}

func TestCreateRejectsMalformedJSON(t *testing.T) {
	srv, _ := newTestServer(t)

	resp, _ := request(t, http.MethodPost, srv.URL+"/api/todos", `{"title":`)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
}

func TestUpdateRejectsInvalidStatus(t *testing.T) {
	srv, store := newTestServer(t)
	now := time.Now().UTC()
	if err := store.Add(todo.Todo{ID: "todo-1", Title: "Task", Status: todo.StatusNotStarted, CreatedAt: now}); err != nil {
		t.Fatalf("Add: %v", err)
	}

	resp, body := request(t, http.MethodPut, srv.URL+"/api/todos/todo-1", `{"status":"不存在"}`)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400\n%s", resp.StatusCode, body)
	}
}

func TestUpdateAndDeleteMissingTodo(t *testing.T) {
	srv, _ := newTestServer(t)

	resp, _ := request(t, http.MethodPut, srv.URL+"/api/todos/missing", `{"status":"已完成"}`)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("update status = %d, want 404", resp.StatusCode)
	}

	resp, _ = request(t, http.MethodDelete, srv.URL+"/api/todos/missing", "")
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("delete status = %d, want 404", resp.StatusCode)
	}
}

func TestUnsupportedMethodReturns405(t *testing.T) {
	srv, _ := newTestServer(t)

	resp, _ := request(t, http.MethodDelete, srv.URL+"/api/todos", "")
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", resp.StatusCode)
	}
}

func decodePanel(t *testing.T, data []byte) todo.Panel {
	t.Helper()
	var panel todo.Panel
	if err := json.Unmarshal(data, &panel); err != nil {
		t.Fatalf("decode panel: %v\n%s", err, data)
	}
	return panel
}

func TestIndexMentionsPanelAPI(t *testing.T) {
	srv, _ := newTestServer(t)

	resp, body := request(t, http.MethodGet, srv.URL+"/", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	html := string(body)
	for _, want := range []string{"/api/panels", "新增分区", "未分组", "panel_id"} {
		if !strings.Contains(html, want) {
			t.Errorf("index.html missing %q", want)
		}
	}
}

func TestPanelsEmptyReturnsJSONArray(t *testing.T) {
	srv, _ := newTestServer(t)

	resp, body := request(t, http.MethodGet, srv.URL+"/api/panels", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if got := strings.TrimSpace(string(body)); got != "[]" {
		t.Errorf("body = %q, want []", got)
	}
}

func TestPanelCreateRenameDeleteFlow(t *testing.T) {
	srv, store := newTestServer(t)

	// Create: name is trimmed and an ID is generated.
	resp, body := request(t, http.MethodPost, srv.URL+"/api/panels", `{"name":"  工作  "}`)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create status = %d, want 201\n%s", resp.StatusCode, body)
	}
	created := decodePanel(t, body)
	if created.ID == "" {
		t.Error("created ID is empty")
	}
	if created.Name != "工作" {
		t.Errorf("name = %q, want 工作", created.Name)
	}
	if created.CreatedAt.IsZero() {
		t.Error("created_at is zero")
	}

	// List sees the new panel.
	resp, body = request(t, http.MethodGet, srv.URL+"/api/panels", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list status = %d, want 200", resp.StatusCode)
	}
	var panels []todo.Panel
	if err := json.Unmarshal(body, &panels); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if len(panels) != 1 || panels[0].ID != created.ID || panels[0].Name != "工作" {
		t.Fatalf("panels = %+v, want the created panel", panels)
	}

	// Rename keeps the ID and creation time.
	resp, body = request(t, http.MethodPut, srv.URL+"/api/panels/"+created.ID, `{"name":"工作（新）"}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("rename status = %d, want 200\n%s", resp.StatusCode, body)
	}
	renamed := decodePanel(t, body)
	if renamed.ID != created.ID || renamed.Name != "工作（新）" {
		t.Errorf("renamed = %+v, want same ID with new name", renamed)
	}
	if !renamed.CreatedAt.Equal(created.CreatedAt) {
		t.Errorf("created_at changed: %v → %v", created.CreatedAt, renamed.CreatedAt)
	}

	// The change is persisted in the backing store too.
	persisted, err := store.ListPanels()
	if err != nil {
		t.Fatalf("store.ListPanels: %v", err)
	}
	if len(persisted) != 1 || persisted[0].Name != "工作（新）" {
		t.Fatalf("persisted = %+v, want renamed panel", persisted)
	}

	// Delete removes it.
	resp, body = request(t, http.MethodDelete, srv.URL+"/api/panels/"+created.ID, "")
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("delete status = %d, want 204\n%s", resp.StatusCode, body)
	}
	persisted, _ = store.ListPanels()
	if len(persisted) != 0 {
		t.Fatalf("persisted = %+v, want empty", persisted)
	}
}

func TestPanelRejectsEmptyOrMalformedRequest(t *testing.T) {
	srv, _ := newTestServer(t)

	resp, body := request(t, http.MethodPost, srv.URL+"/api/panels", `{"name":"   "}`)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400\n%s", resp.StatusCode, body)
	}
	if !strings.Contains(string(body), "分区名称不能为空") {
		t.Errorf("body = %s, want 分区名称不能为空", body)
	}

	resp, _ = request(t, http.MethodPost, srv.URL+"/api/panels", `{"name":`)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("malformed status = %d, want 400", resp.StatusCode)
	}

	resp, body = request(t, http.MethodPost, srv.URL+"/api/panels", `{"name":"工作"}`)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create status = %d, want 201\n%s", resp.StatusCode, body)
	}
	panel := decodePanel(t, body)

	resp, body = request(t, http.MethodPut, srv.URL+"/api/panels/"+panel.ID, `{"name":"  "}`)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("rename empty status = %d, want 400\n%s", resp.StatusCode, body)
	}
}

func TestPanelMissingReturns404(t *testing.T) {
	srv, _ := newTestServer(t)

	resp, _ := request(t, http.MethodPut, srv.URL+"/api/panels/missing", `{"name":"工作"}`)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("update status = %d, want 404", resp.StatusCode)
	}

	resp, _ = request(t, http.MethodDelete, srv.URL+"/api/panels/missing", "")
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("delete status = %d, want 404", resp.StatusCode)
	}
}

func TestAssignTodoToPanelAndDeletePanelUnassigns(t *testing.T) {
	srv, store := newTestServer(t)

	_, body := request(t, http.MethodPost, srv.URL+"/api/todos", `{"title":"写周报"}`)
	created := decodeTodo(t, body)

	_, body = request(t, http.MethodPost, srv.URL+"/api/panels", `{"name":"工作"}`)
	panel := decodePanel(t, body)

	// Assign the todo to the panel.
	resp, body := request(t, http.MethodPut, srv.URL+"/api/todos/"+created.ID,
		`{"panel_id":"`+panel.ID+`"}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("assign status = %d, want 200\n%s", resp.StatusCode, body)
	}
	assigned := decodeTodo(t, body)
	if assigned.PanelID != panel.ID {
		t.Errorf("panel_id = %q, want %q", assigned.PanelID, panel.ID)
	}
	if assigned.Title != "写周报" {
		t.Errorf("assign changed other fields: %+v", assigned)
	}

	items, _ := store.List()
	if len(items) != 1 || items[0].PanelID != panel.ID {
		t.Fatalf("persisted = %+v, want panel_id %q", items, panel.ID)
	}

	// Unknown panels are rejected and do not alter the todo.
	resp, body = request(t, http.MethodPut, srv.URL+"/api/todos/"+created.ID,
		`{"panel_id":"panel-missing"}`)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("invalid panel status = %d, want 400\n%s", resp.StatusCode, body)
	}
	if !strings.Contains(string(body), "分区不存在") {
		t.Errorf("body = %s, want 分区不存在", body)
	}
	items, _ = store.List()
	if items[0].PanelID != panel.ID {
		t.Errorf("invalid assign changed panel_id to %q", items[0].PanelID)
	}

	// Deleting the panel moves the todo back to the ungrouped status columns.
	resp, body = request(t, http.MethodDelete, srv.URL+"/api/panels/"+panel.ID, "")
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("delete panel status = %d, want 204\n%s", resp.StatusCode, body)
	}
	items, _ = store.List()
	if len(items) != 1 || items[0].PanelID != "" {
		t.Fatalf("todo after deleting panel = %+v, want empty panel_id", items)
	}

	// An empty panel_id also moves a todo back explicitly.
	_, body = request(t, http.MethodPost, srv.URL+"/api/panels", `{"name":"生活"}`)
	panel2 := decodePanel(t, body)
	request(t, http.MethodPut, srv.URL+"/api/todos/"+created.ID, `{"panel_id":"`+panel2.ID+`"}`)
	resp, body = request(t, http.MethodPut, srv.URL+"/api/todos/"+created.ID, `{"panel_id":""}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("clear panel status = %d, want 200\n%s", resp.StatusCode, body)
	}
	if cleared := decodeTodo(t, body); cleared.PanelID != "" {
		t.Errorf("panel_id = %q, want empty", cleared.PanelID)
	}
}
