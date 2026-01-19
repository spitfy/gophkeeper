package user

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/exp/slog"
)

// MockRepository is a mock implementation of the Repository interface for testing
type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) Create(ctx context.Context, login, passwordHash string) (int, error) {
	args := m.Called(ctx, login, passwordHash)
	return args.Int(0), args.Error(1)
}

func (m *MockRepository) FindByLogin(ctx context.Context, login string) (User, error) {
	args := m.Called(ctx, login)
	return args.Get(0).(User), args.Error(1)
}

func TestService_Register(t *testing.T) {
	mockRepo := new(MockRepository)
	logger := slog.Default()
	service := NewService(mockRepo, logger)

	login := "testuser"
	password := "testpassword123"

	// Mock the repository call - we can't predict the exact hash, so we'll check that it's called with correct login and non-empty hash
	mockRepo.On("Create", mock.Anything, login, mock.MatchedBy(func(hash string) bool {
		return hash != "" && len(hash) > 0
	})).Return(123, nil)

	userID, err := service.Register(context.Background(), login, password)
	assert.NoError(t, err)
	assert.Equal(t, 123, userID)

	mockRepo.AssertExpectations(t)
}

func TestService_Register_RepositoryError(t *testing.T) {
	mockRepo := new(MockRepository)
	logger := slog.Default()
	service := NewService(mockRepo, logger)

	login := "testuser"
	password := "testpassword123"

	mockRepo.On("Create", mock.Anything, login, mock.AnythingOfType("string")).Return(0, errors.New("database error"))

	_, err := service.Register(context.Background(), login, password)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database error")

	mockRepo.AssertExpectations(t)
}

func TestService_Authenticate_Success(t *testing.T) {
	mockRepo := new(MockRepository)
	logger := slog.Default()
	service := NewService(mockRepo, logger)

	login := "testuser"
	password := "testpassword123"

	// Create a valid hash for the password
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	assert.NoError(t, err)

	user := User{
		ID:       123,
		Login:    login,
		Password: string(hash),
	}

	mockRepo.On("FindByLogin", mock.Anything, login).Return(user, nil)

	authUser, err := service.Authenticate(context.Background(), login, password)
	assert.NoError(t, err)
	assert.Equal(t, user, authUser)

	mockRepo.AssertExpectations(t)
}

func TestService_Authenticate_UserNotFound(t *testing.T) {
	mockRepo := new(MockRepository)
	logger := slog.Default()
	service := NewService(mockRepo, logger)

	login := "nonexistent"
	password := "testpassword123"

	mockRepo.On("FindByLogin", mock.Anything, login).Return(User{}, errors.New("user not found"))

	_, err := service.Authenticate(context.Background(), login, password)
	assert.Error(t, err)
	assert.Equal(t, ErrUserNotFound, err)

	mockRepo.AssertExpectations(t)
}

func TestService_Authenticate_InvalidPassword(t *testing.T) {
	mockRepo := new(MockRepository)
	logger := slog.Default()
	service := NewService(mockRepo, logger)

	login := "testuser"
	correctPassword := "correctpassword"
	wrongPassword := "wrongpassword"

	// Create a valid hash for the correct password
	hash, err := bcrypt.GenerateFromPassword([]byte(correctPassword), bcrypt.DefaultCost)
	assert.NoError(t, err)

	user := User{
		ID:       123,
		Login:    login,
		Password: string(hash),
	}

	mockRepo.On("FindByLogin", mock.Anything, login).Return(user, nil)

	_, err = service.Authenticate(context.Background(), login, wrongPassword)
	assert.Error(t, err)
	assert.Equal(t, ErrInvalidCredentials, err)

	mockRepo.AssertExpectations(t)
}

func TestService_Authenticate_InvalidHash(t *testing.T) {
	mockRepo := new(MockRepository)
	logger := slog.Default()
	service := NewService(mockRepo, logger)

	login := "testuser"
	password := "testpassword123"

	// Create a user with invalid hash (not a bcrypt hash)
	user := User{
		ID:       123,
		Login:    login,
		Password: "invalidhash", // This is not a valid bcrypt hash
	}

	mockRepo.On("FindByLogin", mock.Anything, login).Return(user, nil)

	_, err := service.Authenticate(context.Background(), login, password)
	assert.Error(t, err)
	assert.Equal(t, ErrInvalidCredentials, err)

	mockRepo.AssertExpectations(t)
}

// Test table-driven tests for edge cases
func TestService_Register_EdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		login       string
		password    string
		expectError bool
	}{
		{
			name:        "Empty login",
			login:       "",
			password:    "password123",
			expectError: false,
		},
		{
			name:        "Empty password",
			login:       "testuser",
			password:    "",
			expectError: false,
		},
		{
			name:        "Empty both",
			login:       "",
			password:    "",
			expectError: false,
		},
		{
			name:        "Valid credentials",
			login:       "testuser",
			password:    "password123",
			expectError: false,
		},
		{
			name:        "Short password",
			login:       "testuser",
			password:    "123",
			expectError: false,
		},
		{
			name:        "Long password (50 bytes)",
			login:       "testuser",
			password:    "verylongpassword1234567890123456789012345678901234",
			expectError: false,
		},
		{
			name:        "Very long password (100 bytes)",
			login:       "testuser",
			password:    "verylongpassword1234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockRepository)
			logger := slog.Default()
			service := NewService(mockRepo, logger)

			if !tt.expectError {
				mockRepo.On("Create", mock.Anything, tt.login, mock.AnythingOfType("string")).Return(123, nil)
			}

			_, err := service.Register(context.Background(), tt.login, tt.password)
			if tt.expectError {
				assert.Error(t, err)
				// Don't expect mock calls when there's an error
			} else {
				assert.NoError(t, err)
				mockRepo.AssertExpectations(t)
			}
		})
	}
}

func TestService_Authenticate_EdgeCases(t *testing.T) {
	tests := []struct {
		name          string
		login         string
		password      string
		setupMock     bool
		expectError   bool
		expectedError error
	}{
		{
			name:          "Empty login",
			login:         "",
			password:      "password123",
			setupMock:     false,
			expectError:   true,
			expectedError: ErrUserNotFound,
		},
		{
			name:          "Empty password",
			login:         "testuser",
			password:      "",
			setupMock:     true,
			expectError:   false, // bcrypt can handle empty passwords
			expectedError: nil,
		},
		{
			name:          "Empty both",
			login:         "",
			password:      "",
			setupMock:     false,
			expectError:   true,
			expectedError: ErrUserNotFound,
		},
		{
			name:          "Valid credentials",
			login:         "testuser",
			password:      "password123",
			setupMock:     true,
			expectError:   false,
			expectedError: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockRepository)
			logger := slog.Default()
			service := NewService(mockRepo, logger)

			if tt.setupMock {
				hash, err := bcrypt.GenerateFromPassword([]byte(tt.password), bcrypt.DefaultCost)
				assert.NoError(t, err)
				user := User{
					ID:       123,
					Login:    tt.login,
					Password: string(hash),
				}
				mockRepo.On("FindByLogin", mock.Anything, tt.login).Return(user, nil)
			} else {
				mockRepo.On("FindByLogin", mock.Anything, tt.login).Return(User{}, errors.New("user not found"))
			}

			_, err := service.Authenticate(context.Background(), tt.login, tt.password)
			if tt.expectError {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError, err)
			} else {
				assert.NoError(t, err)
				mockRepo.AssertExpectations(t)
			}
		})
	}
}
