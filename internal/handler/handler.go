//регистрация, аутентификация и авторизация пользователей;
//хранение приватных данных;
//синхронизация данных между несколькими авторизованными клиентами одного владельца;
//передача приватных данных владельцу по запросу.

//POST /user/register     # Регистрация (публичный)
//POST /user/login        # Логин (публичный)
//POST /api/records       # Создать запись (auth)
//GET  /api/records       # Список записей (auth)
//GET  /api/records/{id}  # Получить запись (auth)
//PUT  /api/records/{id}  # Обновить запись (auth)
//DELETE /api/records/{id} # Удалить запись (auth)

package handler

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	"golang.org/x/exp/slog"
	"gophkeeper/internal/domain/user"
	"gophkeeper/internal/storage"
	"net/http"
	"time"
)

type UserServicer interface {
	CreateUser(ctx context.Context, login, passwordHash string) (int, error)
	AuthUser(ctx context.Context, login, password string) (int, string, error)
}

type RecordServicer interface {
	ListRecords(ctx context.Context, userID int) ([]storage.Record, error)
	CreateRecord(ctx context.Context, userID int, typ, encryptedData string, meta json.RawMessage) (int, error)
	GetRecord(ctx context.Context, userID, recordID int) (*storage.Record, error)
	UpdateRecord(ctx context.Context, userID, recordID int, typ, encryptedData string, meta json.RawMessage) error
	DeleteRecord(ctx context.Context, userID, recordID int) error
}

type SessionServicer interface {
	CreateSession(ctx context.Context, userID int, tokenHash string, expiresAt time.Time) error
	ValidateSession(ctx context.Context, tokenHash string) (int, error)
}

// API содержит зависимости для хендлеров
type API struct {
	log     *slog.Logger
	storage storage.Storage
}

type Handler struct {
	User *user.Handler
}

// NewAPI создает API с ВСЕМИ операциями через huma.Register
func NewAPI(log *slog.Logger, storage storage.Storage, handler Handler) *chi.Mux {
	mux := chi.NewMux()
	api := &API{log: log, storage: storage}

	config := huma.DefaultConfig("Gophkeeper API", "1.0.0")
	config.Components.SecuritySchemes = map[string]*huma.SecurityScheme{
		"bearer": {Type: "http", Scheme: "bearer"},
	}

	// Публичные операции регистрируем на mux
	publicAPI := humachi.New(mux, config)

	// Защищенные операции регистрируем на защищенном роутере с middleware
	protectedRouter := chi.NewRouter()
	protectedRouter.Use(api.authMiddleware)
	// Монтируем защищенный роутер на /api в корневом роутере
	mux.Mount("/api", protectedRouter)
	protectedAPI := humachi.New(protectedRouter, config)

	// Регистрируем публичные операции
	//huma.Register(publicAPI, api.userRegisterOp(), api.userRegister)
	//huma.Register(publicAPI, api.userLoginOp(), api.userLogin)
	handler.User.SetupRoutes(publicAPI)

	// Регистрируем защищенные операции
	huma.Register(protectedAPI, api.recordsListOp(), api.recordsList)
	huma.Register(protectedAPI, api.recordsCreateOp(), api.recordsCreate)
	huma.Register(protectedAPI, api.recordsGetOp(), api.recordsGet)
	huma.Register(protectedAPI, api.recordsUpdateOp(), api.recordsUpdate)
	huma.Register(protectedAPI, api.recordsDeleteOp(), api.recordsDelete)

	return mux
}

// === OPERATIONS ===

type contextKey string

const userIDKey contextKey = "userID"

