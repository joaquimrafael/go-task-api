package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/joaquimrafael/go-task-api/internal/handler"
	"github.com/joaquimrafael/go-task-api/internal/model"
	"github.com/joaquimrafael/go-task-api/internal/repository"
	"github.com/joaquimrafael/go-task-api/internal/service"
)

func newIntegrationRouter(t *testing.T) http.Handler {
	t.Helper()

	databasePath := filepath.Join(t.TempDir(), "tasks.db")
	database, err := repository.OpenSQLite(databasePath)
	if err != nil {
		t.Fatalf("OpenSQLite() error = %v", err)
	}
	t.Cleanup(func() {
		if err := database.Close(); err != nil {
			t.Errorf("close test database: %v", err)
		}
	})

	taskRepository, err := repository.NewSQLiteTaskRepository(database)
	if err != nil {
		t.Fatalf("NewSQLiteTaskRepository() error = %v", err)
	}
	taskService, err := service.NewTaskService(taskRepository)
	if err != nil {
		t.Fatalf("NewTaskService() error = %v", err)
	}
	taskHandler, err := handler.NewTaskHandler(taskService)
	if err != nil {
		t.Fatalf("NewTaskHandler() error = %v", err)
	}
	healthHandler, err := handler.NewHealthHandler(database)
	if err != nil {
		t.Fatalf("NewHealthHandler() error = %v", err)
	}

	return newRouter(taskHandler, healthHandler)
}

func requestAPI(
	t *testing.T,
	router http.Handler,
	method string,
	path string,
	body string,
	wantStatus int,
) []byte {
	t.Helper()

	request := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != wantStatus {
		t.Fatalf(
			"%s %s status = %d, want %d; body = %q",
			method,
			path,
			recorder.Code,
			wantStatus,
			recorder.Body.String(),
		)
	}

	return recorder.Body.Bytes()
}

func decodeResponse[T any](t *testing.T, body []byte) T {
	t.Helper()

	var value T
	if err := json.Unmarshal(body, &value); err != nil {
		t.Fatalf("decode response %q: %v", body, err)
	}
	return value
}

func TestAPIIntegration(t *testing.T) {
	router := newIntegrationRouter(t)

	healthBody := requestAPI(t, router, http.MethodGet, "/health", "", http.StatusOK)
	health := decodeResponse[map[string]string](t, healthBody)
	if health["status"] != "ok" {
		t.Errorf("health status = %q, want %q", health["status"], "ok")
	}

	createBody := requestAPI(
		t,
		router,
		http.MethodPost,
		"/tasks",
		`{"title":"  Learn integration tests  ","description":"Test the full stack"}`,
		http.StatusCreated,
	)
	created := decodeResponse[model.Task](t, createBody)
	if created.ID <= 0 {
		t.Fatalf("created task ID = %d, want a positive ID", created.ID)
	}
	if created.Title != "Learn integration tests" {
		t.Errorf("created title = %q, want trimmed title", created.Title)
	}
	if created.CreatedAt.IsZero() || created.UpdatedAt.IsZero() {
		t.Error("created task timestamps were not populated")
	}

	taskPath := fmt.Sprintf("/tasks/%d", created.ID)
	getBody := requestAPI(t, router, http.MethodGet, taskPath, "", http.StatusOK)
	got := decodeResponse[model.Task](t, getBody)
	if got.ID != created.ID || got.Title != created.Title || got.Description != created.Description {
		t.Errorf("retrieved task = %#v, want created task %#v", got, created)
	}

	listBody := requestAPI(t, router, http.MethodGet, "/tasks", "", http.StatusOK)
	tasks := decodeResponse[[]model.Task](t, listBody)
	if len(tasks) != 1 || tasks[0].ID != created.ID {
		t.Errorf("listed tasks = %#v, want only task %d", tasks, created.ID)
	}

	updateBody := requestAPI(
		t,
		router,
		http.MethodPut,
		taskPath,
		`{"title":"Integration tests complete","description":"Test the full stack","completed":true}`,
		http.StatusOK,
	)
	updated := decodeResponse[model.Task](t, updateBody)
	if updated.Title != "Integration tests complete" || !updated.Completed {
		t.Errorf("updated task = %#v, want updated title and completed state", updated)
	}

	deleteBody := requestAPI(t, router, http.MethodDelete, taskPath, "", http.StatusNoContent)
	if len(deleteBody) != 0 {
		t.Errorf("delete response body = %q, want empty body", deleteBody)
	}

	missingBody := requestAPI(t, router, http.MethodGet, taskPath, "", http.StatusNotFound)
	missing := decodeResponse[map[string]string](t, missingBody)
	if missing["error"] != "task not found" {
		t.Errorf("missing task error = %q, want %q", missing["error"], "task not found")
	}
}
