package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	"worker-pool/api"
	"worker-pool/internal/config"
	"worker-pool/internal/db"
	"worker-pool/internal/handler"
	"worker-pool/internal/redis"
	"worker-pool/internal/services"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"
)

func main() {

	log.Info().Msg("Starting server...")

	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatal().Err(err).Msg("Error loading config")
	}

	store, err := db.InitPostgres(cfg.DatabaseURL)
	if err != nil {
		log.Fatal().Err(err).Msg("Error initializing database")
	}

	if err := redis.InitRedis(redis.RedisConfig{
		Address:  cfg.Redis.Address,
		Password: cfg.Redis.Password,
	}); err != nil {
		log.Warn().Err(err).Msg("Failed to connect to Redis, continuing without Redis support")
	} else {
		log.Info().Msg("Redis connected successfully")
	}

	ws := services.NewWebhookService(store)

	h := handler.NewHandler(cfg, ws)

	e := echo.New()
	e.HideBanner = true

	e.Use(middleware.RequestLogger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{http.MethodGet, http.MethodPost, http.MethodPatch, http.MethodDelete, http.MethodOptions},
		AllowHeaders:     []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, echo.HeaderCookie},
		AllowCredentials: false,
	}))

	api.RegisterHandlers(e, h)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var g errgroup.Group

	g.Go(func() error {
		serverAddr := ":" + cfg.Port
		log.Info().Str("address", serverAddr).Msg("Starting HTTP server")
		if err := e.Start(serverAddr); err != nil && err != http.ErrServerClosed {
			log.Error().Err(err).Msg("HTTP server error")
			return err
		}
		return nil
	})

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	g.Go(func() error {
		select {
		case <-sigChan:
			log.Info().Msg("Received termination signal, shutting down...")
			cancel()
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	})

	g.Go(func() error {
		<-ctx.Done()
		log.Info().Msg("Shutting down server...")

		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()

		if err := e.Shutdown(shutdownCtx); err != nil {
			log.Error().Err(err).Msg("Error during server shutdown")
			return err
		}

		log.Info().Msg("Server shutdown complete")
		return nil
	})

	if err := g.Wait(); err != nil {
		log.Fatal().Err(err).Msg("Server shut down with error")
	}

	log.Info().Msg("Server gracefully stopped")
}
