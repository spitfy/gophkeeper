package postgres

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"gophkeeper/internal/app/server/config"
	"gophkeeper/internal/infrastructure/migration"
)

type Storage struct {
	pool *pgxpool.Pool
}

func New(cfg *config.Config) (*Storage, error) {
	pool, err := pgxpool.New(context.Background(), cfg.DB.DatabaseURI)
	if err != nil {
		return nil, fmt.Errorf("create pool: %w", err)
	}
	mg := migration.NewMigration(cfg)
	err = mg.Up()
	if err != nil {
		return nil, fmt.Errorf("migration error: %w", err)
	}
	return &Storage{pool: pool}, nil
}

func (s *Storage) Close() error {
	s.pool.Close()
	return nil
}

func (s *Storage) Pool() *pgxpool.Pool {
	return s.pool
}
