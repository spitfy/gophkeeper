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
	"gophkeeper/internal/app/server/api/http/record"
	"gophkeeper/internal/domain/user"
)

type Handler struct {
	User   *user.Handler
	Record *record.Handler
}

// New создает *chi.Mux с ВСЕМИ операциями через huma.Register
func New(
	handler Handler,
) *chi.Mux {
	mux := chi.NewMux()

	config := huma.DefaultConfig("Gophkeeper API", "1.0.0")
	config.Components.SecuritySchemes = map[string]*huma.SecurityScheme{
		"bearer": {Type: "http", Scheme: "bearer"},
	}

	API := humachi.New(mux, config)

	handler.User.SetupRoutes(API)
	handler.Record.SetupRoutes(API)

	return mux
}
