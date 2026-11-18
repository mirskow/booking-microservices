package health

import (
	"context"
	"encoding/json"
	"net/http"
)

type DBChecker interface {
	Ping(ctx context.Context) error
}

type Handler struct {
	db DBChecker
}

func NewHandler(db DBChecker) *Handler {
	return &Handler{
		db: db,
	}
}

type healthResponse struct {
	Status   string `json:"status"`
	Database string `json:"database,omitempty"`
}

func (h *Handler) Live(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, healthResponse{
		Status: "alive",
	})
}

func (h *Handler) Ready(w http.ResponseWriter, r *http.Request) {
	if err := h.db.Ping(r.Context()); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, healthResponse{
			Status:   "not_ready",
			Database: "unavailable",
		})
		return
	}

	writeJSON(w, http.StatusOK, healthResponse{
		Status:   "ready",
		Database: "connected",
	})
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	_ = json.NewEncoder(w).Encode(data)
}
