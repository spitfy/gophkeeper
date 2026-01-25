package health

import (
	"context"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/stretchr/testify/assert"
	"golang.org/x/exp/slog"
)

func TestHandler_healthCheck(t *testing.T) {
	tests := []struct {
		name           string
		expectedStatus string
	}{
		{
			name:           "health check returns OK",
			expectedStatus: "OK",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			log := slog.Default()
			middleware := huma.Middlewares{}
			handler := NewHandler(log, middleware)
			ctx := context.Background()
			input := &Input{}

			// Act
			output, err := handler.healthCheck(ctx, input)

			// Assert
			assert.NoError(t, err)
			assert.NotNil(t, output)
			assert.Equal(t, tt.expectedStatus, output.Body.Status)
		})
	}
}

func TestNewHandler(t *testing.T) {
	// Arrange
	log := slog.Default()
	middleware := huma.Middlewares{}

	// Act
	handler := NewHandler(log, middleware)

	// Assert
	assert.NotNil(t, handler)
	assert.NotNil(t, handler.log)
	assert.NotNil(t, handler.middleware)
}
