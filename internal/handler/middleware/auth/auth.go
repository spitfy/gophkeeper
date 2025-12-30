package auth

import (
	"context"
	"encoding/json"
	"golang.org/x/exp/slog"
	"gophkeeper/internal/domain/session"
	"net/http"
)

type Auth struct {
	session session.Servicer
	log     *slog.Logger
}

func New(session session.Servicer, log *slog.Logger) *Auth {
	return &Auth{
		session: session,
		log:     log,
	}
}

type contextKey string

const userIDKey contextKey = "userID"

// === MIDDLEWARE ===
func (a *Auth) Proceed(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("Authorization")
		if len(token) < 7 || token[:7] != "Bearer " {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"error": "Unauthorized",
			})
			return
		}
		userID, err := a.session.Validate(r.Context(), token[7:])
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"error": "Unauthorized",
			})
			return
		}
		ctx := context.WithValue(r.Context(), userIDKey, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
