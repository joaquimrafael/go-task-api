package handler

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// DatabasePinger describes the database availability check required by the health handler.
type DatabasePinger interface {
	PingContext(ctx context.Context) error
}

type healthResponse struct {
	Status string `json:"status"`
}

// NewHealthHandler returns a handler that reports whether the database is reachable.
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
