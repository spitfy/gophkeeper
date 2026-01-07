package user

import (
	"github.com/danielgtaylor/huma/v2"
	"net/http"
)

func (h *Handler) registerOp() huma.Operation {
	return huma.Operation{
		OperationID: "user-register",
		Method:      http.MethodPost,
		Path:        "/user/register",
		Summary:     "Регистрация пользователя",
		Tags:        []string{"users"},
		Middlewares: h.middleware,
	}
}

func (h *Handler) loginOp() huma.Operation {
	return huma.Operation{
		OperationID: "user-login",
		Method:      http.MethodPost,
		Path:        "/user/login",
		Summary:     "Авторизация пользователя",
		Tags:        []string{"users"},
		Middlewares: h.middleware,
	}
}
