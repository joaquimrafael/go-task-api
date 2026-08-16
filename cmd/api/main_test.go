package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/joaquimrafael/go-task-api/internal/handler"
	"github.com/joaquimrafael/go-task-api/internal/model"
)

type recordingTaskService struct {
	method string
	id     int64
	input  model.TaskInput
}

func (s *recordingTaskService) List(context.Context) ([]model.Task, error) {
	s.method = "List"
	return []model.Task{}, nil
}

func (s *recordingTaskService) GetByID(_ context.Context, id int64) (*model.Task, error) {
	s.method = "GetByID"
	s.id = id
	return &model.Task{ID: id, Title: "Task"}, nil
}

func (s *recordingTaskService) Create(_ context.Context, input model.TaskInput) (*model.Task, error) {
	s.method = "Create"
	s.input = input
	return &model.Task{ID: 1, Title: input.Title}, nil
}

func (s *recordingTaskService) Update(_ context.Context, id int64, input model.TaskInput) (*model.Task, error) {
	s.method = "Update"
	s.id = id
	s.input = input
	return &model.Task{ID: id, Title: input.Title}, nil
}

func (s *recordingTaskService) Delete(_ context.Context, id int64) error {
	s.method = "Delete"
	s.id = id
	return nil
}

func newTestRouter(t *testing.T, service *recordingTaskService) http.Handler {
	t.Helper()

	taskHandler, err := handler.NewTaskHandler(service)
	if err != nil {
		t.Fatalf("NewTaskHandler() error = %v", err)
	}
	healthHandler := func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}

	return newRouter(taskHandler, healthHandler)
}

func TestNewRouterRoutesRequests(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		path       string
		body       string
		wantStatus int
		wantCall   string
		wantID     int64
		wantInput  model.TaskInput
	}{
		{
			name:       "health",
			method:     http.MethodGet,
			path:       "/health",
			wantStatus: http.StatusOK,
		},
		{
			name:       "list tasks",
			method:     http.MethodGet,
			path:       "/tasks",
			wantStatus: http.StatusOK,
			wantCall:   "List",
		},
		{
			name:       "get task",
			method:     http.MethodGet,
			path:       "/tasks/17",
			wantStatus: http.StatusOK,
			wantCall:   "GetByID",
			wantID:     17,
		},
		{
			name:       "create task",
			method:     http.MethodPost,
			path:       "/tasks",
			body:       `{"title":"Created"}`,
			wantStatus: http.StatusCreated,
			wantCall:   "Create",
			wantInput:  model.TaskInput{Title: "Created"},
		},
		{
			name:       "update task",
			method:     http.MethodPut,
			path:       "/tasks/23",
			body:       `{"title":"Updated","completed":true}`,
			wantStatus: http.StatusOK,
			wantCall:   "Update",
			wantID:     23,
			wantInput:  model.TaskInput{Title: "Updated", Completed: true},
		},
		{
			name:       "delete task",
			method:     http.MethodDelete,
			path:       "/tasks/29",
			wantStatus: http.StatusNoContent,
			wantCall:   "Delete",
			wantID:     29,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service := &recordingTaskService{}
			request := httptest.NewRequest(test.method, test.path, stringReader(test.body))
			recorder := httptest.NewRecorder()

			newTestRouter(t, service).ServeHTTP(recorder, request)

			if recorder.Code != test.wantStatus {
				t.Errorf("status = %d, want %d", recorder.Code, test.wantStatus)
			}
			if service.method != test.wantCall {
				t.Errorf("service call = %q, want %q", service.method, test.wantCall)
			}
			if service.id != test.wantID {
				t.Errorf("service id = %d, want %d", service.id, test.wantID)
			}
			if service.input != test.wantInput {
				t.Errorf("service input = %#v, want %#v", service.input, test.wantInput)
			}
		})
	}
}

func TestNewRouterRejectsUnmatchedRequests(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		path       string
		wantStatus int
		wantAllow  string
	}{
		{
			name:       "unknown path",
			method:     http.MethodGet,
			path:       "/unknown",
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "unsupported collection method",
			method:     http.MethodPatch,
			path:       "/tasks",
			wantStatus: http.StatusMethodNotAllowed,
			wantAllow:  "GET, HEAD, POST",
		},
		{
			name:       "unsupported item method",
			method:     http.MethodPost,
			path:       "/tasks/1",
			wantStatus: http.StatusMethodNotAllowed,
			wantAllow:  "DELETE, GET, HEAD, PUT",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service := &recordingTaskService{}
			request := httptest.NewRequest(test.method, test.path, nil)
			recorder := httptest.NewRecorder()

			newTestRouter(t, service).ServeHTTP(recorder, request)

			if recorder.Code != test.wantStatus {
				t.Errorf("status = %d, want %d", recorder.Code, test.wantStatus)
			}
			if got := recorder.Header().Get("Allow"); got != test.wantAllow {
				t.Errorf("Allow = %q, want %q", got, test.wantAllow)
			}
			if service.method != "" {
				t.Errorf("unexpected service call: %s", service.method)
			}
		})
	}
}

func stringReader(value string) *strings.Reader {
	return strings.NewReader(value)
}
