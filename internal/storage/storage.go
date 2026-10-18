// Package storage defines the persistence interface and implementations for todos.
package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/adrg/xdg"
	"todo-list/internal/todo"
)

// ErrNotFound is returned when an operation references a todo that does not
// exist in storage.
var ErrNotFound = errors.New("todo not found")

// ErrPanelNotFound is returned when an operation references a panel that does
// not exist in storage.
var ErrPanelNotFound = errors.New("panel not found")

// Storage abstracts persistence so future providers can be swapped in without
// changing consumers.
type Storage interface {
	Add(t todo.Todo) error
	Update(t todo.Todo) error
	Delete(id string) error
	List() ([]todo.Todo, error)

	AddPanel(p todo.Panel) error
	UpdatePanel(p todo.Panel) error
	DeletePanel(id string) error
	ListPanels() ([]todo.Panel, error)
}

// JSONStorage persists todos as an indented JSON array in a local file.
// User-defined panels live in a sibling panels.json file so that todos.json
// keeps its historic top-level array format and old files stay readable.
type JSONStorage struct {
	mu          sync.RWMutex
	path        string
	panelsPath  string
	todos       []todo.Todo
	panels      []todo.Panel
	nextID      int
	nextPanelID int
}

// NewJSONStorage opens or creates the JSON-backed store at path. Panels are
// stored in panels.json in the same directory.
func NewJSONStorage(path string) (*JSONStorage, error) {
	s := &JSONStorage{
		path:        path,
		panelsPath:  filepath.Join(filepath.Dir(path), "panels.json"),
		nextID:      1,
		nextPanelID: 1,
	}
	if err := s.load(); err != nil {
		return nil, err
	}
	if err := s.loadPanels(); err != nil {
		return nil, err
	}
	return s, nil
}

// NewXDGStorage returns a JSON-backed store located in the user's XDG config
// directory at todo-list/todos.json.
func NewXDGStorage() (*JSONStorage, error) {
	dir := filepath.Join(xdg.ConfigHome, "todo-list")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create config dir: %w", err)
	}
	return NewJSONStorage(filepath.Join(dir, "todos.json"))
}

// Add inserts a new todo. Empty fields are normalized: an empty ID gets a
// generated sequential ID, an empty status becomes 未开始, and a zero time
// becomes the current UTC time.
func (s *JSONStorage) Add(t todo.Todo) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if t.ID == "" {
		t.ID = s.nextIDString()
		s.nextID++
	} else if id, ok := parseID(t.ID); ok && id >= s.nextID {
		s.nextID = id + 1
	}
	if t.Status == "" {
		t.Status = todo.StatusNotStarted
	}
	if t.CreatedAt.IsZero() {
		t.CreatedAt = time.Now().UTC()
	}

	s.todos = append(s.todos, t)
	return s.save()
}

// Update replaces an existing todo with the provided value.
func (s *JSONStorage) Update(t todo.Todo) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, existing := range s.todos {
		if existing.ID == t.ID {
			s.todos[i] = t
			return s.save()
		}
	}
	return ErrNotFound
}

// Delete removes the todo with the given ID.
func (s *JSONStorage) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, t := range s.todos {
		if t.ID == id {
			s.todos = append(s.todos[:i], s.todos[i+1:]...)
			return s.save()
		}
	}
	return ErrNotFound
}

// List returns a copy of all stored todos.
func (s *JSONStorage) List() ([]todo.Todo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]todo.Todo, len(s.todos))
	copy(result, s.todos)
	return result, nil
}

// AddPanel inserts a new panel. An empty ID gets a generated sequential ID and
// a zero CreatedAt becomes the current UTC time.
func (s *JSONStorage) AddPanel(p todo.Panel) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if p.ID == "" {
		p.ID = s.nextPanelIDString()
		s.nextPanelID++
	} else if id, ok := parsePanelID(p.ID); ok && id >= s.nextPanelID {
		s.nextPanelID = id + 1
	}
	if p.CreatedAt.IsZero() {
		p.CreatedAt = time.Now().UTC()
	}

	s.panels = append(s.panels, p)
	return s.savePanels()
}

