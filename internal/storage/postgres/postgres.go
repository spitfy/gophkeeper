package postgres

import (
	"context"
	"fmt"
	"gophkeeper/internal/config"
	"gophkeeper/internal/migration"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Storage struct {
	conf *config.Config
	pool *pgxpool.Pool
}

func New(conf *config.Config) (*Storage, error) {
	const op = "storage.postgres.New"

	if err := migrate(conf); err != nil {
		return nil, err
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, conf.DB.DatabaseURI)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &Storage{
		conf,
		pool,
	}, nil
}

func (s *Storage) Close() {
	s.pool.Close()
}

func migrate(cfg *config.Config) error {
	m := migration.NewMigration(cfg)
	if err := m.Up(); err != nil {
		return err
	}
	return nil
}
