package storage

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/adrg/xdg"
	"todo-list/internal/todo"
)

func TestJSONStorageImplementsStorageInterface(t *testing.T) {
	var _ Storage = (*JSONStorage)(nil)
}

func TestJSONStorageAddAndList(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "todos.json")

	store, err := NewJSONStorage(path)
	if err != nil {
		t.Fatalf("NewJSONStorage: %v", err)
	}

	created := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	if err := store.Add(todo.Todo{Title: "Task 1", Description: "描述一", CreatedAt: created}); err != nil {
		t.Fatalf("Add first: %v", err)
	}
	if err := store.Add(todo.Todo{Title: "Task 2", CreatedAt: created}); err != nil {
		t.Fatalf("Add second: %v", err)
	}

	items, err := store.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("len(items) = %d, want 2", len(items))
	}
	if items[0].Title != "Task 1" {
		t.Errorf("items[0].Title = %q, want Task 1", items[0].Title)
	}
	if items[0].Description != "描述一" {
		t.Errorf("items[0].Description = %q, want 描述一", items[0].Description)
	}
	if items[0].Status != todo.StatusNotStarted {
		t.Errorf("items[0].Status = %q, want %q", items[0].Status, todo.StatusNotStarted)
	}
	if !items[0].CreatedAt.Equal(created) {
		t.Errorf("items[0].CreatedAt = %v, want %v", items[0].CreatedAt, created)
	}
	if items[1].Title != "Task 2" {
		t.Errorf("items[1].Title = %q, want Task 2", items[1].Title)
	}
	if items[1].Description != "" {
		t.Errorf("items[1].Description = %q, want empty", items[1].Description)
	}
}

func TestJSONStoragePersistence(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "todos.json")

	store, err := NewJSONStorage(path)
	if err != nil {
		t.Fatalf("NewJSONStorage first: %v", err)
	}

	created := time.Date(2025, 6, 15, 10, 0, 0, 0, time.UTC)
	item := todo.Todo{
		ID:          "todo-1",
		Title:       "Persisted task",
		Description: "持久化描述",
		Status:      todo.StatusNotStarted,
		CreatedAt:   created,
	}
	if err := store.Add(item); err != nil {
		t.Fatalf("Add: %v", err)
	}

	store2, err := NewJSONStorage(path)
	if err != nil {
		t.Fatalf("NewJSONStorage second: %v", err)
	}

	items, err := store2.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(items))
	}
	if items[0].ID != item.ID {
		t.Errorf("ID = %q, want %q", items[0].ID, item.ID)
	}
	if items[0].Title != item.Title {
		t.Errorf("Title = %q, want %q", items[0].Title, item.Title)
	}
	if items[0].Description != item.Description {
		t.Errorf("Description = %q, want %q", items[0].Description, item.Description)
	}
	if items[0].Status != item.Status {
		t.Errorf("Status = %q, want %q", items[0].Status, item.Status)
	}
	if !items[0].CreatedAt.Equal(item.CreatedAt) {
		t.Errorf("CreatedAt = %v, want %v", items[0].CreatedAt, item.CreatedAt)
	}
}

func TestJSONStorageHumanReadable(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "todos.json")

	store, err := NewJSONStorage(path)
	if err != nil {
		t.Fatalf("NewJSONStorage: %v", err)
	}

	created := time.Date(2025, 6, 15, 10, 0, 0, 0, time.UTC)
	if err := store.Add(todo.Todo{
		ID:          "todo-1",
		Title:       "Readable task",
		Description: "人可读的描述",
		Status:      todo.StatusNotStarted,
		CreatedAt:   created,
	}); err != nil {
		t.Fatalf("Add: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	var raw []map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("parse JSON: %v", err)
	}
	if len(raw) != 1 {
		t.Fatalf("len(raw) = %d, want 1", len(raw))
	}
	if raw[0]["id"] != "todo-1" {
		t.Errorf("id = %v, want todo-1", raw[0]["id"])
	}
	if raw[0]["title"] != "Readable task" {
		t.Errorf("title = %v, want Readable task", raw[0]["title"])
	}
	if raw[0]["description"] != "人可读的描述" {
		t.Errorf("description = %v, want 人可读的描述", raw[0]["description"])
	}
	if raw[0]["status"] != "未开始" {
		t.Errorf("status = %v, want 未开始", raw[0]["status"])
	}
	if raw[0]["created_at"] != "2025-06-15T10:00:00Z" {
		t.Errorf("created_at = %v, want 2025-06-15T10:00:00Z", raw[0]["created_at"])
	}

	if !strings.Contains(string(data), "\n") {
		t.Errorf("JSON file is not indented / human-readable")
	}
}

