package handler_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/joaquimrafael/go-task-api/internal/handler"
	"github.com/joaquimrafael/go-task-api/internal/model"
)

type fakeTaskService struct {
	listFn    func(context.Context) ([]model.Task, error)
	getByIDFn func(context.Context, int64) (*model.Task, error)
	createFn  func(context.Context, model.TaskInput) (*model.Task, error)
	updateFn  func(context.Context, int64, model.TaskInput) (*model.Task, error)
	deleteFn  func(context.Context, int64) error
}

func (f *fakeTaskService) List(ctx context.Context) ([]model.Task, error) {
	return f.listFn(ctx)
}

func (f *fakeTaskService) GetByID(ctx context.Context, id int64) (*model.Task, error) {
	return f.getByIDFn(ctx, id)
}

func (f *fakeTaskService) Create(ctx context.Context, input model.TaskInput) (*model.Task, error) {
	return f.createFn(ctx, input)
}

func (f *fakeTaskService) Update(ctx context.Context, id int64, input model.TaskInput) (*model.Task, error) {
	return f.updateFn(ctx, id, input)
}

func (f *fakeTaskService) Delete(ctx context.Context, id int64) error {
	return f.deleteFn(ctx, id)
}

func newFakeTaskService(t *testing.T) *fakeTaskService {
	t.Helper()

	unexpected := func(method string) {
		t.Helper()
		t.Fatalf("unexpected service call: %s", method)
	}

	return &fakeTaskService{
		listFn: func(context.Context) ([]model.Task, error) {
			unexpected("List")
			return nil, nil
		},
		getByIDFn: func(context.Context, int64) (*model.Task, error) {
			unexpected("GetByID")
			return nil, nil
		},
		createFn: func(context.Context, model.TaskInput) (*model.Task, error) {
			unexpected("Create")
			return nil, nil
		},
		updateFn: func(context.Context, int64, model.TaskInput) (*model.Task, error) {
			unexpected("Update")
			return nil, nil
		},
		deleteFn: func(context.Context, int64) error {
			unexpected("Delete")
			return nil
		},
	}
}

func newTaskMux(t *testing.T, service handler.TaskService) http.Handler {
	t.Helper()

	taskHandler, err := handler.NewTaskHandler(service)
	if err != nil {
		t.Fatalf("NewTaskHandler() error = %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /tasks", taskHandler.List)
	mux.HandleFunc("GET /tasks/{id}", taskHandler.GetByID)
	mux.HandleFunc("POST /tasks", taskHandler.Create)
	mux.HandleFunc("PUT /tasks/{id}", taskHandler.Update)
	mux.HandleFunc("DELETE /tasks/{id}", taskHandler.Delete)
	return mux
}

func performRequest(t *testing.T, h http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()

	request := httptest.NewRequest(method, path, strings.NewReader(body))
	request = request.WithContext(context.WithValue(request.Context(), requestContextKey{}, t.Name()))
	recorder := httptest.NewRecorder()
	h.ServeHTTP(recorder, request)
	return recorder
}

func assertServiceContext(t *testing.T, ctx context.Context) {
	t.Helper()

	if got := ctx.Value(requestContextKey{}); got != t.Name() {
		t.Errorf("context marker = %#v, want %q", got, t.Name())
	}
}

func assertEqual[T any](t *testing.T, got, want T) {
	t.Helper()

	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %#v, want %#v", got, want)
	}
}

func assertJSONResponse(t *testing.T, recorder *httptest.ResponseRecorder, wantStatus int, wantBody any) {
	t.Helper()

	if recorder.Code != wantStatus {
		t.Errorf("status = %d, want %d", recorder.Code, wantStatus)
	}
	if got := recorder.Header().Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type = %q, want %q", got, "application/json")
	}

	wantJSON, err := json.Marshal(wantBody)
	if err != nil {
		t.Fatalf("marshal expected response: %v", err)
	}

	var gotValue any
	if err := json.Unmarshal(recorder.Body.Bytes(), &gotValue); err != nil {
		t.Fatalf("response is not valid JSON: %v; body = %q", err, recorder.Body.String())
	}

	var wantValue any
	if err := json.Unmarshal(wantJSON, &wantValue); err != nil {
		t.Fatalf("decode expected response: %v", err)
	}

	assertEqual(t, gotValue, wantValue)
}

