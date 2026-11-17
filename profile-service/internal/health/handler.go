package health

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

type DBChecker interface {
	Ping(ctx context.Context) error
}

type Handler struct {
	db DBChecker
}

type Response struct {
	Status   string `json:"status"`
	Database string `json:"database,omitempty"`
}

func NewHandler(db DBChecker) *Handler {
	return &Handler{
		db: db,
	}
}

// Live checks whether the application process is alive.
func (h *Handler) Live(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, Response{
		Status: "alive",
	})
}

// Ready checks whether the application can handle requests.
func (h *Handler) Ready(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	if err := h.db.Ping(ctx); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, Response{
			Status:   "not_ready",
			Database: "unavailable",
		})
		return
	}

	writeJSON(w, http.StatusOK, Response{
		Status:   "ready",
		Database: "connected",
	})
}

func writeJSON(w http.ResponseWriter, status int, response Response) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)

	_ = json.NewEncoder(w).Encode(response)
}
