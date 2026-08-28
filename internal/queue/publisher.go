package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	BookingExchange     = "booking.events"
	NotificationQueue	= "booking.notifications"
	DocumentQueue       = "booking.documents"
)

// BookingEvent encapsulates the data published when a cargo booking status changes.
type BookingEvent struct {
	BookingID  string    `json:"booking_id"`
	VoyageID   string    `json:"voyage_id"`
	UserID     string    `json:"user_id"`
	Status 	   string    `json:"status"`
	ItemCount  int       `json:"item_count"`
	OccurredAt time.Time `json:"occurred_at"`
}

// Publisher handles creating AMQP channels and publishing message events to RabbitMQ.
type Publisher struct {
	ch *amqp.Channel
}

// NewPublisher opens an AMQP channel on the provided connection and ensures the target queue is declared.
func NewPublisher(conn *amqp.Connection) (*Publisher, error) {
	ch, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("open channel: %w", err)
	}

	err = ch.ExchangeDeclare(
		BookingExchange,
		"fanout", // each message is copied to all bound queues
		true,  // durable
		false, // autoDelete
		false, // exclusive
		false, // noWait
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("declare exchange: %w", err)
	}

	// Declare queues and bind them to the exchange
	for _, queueName := range []string{NotificationQueue, DocumentQueue} {
		_, err = ch.QueueDeclare(queueName, true, false, false, false, nil)
		if err != nil {
			return nil, fmt.Errorf("declare queue %s: %w", queueName, err)
		}

		err = ch.QueueBind(queueName, "", BookingExchange, false, nil)
		if err != nil {
			return nil, fmt.Errorf("bind queue %s: %w", queueName, err)
		}
	}

	return &Publisher{ch: ch}, nil
}

// PublishBookingEvent marshals the event metadata into JSON and pushes it to the RabbitMQ exchange.
func (p *Publisher) PublishBookingEvent(ctx context.Context, event BookingEvent) error {
	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}

	return p.ch.PublishWithContext(ctx,
		BookingExchange,
		"", // routing key is empty for fanout
		false, // mandatory
		false, // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent, // The message will survive a RabbitMQ restart.
			Body:         body,
		},
	)
}

// Close gracefully releases the underlying AMQP channel resource.
func (p *Publisher) Close() error {
	return p.ch.Close()
}