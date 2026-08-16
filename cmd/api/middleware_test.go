package main

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequestLogger(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		path       string
		handler    http.Handler
		wantStatus int
	}{
		{
			name:   "implicit 200",
			method: http.MethodGet,
			path:   "/tasks",
			handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte("ok"))
			}),
			wantStatus: http.StatusOK,
		},
		{
			name:   "explicit 201",
			method: http.MethodPost,
			path:   "/tasks",
			handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusCreated)
			}),
			wantStatus: http.StatusCreated,
		},
		{
			name:   "explicit 204",
			method: http.MethodDelete,
			path:   "/tasks/42",
			handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusNoContent)
			}),
			wantStatus: http.StatusNoContent,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var logs bytes.Buffer

			logger := slog.New(slog.NewJSONHandler(&logs, nil))
			handler := requestLogger(logger, test.handler)

			request := httptest.NewRequest(test.method, test.path, nil)
			recorder := httptest.NewRecorder()

			handler.ServeHTTP(recorder, request)

			if recorder.Code != test.wantStatus {
				t.Errorf(
					"response status = %d, want %d",
					recorder.Code,
					test.wantStatus,
				)
			}

			var entry map[string]any
			if err := json.Unmarshal(logs.Bytes(), &entry); err != nil {
				t.Fatalf("decode log: %v\nlog: %s", err, logs.String())
			}

			if got := entry["method"]; got != test.method {
				t.Errorf("logged method = %v, want %s", got, test.method)
			}

			if got := entry["path"]; got != test.path {
				t.Errorf("logged path = %v, want %s", got, test.path)
			}

			if got := entry["status"]; got != float64(test.wantStatus) {
				t.Errorf(
					"logged status = %v, want %d",
					got,
					test.wantStatus,
				)
			}

			if _, exists := entry["duration"]; !exists {
				t.Error("log does not contain duration")
			}
		})
	}
}
