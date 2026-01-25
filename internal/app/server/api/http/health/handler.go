package health

import (
	"context"

	"github.com/danielgtaylor/huma/v2"
	"golang.org/x/exp/slog"
)

type Handler struct {
	log        *slog.Logger
	middleware huma.Middlewares
}

func NewHandler(log *slog.Logger, middleware huma.Middlewares) *Handler {
	return &Handler{
		log:        log,
		middleware: middleware,
	}
}

func (h *Handler) SetupRoutes(api huma.API) {
	huma.Register(api, h.healthCheckOp(), h.healthCheck)
}

func (h *Handler) healthCheck(_ context.Context, _ *Input) (*Output, error) {
	h.log.Debug("health check request received")

	return &Output{
		Body: HResponse{
			Status: "OK",
		},
	}, nil
}