// UpdatePanel replaces an existing panel with the provided value.
func (s *JSONStorage) UpdatePanel(p todo.Panel) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, existing := range s.panels {
		if existing.ID == p.ID {
			s.panels[i] = p
			return s.savePanels()
		}
	}
	return ErrPanelNotFound
}

// DeletePanel removes the panel and clears PanelID on every todo that pointed
// at it, so no todo is left referencing a panel that no longer exists.
func (s *JSONStorage) DeletePanel(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	idx := -1
	for i, p := range s.panels {
		if p.ID == id {
			idx = i
			break
		}
	}
	if idx == -1 {
		return ErrPanelNotFound
	}

	s.panels = append(s.panels[:idx], s.panels[idx+1:]...)

	unassigned := false
	for i, t := range s.todos {
		if t.PanelID == id {
			s.todos[i].PanelID = ""
			unassigned = true
		}
	}
	if unassigned {
		if err := s.save(); err != nil {
			return err
		}
	}
	return s.savePanels()
}

// ListPanels returns a copy of all stored panels.
func (s *JSONStorage) ListPanels() ([]todo.Panel, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]todo.Panel, len(s.panels))
	copy(result, s.panels)
	return result, nil
}

// Path returns the underlying JSON file path.
func (s *JSONStorage) Path() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.path
}

func (s *JSONStorage) load() error {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			s.todos = []todo.Todo{}
			return nil
		}
		return fmt.Errorf("read storage file: %w", err)
	}

	if len(data) == 0 {
		s.todos = []todo.Todo{}
		return nil
	}

	if err := json.Unmarshal(data, &s.todos); err != nil {
		return fmt.Errorf("parse storage file: %w", err)
	}

	s.nextID = 1
	for _, t := range s.todos {
		if id, ok := parseID(t.ID); ok && id >= s.nextID {
			s.nextID = id + 1
		}
	}
	return nil
}

func (s *JSONStorage) save() error {
	data, err := json.MarshalIndent(s.todos, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal todos: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0755); err != nil {
		return fmt.Errorf("create storage dir: %w", err)
	}
	if err := os.WriteFile(s.path, data, 0644); err != nil {
		return fmt.Errorf("write storage file: %w", err)
	}
	return nil
}

func (s *JSONStorage) loadPanels() error {
	data, err := os.ReadFile(s.panelsPath)
	if err != nil {
		if os.IsNotExist(err) {
			s.panels = []todo.Panel{}
			return nil
		}
		return fmt.Errorf("read panels file: %w", err)
	}

	if len(data) == 0 {
		s.panels = []todo.Panel{}
		return nil
	}

	if err := json.Unmarshal(data, &s.panels); err != nil {
		return fmt.Errorf("parse panels file: %w", err)
	}

	s.nextPanelID = 1
	for _, p := range s.panels {
		if id, ok := parsePanelID(p.ID); ok && id >= s.nextPanelID {
			s.nextPanelID = id + 1
		}
	}
	return nil
}

func (s *JSONStorage) savePanels() error {
	data, err := json.MarshalIndent(s.panels, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal panels: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(s.panelsPath), 0755); err != nil {
		return fmt.Errorf("create storage dir: %w", err)
	}
	if err := os.WriteFile(s.panelsPath, data, 0644); err != nil {
		return fmt.Errorf("write panels file: %w", err)
	}
	return nil
}

func (s *JSONStorage) nextIDString() string {
	return fmt.Sprintf("todo-%d", s.nextID)
}

func (s *JSONStorage) nextPanelIDString() string {
	return fmt.Sprintf("panel-%d", s.nextPanelID)
}

func parseID(id string) (int, bool) {
	var n int
	_, err := fmt.Sscanf(id, "todo-%d", &n)
	return n, err == nil
}

func parsePanelID(id string) (int, bool) {
	var n int
	_, err := fmt.Sscanf(id, "panel-%d", &n)
	return n, err == nil
}
