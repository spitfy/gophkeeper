package sync

import (
	"context"

	"github.com/danielgtaylor/huma/v2"
	"golang.org/x/exp/slog"
	"gophkeeper/internal/domain/sync"
)

type Handler struct {
	service    sync.Servicer
	log        *slog.Logger
	middleware huma.Middlewares
}

func NewHandler(service sync.Servicer, log *slog.Logger, middleware huma.Middlewares) *Handler {
	return &Handler{
		service:    service,
		log:        log,
		middleware: middleware,
	}
}

func (h *Handler) SetupRoutes(api huma.API) {
	huma.Register(api, h.getChangesOp(), h.getChanges)
	huma.Register(api, h.batchSyncOp(), h.batchSync)
	huma.Register(api, h.getStatusOp(), h.getStatus)
	huma.Register(api, h.getConflictsOp(), h.getConflicts)
	huma.Register(api, h.resolveConflictOp(), h.resolveConflict)
	huma.Register(api, h.getDevicesOp(), h.getDevices)
	huma.Register(api, h.removeDeviceOp(), h.removeDevice)
}

func (h *Handler) getChanges(ctx context.Context, input *getChangesInput) (*getChangesOutput, error) {
	response, err := h.service.GetChanges(ctx, input.Body)
	if err != nil {
		return &getChangesOutput{
			Body: sync.GetChangesResponse{
				Status: "Error",
				Error:  err.Error(),
			},
		}, nil
	}

	return &getChangesOutput{
		Body: *response,
	}, nil
}

func (h *Handler) batchSync(ctx context.Context, input *batchSyncInput) (*batchSyncOutput, error) {
	response, err := h.service.ProcessBatch(ctx, input.Body)
	if err != nil {
		return &batchSyncOutput{
			Body: sync.BatchSyncResponse{
				Status: "Error",
				Error:  err.Error(),
			},
		}, nil
	}

	return &batchSyncOutput{
		Body: *response,
	}, nil
}

func (h *Handler) getStatus(ctx context.Context, _ *getStatusInput) (*getStatusOutput, error) {
	response, err := h.service.GetStatus(ctx)
	if err != nil {
		return &getStatusOutput{
			Body: sync.GetStatusResponse{
				Status: "Error",
				Error:  err.Error(),
			},
		}, nil
	}

	return &getStatusOutput{
		Body: *response,
	}, nil
}

func (h *Handler) getConflicts(ctx context.Context, _ *getConflictsInput) (*getConflictsOutput, error) {
	response, err := h.service.GetConflicts(ctx)
	if err != nil {
		return &getConflictsOutput{
			Body: sync.GetConflictsResponse{
				Status: "Error",
				Error:  err.Error(),
			},
		}, nil
	}

	return &getConflictsOutput{
		Body: sync.GetConflictsResponse{
			Status: "Ok",
			Data:   response,
		},
	}, nil
}

func (h *Handler) resolveConflict(ctx context.Context, input *resolveConflictInput) (*resolveConflictOutput, error) {
	response, err := h.service.ResolveConflict(ctx, input.ID, input.Body)
	if err != nil {
		return &resolveConflictOutput{
			Body: sync.ResolveConflictResponse{
				Status: "Error",
				Error:  err.Error(),
			},
		}, nil
	}

	return &resolveConflictOutput{
		Body: *response,
	}, nil
}

func (h *Handler) getDevices(ctx context.Context, _ *getDevicesInput) (*getDevicesOutput, error) {
	response, err := h.service.GetDevices(ctx)
	if err != nil {
		return &getDevicesOutput{
			Body: sync.GetDevicesResponse{
				Status: "Error",
				Error:  err.Error(),
			},
		}, nil
	}

	return &getDevicesOutput{
		Body: sync.GetDevicesResponse{
			Status: "Ok",
			Data:   response,
		},
	}, nil
}

func (h *Handler) removeDevice(ctx context.Context, input *removeDeviceInput) (*removeDeviceOutput, error) {
	response, err := h.service.RemoveDevice(ctx, input.ID)
	if err != nil {
		return &removeDeviceOutput{
			Body: sync.RemoveDeviceResponse{
				Status: "Error",
				Error:  err.Error(),
			},
		}, nil
	}

	return &removeDeviceOutput{
		Body: *response,
	}, nil
}