func assertNoContentResponse(t *testing.T, recorder *httptest.ResponseRecorder) {
	t.Helper()

	if recorder.Code != http.StatusNoContent {
		t.Errorf("status = %d, want %d", recorder.Code, http.StatusNoContent)
	}
	if recorder.Body.Len() != 0 {
		t.Errorf("body = %q, want empty body", recorder.Body.String())
	}
	if got := recorder.Header().Get("Content-Type"); got != "" {
		t.Errorf("Content-Type = %q, want no Content-Type", got)
	}
}

func errorBody(message string) map[string]string {
	return map[string]string{"error": message}
}

func TestNewTaskHandler(t *testing.T) {
	tests := []struct {
		name    string
		service handler.TaskService
		wantErr bool
	}{
		{
			name:    "accepts service",
			service: newFakeTaskService(t),
		},
		{
			name:    "rejects nil service",
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := handler.NewTaskHandler(test.service)
			if test.wantErr {
				if err == nil {
					t.Fatal("NewTaskHandler() error = nil, want an error")
				}
				if got != nil {
					t.Errorf("NewTaskHandler() handler = %#v, want nil", got)
				}
				return
			}

			if err != nil {
				t.Fatalf("NewTaskHandler() error = %v", err)
			}
			if got == nil {
				t.Fatal("NewTaskHandler() handler = nil, want a handler")
			}
		})
	}
}

func TestTaskHandlerList(t *testing.T) {
	tasks := []model.Task{
		{ID: 1, Title: "First task"},
		{ID: 2, Title: "Second task", Completed: true},
	}
	repositoryErr := errors.New("database unavailable")

	tests := []struct {
		name        string
		serviceTask []model.Task
		serviceErr  error
		wantStatus  int
		wantBody    any
	}{
		{
			name:        "lists tasks",
			serviceTask: tasks,
			wantStatus:  http.StatusOK,
			wantBody:    tasks,
		},
		{
			name:        "lists no tasks",
			serviceTask: []model.Task{},
			wantStatus:  http.StatusOK,
			wantBody:    []model.Task{},
		},
		{
			name:       "returns 500 for service failure",
			serviceErr: repositoryErr,
			wantStatus: http.StatusInternalServerError,
			wantBody:   errorBody("internal server error"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service := newFakeTaskService(t)
			calls := 0
			service.listFn = func(ctx context.Context) ([]model.Task, error) {
				calls++
				assertServiceContext(t, ctx)
				return test.serviceTask, test.serviceErr
			}

			recorder := performRequest(t, newTaskMux(t, service), http.MethodGet, "/tasks", "")

			assertJSONResponse(t, recorder, test.wantStatus, test.wantBody)
			assertEqual(t, calls, 1)
		})
	}
}

func TestTaskHandlerGetByID(t *testing.T) {
	task := &model.Task{ID: 42, Title: "Read Go docs"}
	serviceErr := errors.New("service failed")

	tests := []struct {
		name        string
		path        string
		serviceID   *int64
		serviceTask *model.Task
		serviceErr  error
		wantStatus  int
		wantBody    any
	}{
		{
			name:        "gets task",
			path:        "/tasks/42",
			serviceID:   int64Pointer(42),
			serviceTask: task,
			wantStatus:  http.StatusOK,
			wantBody:    task,
		},
		{
			name:       "returns 400 for non-numeric id",
			path:       "/tasks/not-a-number",
			wantStatus: http.StatusBadRequest,
			wantBody:   errorBody("invalid task id"),
		},
		{
			name:       "returns 400 for zero id",
			path:       "/tasks/0",
			wantStatus: http.StatusBadRequest,
			wantBody:   errorBody("invalid task id"),
		},
		{
			name:       "returns 400 for negative id",
			path:       "/tasks/-1",
			wantStatus: http.StatusBadRequest,
			wantBody:   errorBody("invalid task id"),
		},
		{
			name:       "returns 400 for overflowing id",
			path:       "/tasks/9223372036854775808",
			wantStatus: http.StatusBadRequest,
			wantBody:   errorBody("invalid task id"),
		},
		{
			name:       "returns 404 for missing task",
			path:       "/tasks/99",
			serviceID:  int64Pointer(99),
			serviceErr: fmt.Errorf("get task: %w", model.ErrTaskNotFound),
			wantStatus: http.StatusNotFound,
			wantBody:   errorBody("task not found"),
		},
		{
			name:       "returns 500 for service failure",
			path:       "/tasks/42",
			serviceID:  int64Pointer(42),
			serviceErr: serviceErr,
			wantStatus: http.StatusInternalServerError,
			wantBody:   errorBody("internal server error"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service := newFakeTaskService(t)
			calls := 0
			if test.serviceID != nil {
				service.getByIDFn = func(ctx context.Context, id int64) (*model.Task, error) {
					calls++
					assertServiceContext(t, ctx)
					assertEqual(t, id, *test.serviceID)
					return test.serviceTask, test.serviceErr
				}
			}

			recorder := performRequest(t, newTaskMux(t, service), http.MethodGet, test.path, "")

			assertJSONResponse(t, recorder, test.wantStatus, test.wantBody)
			if test.serviceID != nil {
				assertEqual(t, calls, 1)
			}
		})
	}
}

