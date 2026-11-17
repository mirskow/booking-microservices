package user

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

const maxRequestBodySize = 1 << 20 // 1 MiB

type ProfileService interface {
	Create(ctx context.Context, req CreateProfileRequest) (*Profile, error)
	GetByID(ctx context.Context, id uuid.UUID) (*Profile, error)
	Update(ctx context.Context, id uuid.UUID, req UpdateProfileRequest) (*Profile, error)
}

type Handler struct {
	service ProfileService
}

func NewHandler(service ProfileService) *Handler {
	return &Handler{
		service: service,
	}
}

type errorResponse struct {
	Error string `json:"error"`
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateProfileRequest

	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	profile, err := h.service.Create(r.Context(), req)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, profile)
}

func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := parseProfileID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	profile, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, profile)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := parseProfileID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	var req UpdateProfileRequest

	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	profile, err := h.service.Update(r.Context(), id, req)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, profile)
}

func parseProfileID(r *http.Request) (uuid.UUID, error) {
	rawID := r.PathValue("id")

	id, err := uuid.Parse(rawID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid profile ID")
	}

	return id, nil
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	contentType := r.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "application/json") {
		return fmt.Errorf("Content-Type must be application/json")
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(dst); err != nil {
		return fmt.Errorf("invalid JSON request body")
	}

	// Проверяем, что после первого JSON-объекта нет второго.
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return fmt.Errorf("request body must contain a single JSON value")
	}

	return nil
}

func (h *Handler) handleServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrInvalidInput):
		writeError(w, http.StatusBadRequest, err.Error())

	case errors.Is(err, ErrNotFound):
		writeError(w, http.StatusNotFound, "profile not found")

	case errors.Is(err, ErrEmailExists):
		writeError(w, http.StatusConflict, "email already exists")

	default:
		writeError(w, http.StatusInternalServerError, "internal server error")
	}
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		// После WriteHeader изменить HTTP-статус уже нельзя.
		// Ошибку записи здесь следует логировать на уровне инфраструктуры.
		return
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, errorResponse{
		Error: message,
	})
}
