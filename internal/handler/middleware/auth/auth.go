package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"golang.org/x/exp/slog"
	"gophkeeper/internal/domain/session"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
)

type Auth struct {
	session session.Servicer
	log     *slog.Logger
}

func New(session session.Servicer, log *slog.Logger) *Auth {
	return &Auth{
		session: session,
		log:     log.With("auth middleware"),
	}
}

type contextKey string

const UserIDKey contextKey = "userID"

// Middleware возвращает middleware для Huma с сигнатурой func(ctx Context, next func(Context))
func (a *Auth) Middleware() func(huma.Context, func(huma.Context)) {
	return func(ctx huma.Context, next func(huma.Context)) {
		token := ctx.Header("Authorization")

		if len(token) < 7 || token[:7] != "Bearer " {
			a.log.Error("wrong Bearer: ", token)
			ctx.SetStatus(http.StatusUnauthorized)
			ctx.SetHeader("Content-Type", "application/json")

			w := ctx.BodyWriter()
			json.NewEncoder(w).Encode(map[string]string{
				"error": "Unauthorized",
			})
			return
		}

		// Валидируем токен
		userID, err := a.session.Validate(ctx.Context(), token[7:])
		if err != nil {
			a.log.Error(fmt.Sprintf("validate error: %w", err))
			ctx.SetStatus(http.StatusUnauthorized)
			ctx.SetHeader("Content-Type", "application/json")

			w := ctx.BodyWriter()
			err = json.NewEncoder(w).Encode(map[string]string{
				"error": "Unauthorized",
			})
			if err != nil {
				a.log.Error(fmt.Sprintf("json encod: %w", err))
			}
			return
		}

		newCtx := context.WithValue(ctx.Context(), UserIDKey, userID)
		newHumaCtx := huma.WithContext(ctx, newCtx)

		next(newHumaCtx)
	}
}

func GetUserID(ctx context.Context) (int, bool) {
	userID, ok := ctx.Value(UserIDKey).(int)
	return userID, ok
}
