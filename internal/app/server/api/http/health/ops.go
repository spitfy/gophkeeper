package health

import (
	"net/http"

	"github.com/danielgtaylor/huma/v2"
)

func (h *Handler) healthCheckOp() huma.Operation {
	return huma.Operation{
		OperationID: "health-check",
		Method:      http.MethodGet,
		Path:        "/api/v1/health",
		Summary:     "Health check endpoint",
		Description: "Returns the health status of the service",
		Tags:        []string{"health"},
		Middlewares: h.middleware,
	}
}
