package postgres

import (
	"context"
	"fmt"
	"golang.org/x/exp/slog"
	"gophkeeper/internal/domain/user"
)

func NewUserRepository(db *Storage, log *slog.Logger) *UserRepository {
	return &UserRepository{
		db:  db,
		log: log,
	}
}

type UserRepository struct {
	db  *Storage
	log *slog.Logger
}

func (r *UserRepository) Create(ctx context.Context, login, passwordHash string) (int, error) {
	var userID int
	err := r.db.Pool().QueryRow(ctx,
		`INSERT INTO users (login, password_hash) VALUES ($1, $2) RETURNING id`,
		login, passwordHash).Scan(&userID)
	return userID, err
}

func (r *UserRepository) FindByLogin(ctx context.Context, login string) (user.User, error) {
	var u user.User
	err := r.db.Pool().QueryRow(ctx,
		`SELECT id, password_hash FROM users WHERE login = $1`, login).
		Scan(&u.ID, &u.Password)
	if err != nil {
		return u, fmt.Errorf("user not found")
	}

	return u, nil
}
