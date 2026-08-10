package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/joaquimrafael/go-task-api/internal/model"
	"github.com/joaquimrafael/go-task-api/internal/service"
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

type DatabasePinger interface {
	PingContext(ctx context.Context) error
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

type healthResponse struct {
	Status string `json:"status"`
}

func NewHealthHandler(database DatabasePinger) (http.HandlerFunc, error) {
	if database == nil {
		return nil, fmt.Errorf("database must not be nil")
	}

	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		if err := database.PingContext(ctx); err != nil {
			_ = writeJSONError(
				w,
				http.StatusServiceUnavailable,
				"database unavailable",
			)
			return
		}

		_ = writeJSON(w, http.StatusOK, healthResponse{
			Status: "ok",
		})
	}, nil
}

var _ TaskService = (*service.TaskService)(nil)

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
