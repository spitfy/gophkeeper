package logger

import (
	"time"

	"github.com/danielgtaylor/huma/v2"
	"golang.org/x/exp/slog"
)

// Logger middleware для логирования входящих HTTP запросов
type Logger struct {
	log *slog.Logger
}

// New создает новый экземпляр Logger middleware
func New(log *slog.Logger) *Logger {
	return &Logger{
		log: log.With(slog.String("component", "http_logger")),
	}
}

// Middleware возвращает middleware функцию для логирования HTTP запросов
func (l *Logger) Middleware() func(huma.Context, func(huma.Context)) {
	return func(ctx huma.Context, next func(huma.Context)) {
		start := time.Now()

		method := ctx.Method()
		path := ctx.URL().Path
		remoteAddr := ctx.RemoteAddr()
		contentType := ctx.Header("Content-Type")

		// Вызываем следующий обработчик
		next(ctx)

		duration := time.Since(start)
		status := ctx.Status()

		attrs := []any{
			slog.String("method", method),
			slog.String("path", path),
			slog.Int("status", status),
			slog.Duration("duration", duration),
			slog.String("remote_addr", remoteAddr),
			slog.String("content_type", contentType),
		}

		if userID := ctx.Header("X-User-ID"); userID != "" {
			attrs = append(attrs, slog.String("user_id", userID))
		}

		switch {
		case status >= 500:
			l.log.Error("HTTP request", attrs...)
		case status >= 400:
			l.log.Warn("HTTP request", attrs...)
		default:
			l.log.Info("HTTP request", attrs...)
		}
	}
}
