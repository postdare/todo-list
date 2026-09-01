// Package todo contains the domain models for the todo list.
package todo

import "time"

// Status represents the lifecycle state of a todo item.
type Status string

// Valid todo statuses.
const (
	StatusNotStarted Status = "未开始"
	StatusInProgress Status = "进行中"
	StatusDone       Status = "已完成"
)

// Todo is the core domain entity. Its JSON tags ensure the persisted file
// uses the exact keys required by the contract.
type Todo struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Status      Status    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
}
