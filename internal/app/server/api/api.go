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

package api

import (
	healthAPI "gophkeeper/internal/app/server/api/http/health"
	"gophkeeper/internal/app/server/api/http/middleware"
	"gophkeeper/internal/app/server/api/http/middleware/auth"
	"gophkeeper/internal/app/server/api/http/middleware/logger"
	recordAPI "gophkeeper/internal/app/server/api/http/record"
	syncAPI "gophkeeper/internal/app/server/api/http/sync"
	userAPI "gophkeeper/internal/app/server/api/http/user"
	"gophkeeper/internal/domain/record"
	"gophkeeper/internal/domain/session"
	"gophkeeper/internal/domain/sync"
	"gophkeeper/internal/domain/user"
	"gophkeeper/internal/infrastructure/storage/postgres"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	"golang.org/x/exp/slog"
)

type Handlers struct {
	Health *healthAPI.Handler
	User   *userAPI.Handler
	Record *recordAPI.Handler
	Sync   *syncAPI.Handler
}

// New создает *chi.Mux с ВСЕМИ операциями через huma.Register
func New(storage *postgres.Storage, log *slog.Logger) *chi.Mux {
	mux := chi.NewMux()

	config := huma.DefaultConfig("Gophkeeper API", "1.0.0")
	config.Components.SecuritySchemes = map[string]*huma.SecurityScheme{
		"bearer": {Type: "http", Scheme: "bearer"},
	}

	API := humachi.New(mux, config)

	h := handlers(storage, log)
	h.Health.SetupRoutes(API)
	h.User.SetupRoutes(API)
	h.Record.SetupRoutes(API)
	h.Sync.SetupRoutes(API)

	return mux
}

func handlers(storage *postgres.Storage, log *slog.Logger) *Handlers {
	sessionRepo := postgres.NewSessionRepository(storage, log)
	sessionService := session.NewService(sessionRepo, log)
	authMW := auth.New(sessionService, log)
	loggerMW := logger.New(log)
	middlewares := middleware.NewContainer()

	middlewares.Add(loggerMW.Middleware())
	healthHandler := healthAPI.NewHandler(log, middlewares.GetAllAndClear())

	userRepo := postgres.NewUserRepository(storage, log)
	userService := user.NewService(userRepo, log)
	middlewares.Add(loggerMW.Middleware())
	userHandler := userAPI.NewHandler(userService, sessionService, log, middlewares.GetAllAndClear())

	recordRepo := postgres.NewRecordRepository(storage, log)
	recordFactory := record.NewRecordFactory()
	recordService := record.NewService(recordRepo, recordFactory, log)
	middlewares.Add(authMW.Middleware())
	middlewares.Add(loggerMW.Middleware())
	recordHandler := recordAPI.NewHandler(recordService, log, middlewares.GetAllAndClear())

	syncRepo := postgres.NewSyncRepository(storage, log)
	syncService := sync.NewService(syncRepo, log, nil)
	middlewares.Add(authMW.Middleware())
	middlewares.Add(loggerMW.Middleware())
	syncHandler := syncAPI.NewHandler(syncService, log, middlewares.GetAllAndClear())

	return &Handlers{
		Health: healthHandler,
		User:   userHandler,
		Record: recordHandler,
		Sync:   syncHandler,
	}
}
