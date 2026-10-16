package model

// Task status.
const (
	StatusIncomplete = 0
	StatusComplete   = 1
)

// Task is the core data structure of the API, passed between all layers.
type Task struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Status int    `json:"status"`
}
