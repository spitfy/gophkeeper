package logger

import (
	"context"
	"testing"

	"golang.org/x/exp/slog"
	"gophkeeper/internal/app/server/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name           string
		env            string
		expectedLevel  slog.Level
		expectedPretty bool
	}{
		{
			name:           "local environment",
			env:            config.EnvLocal,
			expectedLevel:  slog.LevelDebug,
			expectedPretty: true,
		},
		{
			name:           "dev environment",
			env:            config.EnvDev,
			expectedLevel:  slog.LevelDebug,
			expectedPretty: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := New(tt.env)
			require.NotNil(t, logger)
			ctx := context.Background()
			assert.Equal(t, tt.expectedLevel <= 0, logger.Enabled(ctx, slog.LevelDebug))
			assert.Equal(t, tt.expectedLevel <= slog.LevelInfo, logger.Enabled(ctx, slog.LevelInfo))
		})
	}
}

func TestSetupPrettySlog(t *testing.T) {
	logger := setupPrettySlog()
	require.NotNil(t, logger)

	ctx := context.Background()
	assert.True(t, logger.Enabled(ctx, slog.LevelDebug))
}

func TestNewLoggerLevels(t *testing.T) {
	ctx := context.Background()

	// Prod - только INFO и выше
	prodLogger := New(config.EnvProd)
	assert.False(t, prodLogger.Enabled(ctx, slog.LevelDebug))
	assert.True(t, prodLogger.Enabled(ctx, slog.LevelInfo))

	// Dev - DEBUG и выше
	devLogger := New(config.EnvDev)
	assert.True(t, devLogger.Enabled(ctx, slog.LevelDebug))
	assert.True(t, devLogger.Enabled(ctx, slog.LevelInfo))

	// Local - DEBUG (pretty)
	localLogger := New(config.EnvLocal)
	assert.True(t, localLogger.Enabled(ctx, slog.LevelDebug))
}
