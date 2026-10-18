// Package web exposes the todo list over HTTP so the same store can be used
// from a browser in addition to the terminal UI.
package web

import (
	_ "embed"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync"

	"todo-list/internal/storage"
	"todo-list/internal/todo"
)

//go:embed index.html
var indexHTML []byte

// Server is an http.Handler that serves the single-page web UI and its JSON
// API, backed by the same storage used by the terminal UI.
type Server struct {
	store storage.Storage
	mux   *http.ServeMux
	// mu serializes mutations so "add, then read back the created record" is
	// atomic with respect to other requests. Personal-scale server; a single
	// mutex keeps the implementation simple.
	mu sync.Mutex
}

// NewServer wires the HTTP routes to store.
func NewServer(store storage.Storage) *Server {
	s := &Server{store: store, mux: http.NewServeMux()}
	s.mux.HandleFunc("GET /{$}", s.handleIndex)
	s.mux.HandleFunc("GET /api/todos", s.handleList)
	s.mux.HandleFunc("POST /api/todos", s.handleCreate)
	s.mux.HandleFunc("PUT /api/todos/{id}", s.handleUpdate)
	s.mux.HandleFunc("DELETE /api/todos/{id}", s.handleDelete)
	s.mux.HandleFunc("GET /api/panels", s.handleListPanels)
	s.mux.HandleFunc("POST /api/panels", s.handleCreatePanel)
	s.mux.HandleFunc("PUT /api/panels/{id}", s.handleUpdatePanel)
	s.mux.HandleFunc("DELETE /api/panels/{id}", s.handleDeletePanel)
	return s
}

// ServeHTTP implements http.Handler.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(indexHTML)
}

func (s *Server) handleList(w http.ResponseWriter, r *http.Request) {
	items, err := s.store.List()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "读取失败："+err.Error())
		return
	}
	if items == nil {
		items = []todo.Todo{}
	}
	writeJSON(w, http.StatusOK, items)
}

// createRequest is the payload accepted by POST /api/todos.
type createRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

func (s *Server) handleCreate(w http.ResponseWriter, r *http.Request) {
	var req createRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "请求格式错误")
		return
	}
	title := strings.TrimSpace(req.Title)
	if title == "" {
		writeError(w, http.StatusBadRequest, "标题不能为空")
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.store.Add(todo.Todo{
		Title:       title,
		Description: strings.TrimSpace(req.Description),
		Status:      todo.StatusNotStarted,
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "保存失败："+err.Error())
		return
	}

	items, err := s.store.List()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "读取失败："+err.Error())
		return
	}
	if len(items) == 0 {
		writeError(w, http.StatusInternalServerError, "保存后未找到任务")
		return
	}
	writeJSON(w, http.StatusCreated, items[len(items)-1])
}

// updateRequest is the payload accepted by PUT /api/todos/{id}. Omitted
// fields (nil pointers) are left unchanged. An empty panel_id moves the todo
// back to the ungrouped status columns.
type updateRequest struct {
	Title       *string `json:"title"`
	Description *string `json:"description"`
	Status      *string `json:"status"`
	PanelID     *string `json:"panel_id"`
}

