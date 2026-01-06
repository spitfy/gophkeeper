package record

import (
	"context"
	"github.com/danielgtaylor/huma/v2"
	"golang.org/x/exp/slog"
	"gophkeeper/internal/infrastructure/middleware/auth"
)

type Handler struct {
	service    Servicer
	log        *slog.Logger
	middleware huma.Middlewares
}

func NewHandler(service Servicer, log *slog.Logger, mws huma.Middlewares) *Handler {
	return &Handler{
		service:    service,
		log:        log,
		middleware: mws,
	}
}

func (h *Handler) SetupRoutes(api huma.API) {
	huma.Register(api, h.listOp(), h.list)
	huma.Register(api, h.createOp(), h.Create)
	huma.Register(api, h.findOp(), h.find)
	huma.Register(api, h.updateOp(), h.update)
	huma.Register(api, h.deleteOp(), h.delete)
}

func (h *Handler) list(ctx context.Context, _ *struct{}) (*listOutput, error) {
	userID, ok := auth.GetUserID(ctx)
	if !ok {
		return nil, huma.Error401Unauthorized("Unauthorized")
	}

	records, err := h.service.list(ctx, userID)
	if err != nil {
		return nil, err
	}

	return &listOutput{
		Body: records,
	}, nil
}

func (h *Handler) find(ctx context.Context, input *findInput) (*findOutput, error) {
	userID, ok := auth.GetUserID(ctx)
	if !ok {
		return nil, huma.Error401Unauthorized("Unauthorized")
	}

	record, err := h.service.Find(ctx, userID, input.ID)
	if err != nil {
		return &findOutput{
			Body: FindResponse{
				Status: "Error",
			},
		}, err
	}

	return &findOutput{
		Body: FindResponse{
			Status: "Ok",
			Record: record,
		},
	}, nil
}

func (h *Handler) Create(ctx context.Context, input *createInput) (*Output, error) {
	userID, ok := auth.GetUserID(ctx)
	if !ok {
		return nil, huma.Error401Unauthorized("Unauthorized")
	}

	response, err := h.service.create(ctx, userID, input.Body.Type, input.Body.EncryptedData, input.Body.Meta)
	if err != nil {
		return &Output{
			Body: response,
		}, err
	}

	return &Output{
		Body: response,
	}, nil
}

func (h *Handler) update(ctx context.Context, input *updateInput) (*Output, error) {
	userID, ok := auth.GetUserID(ctx)
	if !ok {
		return nil, huma.Error401Unauthorized("Unauthorized")
	}

	err := h.service.Update(ctx, userID, input.ID, input.Body.Type, input.Body.EncryptedData, input.Body.Meta)
	if err != nil {
		return &Output{
			Body: Response{
				ID:     input.ID,
				Status: "Error",
			},
		}, err
	}
	return &Output{
		Body: Response{
			ID:     input.ID,
			Status: "Ok",
		},
	}, nil
}

func (h *Handler) delete(ctx context.Context, input *updateInput) (*Output, error) {
	userID, ok := auth.GetUserID(ctx)
	if !ok {
		return nil, huma.Error401Unauthorized("Unauthorized")
	}

	err := h.service.Delete(ctx, userID, input.ID)
	if err != nil {
		return &Output{
			Body: Response{
				Status: "Error",
			},
		}, err
	}
	return &Output{
		Body: Response{
			Status: "Ok",
		},
	}, nil
}
