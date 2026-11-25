package database

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type DB struct {
	Pool *pgxpool.Pool
}

func Connect(ctx context.Context, databaseURL string, maxOpen, maxIdle int, maxLifetime time.Duration) (*DB, error) {
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database config: %w", err)
	}

	config.MaxConns = int32(maxOpen)
	config.MinConns = int32(maxIdle)
	config.MaxConnLifetime = maxLifetime

	ctxTimeout, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(ctxTimeout, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	if err := pool.Ping(ctxTimeout); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	log.Println("Database connection pool established successfully")
	return &DB{Pool: pool}, nil
}

func (db *DB) Close() {
	if db != nil && db.Pool != nil {
		db.Pool.Close()
		log.Println("Database connection pool closed")
	}
}
