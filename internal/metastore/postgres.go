package metastore

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/npclaudiu/git-server-poc/internal/config"
)

// Metastore wraps the database connection pool.
type Metastore struct {
	pool *pgxpool.Pool
}

// New creates a new Metastore with a connection to the database.
func New(ctx context.Context, dbConfig config.DatabaseConfig) (*Metastore, error) {
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		dbConfig.User,
		dbConfig.Password,
		dbConfig.Host,
		dbConfig.Port,
		dbConfig.DBName,
		dbConfig.SSLMode,
	)

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to create database pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &Metastore{pool: pool}, nil
}

// Close gracefully closes the database connection pool.
func (m *Metastore) Close() {
	m.pool.Close()
}
