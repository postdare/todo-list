package todo

import (
	"encoding/json"
	"testing"
	"time"
)

func TestStatusConstants(t *testing.T) {
	if StatusNotStarted != "未开始" {
		t.Errorf("StatusNotStarted = %q, want 未开始", StatusNotStarted)
	}
	if StatusInProgress != "进行中" {
		t.Errorf("StatusInProgress = %q, want 进行中", StatusInProgress)
	}
	if StatusDone != "已完成" {
		t.Errorf("StatusDone = %q, want 已完成", StatusDone)
	}
}

func TestTodoJSONMarshaling(t *testing.T) {
	now := time.Date(2025, 1, 2, 3, 4, 5, 0, time.UTC)
	item := Todo{
		ID:          "1",
		Title:       "Buy milk",
		Description: "从楼下便利店买一盒",
		Status:      StatusNotStarted,
		CreatedAt:   now,
	}

	data, err := json.Marshal(item)
	if err != nil {
		t.Fatalf("marshal todo: %v", err)
	}

	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unmarshal to map: %v", err)
	}

	if raw["id"] != "1" {
		t.Errorf("id = %v, want 1", raw["id"])
	}
	if raw["title"] != "Buy milk" {
		t.Errorf("title = %v, want Buy milk", raw["title"])
	}
	if raw["description"] != "从楼下便利店买一盒" {
		t.Errorf("description = %v, want 从楼下便利店买一盒", raw["description"])
	}
	if raw["status"] != "未开始" {
		t.Errorf("status = %v, want 未开始", raw["status"])
	}
	if raw["created_at"] != "2025-01-02T03:04:05Z" {
		t.Errorf("created_at = %v, want 2025-01-02T03:04:05Z", raw["created_at"])
	}
}

func TestTodoJSONMarshalingEmptyDescription(t *testing.T) {
	// 没有描述的任务序列化后 description 键必须存在且为空字符串,
	// 保证持久化格式稳定(所有记录都带 description 键)。
	item := Todo{
		ID:        "1",
		Title:     "No desc",
		Status:    StatusNotStarted,
		CreatedAt: time.Date(2025, 1, 2, 3, 4, 5, 0, time.UTC),
	}

	data, err := json.Marshal(item)
	if err != nil {
		t.Fatalf("marshal todo: %v", err)
	}

	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unmarshal to map: %v", err)
	}
	if v, ok := raw["description"]; !ok {
		t.Errorf("description key missing for empty description")
	} else if v != "" {
		t.Errorf("description = %v, want empty string", v)
	}
}

func TestTodoJSONUnmarshaling(t *testing.T) {
	data := []byte(`{
		"id": "2",
		"title": "Read book",
		"description": "重点看第三章",
		"status": "进行中",
		"created_at": "2025-06-15T10:30:00Z"
	}`)

	var item Todo
	if err := json.Unmarshal(data, &item); err != nil {
		t.Fatalf("unmarshal todo: %v", err)
	}

	if item.ID != "2" {
		t.Errorf("ID = %q, want 2", item.ID)
	}
	if item.Title != "Read book" {
		t.Errorf("Title = %q, want Read book", item.Title)
	}
	if item.Description != "重点看第三章" {
		t.Errorf("Description = %q, want 重点看第三章", item.Description)
	}
	if item.Status != StatusInProgress {
		t.Errorf("Status = %q, want 进行中", item.Status)
	}
	expectedTime := time.Date(2025, 6, 15, 10, 30, 0, 0, time.UTC)
	if !item.CreatedAt.Equal(expectedTime) {
		t.Errorf("CreatedAt = %v, want %v", item.CreatedAt, expectedTime)
	}
}

func TestTodoJSONUnmarshalingLegacyWithoutDescription(t *testing.T) {
	// 旧版本 todos.json 的记录没有 description 键,加载后必须为空字符串(向后兼容)。
	data := []byte(`{
		"id": "3",
		"title": "Legacy task",
		"status": "未开始",
		"created_at": "2024-01-01T00:00:00Z"
	}`)

	var item Todo
	if err := json.Unmarshal(data, &item); err != nil {
		t.Fatalf("unmarshal legacy todo: %v", err)
	}

	if item.ID != "3" {
		t.Errorf("ID = %q, want 3", item.ID)
	}
	if item.Title != "Legacy task" {
		t.Errorf("Title = %q, want Legacy task", item.Title)
	}
	if item.Description != "" {
		t.Errorf("Description = %q, want empty (legacy record has no description key)", item.Description)
	}
	if item.Status != StatusNotStarted {
		t.Errorf("Status = %q, want 未开始", item.Status)
	}
}