// === MIDDLEWARE ===
func (a *API) authMiddleware(next http.Handler) http.Handler {
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
		userID, err := a.validateSession(r.Context(), token[7:])
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

func (a *API) validateSession(ctx context.Context, token string) (int, error) {
	tokenHash := sha256.Sum256([]byte(token))
	return a.storage.ValidateSession(ctx, hex.EncodeToString(tokenHash[:]))
}

// === ПОЛЬЗОВАТЕЛИ ===

// === ЗАПИСИ ===
type RecordsListOutput struct {
	Body struct {
		Records []RecordItem `json:"records"`
	}
}

type RecordItem struct {
	ID           int             `json:"id"`
	Type         string          `json:"type"`
	Meta         json.RawMessage `json:"meta"`
	Version      int             `json:"version"`
	LastModified time.Time       `json:"last_modified"`
}

func (a *API) recordsList(ctx context.Context, _ *struct{}) (*RecordsListOutput, error) {
	userID := ctx.Value("userID").(int)
	records, err := a.storage.ListRecords(ctx, userID)
	if err != nil {
		return nil, err
	}

	items := make([]RecordItem, len(records))
	for i, r := range records {
		items[i] = RecordItem{
			ID:           r.ID,
			Type:         r.Type,
			Meta:         r.Meta,
			Version:      r.Version,
			LastModified: r.LastModified,
		}
	}

	return &RecordsListOutput{
		Body: struct {
			Records []RecordItem `json:"records"`
		}{Records: items},
	}, nil
}

type RecordCreateInput struct {
	Body struct {
		Type          string          `json:"type"`
		EncryptedData string          `json:"data" format:"binary"` // base64
		Meta          json.RawMessage `json:"meta"`
	}
}

type RecordOutput struct {
	Body struct {
		ID     int    `json:"id"`
		Status string `json:"status"`
		Error  string `json:"error"`
	}
}

func (a *API) recordsCreate(ctx context.Context, input *RecordCreateInput) (*RecordOutput, error) {
	userID := ctx.Value("userID").(int)

	// TODO: расшифровка/проверка data на клиенте, здесь только хранение
	recordID, err := a.storage.CreateRecord(ctx, userID, input.Body.Type,
		input.Body.EncryptedData, input.Body.Meta)
	if err != nil {
		return &RecordOutput{
			Body: struct {
				ID     int    `json:"id"`
				Status string `json:"status"`
				Error  string `json:"error"`
			}(struct {
				ID            int
				Status, Error string
			}{
				Status: "Error", Error: err.Error(),
			}),
		}, nil
	}

	return &RecordOutput{
		Body: struct {
			ID     int    `json:"id"`
			Status string `json:"status"`
			Error  string `json:"error"`
		}(struct {
			ID            int
			Status, Error string
		}{
			ID:     recordID,
			Status: "Ok",
		}),
	}, nil
}

// Аналогично для Get/Update/Delete (упрощено)
func (a *API) recordsGet(ctx context.Context, input *struct {
	ID int `path:"id" example:"1" doc:"ID записи"`
}) (*RecordOutput, error) {
	// Реализация через storage.GetRecord(ctx, userID, input.ID)
	return &RecordOutput{
		Body: struct {
			ID     int    `json:"id"`
			Status string `json:"status"`
			Error  string `json:"error"`
		}(struct {
			ID            int
			Status, Error string
		}{
			Status: "Ok",
		}),
	}, nil
}

func (a *API) recordsUpdate(ctx context.Context, input *struct {
	ID int `path:"id"`
}) (*RecordOutput, error) {
	// Реализация через storage.UpdateRecord
	return &RecordOutput{
		Body: struct {
			ID     int    `json:"id"`
			Status string `json:"status"`
			Error  string `json:"error"`
		}(struct {
			ID            int
			Status, Error string
		}{
			Status: "Ok",
		}),
	}, nil
}

func (a *API) recordsDelete(ctx context.Context, input *struct {
	ID int `path:"id"`
}) (*RecordOutput, error) {
	// Реализация через storage.DeleteRecord
	return &RecordOutput{
		Body: struct {
			ID     int    `json:"id"`
			Status string `json:"status"`
			Error  string `json:"error"`
		}(struct {
			ID            int
			Status, Error string
		}{
			Status: "Ok",
		}),
	}, nil
}
