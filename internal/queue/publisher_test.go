package queue

import (
	"encoding/json"
	"testing"
	"time"
)

// TestBookingEvent_Marshal verifies that a BookingEvent is accurately encoded into JSON 
// and correctly restored back into a structural model with matching fields.
func TestBookingEvent_Marshal(t *testing.T) {
	event := BookingEvent{
		BookingID:  "booking-001",
		VoyageID:   "voyage-001",
		UserID:     "user-001",
		Status:     "confirmed",
		ItemCount:  3,
		OccurredAt: time.Date(2026, 9, 15, 10, 0, 0, 0, time.UTC),
	}

	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var decoded BookingEvent
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if decoded.BookingID != event.BookingID {
		t.Errorf("booking_id: want %s, got %s", event.BookingID, decoded.BookingID)
	}

	if decoded.Status != event.Status {
		t.Errorf("status: want %s, got %s", event.Status, decoded.Status)
	}

	if decoded.ItemCount != event.ItemCount {
		t.Errorf("item_count: want %d, got %d", event.ItemCount, decoded.ItemCount)
	}
}

// TestBookingEvent_UnmarshalInvalid asserts that decoding a JSON payload with mismatched data types 
// (e.g., an integer assigned to a string field) returns a parsing error.
func TestBookingEvent_UnmarshalInvalid(t *testing.T) {
	invalid := []byte(`{"booking_id": 123}`) // booking_id must be string

	var event BookingEvent
	err := json.Unmarshal(invalid, &event)

	if err == nil {
		t.Error("expected error for invalid json, got nil")
	}
}