func TestTaskHandlerCreate(t *testing.T) {
	createdAt := time.Date(2026, time.August, 15, 12, 0, 0, 0, time.UTC)
	input := model.TaskInput{Title: "Learn handlers", Description: "Write tests"}
	task := &model.Task{
		ID:          7,
		Title:       input.Title,
		Description: input.Description,
		CreatedAt:   createdAt,
		UpdatedAt:   createdAt,
	}
	serviceErr := errors.New("service failed")

	tests := []struct {
		name         string
		body         string
		serviceInput *model.TaskInput
		serviceTask  *model.Task
		serviceErr   error
		wantStatus   int
		wantBody     any
	}{
		{
			name:         "creates task",
			body:         `{"title":"Learn handlers","description":"Write tests","completed":false}`,
			serviceInput: &input,
			serviceTask:  task,
			wantStatus:   http.StatusCreated,
			wantBody:     task,
		},
		{
			name:       "returns 400 for empty body",
			wantStatus: http.StatusBadRequest,
			wantBody:   errorBody("could not decode body"),
		},
		{
			name:       "returns 400 for malformed JSON",
			body:       `{"title":`,
			wantStatus: http.StatusBadRequest,
			wantBody:   errorBody("could not decode body"),
		},
		{
			name:       "returns 400 for JSON null",
			body:       `null`,
			wantStatus: http.StatusBadRequest,
			wantBody:   errorBody("could not decode body"),
		},
		{
			name:       "returns 400 for unknown field",
			body:       `{"title":"Test","priority":1}`,
			wantStatus: http.StatusBadRequest,
			wantBody:   errorBody("could not decode body"),
		},
		{
			name:       "returns 400 for a second JSON value",
			body:       `{"title":"First"} {"title":"Second"}`,
			wantStatus: http.StatusBadRequest,
			wantBody:   errorBody("could not decode body"),
		},
		{
			name:         "returns 422 for invalid task",
			body:         `{"title":""}`,
			serviceInput: &model.TaskInput{},
			serviceErr:   fmt.Errorf("create task: %w", model.ErrInvalidTask),
			wantStatus:   http.StatusUnprocessableEntity,
			wantBody:     errorBody("unprocessable entity"),
		},
		{
			name:         "returns 500 for service failure",
			body:         `{"title":"Learn handlers","description":"Write tests"}`,
			serviceInput: &input,
			serviceErr:   serviceErr,
			wantStatus:   http.StatusInternalServerError,
			wantBody:     errorBody("internal server error"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service := newFakeTaskService(t)
			calls := 0
			if test.serviceInput != nil {
				service.createFn = func(ctx context.Context, gotInput model.TaskInput) (*model.Task, error) {
					calls++
					assertServiceContext(t, ctx)
					assertEqual(t, gotInput, *test.serviceInput)
					return test.serviceTask, test.serviceErr
				}
			}

			recorder := performRequest(t, newTaskMux(t, service), http.MethodPost, "/tasks", test.body)

			assertJSONResponse(t, recorder, test.wantStatus, test.wantBody)
			if test.serviceInput != nil {
				assertEqual(t, calls, 1)
			}
		})
	}
}