func (s *Server) handleUpdate(w http.ResponseWriter, r *http.Request) {
	var req updateRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "请求格式错误")
		return
	}
	if req.Title != nil && strings.TrimSpace(*req.Title) == "" {
		writeError(w, http.StatusBadRequest, "标题不能为空")
		return
	}
	if req.Status != nil && !validStatus(todo.Status(*req.Status)) {
		writeError(w, http.StatusBadRequest, "无效的状态："+*req.Status)
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	id := r.PathValue("id")
	items, err := s.store.List()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "读取失败："+err.Error())
		return
	}
	found := false
	var updated todo.Todo
	for _, t := range items {
		if t.ID != id {
			continue
		}
		updated, found = t, true
		break
	}
	if !found {
		writeError(w, http.StatusNotFound, "任务不存在")
		return
	}

	if req.Title != nil {
		updated.Title = strings.TrimSpace(*req.Title)
	}
	if req.Description != nil {
		updated.Description = strings.TrimSpace(*req.Description)
	}
	if req.Status != nil {
		updated.Status = todo.Status(*req.Status)
	}
	if req.PanelID != nil {
		panelID := strings.TrimSpace(*req.PanelID)
		if panelID != "" {
			panels, err := s.store.ListPanels()
			if err != nil {
				writeError(w, http.StatusInternalServerError, "读取失败："+err.Error())
				return
			}
			found := false
			for _, p := range panels {
				if p.ID == panelID {
					found = true
					break
				}
			}
			if !found {
				writeError(w, http.StatusBadRequest, "分区不存在："+panelID)
				return
			}
		}
		updated.PanelID = panelID
	}

	if err := s.store.Update(updated); err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			writeError(w, http.StatusNotFound, "任务不存在")
			return
		}
		writeError(w, http.StatusInternalServerError, "保存失败："+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (s *Server) handleDelete(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.store.Delete(r.PathValue("id")); err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			writeError(w, http.StatusNotFound, "任务不存在")
			return
		}
		writeError(w, http.StatusInternalServerError, "删除失败："+err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleListPanels(w http.ResponseWriter, r *http.Request) {
	panels, err := s.store.ListPanels()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "读取失败："+err.Error())
		return
	}
	if panels == nil {
		panels = []todo.Panel{}
	}
	writeJSON(w, http.StatusOK, panels)
}

// panelRequest is the payload accepted by POST /api/panels and
// PUT /api/panels/{id}.
type panelRequest struct {
	Name string `json:"name"`
}

func (s *Server) handleCreatePanel(w http.ResponseWriter, r *http.Request) {
	var req panelRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "请求格式错误")
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		writeError(w, http.StatusBadRequest, "分区名称不能为空")
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.store.AddPanel(todo.Panel{Name: name}); err != nil {
		writeError(w, http.StatusInternalServerError, "保存失败："+err.Error())
		return
	}

	panels, err := s.store.ListPanels()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "读取失败："+err.Error())
		return
	}
	if len(panels) == 0 {
		writeError(w, http.StatusInternalServerError, "保存后未找到分区")
		return
	}
	writeJSON(w, http.StatusCreated, panels[len(panels)-1])
}

func (s *Server) handleUpdatePanel(w http.ResponseWriter, r *http.Request) {
	var req panelRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "请求格式错误")
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		writeError(w, http.StatusBadRequest, "分区名称不能为空")
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	id := r.PathValue("id")
	panels, err := s.store.ListPanels()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "读取失败："+err.Error())
		return
	}
	found := false
	var updated todo.Panel
	for _, p := range panels {
		if p.ID == id {
			updated, found = p, true
			break
		}
	}
	if !found {
		writeError(w, http.StatusNotFound, "分区不存在")
		return
	}

	updated.Name = name
	if err := s.store.UpdatePanel(updated); err != nil {
		if errors.Is(err, storage.ErrPanelNotFound) {
			writeError(w, http.StatusNotFound, "分区不存在")
			return
		}
		writeError(w, http.StatusInternalServerError, "保存失败："+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (s *Server) handleDeletePanel(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.store.DeletePanel(r.PathValue("id")); err != nil {
		if errors.Is(err, storage.ErrPanelNotFound) {
			writeError(w, http.StatusNotFound, "分区不存在")
			return
		}
		writeError(w, http.StatusInternalServerError, "删除失败："+err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// decodeJSON decodes a single JSON value from the request body with a size
// cap, rejecting malformed payloads.
func decodeJSON(w http.ResponseWriter, r *http.Request, v any) error {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	return dec.Decode(v)
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}

func validStatus(s todo.Status) bool {
	switch s {
	case todo.StatusNotStarted, todo.StatusInProgress, todo.StatusDone:
		return true
	}
	return false
}
