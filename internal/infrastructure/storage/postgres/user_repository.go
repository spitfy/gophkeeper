package postgres

import (
	"context"
	"fmt"
	"gophkeeper/internal/domain/user"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/exp/slog"
)

func NewUserRepository(pool *pgxpool.Pool, log *slog.Logger) *UserRepository {
	return &UserRepository{
		pool: pool,
		log:  log,
	}
}

type UserRepository struct {
	pool *pgxpool.Pool
	log  *slog.Logger
}

func (r *UserRepository) Create(ctx context.Context, login, passwordHash string) (int, error) {
	var userID int
	err := r.pool.QueryRow(ctx,
		`INSERT INTO users (login, password_hash) VALUES ($1, $2) RETURNING id`,
		login, passwordHash).Scan(&userID)
	return userID, err
}

func (r *UserRepository) FindByLogin(ctx context.Context, login string) (user.User, error) {
	var u user.User
	err := r.pool.QueryRow(ctx,
		`SELECT id, password_hash FROM users WHERE login = $1`, login).
		Scan(&u.ID, &u.Password)
	if err != nil {
		return u, fmt.Errorf("user not found")
	}

	return u, nil
}
