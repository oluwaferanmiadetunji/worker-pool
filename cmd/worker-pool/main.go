package main

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	sqlc "worker-pool/internal/db/sqlc/generated"
	"worker-pool/internal/config"
	"worker-pool/internal/db"

	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"
)

const (
	defaultWorkerCount   = 5
	defaultPollInterval  = 2 * time.Second
	defaultProcessDelay  = 100 * time.Millisecond
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatal().Err(err).Msg("Error loading config")
	}

	workerCount := intEnv("WORKER_POOL_SIZE", defaultWorkerCount)
	pollInterval := durationEnv("WORKER_POLL_INTERVAL", defaultPollInterval)
	processDelay := durationEnv("WORKER_PROCESS_DELAY", defaultProcessDelay)

	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})

	store, err := db.InitPostgres(cfg.DatabaseURL)
	if err != nil {
		log.Fatal().Err(err).Msg("Error initializing database")
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	log.Info().
		Int("workers", workerCount).
		Dur("poll_interval", pollInterval).
		Dur("process_delay", processDelay).
		Msg("Starting worker pool")

	g, gCtx := errgroup.WithContext(ctx)
	for i := range workerCount {
		workerID := i + 1
		g.Go(func() error {
			return runWorker(gCtx, store, workerID, pollInterval, processDelay)
		})
	}

	if err := g.Wait(); err != nil && !errors.Is(err, context.Canceled) {
		log.Fatal().Err(err).Msg("Worker pool stopped with error")
	}
	log.Info().Msg("Worker pool stopped")
}

func runWorker(ctx context.Context, store db.Store, workerID int, pollInterval, processDelay time.Duration) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		event, err := store.ClaimNextWebhook(ctx)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(pollInterval):
				}
				continue
			}
			return err
		}

		log.Debug().
			Int("worker", workerID).
			Str("event_id", event.EventID).
			Msg("Claimed webhook")

		if err := processWebhook(ctx, store, event, processDelay); err != nil {
			log.Warn().Err(err).Str("event_id", event.EventID).Msg("Processing failed")
		}
	}
}

func processWebhook(ctx context.Context, store db.Store, event sqlc.WebhookEvent, processDelay time.Duration) error {
	var payload map[string]interface{}
	if len(event.Payload) > 0 {
		_ = json.Unmarshal(event.Payload, &payload)
	}

	log.Info().
		Str("event_id", event.EventID).
		Interface("type", event.Type).
		Interface("payload", payload).
		Msg("Processing webhook")

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(processDelay):
	}

	_, err := store.MarkWebhookDone(ctx, event.ID)
	if err != nil {
		errStr := err.Error()
		_, _ = store.MarkWebhookFailed(ctx, sqlc.MarkWebhookFailedParams{ID: event.ID, LastError: &errStr})
		return err
	}

	log.Info().Str("event_id", event.EventID).Msg("Webhook marked done")
	return nil
}

func intEnv(key string, defaultVal int) int {
	s := os.Getenv(key)
	if s == "" {
		return defaultVal
	}
	n, err := strconv.Atoi(s)
	if err != nil || n < 1 {
		return defaultVal
	}
	return n
}

func durationEnv(key string, defaultVal time.Duration) time.Duration {
	s := os.Getenv(key)
	if s == "" {
		return defaultVal
	}
	d, err := time.ParseDuration(s)
	if err != nil || d < 0 {
		return defaultVal
	}
	return d
}
