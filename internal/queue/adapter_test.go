package queue

import (
	"context"
	"testing"
	"time"

	"github.com/ove4lo/ship-cargo-service/internal/service"
)

// spyPublisher acts as a test spy to capture the parameters of internal message publishing.
type spyPublisher struct {
	lastEvent BookingEvent
	called    bool
}

// PublishBookingEvent records the event payload and flips the called invocation flag to true.
func (s *spyPublisher) PublishBookingEvent(_ context.Context, event BookingEvent) error {
	s.lastEvent = event
	s.called = true
	return nil
}

// TestPublisherAdapter_MapsAllFields asserts that the PublisherAdapter accurately 
// translates all core fields from a service.BookingEvent into an infrastructure queue.BookingEvent.
func TestPublisherAdapter_MapsAllFields(t *testing.T) {
	spy := &spyPublisher{}

	adapter := &PublisherAdapter{pub: spy}
	
	input := service.BookingEvent{
		BookingID:  "b-001",
		VoyageID:   "v-001",
		UserID:     "u-001",
		Status:     "partial",
		ItemCount:  5,
		OccurredAt: time.Date(2026, 8, 28, 12, 0, 0, 0, time.UTC),
	}

	err := adapter.PublishBookingEvent(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected adapter error: %v", err)
	}

	if !spy.called {
		t.Fatalf("publisher wasn't called")
	}

	if spy.lastEvent.BookingID != input.BookingID {
		t.Errorf("booking_id: want %s, got %s", input.BookingID, spy.lastEvent.BookingID)
	}

	if spy.lastEvent.VoyageID != input.VoyageID {
		t.Errorf("voyage_id: want %s, got %s", input.VoyageID, spy.lastEvent.VoyageID)
	}

	if spy.lastEvent.UserID != input.UserID {
		t.Errorf("user_id: want %s, got %s", input.UserID, spy.lastEvent.UserID)
	}

	if spy.lastEvent.Status != input.Status {
		t.Errorf("status: want %s, got %s", input.Status, spy.lastEvent.Status)
	}

	if spy.lastEvent.ItemCount != input.ItemCount {
		t.Errorf("item_count: want %d, got %d", input.ItemCount, spy.lastEvent.ItemCount)
	}
}