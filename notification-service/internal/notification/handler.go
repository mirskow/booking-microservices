package notification

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"strconv"

	"github.com/google/uuid"
)

type NotificationService interface {
	ProcessEvent(
		ctx context.Context,
		event BookingEvent,
	) (*EventResponse, error)

	GetByID(
		ctx context.Context,
		id uuid.UUID,
	) (*Notification, error)

	List(
		ctx context.Context,
		recipientID uuid.UUID,
		limit int,
		offset int,
	) ([]Notification, error)
}

type Handler struct {
	service NotificationService
}

func NewHandler(service NotificationService) *Handler {
	return &Handler{
		service: service,
	}
}

const maxRequestBodySize = 1 << 20 // 1 MiB

func (h *Handler) ProcessEvent(w http.ResponseWriter, r *http.Request) {
	var event BookingEvent

	if err := decodeJSON(w, r, &event); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON request")
		return
	}

	response, err := h.service.ProcessEvent(r.Context(), event)
	if err != nil {
		log.Printf("ProcessEvent failed: event_id=%s, error=%v", event.EventID, err)
		if errors.Is(err, ErrInvalidInput) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}

		if errors.Is(err, ErrDeliveryFailed) {
			writeJSON(w, http.StatusBadGateway, map[string]any{
				"error":         "some notifications could not be delivered",
				"event_id":      response.EventID,
				"notifications": response.Notifications,
			})
			return
		}

		writeError(w, http.StatusInternalServerError, "failed to process event")
		return
	}

	writeJSON(w, http.StatusOK, response)
}

func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid notification ID")
		return
	}

	notification, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			writeError(w, http.StatusNotFound, "notification not found")
			return
		}

		if errors.Is(err, ErrInvalidInput) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}

		writeError(w, http.StatusInternalServerError, "failed to get notification")
		return
	}

	writeJSON(w, http.StatusOK, notification)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	recipientID, err := uuid.Parse(query.Get("recipient_id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid recipient_id")
		return
	}

	limit := 20
	offset := 0

	if value := query.Get("limit"); value != "" {
		limit, err = strconv.Atoi(value)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid limit")
			return
		}
	}

	if value := query.Get("offset"); value != "" {
		offset, err = strconv.Atoi(value)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid offset")
			return
		}
	}

	notifications, err := h.service.List(
		r.Context(),
		recipientID,
		limit,
		offset,
	)
	if err != nil {
		if errors.Is(err, ErrInvalidInput) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}

		writeError(w, http.StatusInternalServerError, "failed to list notifications")
		return
	}

	writeJSON(w, http.StatusOK, notifications)
}

type errorResponse struct {
	Error string `json:"error"`
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		// Response уже отправлен, изменить HTTP status невозможно.
		return
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, errorResponse{
		Error: message,
	})
}

func decodeJSON(
	w http.ResponseWriter,
	r *http.Request,
	dst any,
) error {
	contentType := r.Header.Get("Content-Type")

	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil || mediaType != "application/json" {
		return fmt.Errorf("content type must be application/json")
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(dst); err != nil {
		return fmt.Errorf("decode JSON: %w", err)
	}

	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return fmt.Errorf("request must contain a single JSON object")
	}

	return nil
}
