package main

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/ove4lo/ship-cargo-service/internal/config"
	"github.com/ove4lo/ship-cargo-service/internal/handler"
	"github.com/ove4lo/ship-cargo-service/internal/repository"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	cfg, err := config.Load() // NOTE: load the main configuration
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	db, err := sql.Open("pgx", cfg.Postgres.DSN()) // NOTE: connect to database
	if err != nil {
		slog.Error("failed to open db", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	if err := db.PingContext(context.Background()); err != nil {
		slog.Error("failed to ping db", "error", err)
		os.Exit(1)
	}

	slog.Info("connected to postgresql")

	userRepo := repository.NewUserRepository(db)
	authHandler := handler.NewAuthHandler(userRepo, cfg.JWT.Secret, cfg.JWT.Expiration)

	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"ok", "env":"%s"}`, cfg.App.Env)
	})

	mux.HandleFunc("POST /register", authHandler.Register)
	mux.HandleFunc("POST /login", authHandler.Login)

	server := &http.Server{
		Addr: ":" + cfg.App.Port,
		Handler: mux,
		ReadTimeout: 10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout: 60 * time.Second,
	}

	// NOTE: graceful application shutdown
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// 1. async startup of an HTTP server
	go func() {
		slog.Info("server starting", "port", cfg.App.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	// 2. Waiting for a signal from the OS (Ctrl+C or SIGTERM)
	<-ctx.Done()
	slog.Info("shutting down...")

	// 3. Create a context with a shutdown timeout
	// WHY: This ensures the server doesn't hang indefinitely if a request gets stuck
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 4. Send the server a command to shut down and wait for current connections to close
	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("shutdown error", "error", err)
	}

	slog.Info("server stopped")
}