func TestJSONStorageUpdate(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "todos.json")
	store, err := NewJSONStorage(path)
	if err != nil {
		t.Fatalf("NewJSONStorage: %v", err)
	}

	item := todo.Todo{
		ID:        "todo-1",
		Title:     "Old title",
		Status:    todo.StatusNotStarted,
		CreatedAt: time.Now(),
	}
	if err := store.Add(item); err != nil {
		t.Fatalf("Add: %v", err)
	}

	updated := todo.Todo{
		ID:        "todo-1",
		Title:     "New title",
		Status:    todo.StatusDone,
		CreatedAt: item.CreatedAt,
	}
	if err := store.Update(updated); err != nil {
		t.Fatalf("Update: %v", err)
	}

	items, err := store.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(items))
	}
	if items[0].Title != "New title" {
		t.Errorf("Title = %q, want New title", items[0].Title)
	}
	if items[0].Status != todo.StatusDone {
		t.Errorf("Status = %q, want %q", items[0].Status, todo.StatusDone)
	}

	err = store.Update(todo.Todo{ID: "missing"})
	if err != ErrNotFound {
		t.Errorf("Update missing = %v, want ErrNotFound", err)
	}
}

func TestJSONStorageDelete(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "todos.json")
	store, err := NewJSONStorage(path)
	if err != nil {
		t.Fatalf("NewJSONStorage: %v", err)
	}

	if err := store.Add(todo.Todo{ID: "todo-1", Title: "Task 1", CreatedAt: time.Now()}); err != nil {
		t.Fatalf("Add first: %v", err)
	}
	if err := store.Add(todo.Todo{ID: "todo-2", Title: "Task 2", CreatedAt: time.Now()}); err != nil {
		t.Fatalf("Add second: %v", err)
	}

	if err := store.Delete("todo-1"); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	items, err := store.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(items))
	}
	if items[0].ID != "todo-2" {
		t.Errorf("ID = %q, want todo-2", items[0].ID)
	}

	err = store.Delete("missing")
	if err != ErrNotFound {
		t.Errorf("Delete missing = %v, want ErrNotFound", err)
	}
}

func TestJSONStorageListReturnsCopy(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "todos.json")
	store, err := NewJSONStorage(path)
	if err != nil {
		t.Fatalf("NewJSONStorage: %v", err)
	}

	if err := store.Add(todo.Todo{ID: "todo-1", Title: "Task 1", CreatedAt: time.Now()}); err != nil {
		t.Fatalf("Add: %v", err)
	}

	items, err := store.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(items))
	}

	// Mutating the slice returned by List must not affect the store.
	items[0].Title = "Mutated"
	items = append(items, todo.Todo{ID: "todo-2", Title: "Task 2", CreatedAt: time.Now()})

	reloaded, err := store.List()
	if err != nil {
		t.Fatalf("List after mutation: %v", err)
	}
	if len(reloaded) != 1 {
		t.Errorf("len(reloaded) = %d, want 1", len(reloaded))
	}
	if reloaded[0].Title != "Task 1" {
		t.Errorf("Title = %q, want Task 1", reloaded[0].Title)
	}
}

func TestJSONStorageEmptyFileAndJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "todos.json")

	// An empty JSON file should load as an empty store.
	if err := os.WriteFile(path, []byte("[]"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	store, err := NewJSONStorage(path)
	if err != nil {
		t.Fatalf("NewJSONStorage: %v", err)
	}

	items, err := store.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(items) != 0 {
		t.Errorf("len(items) = %d, want 0", len(items))
	}

	// After adding an item, the file must be a readable JSON array.
	if err := store.Add(todo.Todo{ID: "todo-1", Title: "Only task", CreatedAt: time.Now()}); err != nil {
		t.Fatalf("Add: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	var raw []map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("parse JSON: %v", err)
	}
	if len(raw) != 1 {
		t.Errorf("len(raw) = %d, want 1", len(raw))
	}
}

func TestJSONStorageLoadsLegacyFileWithoutDescription(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "todos.json")

	// 旧版本持久化文件:记录里没有 description 键,必须能直接加载且描述为空。
	legacy := `[
  {
    "id": "todo-1",
    "title": "Legacy task",
    "status": "进行中",
    "created_at": "2024-01-01T00:00:00Z"
  }
]`
	if err := os.WriteFile(path, []byte(legacy), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	store, err := NewJSONStorage(path)
	if err != nil {
		t.Fatalf("NewJSONStorage: %v", err)
	}

	items, err := store.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(items))
	}
	if items[0].ID != "todo-1" {
		t.Errorf("ID = %q, want todo-1", items[0].ID)
	}
	if items[0].Title != "Legacy task" {
		t.Errorf("Title = %q, want Legacy task", items[0].Title)
	}
	if items[0].Description != "" {
		t.Errorf("Description = %q, want empty (legacy file has no description key)", items[0].Description)
	}
	if items[0].Status != todo.StatusInProgress {
		t.Errorf("Status = %q, want 进行中", items[0].Status)
	}
}

func TestJSONStorageAddNormalizesEmptyFields(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "todos.json")
	store, err := NewJSONStorage(path)
	if err != nil {
		t.Fatalf("NewJSONStorage: %v", err)
	}

	// Add a todo with no ID, status, or CreatedAt.
	if err := store.Add(todo.Todo{Title: "Auto fields"}); err != nil {
		t.Fatalf("Add: %v", err)
	}

	items, err := store.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(items))
	}
	if items[0].ID == "" {
		t.Errorf("ID was not generated")
	}
	if items[0].Status != todo.StatusNotStarted {
		t.Errorf("Status = %q, want %q", items[0].Status, todo.StatusNotStarted)
	}
	if items[0].CreatedAt.IsZero() {
		t.Errorf("CreatedAt was not set")
	}

	// The generated IDs should be sequential.
	if err := store.Add(todo.Todo{Title: "Auto fields 2"}); err != nil {
		t.Fatalf("Add second: %v", err)
	}
	items, _ = store.List()
	if len(items) != 2 {
		t.Fatalf("len(items) = %d, want 2", len(items))
	}
	id1, ok1 := parseID(items[0].ID)
	id2, ok2 := parseID(items[1].ID)
	if !ok1 || !ok2 || id2 != id1+1 {
		t.Errorf("IDs not sequential: %q, %q", items[0].ID, items[1].ID)
	}
}

func TestNewXDGStorage(t *testing.T) {
	orig := os.Getenv("XDG_CONFIG_HOME")
	defer func() {
		if orig == "" {
			os.Unsetenv("XDG_CONFIG_HOME")
		} else {
			os.Setenv("XDG_CONFIG_HOME", orig)
		}
		xdg.Reload()
	}()

	tmp := t.TempDir()
	os.Setenv("XDG_CONFIG_HOME", tmp)
	xdg.Reload()

	store, err := NewXDGStorage()
	if err != nil {
		t.Fatalf("NewXDGStorage: %v", err)
	}

	expectedPath := filepath.Join(tmp, "todo-list", "todos.json")
	if store.Path() != expectedPath {
		t.Errorf("Path = %q, want %q", store.Path(), expectedPath)
	}

	if _, err := os.Stat(filepath.Join(tmp, "todo-list")); err != nil {
		t.Errorf("config dir not created: %v", err)
	}

	if err := store.Add(todo.Todo{Title: "XDG task", CreatedAt: time.Now()}); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if _, err := os.Stat(expectedPath); err != nil {
		t.Errorf("storage file not created: %v", err)
	}
}
