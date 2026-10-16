package dto

import "taskapi/internal/model"

// TaskRequest is the body of POST /tasks and PUT /tasks/{id}. Clients cannot set the id.
//
// Status is *int rather than int: validator's required treats 0 as missing,
// but 0 (incomplete) is valid, and only a pointer tells "not sent" apart from "sent 0".
type TaskRequest struct {
	Name   string `json:"name"   validate:"required,max=100"`
	Status *int   `json:"status" validate:"omitempty,oneof=0 1"`
}

// ToModel converts the request to a model.Task.
func (r TaskRequest) ToModel() model.Task {
	status := model.StatusIncomplete
	if r.Status != nil {
		status = *r.Status
	}
	return model.Task{Name: r.Name, Status: status}
}