func TestTaskHandlerUpdate(t *testing.T) {
	input := model.TaskInput{Title: "Updated task", Description: "Done", Completed: true}
	task := &model.Task{ID: 8, Title: input.Title, Description: input.Description, Completed: true}
	serviceErr := errors.New("service failed")

	tests := []struct {
		name         string
		path         string
		body         string
		serviceID    *int64
		serviceInput model.TaskInput
		serviceTask  *model.Task
		serviceErr   error
		wantStatus   int
		wantBody     any
	}{
		{
			name:         "updates task",
			path:         "/tasks/8",
			body:         `{"title":"Updated task","description":"Done","completed":true}`,
			serviceID:    int64Pointer(8),
			serviceInput: input,
			serviceTask:  task,
			wantStatus:   http.StatusOK,
			wantBody:     task,
		},
		{
			name:       "returns 400 for invalid id",
			path:       "/tasks/not-a-number",
			body:       `{"title":"Updated task"}`,
			wantStatus: http.StatusBadRequest,
			wantBody:   errorBody("invalid task id"),
		},
		{
			name:       "returns 400 for malformed JSON",
			path:       "/tasks/8",
			body:       `{"title":`,
			wantStatus: http.StatusBadRequest,
			wantBody:   errorBody("could not decode body"),
		},
		{
			name:       "returns 400 for unknown field",
			path:       "/tasks/8",
			body:       `{"title":"Updated","priority":1}`,
			wantStatus: http.StatusBadRequest,
			wantBody:   errorBody("could not decode body"),
		},
		{
			name:       "returns 400 for a second JSON value",
			path:       "/tasks/8",
			body:       `{"title":"First"} {"title":"Second"}`,
			wantStatus: http.StatusBadRequest,
			wantBody:   errorBody("could not decode body"),
		},
		{
			name:         "returns 422 for invalid task",
			path:         "/tasks/8",
			body:         `{"title":""}`,
			serviceID:    int64Pointer(8),
			serviceInput: model.TaskInput{},
			serviceErr:   fmt.Errorf("update task: %w", model.ErrInvalidTask),
			wantStatus:   http.StatusUnprocessableEntity,
			wantBody:     errorBody("unprocessable entity"),
		},
		{
			name:         "returns 404 for missing task",
			path:         "/tasks/99",
			body:         `{"title":"Updated task","description":"Done","completed":true}`,
			serviceID:    int64Pointer(99),
			serviceInput: input,
			serviceErr:   fmt.Errorf("update task: %w", model.ErrTaskNotFound),
			wantStatus:   http.StatusNotFound,
			wantBody:     errorBody("task not found"),
		},
		{
			name:         "returns 500 for service failure",
			path:         "/tasks/8",
			body:         `{"title":"Updated task","description":"Done","completed":true}`,
			serviceID:    int64Pointer(8),
			serviceInput: input,
			serviceErr:   serviceErr,
			wantStatus:   http.StatusInternalServerError,
			wantBody:     errorBody("internal server error"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service := newFakeTaskService(t)
			calls := 0
			if test.serviceID != nil {
				service.updateFn = func(ctx context.Context, id int64, gotInput model.TaskInput) (*model.Task, error) {
					calls++
					assertServiceContext(t, ctx)
					assertEqual(t, id, *test.serviceID)
					assertEqual(t, gotInput, test.serviceInput)
					return test.serviceTask, test.serviceErr
				}
			}

			recorder := performRequest(t, newTaskMux(t, service), http.MethodPut, test.path, test.body)

			assertJSONResponse(t, recorder, test.wantStatus, test.wantBody)
			if test.serviceID != nil {
				assertEqual(t, calls, 1)
			}
		})
	}
}

