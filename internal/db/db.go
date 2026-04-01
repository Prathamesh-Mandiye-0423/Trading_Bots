package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
)

// Pool is the shared connection pool used by all writers.
// pgxpool is safe for concurrent use — one pool for the whole process.
var Pool *pgxpool.Pool

// Connect initialises the connection pool and verifies connectivity.
// dsn example: "postgres://trading:trading@localhost:5432/trading"

func Connect(ctx context.Context, dsn string) error {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return fmt.Errorf("failed to parse database config: %w", err)
	}
	// Pool sizing - 5 connetions is plenty for a single-node dev setup
	cfg.MaxConns = 10
	cfg.MinConns = 2

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return fmt.Errorf("failed to create database connection pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}
	Pool = pool
	log.Info().Str("dsn", maskPassword(dsn)).Msg("database connected")
	return nil
}

func Close() {
	if Pool != nil {
		Pool.Close()
		log.Info().Msg("database connection pool closed")
	}
}
func maskPassword(dsn string) string {
	// NOTE: Simple mask->Update using proper URL parser logic
	return "postgres://***@" + dsn[len("postgres://"):]
}
