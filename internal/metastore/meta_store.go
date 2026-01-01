package metastore

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/npclaudiu/git-server-poc/internal/metastore/pg"
)

type MetaStore struct {
	pool    *pgxpool.Pool
	queries *pg.Queries
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

	return &MetaStore{
		pool:    pool,
		queries: pg.New(pool),
	}, nil
}

func (m *MetaStore) Close() {
	m.pool.Close()
}

func (m *MetaStore) Ping(ctx context.Context) error {
	return m.pool.Ping(ctx)
}

func (m *MetaStore) CreateRepository(ctx context.Context, name string) (pg.Repository, error) {
	return m.queries.CreateRepository(ctx, name)
}

func (m *MetaStore) ListRepositories(ctx context.Context) ([]pg.Repository, error) {
	return m.queries.ListRepositories(ctx)
}

func (m *MetaStore) GetRepository(ctx context.Context, name string) (pg.Repository, error) {
	return m.queries.GetRepository(ctx, name)
}

func (m *MetaStore) UpdateRepository(ctx context.Context, oldName string, newName string) (pg.Repository, error) {
	return m.queries.UpdateRepository(ctx, pg.UpdateRepositoryParams{
		NewName: newName,
		OldName: oldName,
	})
}

func (m *MetaStore) DeleteRepository(ctx context.Context, name string) error {
	return m.queries.DeleteRepository(ctx, name)
}

func (m *MetaStore) GetRef(ctx context.Context, repoName, refName string) (pg.Ref, error) {
	return m.queries.GetRef(ctx, pg.GetRefParams{
		RepoName: repoName,
		RefName:  refName,
	})
}

func (m *MetaStore) ListRefs(ctx context.Context, repoName string) ([]pg.Ref, error) {
	return m.queries.ListRefs(ctx, repoName)
}

func (m *MetaStore) PutRef(ctx context.Context, repoName, refName, refType, hash, target string) error {
	return m.queries.PutRef(ctx, pg.PutRefParams{
		RepoName: repoName,
		RefName:  refName,
		Type:     refType,
		Hash:     pgtype.Text{String: hash, Valid: hash != ""},
		Target:   pgtype.Text{String: target, Valid: target != ""},
	})
}

func (m *MetaStore) DeleteRef(ctx context.Context, repoName, refName string) error {
	return m.queries.DeleteRef(ctx, pg.DeleteRefParams{
		RepoName: repoName,
		RefName:  refName,
	})
}
