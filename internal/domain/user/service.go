package user

import (
	"context"
	"fmt"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/exp/slog"
	"unicode"
)

const (
	minLoginLen    = 3
	maxLoginLen    = 32
	minPasswordLen = 4
)

type Servicer interface {
	Register(ctx context.Context, login, password string) (int, error)
	Authenticate(ctx context.Context, login, password string) (User, error)
}

type Service struct {
	repo      Repository
	validator Validator
	log       *slog.Logger
}

func NewService(repo Repository, validator Validator, log *slog.Logger) *Service {
	return &Service{
		repo:      repo,
		validator: validator,
		log:       log,
	}
}

func (s *Service) Register(ctx context.Context, login, password string) (int, error) {
	if err := s.validator.ValidateRegister(login, password); err != nil {
		s.log.Debug("validation failed", "login", login, "error", err)
		return 0, fmt.Errorf("%w: %v", ErrInvalidInput, err)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return 0, fmt.Errorf("Хэш пароля: %w", err)
	}

	return s.repo.Create(ctx, login, string(hash))
}

func (s *Service) Authenticate(ctx context.Context, login, password string) (User, error) {
	if err := s.validator.ValidateLogin(login); err != nil {
		return User{}, ErrInvalidAuth
	}

	var user User
	user, err := s.repo.FindByLogin(ctx, login)
	if err != nil {
		return user, ErrNotFound
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return user, ErrInvalidAuth
	}

	return user, nil
}

func validatePassword(password string) error {
	if len(password) < minPasswordLen {
		return fmt.Errorf("password must be at least %d characters", minPasswordLen)
	}

	hasLower := false
	hasUpper := false
	hasDigit := false
	hasSpecial := false

	for _, r := range password {
		switch {
		case unicode.IsLower(r):
			hasLower = true
		case unicode.IsUpper(r):
			hasUpper = true
		case unicode.IsDigit(r):
			hasDigit = true
		case unicode.IsPunct(r) || unicode.IsSymbol(r):
			hasSpecial = true
		}
	}

	if !hasLower {
		return fmt.Errorf("password must contain at least one lowercase letter")
	}
	if !hasUpper {
		return fmt.Errorf("password must contain at least one uppercase letter")
	}
	if !hasDigit {
		return fmt.Errorf("password must contain at least one digit")
	}
	if !hasSpecial {
		return fmt.Errorf("password must contain at least one special character")
	}

	return nil
}
