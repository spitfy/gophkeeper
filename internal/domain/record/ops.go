package record

import (
	"github.com/danielgtaylor/huma/v2"
	"net/http"
)

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

func (h *Handler) findOp() huma.Operation {
	return huma.Operation{
		OperationID: "records-find",
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
