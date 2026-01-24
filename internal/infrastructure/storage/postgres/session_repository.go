package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/exp/slog"
)

type SessionRepository struct {
	pool *pgxpool.Pool
	log  *slog.Logger
}

func NewSessionRepository(pool *pgxpool.Pool, log *slog.Logger) *SessionRepository {
	return &SessionRepository{
		pool: pool,
		log:  log,
	}
}

func (r *SessionRepository) Create(ctx context.Context, userID int, tokenHash string, expiresAt time.Time) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO sessions (user_id, token_hash, expires_at) 
         VALUES ($1, decode($2, 'hex'), $3)`,
		userID, tokenHash, expiresAt)
	return err
}

func (r *SessionRepository) Validate(ctx context.Context, tokenHash string) (int, error) {
	var userID int
	err := r.pool.QueryRow(ctx,
		`SELECT user_id FROM sessions 
         WHERE token_hash = decode($1, 'hex') AND expires_at > NOW()`,
		tokenHash).Scan(&userID)

	if err != nil {
		return 0, fmt.Errorf("invalid session")
	}
	return userID, nil
}
