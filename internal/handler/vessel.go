package handler

import (
	"encoding/json"
	"net/http"

	"github.com/ove4lo/ship-cargo-service/internal/model"
	"github.com/ove4lo/ship-cargo-service/internal/repository"
)

// VesselHandler handles HTTP requests related to vessel management
type VesselHandler struct {
	vesselRepo *repository.VesselRepository
}

// NewVesselHandler creates a new instance of VesselHandler
func NewVesselHandler(vesselRepo *repository.VesselRepository) *VesselHandler {
	return &VesselHandler{vesselRepo: vesselRepo}
}

// createVesselRequest represents the HTTP request payload for creating a vessel
type createVesselRequest struct {
	Name string `json:"name"`
	MaxWeightKg float64 `json:"max_weight_kg"`
	MaxVolumeM3 float64 `json:"max_volume_m3"`
}

// Create handles the HTTP request to create and store a new vessel
func (h *VesselHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createVesselRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid json"}`, http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		http.Error(w, `{"error":"name is required"}`, http.StatusBadRequest)
		return
	}

	if req.MaxWeightKg <= 0 || req.MaxVolumeM3 <= 0 {
		http.Error(w, `{"error":"weight and volume must be positive"}`, http.StatusBadRequest)
		return
	}

	vessel := &model.Vessel{
		Name: req.Name,
		MaxWeightKg: req.MaxWeightKg,
		MaxVolumeM3: req.MaxVolumeM3,
	}

	if err := h.vesselRepo.Create(r.Context(), vessel); err != nil {
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(vessel)
}

// GetAll handles the HTTP request to retrieve all vessels from the database
func (h *VesselHandler) GetAll(w http.ResponseWriter, r *http.Request) {
	vessels, err := h.vesselRepo.GetAll(r.Context())
	if err != nil {
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}

	// If an empty slice (nil) is returned from the database, this code initializes it as an empty array []
	if vessels == nil {
		vessels = []model.Vessel{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(vessels)
}