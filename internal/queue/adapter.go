package queue

import (
	"context"

	"github.com/ove4lo/ship-cargo-service/internal/service"
)

// PublisherAdapter bridges the infrastructure queue publisher with the domain service layer interface.
type PublisherAdapter struct {
	pub *Publisher
}

// NewPublisherAdapter constructs a new instance of PublisherAdapter wrapping a raw Publisher.
func NewPublisherAdapter(pub *Publisher) *PublisherAdapter {
	return &PublisherAdapter{pub: pub}
}

// PublishBookingEvent translates a domain service.BookingEvent into an infrastructure queue.BookingEvent 
// and pushes it onto the AMQP message broker exchange.
func (a *PublisherAdapter) PublishBookingEvent(ctx context.Context, event service.BookingEvent) error {
	return a.pub.PublishBookingEvent(ctx, BookingEvent{
		BookingID:  event.BookingID,
		VoyageID:   event.VoyageID,
		UserID:     event.UserID,
		Status:     event.Status,
		ItemCount:  event.ItemCount,
		OccurredAt: event.OccurredAt,
	})
}