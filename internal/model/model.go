package model

import "time"

type Role string

const (
	RoleSupplier Role = "supplier"
	RoleManager Role = "manager"
)

/** Statuses defined as typed strings
	The compiler will prevent you from accidentally passing a `BookingStatus`
	where a `VoyageStatus` is expected
*/
type VoyageStatus string

const (
	VoyageStatusPlanned VoyageStatus = "planned"
	VoyageStatusLoading VoyageStatus = "loading"
	VoyageStatusDeparted VoyageStatus = "departed"
	VoyageStatusCompleted VoyageStatus = "completed"
)

type BookingStatus string

const (
	BookingStatusPending BookingStatus = "pending"
	BookingStatusConfirmed BookingStatus = "confirmed"
	BookingStatusPartial BookingStatus = "partial"
	BookingStatusRejected BookingStatus = "rejected"
	BookingStatusCancelled BookingStatus = "cancelled"
)

type BookingPriority string

const (
	PriorityUrgent BookingPriority = "urgent"
	PriorityNormal BookingPriority = "normal"
	PriorityLow BookingPriority = "low"
)

type ItemStatus string

const (
	ItemStatusPending ItemStatus = "pending"
	ItemStatusPlaced ItemStatus = "placed"
	ItemStatusWaitlisted ItemStatus = "waitlisted"
)

// represents information of user
type User struct {
	ID string `json:"id"`
	Name string `json:"name"`
	Email string `json:"email"`
	PasswordHash string `json:"-"` // NOTE: When serializing to JSON, this field won't be included in the response
	Role Role `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

// represents information of ship
type Vessel struct {
	ID string `json:"id"`
	Name string `json:"name"`
	MaxWeightKg float64 `json:"max_weight_kg"`
	MaxVolumeM3 float64 `json:"max_volume_m3"`
	CreatedAt time.Time `json:"created_at"`
}

// represents information of voyage (departs from and to)
type Voyage struct {
	ID string `json:"id"`
	VesselID string `json:"vessel_id"`
	Route string `json:"route"`
	DepartureDate time.Time `json:"departure_date"`
	Status VoyageStatus `json:"status"`
	ReservedWeightKg float64 `json:"reserved_weight_kg"`
	ReservedVolumeM3 float64 `json:"reserved_volume_m3"`
	CreatedAt time.Time `json:"created_at"`

	// Joined fields(JOIN with vessels)
	VesselName string `json:"vessel_name,omitempty"`
	MaxWeightKg float64 `json:"max_weight_kg,omitempty"`
	MaxVolumeM3 float64 `json:"max_volume_m3,omitempty"`
}

// represents information of seat reservation
type Booking struct {
	ID string `json:"id"`
	VoyageID string `json:"voyage_id"`
	UserID string `json:"user_id"`
	Priority BookingPriority `json:"priority"`
	Status BookingStatus `json:"status"`
	IdempotencyKey string `json:"idempotency_key"`
	CreatedAt time.Time `json:"created_at"`
	Items []BookingItem `json:"items,omitempty"`
}

// represents information of seat reservation's item
type BookingItem struct {
	ID string `json:"id"`
	BookingID string `json:"booking_id"`
	Description string `json:"description"`
	WeightKg float64 `json:"weight_kg"`
	VolumeM3 float64 `json:"volume_m3"`
	Status ItemStatus `json:"status"`
}

// FreeWeightKg calculates free weight
func (v Voyage) FreeWeightKg() float64 {
	return v.MaxWeightKg - v.ReservedWeightKg
}

// FreeVolumeM3 calculates free volume
func (v Voyage) FreeVolumeM3() float64 {
	return v.MaxVolumeM3 - v.ReservedVolumeM3
}

