package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/its-the-vibe/eventhorizon/internal/config"
	"github.com/its-the-vibe/eventhorizon/internal/hub"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Load application config.
	cfgPath := "config.yaml"
	if v := os.Getenv("CONFIG_PATH"); v != "" {
		cfgPath = v
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		logger.Error("failed to load config", "err", err)
		os.Exit(1)
	}

	// Build Redis client.
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.RedisAddr(),
		Password: os.Getenv("REDIS_PASSWORD"),
		DB:       cfg.Redis.DB,
	})

	// Create the SSE hub.
	h := hub.New()

	// Start the Redis subscriber goroutine.
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	go subscribeRedis(ctx, logger, rdb, cfg.Redis.Channel, h)

	// Set up HTTP routes.
	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(http.Dir("static")))
	mux.HandleFunc("/events", sseHandler(logger, h))

	srv := &http.Server{
		Addr:         cfg.Server.Addr(),
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 0, // SSE streams must not time out writes
		IdleTimeout:  60 * time.Second,
	}

	// Start the server in a goroutine.
	go func() {
		logger.Info("server listening", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server error", "err", err)
			cancel()
		}
	}()

	// Wait for shutdown signal.
	<-ctx.Done()
	logger.Info("shutting down")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("shutdown error", "err", err)
	}
}

// subscribeRedis subscribes to a Redis Pub/Sub channel and broadcasts each
// received message to the SSE hub.
func subscribeRedis(ctx context.Context, logger *slog.Logger, rdb *redis.Client, channel string, h *hub.Hub) {
	for {
		if err := runSubscription(ctx, logger, rdb, channel, h); err != nil {
			if ctx.Err() != nil {
				return
			}
			logger.Error("redis subscription error, retrying in 5 s", "err", err)
			select {
			case <-ctx.Done():
				return
			case <-time.After(5 * time.Second):
			}
		} else {
			return
		}
	}
}

func runSubscription(ctx context.Context, logger *slog.Logger, rdb *redis.Client, channel string, h *hub.Hub) error {
	pubsub := rdb.Subscribe(ctx, channel)
	defer pubsub.Close()

	// Confirm subscription before reading messages.
	if _, err := pubsub.Receive(ctx); err != nil {
		return fmt.Errorf("subscribe to %q: %w", channel, err)
	}
	logger.Info("subscribed to redis channel", "channel", channel)

	msgCh := pubsub.Channel()
	for {
		select {
		case <-ctx.Done():
			return nil
		case msg, ok := <-msgCh:
			if !ok {
				return fmt.Errorf("redis channel %q closed", channel)
			}
			logger.Info("received message", "channel", msg.Channel, "payload", msg.Payload)
			h.Broadcast(msg.Payload)
		}
	}
}

// sseHandler returns an HTTP handler that streams Server-Sent Events to the client.
func sseHandler(logger *slog.Logger, h *hub.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming unsupported", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("X-Accel-Buffering", "no")

		client := h.Subscribe()
		defer h.Unsubscribe(client)

		logger.Info("SSE client connected", "remote", r.RemoteAddr)
		defer logger.Info("SSE client disconnected", "remote", r.RemoteAddr)

		for {
			select {
			case <-r.Context().Done():
				return
			case msg, ok := <-client.Channel():
				if !ok {
					return
				}
				fmt.Fprintf(w, "data: %s\n\n", msg)
				flusher.Flush()
			}
		}
	}
}
