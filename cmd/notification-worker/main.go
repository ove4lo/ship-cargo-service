package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/ove4lo/ship-cargo-service/internal/config"
	"github.com/ove4lo/ship-cargo-service/internal/queue"
)

// main initializes and runs the background notification worker.
// It connects to RabbitMQ, consumes booking events from the queue, 
// processes notifications asynchronously, and supports graceful shutdown.
func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	conn, err := amqp.Dial(cfg.RabbitMQ.URL())
	if err != nil {
		slog.Error("failed to open channel", "error", err)
		os.Exit(1)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		slog.Error("failed to open channel", "error", err)
		os.Exit(1)
	}
	defer ch.Close()

	msgs, err := ch.Consume(
		queue.NotificationQueue,
		"", // consumer tag - RabbitMQ generates it itself
		false, // autoAck - off, confirm manually
		false, // exclusive
		false, // noLocal
		false, //noWait
		nil,
	)
	if err != nil {
		slog.Error("failed to start consuming", "error", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	slog.Info("notification worker started, waiting for events...")

	for {
		select {
		case <-ctx.Done():
			slog.Info("shutting down notification worker...")
			return
		case msg, ok := <-msgs:
			if !ok {
				slog.Info("channel closed")
				return
			}
			var event queue.BookingEvent
			if err := json.Unmarshal(msg.Body, &event); err != nil {
				slog.Error("failed to unmarshal event", "error", err)
				if nackErr := msg.Nack(false, false); nackErr != nil {
					slog.Error("failed to nack message", "error", nackErr)
				}
				continue
			}

			// TODO: The actual notification sending will happen here
			slog.Info("booking notification",
				"booking_id", event.BookingID,
				"voyage_id", event.VoyageID,
				"user_id", event.UserID,
				"status", event.Status,
				"item_count", event.ItemCount,
			)

			if err := msg.Ack(false); err != nil {
				slog.Error("failed to ack message", "error", err)
			}
		}
		
	}
}