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
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	"golang.org/x/exp/slog"
	"gophkeeper/internal/app/server/api/http/middleware"
	"gophkeeper/internal/app/server/api/http/middleware/auth"
	recordAPI "gophkeeper/internal/app/server/api/http/record"
	userAPI "gophkeeper/internal/app/server/api/http/user"
	"gophkeeper/internal/domain/record"
	"gophkeeper/internal/domain/session"
	"gophkeeper/internal/domain/user"
	"gophkeeper/internal/infrastructure/storage/postgres"
)

type Handlers struct {
	User   *userAPI.Handler
	Record *recordAPI.Handler
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
	h.User.SetupRoutes(API)
	h.Record.SetupRoutes(API)

	return mux
}

func handlers(storage *postgres.Storage, log *slog.Logger) *Handlers {
	sessionRepo := session.NewRepo(storage, log)
	sessionService := session.NewService(sessionRepo, log)
	authMW := auth.New(sessionService, log)
	middlewares := middleware.NewContainer()

	userRepo := user.NewRepo(storage, log)
	userService := user.NewService(userRepo, log)
	userHandler := userAPI.NewHandler(userService, sessionService, log, middlewares.GetAllAndClear())

	recordRepo := record.NewRepo(storage, log)
	recordService := record.NewService(recordRepo, log)
	middlewares.Add(authMW.Middleware())
	recordHandler := recordAPI.NewHandler(recordService, log, middlewares.GetAllAndClear())

	return &Handlers{User: userHandler, Record: recordHandler}
}
