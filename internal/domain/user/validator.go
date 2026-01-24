package user

import (
	"fmt"
	"unicode"
)

const (
	MinLoginLen    = 3
	MaxLoginLen    = 32
	MinPasswordLen = 8
)

// Validator - интерфейс для валидации пользовательских данных
type Validator interface {
	ValidateRegister(login, password string) error
	ValidateLogin(login string) error
	ValidatePassword(password string) error
}

type PasswordValidator struct {
	requireSpecialChar bool
	requireDigit       bool
	requireUpper       bool
	requireLower       bool
}

// NewPasswordValidator создает новый валидатор
func NewPasswordValidator() *PasswordValidator {
	return &PasswordValidator{
		requireSpecialChar: true,
		requireDigit:       true,
		requireUpper:       true,
		requireLower:       true,
	}
}

// ValidateRegister валидирует данные для регистрации
func (v *PasswordValidator) ValidateRegister(login, password string) error {
	if err := v.ValidateLogin(login); err != nil {
		return fmt.Errorf("login validation failed: %w", err)
	}

	if err := v.ValidatePassword(password); err != nil {
		return fmt.Errorf("password validation failed: %w", err)
	}

	return nil
}

// ValidateLogin валидирует логин
func (v *PasswordValidator) ValidateLogin(login string) error {
	if len(login) < MinLoginLen {
		return fmt.Errorf("login must be at least %d characters", MinLoginLen)
	}

	if len(login) > MaxLoginLen {
		return fmt.Errorf("login must be at most %d characters", MaxLoginLen)
	}

	for _, r := range login {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' && r != '-' && r != '.' {
			return fmt.Errorf("login can only contain letters, digits, '_', '-', '.'")
		}
	}

	return nil
}

// ValidatePassword валидирует пароль
func (v *PasswordValidator) ValidatePassword(password string) error {
	if len(password) < MinPasswordLen {
		return fmt.Errorf("password must be at least %d characters", MinPasswordLen)
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

		if hasLower && hasUpper && hasDigit && hasSpecial {
			break
		}
	}

	if v.requireLower && !hasLower {
		return fmt.Errorf("password must contain at least one lowercase letter")
	}

	if v.requireUpper && !hasUpper {
		return fmt.Errorf("password must contain at least one uppercase letter")
	}

	if v.requireDigit && !hasDigit {
		return fmt.Errorf("password must contain at least one digit")
	}

	if v.requireSpecialChar && !hasSpecial {
		return fmt.Errorf("password must contain at least one special character")
	}

	return nil
}
