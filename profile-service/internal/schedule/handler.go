package schedule

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/google/uuid"
)

const maxRequestBodySize = 1 << 20 // 1 MiB

type ScheduleService interface {
	SetSchedule(
		ctx context.Context,
		userID uuid.UUID,
		req SetScheduleRequest,
	) ([]AvailabilityRule, error)

	GetSchedule(
		ctx context.Context,
		userID uuid.UUID,
	) ([]AvailabilityRule, error)

	GetSlots(
		ctx context.Context,
		userID uuid.UUID,
		date string,
	) (*SlotsResponse, error)
}

type Handler struct {
	service ScheduleService
}

func NewHandler(service ScheduleService) *Handler {
	return &Handler{
		service: service,
	}
}

type errorResponse struct {
	Error string `json:"error"`
}

func (h *Handler) SetSchedule(w http.ResponseWriter, r *http.Request) {
	userID, err := parseUserID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	var req SetScheduleRequest

	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	rules, err := h.service.SetSchedule(r.Context(), userID, req)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"user_id": userID,
		"rules":   rules,
	})
}

func (h *Handler) GetSchedule(w http.ResponseWriter, r *http.Request) {
	userID, err := parseUserID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	rules, err := h.service.GetSchedule(r.Context(), userID)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"user_id": userID,
		"rules":   rules,
	})
}

func (h *Handler) GetSlots(w http.ResponseWriter, r *http.Request) {
	userID, err := parseUserID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	date := r.URL.Query().Get("date")
	if date == "" {
		writeError(w, http.StatusBadRequest, "date query parameter is required")
		return
	}

	slots, err := h.service.GetSlots(r.Context(), userID, date)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, slots)
}

func parseUserID(r *http.Request) (uuid.UUID, error) {
	rawID := r.PathValue("id")

	id, err := uuid.Parse(rawID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid user ID")
	}

	return id, nil
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	if r.Header.Get("Content-Type") != "application/json" {
		return fmt.Errorf("Content-Type must be application/json")
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(dst); err != nil {
		return fmt.Errorf("invalid JSON request body")
	}

	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return fmt.Errorf("request body must contain a single JSON value")
	}

	return nil
}

func (h *Handler) handleServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrInvalidInput):
		writeError(w, http.StatusBadRequest, err.Error())

	case errors.Is(err, ErrUserNotFound):
		writeError(w, http.StatusNotFound, "user not found")

	default:
		writeError(w, http.StatusInternalServerError, "internal server error")
	}
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)

	_ = json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, errorResponse{
		Error: message,
	})
}
