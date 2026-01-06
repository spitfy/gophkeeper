package session

import (
	"context"
	"fmt"
	"golang.org/x/exp/slog"
	"gophkeeper/internal/infrastructure/storage/postgres"
	"time"
)

type Repository interface {
	Create(ctx context.Context, userID int, tokenHash string, expiresAt time.Time) error
	Validate(ctx context.Context, tokenHash string) (int, error)
}

func NewRepo(db *postgres.Storage, log *slog.Logger) Repository {
	return &repository{
		db:  db,
		log: log,
	}
}

type repository struct {
	db  *postgres.Storage
	log *slog.Logger
}

func (r *repository) Create(ctx context.Context, userID int, tokenHash string, expiresAt time.Time) error {
	_, err := r.db.Pool().Exec(ctx,
		`INSERT INTO sessions (user_id, token_hash, expires_at) 
         VALUES ($1, decode($2, 'hex'), $3)`,
		userID, tokenHash, expiresAt)
	return err
}

func (r *repository) Validate(ctx context.Context, tokenHash string) (int, error) {
	var userID int
	err := r.db.Pool().QueryRow(ctx,
		`SELECT user_id FROM sessions 
         WHERE token_hash = decode($1, 'hex') AND expires_at > NOW()`,
		tokenHash).Scan(&userID)

	if err != nil {
		return 0, fmt.Errorf("invalid session")
	}
	return userID, nil
}
