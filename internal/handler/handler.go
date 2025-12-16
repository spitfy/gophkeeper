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
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/exp/slog"
	"gophkeeper/internal/storage"
	"net/http"
	"time"
)

// API содержит зависимости для хендлеров
type API struct {
	log     *slog.Logger
	storage storage.Storage
}

// NewAPI создает API с ВСЕМИ операциями через huma.Register
func NewAPI(log *slog.Logger, storage storage.Storage) *chi.Mux {
	mux := chi.NewMux()
	api := &API{log: log, storage: storage}

	config := huma.DefaultConfig("Gophkeeper API", "1.0.0")
	config.Components.SecuritySchemes = map[string]*huma.SecurityScheme{
		"bearer": {Type: "http", Scheme: "bearer"},
	}
	// Создаем Huma API на mux
	humaAPI := humachi.New(mux, config)

	// ✅ ПУБЛИЧНЫЕ операции (без security)
	huma.Register(humaAPI, api.userRegisterOp(), api.userRegister)
	huma.Register(humaAPI, api.userLoginOp(), api.userLogin)

	// ✅ ЗАЩИЩЕННЫЕ операции (с security: ["bearer": []])
	huma.Register(humaAPI, api.recordsListOp(), api.recordsList)
	huma.Register(humaAPI, api.recordsCreateOp(), api.recordsCreate)
	huma.Register(humaAPI, api.recordsGetOp(), api.recordsGet)
	huma.Register(humaAPI, api.recordsUpdateOp(), api.recordsUpdate)
	huma.Register(humaAPI, api.recordsDeleteOp(), api.recordsDelete)

	// ✅ Middleware на /api/*
	apiRouter := chi.NewRouter()
	apiRouter.Use(api.authMiddleware)
	apiRouter.Handle("/*", mux)
	mux.Mount("/api", apiRouter)

	return mux
}

// === OPERATIONS ===
func (a *API) userRegisterOp() huma.Operation {
	return huma.Operation{
		OperationID: "user-register",
		Method:      http.MethodPost,
		Path:        "/user/register",
		Summary:     "Регистрация пользователя",
		Tags:        []string{"users"},
	}
}

func (a *API) userLoginOp() huma.Operation {
	return huma.Operation{
		OperationID: "user-login",
		Method:      http.MethodPost,
		Path:        "/user/login",
		Summary:     "Авторизация пользователя",
		Tags:        []string{"users"},
	}
}

func (a *API) recordsListOp() huma.Operation {
	return huma.Operation{
		OperationID: "records-list",
		Method:      http.MethodGet,
		Path:        "/api/records",
		Summary:     "Список записей пользователя",
		Tags:        []string{"records"},
		Security:    []map[string][]string{{"bearer": {}}},
	}
}

func (a *API) recordsCreateOp() huma.Operation {
	return huma.Operation{
		OperationID: "records-create",
		Method:      http.MethodPost,
		Path:        "/api/records",
		Summary:     "Создать запись",
		Tags:        []string{"records"},
		Security:    []map[string][]string{{"bearer": {}}},
	}
}

func (a *API) recordsGetOp() huma.Operation {
	return huma.Operation{
		OperationID: "records-get",
		Method:      http.MethodGet,
		Path:        "/api/records/{id}",
		Summary:     "Получить запись",
		Tags:        []string{"records"},
		Security:    []map[string][]string{{"bearer": {}}},
	}
}

func (a *API) recordsUpdateOp() huma.Operation {
	return huma.Operation{
		OperationID: "records-update",
		Method:      http.MethodPut,
		Path:        "/api/records/{id}",
		Summary:     "Обновить запись",
		Tags:        []string{"records"},
		Security:    []map[string][]string{{"bearer": {}}},
	}
}

func (a *API) recordsDeleteOp() huma.Operation {
	return huma.Operation{
		OperationID: "records-delete",
		Method:      http.MethodDelete,
		Path:        "/api/records/{id}",
		Summary:     "Удалить запись",
		Tags:        []string{"records"},
		Security:    []map[string][]string{{"bearer": {}}},
	}
}

// === MIDDLEWARE ===
func (a *API) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("Authorization")
		if len(token) < 7 || token[:7] != "Bearer " {
			_ = huma.Error401Unauthorized("Unauthorized", nil)
			return
		}
		userID, err := a.validateSession(r.Context(), token[7:])
		if err != nil {
			_ = huma.Error401Unauthorized("Unauthorized", nil)
			return
		}
		ctx := context.WithValue(r.Context(), "userID", userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (a *API) validateSession(ctx context.Context, token string) (int, error) {
	tokenHash := sha256.Sum256([]byte(token))
	return a.storage.ValidateSession(ctx, hex.EncodeToString(tokenHash[:]))
}

// === ПОЛЬЗОВАТЕЛИ ===
type UserRegisterInput struct {
	Body struct {
		Login    string `json:"login" maxLength:"20"`
		Password string `json:"password" minLength:"4" maxLength:"20"`
	}
}

type UserRegisterOutput struct {
	Body struct {
		ID     int    `json:"user_id"`
		Status string `json:"status"`
		Error  string `json:"error,omitempty"`
	}
}

func (a *API) userRegister(ctx context.Context, input *UserRegisterInput) (*UserRegisterOutput, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(input.Body.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	userID, err := a.storage.CreateUser(ctx, input.Body.Login, string(hash))
	if err != nil {
		return &UserRegisterOutput{
			Body: struct {
				ID     int    `json:"user_id"`
				Status string `json:"status"`
				Error  string `json:"error,omitempty"`
			}(struct {
				ID     int    `json:"user_id"`
				Status string `json:"status"`
				Error  string `json:"error"`
			}{Status: "Error", Error: err.Error()}),
		}, nil
	}

	return &UserRegisterOutput{
		Body: struct {
			ID     int    `json:"user_id"`
			Status string `json:"status"`
			Error  string `json:"error,omitempty"`
		}(struct {
			ID     int    `json:"user_id"`
			Status string `json:"status"`
			Error  string `json:"error"`
		}{ID: userID, Status: "Ok"}),
	}, nil
}

type UserLoginInput struct {
	Body struct {
		Login    string `json:"login"`
		Password string `json:"password"`
	}
}

type UserLoginOutput struct {
	Body struct {
		Token  string `json:"token"`
		Status string `json:"status"`
		Error  string `json:"error"`
	}
}

func (a *API) userLogin(ctx context.Context, input *UserLoginInput) (*UserLoginOutput, error) {
	userID, _, err := a.storage.AuthUser(ctx, input.Body.Login, input.Body.Password)
	if err != nil {
		return &UserLoginOutput{
			Body: struct {
				Token  string `json:"token"`
				Status string `json:"status"`
				Error  string `json:"error"`
			}(struct{ Token, Status, Error string }{
				Status: "Error",
				Error:  "Invalid credentials",
			}),
		}, nil
	}

	// Генерируем сессионный токен
	tokenBytes := make([]byte, 32)
	rand.Read(tokenBytes)
	token := base64.URLEncoding.EncodeToString(tokenBytes)
	tokenHash := sha256.Sum256([]byte(token))

	expiresAt := time.Now().Add(24 * time.Hour)
	if err := a.storage.CreateSession(ctx, userID, hex.EncodeToString(tokenHash[:]), expiresAt); err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}

	return &UserLoginOutput{
		Body: struct {
			Token  string `json:"token"`
			Status string `json:"status"`
			Error  string `json:"error"`
		}(struct{ Token, Status, Error string }{
			Token:  token,
			Status: "Ok",
			Error:  err.Error(),
		}),
	}, nil
}

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
