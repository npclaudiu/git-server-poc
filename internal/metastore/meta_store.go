package metastore

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type MetaStore struct {
	pool *pgxpool.Pool
}

type Options struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	SSLMode  string
}

func New(ctx context.Context, options Options) (*MetaStore, error) {
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		options.User,
		options.Password,
		options.Host,
		options.Port,
		options.DBName,
		options.SSLMode,
	)

	pool, err := pgxpool.New(ctx, dsn)

	if err != nil {
		return nil, fmt.Errorf("failed to create database pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &MetaStore{pool: pool}, nil
}

func (m *MetaStore) Close() {
	m.pool.Close()
}

func (m *MetaStore) Ping(ctx context.Context) error {
	return m.pool.Ping(ctx)
}
