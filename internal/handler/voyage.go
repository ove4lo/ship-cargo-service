package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/ove4lo/ship-cargo-service/internal/model"
	"github.com/ove4lo/ship-cargo-service/internal/repository"
)

// VoyageHandler handles HTTP requests related to voyage management
type VoyageHandler struct {
	voyageRepo *repository.VoyageRepository
}

// NewVoyageHandler creates a new instance of VoyageHandler
func NewVoyageHandler(voyageRepo *repository.VoyageRepository) *VoyageHandler {
	return &VoyageHandler{voyageRepo: voyageRepo}
}

// createVoyageRequest represents the HTTP request payload for creating a voyage
type createVoyageRequest struct {
	VesselID string `json:"vessel_id"`
	Route string `json:"route"`
	DepartureDate string `json:"departure_date"`
}

// voyageResponse defines the HTTP JSON output structure for voyage data, 
// including formatted dates and calculated remaining capacities
type voyageResponse struct {
	ID string  `json:"id"`
	VesselID string  `json:"vessel_id"`
	VesselName  string  `json:"vessel_name"`
	Route string  `json:"route"`
	DepartureDate string  `json:"departure_date"`
	Status string  `json:"status"`
	FreeWeightKg float64 `json:"free_weight_kg"`
	FreeVolumeM3 float64 `json:"free_volume_m3"`
	MaxWeightKg float64 `json:"max_weight_kg"`
	MaxVolumeM3 float64 `json:"max_volume_m3"`
}

// toVoyageResponse maps a raw Voyage domain model to a formatted voyageResponse
func toVoyageResponse(v model.Voyage) voyageResponse {
	return voyageResponse{
		ID: v.ID,
		VesselID: v.VesselID,
		VesselName: v.VesselName,
		Route: v.Route,
		DepartureDate: v.DepartureDate.Format("2006-01-02"),
		Status: string(v.Status),
		FreeWeightKg: v.FreeWeightKg(),
		FreeVolumeM3: v.FreeVolumeM3(),
		MaxWeightKg: v.MaxWeightKg,
		MaxVolumeM3: v.MaxVolumeM3,
	}
}

// Create handles the HTTP request to create and store a new voyage
func (h *VoyageHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createVoyageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid json"}`, http.StatusBadRequest)
		return
	}

	if req.VesselID == "" || req.Route == "" || req.DepartureDate == "" {
		http.Error(w, `{"error":"vessel_id, route and departure_date are required"}`, http.StatusBadRequest)
		return
	}

	depDate, err := time.Parse("2006-01-02", req.DepartureDate)
	if err != nil {
		http.Error(w, `{"error":"departure_date must be YYYY-MM-DD"}`, http.StatusBadRequest)
		return
	}

	voyage := &model.Voyage{
		VesselID: req.VesselID,
		Route: req.Route,
		DepartureDate: depDate,
	}

	if err := h.voyageRepo.Create(r.Context(), voyage); err != nil {
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}

	full, err := h.voyageRepo.GetByID(r.Context(), voyage.ID)
	if err != nil {
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	// Convert to voyageResponse before sending to the client
	json.NewEncoder(w).Encode(toVoyageResponse(*full))
}

// GetAll handles the HTTP request to retrieve all voyages from the database
func (h *VoyageHandler) GetAll(w http.ResponseWriter, r *http.Request) {
	voyages, err := h.voyageRepo.GetAll(r.Context())
	if err != nil {
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}

	if voyages == nil {
		voyages = []model.Voyage{}
	}

	response := make([]voyageResponse, len(voyages))
	for i, v := range voyages {
		response[i] = toVoyageResponse(v)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
