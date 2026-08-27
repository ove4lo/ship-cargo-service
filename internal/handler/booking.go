package handler
import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/ove4lo/ship-cargo-service/internal/middleware"
	"github.com/ove4lo/ship-cargo-service/internal/model"
	"github.com/ove4lo/ship-cargo-service/internal/service"
)

// BookingHandler manages HTTP requests for allocating cargo space on voyages.
type BookingHandler struct {
	bookingService *service.BookingService
}

// NewBookingHandler creates and initializes a new BookingHandler with its service dependency.
func NewBookingHandler(bookingService *service.BookingService) *BookingHandler {
	return &BookingHandler{bookingService: bookingService}
}

type createBookingRequest struct {
	VoyageID string `json:"voyage_id"`
	Priority model.BookingPriority `json:"priority"`
	IdempotencyKey string `json:"idempotency_key"`
	Items []bookingItemRequest `json:"items"`
}

type bookingItemRequest struct {
	Description string `json:"description"`
	WeightKg float64 `json:"weight_kg"`
	VolumeM3 float64 `json:"volume_m3"`
}

// Create handles incoming POST requests to book cargo space, enforcing payload constraints.
func (h *BookingHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createBookingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid json"}`, http.StatusBadRequest)
		return
	}

	if req.VoyageID == "" || req.IdempotencyKey == "" || len(req.Items) == 0 {
		http.Error(w, `{"error":"voyage_id, idempotency_key and items are required"}`, http.StatusBadRequest)
		return
	}

	if req.Priority == "" {
		req.Priority = model.PriorityNormal
	}

	for _, item := range req.Items {
		if item.Description == "" || item.WeightKg <= 0 || item.VolumeM3 <= 0 {
			http.Error(w, `{"error":"each item needs description, positive weight_kg and volume_m3"}`, http.StatusBadRequest)
			return
		}
	}

	userID := r.Context().Value(middleware.UserIDKey).(string)

	items := make([]service.BookingItemRequest, len(req.Items))
	for i, ri := range req.Items {
		items[i] = service.BookingItemRequest{
			Description: ri.Description,
			WeightKg: ri.WeightKg,
			VolumeM3: ri.VolumeM3,
		}
	}

	booking, err := h.bookingService.CreateBooking(r.Context(), service.BookingRequest{
		VoyageID: req.VoyageID,
		UserID: userID,
		Priority: req.Priority,
		IdempotencyKey: req.IdempotencyKey,
		Items: items,
	})
	if err != nil {
		switch {
		case errors.Is(err, service.ErrorVoyageNotAvailable):
			http.Error(w, `{"error":"voyage isn't available for booking"}`, http.StatusConflict)
		case errors.Is(err, service.ErrNoCapacity):
			http.Error(w, `{"error":"no capacity available"}`, http.StatusConflict)
		case errors.Is(err, service.ErrVoyageLocked):
			http.Error(w, `{"error":"voyage is being booked, try again"}`, http.StatusConflict)
		default:
			http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(booking)
}