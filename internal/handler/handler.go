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
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	"gophkeeper/internal/domain/record"
	"gophkeeper/internal/domain/user"
	"gophkeeper/internal/handler/middleware/auth"
)

type Handler struct {
	User   *user.Handler
	Record *record.Handler
}

type Middleware struct {
	Auth *auth.Auth
}

// NewAPI создает *chi.Mux с ВСЕМИ операциями через huma.Register
func NewAPI(
	handler Handler,
	middleware Middleware,
) *chi.Mux {
	mux := chi.NewMux()

	config := huma.DefaultConfig("Gophkeeper API", "1.0.0")
	config.Components.SecuritySchemes = map[string]*huma.SecurityScheme{
		"bearer": {Type: "http", Scheme: "bearer"},
	}

	publicAPI := humachi.New(mux, config)

	protectedRouter := chi.NewRouter()
	//protectedRouter.Use(middleware.Auth.Proceed)
	mux.Mount("/api", protectedRouter)

	protectedAPI := humachi.New(protectedRouter, config)

	handler.User.SetupRoutes(publicAPI)
	handler.Record.SetupRoutes(protectedAPI)

	return mux
}
