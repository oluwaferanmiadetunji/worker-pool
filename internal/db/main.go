package db

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresConfig struct {
	User        string
	Password    string
	Port        string
	Name        string
	Host        string
	Environment string
}

func InitPostgres(dsn string) (Store, error) {
	runDBMigration(dsn)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to create pgx pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping pgx pool: %w", err)
	}

	log.Info().Msg("PostgreSQL connection via pgx/v5 successful")

	store := NewStore(pool)
	return store, nil
}
