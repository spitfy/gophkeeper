package user

import (
	"context"
	"fmt"
	"gophkeeper/internal/domain/session"
	"gophkeeper/internal/domain/user"

	"github.com/danielgtaylor/huma/v2"
	"golang.org/x/exp/slog"
)

type Handler struct {
	service    user.Servicer
	session    session.Servicer
	log        *slog.Logger
	middleware huma.Middlewares
}

func NewHandler(service user.Servicer, session session.Servicer, log *slog.Logger, middleware huma.Middlewares) *Handler {
	return &Handler{
		service:    service,
		session:    session,
		log:        log,
		middleware: middleware,
	}
}

func (h *Handler) SetupRoutes(api huma.API) {
	huma.Register(api, h.registerOp(), h.register)
	huma.Register(api, h.loginOp(), h.login)
}

func (h *Handler) register(ctx context.Context, input *registerInput) (*registerOutput, error) {
	userID, err := h.service.Register(ctx, input.Body.Login, input.Body.Password)
	if err != nil {
		return &registerOutput{
			Body: RegisterResponse{Status: "Error", Error: err.Error()},
		}, nil
	}

	return &registerOutput{
		Body: RegisterResponse{ID: userID, Status: "Ok"},
	}, nil
}

func (h *Handler) login(ctx context.Context, input *loginInput) (*loginOutput, error) {
	u, err := h.service.Authenticate(ctx, input.Body.Login, input.Body.Password)
	if err != nil {
		return &loginOutput{
			Body: LoginResponse{
				Status: "Error",
				Error:  "Invalid credentials",
			},
		}, nil
	}

	token, err := h.session.Create(ctx, u.ID)
	if err != nil {
		err = fmt.Errorf("create session: %w", err)
	}

	errMsg := ""
	if err != nil {
		errMsg = err.Error()
	}

	return &loginOutput{
		Body: LoginResponse{
			Token:  token,
			Status: "Ok",
			Error:  errMsg,
		},
	}, nil
}
