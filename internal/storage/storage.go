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

// Storage abstracts persistence so future providers can be swapped in without
// changing consumers.
type Storage interface {
	Add(t todo.Todo) error
	Update(t todo.Todo) error
	Delete(id string) error
	List() ([]todo.Todo, error)
}

// JSONStorage persists todos as an indented JSON array in a local file.
type JSONStorage struct {
	mu     sync.RWMutex
	path   string
	todos  []todo.Todo
	nextID int
}

// NewJSONStorage opens or creates the JSON-backed store at path.
func NewJSONStorage(path string) (*JSONStorage, error) {
	s := &JSONStorage{
		path:   path,
		nextID: 1,
	}
	if err := s.load(); err != nil {
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

func (s *JSONStorage) nextIDString() string {
	return fmt.Sprintf("todo-%d", s.nextID)
}

func parseID(id string) (int, bool) {
	var n int
	_, err := fmt.Sscanf(id, "todo-%d", &n)
	return n, err == nil
}
