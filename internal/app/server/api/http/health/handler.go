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

func (h *Handler) healthCheck(ctx context.Context, input *healthCheckInput) (*healthCheckOutput, error) {
	h.log.Debug("health check request received")

	return &healthCheckOutput{
		Body: HealthCheckResponse{
			Status: "OK",
		},
	}, nil
}
