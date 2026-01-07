package user

import (
	"context"
	"fmt"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/exp/slog"
)

type Servicer interface {
	Register(ctx context.Context, login, password string) (int, error)
	Authenticate(ctx context.Context, login, password string) (User, error)
}

type Service struct {
	repo Repository
	log  *slog.Logger
}

func NewService(repo Repository, log *slog.Logger) Servicer {
	return &Service{
		repo: repo,
		log:  log,
	}
}

func (s *Service) Register(ctx context.Context, login, password string) (int, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return 0, fmt.Errorf("hash password: %w", err)
	}

	return s.repo.Create(ctx, login, string(hash))
}

func (s *Service) Authenticate(ctx context.Context, login, password string) (User, error) {
	var user User
	user, err := s.repo.FindByLogin(ctx, login)
	if err != nil {
		return user, ErrUserNotFound
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return user, ErrInvalidCredentials
	}

	return user, nil
}
