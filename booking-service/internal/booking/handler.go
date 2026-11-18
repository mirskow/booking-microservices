package booking

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/mirskov/booking-microservices/booking-service/internal/clients"
)

const maxRequestBodySize = 1 << 20 // 1 MB

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{
		service: service,
	}
}

// RegisterRoutes регистрирует HTTP-маршруты booking-service.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /bookings", h.Create)
	mux.HandleFunc("GET /bookings/{id}", h.GetByID)
	mux.HandleFunc("GET /bookings", h.ListByUser)
	mux.HandleFunc("DELETE /bookings/{id}", h.Cancel)
}

type errorResponse struct {
	Error string `json:"error"`
}

// POST /bookings
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateBookingRequest

	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	booking, err := h.service.Create(r.Context(), req)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, booking)
}

// GET /bookings/{id}
func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid booking ID")
		return
	}

	booking, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, booking)
}

// GET /bookings?user_id=...&limit=20&offset=0
func (h *Handler) ListByUser(w http.ResponseWriter, r *http.Request) {
	userID, err := parseUUID(r.URL.Query().Get("user_id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid or missing user_id")
		return
	}

	limit, err := parseQueryInt(r, "limit", 20)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid limit")
		return
	}

	offset, err := parseQueryInt(r, "offset", 0)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid offset")
		return
	}

	bookings, err := h.service.ListByUser(
		r.Context(),
		userID,
		limit,
		offset,
	)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	response := BookingListResponse{
		Bookings: bookings,
	}

	writeJSON(w, http.StatusOK, response)
}

// DELETE /bookings/{id}?requester_id=...
func (h *Handler) Cancel(w http.ResponseWriter, r *http.Request) {
	bookingID, err := parseUUID(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid booking ID")
		return
	}

	requesterID, err := parseUUID(
		r.URL.Query().Get("requester_id"),
	)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid or missing requester_id")
		return
	}

	booking, err := h.service.Cancel(
		r.Context(),
		bookingID,
		requesterID,
	)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, booking)
}

// -------------------- Helpers --------------------

func decodeJSON(
	w http.ResponseWriter,
	r *http.Request,
	dst any,
) error {
	if r.Body == nil {
		return errors.New("request body is required")
	}

	r.Body = http.MaxBytesReader(
		w,
		r.Body,
		maxRequestBodySize,
	)

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(dst); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}

	// Запрещаем несколько JSON-объектов в одном body.
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return errors.New("request body must contain only one JSON object")
		}

		return fmt.Errorf("invalid JSON: %w", err)
	}

	return nil
}

func parseUUID(value string) (uuid.UUID, error) {
	return uuid.Parse(strings.TrimSpace(value))
}

func parseQueryInt(
	r *http.Request,
	name string,
	defaultValue int,
) (int, error) {
	value := r.URL.Query().Get(name)

	if value == "" {
		return defaultValue, nil
	}

	return strconv.Atoi(value)
}

func handleServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrInvalidInput):
		writeError(w, http.StatusBadRequest, err.Error())

	case errors.Is(err, ErrUserNotFound):
		writeError(w, http.StatusNotFound, err.Error())

	case errors.Is(err, ErrNotFound):
		writeError(w, http.StatusNotFound, err.Error())

	case errors.Is(err, ErrForbidden):
		writeError(w, http.StatusForbidden, err.Error())

	case errors.Is(err, ErrSlotUnavailable):
		writeError(w, http.StatusConflict, err.Error())

	case errors.Is(err, ErrConflict):
		writeError(w, http.StatusConflict, err.Error())

	case errors.Is(err, clients.ErrDependencyUnavailable):
		log.Printf("booking dependency unavailable: %v", err)
		writeError(
			w,
			http.StatusServiceUnavailable,
			"dependent service unavailable",
		)

	default:
		log.Printf("booking service error: %v", err)
		writeError(
			w,
			http.StatusInternalServerError,
			"internal server error",
		)
	}
}

func writeJSON(
	w http.ResponseWriter,
	status int,
	data any,
) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		// Ошибка возникает уже после отправки HTTP-заголовков.
		return
	}
}

func writeError(
	w http.ResponseWriter,
	status int,
	message string,
) {
	writeJSON(w, status, errorResponse{
		Error: message,
	})
}
