package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/joaquimrafael/go-task-api/internal/model"
)

const jsonContentType = "application/json"

type errorResponse struct {
	Error string `json:"error"`
}

type TaskService interface {
	List(ctx context.Context) ([]model.Task, error)
	GetByID(ctx context.Context, id int64) (*model.Task, error)
	Create(ctx context.Context, input model.TaskInput) (*model.Task, error)
	Update(ctx context.Context, id int64, input model.TaskInput) (*model.Task, error)
	Delete(ctx context.Context, id int64) error
}

func writeJSON(w http.ResponseWriter, status int, value any) error {
	w.Header().Set("Content-Type", jsonContentType)
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(value); err != nil {
		return fmt.Errorf("encode JSON response: %w", err)
	}

	return nil
}

func writeJSONError(w http.ResponseWriter, status int, message string) error {
	return writeJSON(w, status, errorResponse{
		Error: message,
	})
}

type TaskHandler struct {
	service TaskService
}

func NewTaskHandler(service TaskService) (*TaskHandler, error) {
	if service == nil {
		return nil, fmt.Errorf("service must not be nil")
	}
	return &TaskHandler{service: service}, nil
}

func (th *TaskHandler) List(w http.ResponseWriter, r *http.Request) {
	tasks, err := th.service.List(r.Context())
	if err != nil {
		_ = writeJSONError(
			w,
			http.StatusInternalServerError,
			"internal server error",
		)
		return
	}

	_ = writeJSON(w, http.StatusOK, tasks)
}

func (th *TaskHandler) Create(w http.ResponseWriter, r *http.Request) {
	var input model.TaskInput

	err := decodeTaskInput(r.Body, &input)
	if err != nil {
		_ = writeJSONError(
			w,
			http.StatusBadRequest,
			"could not decode body",
		)
		return
	}

	task, err := th.service.Create(r.Context(), input)
	if err != nil {
		writeTaskServiceError(w, err)
		return
	}

	_ = writeJSON(w, http.StatusCreated, task)
}

func (th *TaskHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := parseTaskID(r)
	if err != nil {
		_ = writeJSONError(
			w,
			http.StatusBadRequest,
			"invalid task id",
		)
		return
	}

	task, err := th.service.GetByID(r.Context(), id)
	if err != nil {
		writeTaskServiceError(w, err)
		return
	}

	_ = writeJSON(w, http.StatusOK, task)

}

func (th *TaskHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := parseTaskID(r)
	if err != nil {
		_ = writeJSONError(
			w,
			http.StatusBadRequest,
			"invalid task id",
		)
		return
	}

	var input model.TaskInput

	err = decodeTaskInput(r.Body, &input)
	if err != nil {
		_ = writeJSONError(
			w,
			http.StatusBadRequest,
			"could not decode body",
		)
		return
	}

	task, err := th.service.Update(r.Context(), id, input)
	if err != nil {
		writeTaskServiceError(w, err)
		return
	}

	_ = writeJSON(w, http.StatusOK, task)
}

func (th *TaskHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseTaskID(r)
	if err != nil {
		_ = writeJSONError(
			w,
			http.StatusBadRequest,
			"invalid task id",
		)
		return
	}

	err = th.service.Delete(r.Context(), id)
	if err != nil {
		writeTaskServiceError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func parseTaskID(r *http.Request) (int64, error) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		return 0, errors.New("invalid task id")
	}
	return id, nil
}

func decodeTaskInput(r io.Reader, input *model.TaskInput) error {
	decoder := json.NewDecoder(r)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(input); err != nil {
		return err
	}

	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return errors.New("body must contain exactly one JSON object")
	}

	return nil
}

func writeTaskServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, model.ErrInvalidTask):
		_ = writeJSONError(
			w,
			http.StatusUnprocessableEntity,
			"unprocessable entity",
		)
	case errors.Is(err, model.ErrTaskNotFound):
		_ = writeJSONError(
			w,
			http.StatusNotFound,
			"task not found",
		)
	default:
		_ = writeJSONError(
			w,
			http.StatusInternalServerError,
			"internal server error",
		)
	}
}
