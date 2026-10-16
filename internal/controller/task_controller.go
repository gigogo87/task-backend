package controller

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-playground/validator/v10"

	"taskapi/internal/dto"
	"taskapi/internal/model"
	"taskapi/internal/response"
	"taskapi/internal/service"
)

const maxBodyBytes = 1 << 20 // cap request bodies at 1 MB

type TaskController struct {
	service  service.TaskService
	validate *validator.Validate
	logger   *slog.Logger
}

// NewTaskController receives all its dependencies as parameters, so tests can pass a fake service.
func NewTaskController(svc service.TaskService, validate *validator.Validate, logger *slog.Logger) *TaskController {
	return &TaskController{service: svc, validate: validate, logger: logger}
}

// List handles GET /tasks
func (c *TaskController) List(w http.ResponseWriter, r *http.Request) {
	c.logger.InfoContext(r.Context(), "[CTR IN] List")
	tasks, err := c.service.List(r.Context())
	if err != nil {
		c.fail(w, r, err)
		return
	}
	response.WriteJSON(w, http.StatusOK, map[string]any{"result": tasks})
}

// Create handles POST /tasks
func (c *TaskController) Create(w http.ResponseWriter, r *http.Request) {
	c.logger.InfoContext(r.Context(), "[CTR IN] Create")
	req, err := c.decodeAndValidate(w, r)
	if err != nil {
		c.fail(w, r, err)
		return
	}

	created, err := c.service.Create(r.Context(), req.ToModel())
	if err != nil {
		c.fail(w, r, err)
		return
	}
	response.WriteJSON(w, http.StatusCreated, map[string]any{"result": created})
}

// Update handles PUT /tasks/{id}
func (c *TaskController) Update(w http.ResponseWriter, r *http.Request) {
	c.logger.InfoContext(r.Context(), "[CTR IN] Update")
	id, err := c.parseID(r)
	if err != nil {
		c.fail(w, r, err)
		return
	}
	req, err := c.decodeAndValidate(w, r)
	if err != nil {
		c.fail(w, r, err)
		return
	}

	updated, err := c.service.Update(r.Context(), id, req.ToModel())
	if err != nil {
		c.fail(w, r, err)
		return
	}
	response.WriteJSON(w, http.StatusOK, map[string]any{"result": updated})
}

// Delete handles DELETE /tasks/{id}
func (c *TaskController) Delete(w http.ResponseWriter, r *http.Request) {
	c.logger.InfoContext(r.Context(), "[CTR IN] Delete")
	id, err := c.parseID(r)
	if err != nil {
		c.fail(w, r, err)
		return
	}

	if err := c.service.Delete(r.Context(), id); err != nil {
		c.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent) // 204: success with no body
}

// ---------- Helpers ----------

// parseID reads {id} from the URL and checks that it is a positive integer.
func (c *TaskController) parseID(r *http.Request) (int, error) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || id <= 0 {
		return 0, response.ErrInvalidID
	}
	return id, nil
}

// decodeAndValidate decodes the JSON body into a dto.TaskRequest and checks it with the validator.
func (c *TaskController) decodeAndValidate(w http.ResponseWriter, r *http.Request) (dto.TaskRequest, error) {
	var req dto.TaskRequest
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes))
	dec.DisallowUnknownFields() // reject unknown fields
	if err := dec.Decode(&req); err != nil {
		return dto.TaskRequest{}, decodeBodyError(err)
	}
	// Decode reads only the first JSON value; anything left after it (e.g. {"name":"x"}garbage or a second JSON object)
	// is also malformed. If only whitespace remains, the second Decode returns io.EOF, meaning the body is fully read.
	var extra json.RawMessage
	if err := dec.Decode(&extra); !errors.Is(err, io.EOF) {
		return dto.TaskRequest{}, decodeBodyError(err)
	}

	if err := c.validate.Struct(req); err != nil {
		return dto.TaskRequest{}, dto.ValidationError(err)
	}
	return req, nil
}

// decodeBodyError maps a body decoding error to an API error: PAYLOAD_TOO_LARGE when over the size limit, otherwise (including nil,
// i.e. an extra valid JSON value after the body) INVALID_JSON.
func decodeBodyError(err error) error {
	var tooLarge *http.MaxBytesError
	if errors.As(err, &tooLarge) {
		return response.ErrPayloadTooLarge
	}
	return response.ErrInvalidJSON
}

// fail writes an error response. Unexpected errors (not an AppError) are logged first, because the client only sees "internal server error".
func (c *TaskController) fail(w http.ResponseWriter, r *http.Request, err error) {
	var appErr *model.AppError
	if !errors.As(err, &appErr) {
		c.logger.ErrorContext(r.Context(), "unexpected error",
			"error", err, "method", r.Method, "path", r.URL.Path)
	}
	response.WriteError(w, r, err)
}