func TestTaskHandlerDelete(t *testing.T) {
	serviceErr := errors.New("service failed")

	tests := []struct {
		name       string
		path       string
		serviceID  *int64
		serviceErr error
		wantStatus int
		wantBody   any
	}{
		{
			name:       "deletes task",
			path:       "/tasks/9",
			serviceID:  int64Pointer(9),
			wantStatus: http.StatusNoContent,
		},
		{
			name:       "returns 400 for invalid id",
			path:       "/tasks/not-a-number",
			wantStatus: http.StatusBadRequest,
			wantBody:   errorBody("invalid task id"),
		},
		{
			name:       "returns 404 for missing task",
			path:       "/tasks/99",
			serviceID:  int64Pointer(99),
			serviceErr: fmt.Errorf("delete task: %w", model.ErrTaskNotFound),
			wantStatus: http.StatusNotFound,
			wantBody:   errorBody("task not found"),
		},
		{
			name:       "returns 500 for service failure",
			path:       "/tasks/9",
			serviceID:  int64Pointer(9),
			serviceErr: serviceErr,
			wantStatus: http.StatusInternalServerError,
			wantBody:   errorBody("internal server error"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service := newFakeTaskService(t)
			calls := 0
			if test.serviceID != nil {
				service.deleteFn = func(ctx context.Context, id int64) error {
					calls++
					assertServiceContext(t, ctx)
					assertEqual(t, id, *test.serviceID)
					return test.serviceErr
				}
			}

			recorder := performRequest(t, newTaskMux(t, service), http.MethodDelete, test.path, "")

			if test.wantStatus == http.StatusNoContent {
				assertNoContentResponse(t, recorder)
			} else {
				assertJSONResponse(t, recorder, test.wantStatus, test.wantBody)
			}
			if test.serviceID != nil {
				assertEqual(t, calls, 1)
			}
		})
	}
}

type fakeDatabasePinger struct {
	pingContextFn func(context.Context) error
}

func (f *fakeDatabasePinger) PingContext(ctx context.Context) error {
	return f.pingContextFn(ctx)
}

func TestNewHealthHandler(t *testing.T) {
	database := &fakeDatabasePinger{
		pingContextFn: func(context.Context) error { return nil },
	}

	tests := []struct {
		name     string
		database handler.DatabasePinger
		wantErr  bool
	}{
		{name: "accepts database", database: database},
		{name: "rejects nil database", wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := handler.NewHealthHandler(test.database)
			if test.wantErr {
				if err == nil {
					t.Fatal("NewHealthHandler() error = nil, want an error")
				}
				if got != nil {
					t.Error("NewHealthHandler() handler is non-nil, want nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("NewHealthHandler() error = %v", err)
			}
			if got == nil {
				t.Fatal("NewHealthHandler() handler = nil, want a handler")
			}
		})
	}
}

func TestHealthHandler(t *testing.T) {
	tests := []struct {
		name       string
		pingErr    error
		wantStatus int
		wantBody   any
	}{
		{
			name:       "reports healthy database",
			wantStatus: http.StatusOK,
			wantBody:   map[string]string{"status": "ok"},
		},
		{
			name:       "reports unavailable database",
			pingErr:    errors.New("ping failed"),
			wantStatus: http.StatusServiceUnavailable,
			wantBody:   errorBody("database unavailable"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			calls := 0
			var pingContext context.Context
			database := &fakeDatabasePinger{
				pingContextFn: func(ctx context.Context) error {
					calls++
					pingContext = ctx
					assertServiceContext(t, ctx)
					deadline, ok := ctx.Deadline()
					if !ok {
						t.Error("PingContext() context has no deadline")
					} else if remaining := time.Until(deadline); remaining <= 0 || remaining > 2*time.Second {
						t.Errorf("PingContext() deadline remaining = %v, want within (0, 2s]", remaining)
					}
					return test.pingErr
				},
			}
			healthHandler, err := handler.NewHealthHandler(database)
			if err != nil {
				t.Fatalf("NewHealthHandler() error = %v", err)
			}

			recorder := performRequest(t, healthHandler, http.MethodGet, "/health", "")

			assertJSONResponse(t, recorder, test.wantStatus, test.wantBody)
			assertEqual(t, calls, 1)
			select {
			case <-pingContext.Done():
			default:
				t.Error("PingContext() context was not canceled after the handler returned")
			}
		})
	}
}

func int64Pointer(value int64) *int64 {
	return &value
}

type requestContextKey struct{}
