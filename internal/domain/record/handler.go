package record

import (
	"context"
	"github.com/danielgtaylor/huma/v2"
	"golang.org/x/exp/slog"
)

type Handler struct {
	service Servicer
	log     *slog.Logger
}

func NewHandler(service Servicer, log *slog.Logger) *Handler {
	return &Handler{
		service: service,
		log:     log,
	}
}

func (h *Handler) SetupRoutes(api huma.API) {
	huma.Register(api, h.listOp(), h.list)
	huma.Register(api, h.createOp(), h.create)
	huma.Register(api, h.findOp(), h.find)
	huma.Register(api, h.updateOp(), h.update)
	huma.Register(api, h.deleteOp(), h.delete)
}

func (h *Handler) list(ctx context.Context, _ *struct{}) (*listOutput, error) {
	userID := ctx.Value("userID").(int)
	records, err := h.service.list(ctx, userID)
	if err != nil {
		return nil, err
	}

	return &listOutput{
		Body: records,
	}, nil
}

func (h *Handler) find(ctx context.Context, input *findInput) (*findOutput, error) {
	userID := ctx.Value("userID").(int)

	record, err := h.service.Find(ctx, userID, input.ID)
	if err != nil {
		return &findOutput{
			Body: findResponse{
				Status: "Error",
			},
		}, err
	}

	return &findOutput{
		Body: findResponse{
			Status: "Ok",
			Record: record,
		},
	}, nil
}

func (h *Handler) create(ctx context.Context, input *createInput) (*output, error) {
	userID := ctx.Value("userID").(int)

	response, err := h.service.create(ctx, userID, input.Body.Type, input.Body.EncryptedData, input.Body.Meta)
	if err != nil {
		return &output{
			Body: response,
		}, err
	}

	return &output{
		Body: response,
	}, nil
}

func (h *Handler) update(ctx context.Context, input *updateInput) (*output, error) {
	userID := ctx.Value("userID").(int)

	err := h.service.Update(ctx, userID, input.ID, input.Body.Type, input.Body.EncryptedData, input.Body.Meta)
	if err != nil {
		return &output{
			Body: response{
				ID:     input.ID,
				Status: "Error",
			},
		}, err
	}
	return &output{
		Body: response{
			ID:     input.ID,
			Status: "Ok",
		},
	}, nil
}

func (h *Handler) delete(ctx context.Context, input *updateInput) (*output, error) {
	userID := ctx.Value("userID").(int)

	err := h.service.Delete(ctx, userID, input.ID)
	if err != nil {
		return &output{
			Body: response{
				Status: "Error",
			},
		}, err
	}
	return &output{
		Body: response{
			Status: "Ok",
		},
	}, nil
}
