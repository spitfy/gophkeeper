package user

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPasswordValidator_ValidateLogin(t *testing.T) {
	validator := NewPasswordValidator()

	tests := []struct {
		name        string
		login       string
		wantErr     bool
		expectedErr string
	}{
		{
			name:    "valid login",
			login:   "user123",
			wantErr: false,
		},
		{
			name:        "too short",
			login:       "ab",
			wantErr:     true,
			expectedErr: "login must be at least 3 characters",
		},
		{
			name:        "too long",
			login:       strings.Repeat("a", 33),
			wantErr:     true,
			expectedErr: "login must be at most 32 characters",
		},
		{
			name:    "valid with underscore",
			login:   "user_name",
			wantErr: false,
		},
		{
			name:    "valid with dash",
			login:   "user-name",
			wantErr: false,
		},
		{
			name:    "valid with dot",
			login:   "user.name",
			wantErr: false,
		},
		{
			name:        "invalid space",
			login:       "user name",
			wantErr:     true,
			expectedErr: "login can only contain letters, digits, '_', '-', '.'",
		},
		{
			name:        "invalid special char",
			login:       "user@name",
			wantErr:     true,
			expectedErr: "login can only contain letters, digits, '_', '-', '.'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateLogin(tt.login)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestPasswordValidator_ValidatePassword(t *testing.T) {
	validator := NewPasswordValidator()

	tests := []struct {
		name        string
		password    string
		wantErr     bool
		expectedErr string
	}{
		{
			name:     "valid strong password",
			password: "Abc123!",
			wantErr:  true,
		},
		{
			name:        "too short",
			password:    "Abc123",
			wantErr:     true,
			expectedErr: "password must be at least 8 characters",
		},
		{
			name:        "no uppercase",
			password:    "abc123!@",
			wantErr:     true,
			expectedErr: "password must contain at least one uppercase letter",
		},
		{
			name:        "no lowercase",
			password:    "ABC123!@",
			wantErr:     true,
			expectedErr: "password must contain at least one lowercase letter",
		},
		{
			name:        "no digit",
			password:    "Abcdef!@",
			wantErr:     true,
			expectedErr: "password must contain at least one digit",
		},
		{
			name:        "no special char",
			password:    "Abcdef12",
			wantErr:     true,
			expectedErr: "password must contain at least one special character",
		},
		{
			name:     "strong with multiple chars",
			password: "P@ssw0rd123!",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidatePassword(tt.password)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestPasswordValidator_ValidateRegister(t *testing.T) {
	validator := NewPasswordValidator()

	tests := []struct {
		name           string
		login          string
		password       string
		wantErr        bool
		expectedErrMsg string
	}{
		{
			name:     "valid registration",
			login:    "user123",
			password: "Abc123!",
			wantErr:  true,
		},
		{
			name:           "invalid login",
			login:          "ab",
			password:       "Abc123!",
			wantErr:        true,
			expectedErrMsg: "login validation failed",
		},
		{
			name:           "invalid password",
			login:          "user123",
			password:       "abc",
			wantErr:        true,
			expectedErrMsg: "password validation failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateRegister(tt.login, tt.password)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErrMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestNewPasswordValidator(t *testing.T) {
	v := NewPasswordValidator()
	assert.True(t, v.requireSpecialChar)
	assert.True(t, v.requireDigit)
	assert.True(t, v.requireUpper)
	assert.True(t, v.requireLower)
}
