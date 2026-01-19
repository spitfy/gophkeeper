package session

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/exp/slog"
)

// MockRepository is a mock implementation of the Repository interface for testing
type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) Create(ctx context.Context, userID int, tokenHash string, expiresAt time.Time) error {
	args := m.Called(ctx, userID, tokenHash, expiresAt)
	return args.Error(0)
}

func (m *MockRepository) Validate(ctx context.Context, tokenHash string) (int, error) {
	args := m.Called(ctx, tokenHash)
	return args.Int(0), args.Error(1)
}

func TestService_Create(t *testing.T) {
	mockRepo := new(MockRepository)
	logger := slog.Default()
	service := NewService(mockRepo, logger)

	userID := 123

	// Mock the repository call - we can't predict the exact hash, so we'll check that it's called with correct userID and non-empty hash
	mockRepo.On("Create", mock.Anything, userID, mock.MatchedBy(func(hash string) bool {
		return hash != "" && len(hash) > 0
	}), mock.MatchedBy(func(expiresAt time.Time) bool {
		return !expiresAt.IsZero() && expiresAt.After(time.Now())
	})).Return(nil)

	token, err := service.Create(context.Background(), userID)
	assert.NoError(t, err)
	assert.NotEmpty(t, token)
	// base64 encoded 32 bytes should be 44 characters (32*8/6 = 42.67, rounded up to 44 with padding)
	assert.Len(t, token, 44)

	mockRepo.AssertExpectations(t)
}

func TestService_Create_RepositoryError(t *testing.T) {
	mockRepo := new(MockRepository)
	logger := slog.Default()
	service := NewService(mockRepo, logger)

	userID := 123

	mockRepo.On("Create", mock.Anything, userID, mock.AnythingOfType("string"), mock.AnythingOfType("time.Time")).Return(errors.New("database error"))

	_, err := service.Create(context.Background(), userID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database error")

	mockRepo.AssertExpectations(t)
}

func TestService_Validate(t *testing.T) {
	mockRepo := new(MockRepository)
	logger := slog.Default()
	service := NewService(mockRepo, logger)

	// First create a token to validate
	userID := 123
	token := "test_token_123"

	// Mock the repository validation
	mockRepo.On("Validate", mock.Anything, mock.MatchedBy(func(hash string) bool {
		return hash != "" && len(hash) > 0
	})).Return(userID, nil)

	validatedUserID, err := service.Validate(context.Background(), token)
	assert.NoError(t, err)
	assert.Equal(t, userID, validatedUserID)

	mockRepo.AssertExpectations(t)
}

func TestService_Validate_InvalidToken(t *testing.T) {
	mockRepo := new(MockRepository)
	logger := slog.Default()
	service := NewService(mockRepo, logger)

	token := "invalid_token"

	// Mock the repository to return an error
	mockRepo.On("Validate", mock.Anything, mock.AnythingOfType("string")).Return(0, errors.New("invalid token"))

	_, err := service.Validate(context.Background(), token)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid token")

	mockRepo.AssertExpectations(t)
}

func TestService_Validate_RepositoryError(t *testing.T) {
	mockRepo := new(MockRepository)
	logger := slog.Default()
	service := NewService(mockRepo, logger)

	token := "test_token"

	// Mock the repository to return an error
	mockRepo.On("Validate", mock.Anything, mock.AnythingOfType("string")).Return(0, errors.New("database error"))

	_, err := service.Validate(context.Background(), token)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database error")

	mockRepo.AssertExpectations(t)
}

// Test that Create and Validate work together
func TestService_CreateAndValidate(t *testing.T) {
	mockRepo := new(MockRepository)
	logger := slog.Default()
	service := NewService(mockRepo, logger)

	userID := 123

	// Mock Create
	mockRepo.On("Create", mock.Anything, userID, mock.AnythingOfType("string"), mock.AnythingOfType("time.Time")).Return(nil)

	// Create token
	token, err := service.Create(context.Background(), userID)
	assert.NoError(t, err)
	assert.NotEmpty(t, token)

	// Mock Validate with the same hash that would be generated from the token
	mockRepo.On("Validate", mock.Anything, mock.AnythingOfType("string")).Return(userID, nil)

	// Validate the token
	validatedUserID, err := service.Validate(context.Background(), token)
	assert.NoError(t, err)
	assert.Equal(t, userID, validatedUserID)

	mockRepo.AssertExpectations(t)
}

// Test edge cases for Create
func TestService_Create_EdgeCases(t *testing.T) {
	tests := []struct {
		name   string
		userID int
	}{
		{
			name:   "Zero user ID",
			userID: 0,
		},
		{
			name:   "Negative user ID",
			userID: -1,
		},
		{
			name:   "Positive user ID",
			userID: 123,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockRepository)
			logger := slog.Default()
			service := NewService(mockRepo, logger)

			mockRepo.On("Create", mock.Anything, tt.userID, mock.AnythingOfType("string"), mock.AnythingOfType("time.Time")).Return(nil)

			token, err := service.Create(context.Background(), tt.userID)
			assert.NoError(t, err)
			assert.NotEmpty(t, token)

			mockRepo.AssertExpectations(t)
		})
	}
}

// Test edge cases for Validate
func TestService_Validate_EdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		token       string
		expectError bool
	}{
		{
			name:        "Empty token",
			token:       "",
			expectError: false,
		},
		{
			name:        "Normal token",
			token:       "normal_token_123",
			expectError: false,
		},
		{
			name:        "Long token",
			token:       "verylongtoken12345678901234567890123456789012345678901234567890",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockRepository)
			logger := slog.Default()
			service := NewService(mockRepo, logger)

			if !tt.expectError {
				mockRepo.On("Validate", mock.Anything, mock.AnythingOfType("string")).Return(123, nil)
			} else {
				mockRepo.On("Validate", mock.Anything, mock.AnythingOfType("string")).Return(0, errors.New("validation error"))
			}

			_, err := service.Validate(context.Background(), tt.token)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}
