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
	"github.com/ove4lo/ship-cargo-service/internal/lock"
	"github.com/ove4lo/ship-cargo-service/internal/middleware"
	"github.com/ove4lo/ship-cargo-service/internal/model"
	"github.com/ove4lo/ship-cargo-service/internal/repository"
	"github.com/ove4lo/ship-cargo-service/internal/service"
	"github.com/redis/go-redis/v9"
)

// applyMiddleware wraps an HTTP handler with a chain of middleware functions,
// executing them in the order they are provided.
func applyMiddleware(h http.HandlerFunc, middlewares ...func(http.Handler) http.Handler) http.Handler {
	var handler http.Handler = h
	for i := len(middlewares) - 1; i >= 0; i-- {
		handler = middlewares[i](handler)
	}
	return handler 
}

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

	rdb := redis.NewClient(&redis.Options{
		Addr: cfg.Redis.Addr(),
	})
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		slog.Error("failed to ping redis", "error", err)
		os.Exit(1)
	}
	defer rdb.Close()

	slog.Info("connected to redis")

	userRepo := repository.NewUserRepository(db)
	vesselRepo := repository.NewVesselRepository(db)
	voyageRepo := repository.NewVoyageRepository(db)
	bookingRepo := repository.NewBookingRepository(db)

	voyageLock := lock.NewRedisLock(rdb)
	bookingService := service.NewBookingService(bookingRepo, voyageRepo, voyageLock)

	authHandler := handler.NewAuthHandler(userRepo, cfg.JWT.Secret, cfg.JWT.Expiration)
	vesselHandler := handler.NewVesselHandler(vesselRepo)
	voyageHandler := handler.NewVoyageHandler(voyageRepo)
	bookingHandler := handler.NewBookingHandler(bookingService)

	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"ok", "env":"%s"}`, cfg.App.Env)
	})

	mux.HandleFunc("POST /register", authHandler.Register)
	mux.HandleFunc("POST /login", authHandler.Login)

	authMw := middleware.Auth(cfg.JWT.Secret)
	managerOnly := middleware.RequireRole(string(model.RoleManager))

	// Vessels - only manager
	mux.Handle("POST /vessels", applyMiddleware(vesselHandler.Create, authMw, managerOnly))
	mux.Handle("GET /vessels", applyMiddleware(vesselHandler.GetAll, authMw))

	// Voyages - create: manager, view: all authorized users
	mux.Handle("POST /voyages", applyMiddleware(voyageHandler.Create, authMw, managerOnly))
	mux.Handle("GET /voyages", applyMiddleware(voyageHandler.GetAll, authMw))

	// Bookings - all authorized users
	mux.Handle("POST /bookings", applyMiddleware(bookingHandler.Create, authMw))

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

	// 1. Async startup of an HTTP server
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