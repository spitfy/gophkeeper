package record

import (
	"context"
	"github.com/danielgtaylor/huma/v2"
	"golang.org/x/exp/slog"
	"gophkeeper/internal/domain/session"
	"net/http"
)

type Handler struct {
	service Servicer
	session session.Servicer
	log     *slog.Logger
}

func (h *Handler) SetupRoutes(api huma.API) {
	huma.Register(api, h.listOp(), h.list)
	huma.Register(api, h.createOp(), h.create)
	huma.Register(api, h.getOp(), h.get)
	huma.Register(api, h.updateOp(), h.update)
	huma.Register(api, h.deleteOp(), h.delete)
}

func (h *Handler) listOp() huma.Operation {
	return huma.Operation{
		OperationID: "records-list",
		Method:      http.MethodGet,
		Path:        "/api/records",
		Summary:     "Список записей пользователя",
		Tags:        []string{"records"},
		Security:    []map[string][]string{{"bearer": {}}},
	}
}

func (h *Handler) createOp() huma.Operation {
	return huma.Operation{
		OperationID: "records-create",
		Method:      http.MethodPost,
		Path:        "/api/records",
		Summary:     "Создать запись",
		Tags:        []string{"records"},
		Security:    []map[string][]string{{"bearer": {}}},
	}
}

func (h *Handler) getOp() huma.Operation {
	return huma.Operation{
		OperationID: "records-get",
		Method:      http.MethodGet,
		Path:        "/api/records/{id}",
		Summary:     "Получить запись",
		Tags:        []string{"records"},
		Security:    []map[string][]string{{"bearer": {}}},
	}
}

func (h *Handler) updateOp() huma.Operation {
	return huma.Operation{
		OperationID: "records-update",
		Method:      http.MethodPut,
		Path:        "/api/records/{id}",
		Summary:     "Обновить запись",
		Tags:        []string{"records"},
		Security:    []map[string][]string{{"bearer": {}}},
	}
}

func (h *Handler) deleteOp() huma.Operation {
	return huma.Operation{
		OperationID: "records-delete",
		Method:      http.MethodDelete,
		Path:        "/api/records/{id}",
		Summary:     "Удалить запись",
		Tags:        []string{"records"},
		Security:    []map[string][]string{{"bearer": {}}},
	}
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

func (h *Handler) create(ctx context.Context, input *createInput) (*createOutput, error) {
	userID := ctx.Value("userID").(int)

	// TODO: расшифровка/проверка data на клиенте, здесь только хранение
	response, err := h.service.create(ctx, userID, input.Body.Type,
		input.Body.EncryptedData, input.Body.Meta)
	if err != nil {
		return &createOutput{
			Body: response,
		}, err
	}

	return &createOutput{
		Body: response,
	}, nil
}
