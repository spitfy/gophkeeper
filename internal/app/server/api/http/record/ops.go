package record

import (
	"net/http"

	"github.com/danielgtaylor/huma/v2"
)

func (h *Handler) listOp() huma.Operation {
	return huma.Operation{
		OperationID: "records-list",
		Method:      http.MethodGet,
		Path:        "/api/records",
		Summary:     "Список записей пользователя",
		Tags:        []string{"records"},
		Security:    []map[string][]string{{"bearer": {}}},
		Middlewares: h.middleware,
	}
}

func (h *Handler) createOp() huma.Operation {
	return huma.Operation{
		OperationID: "records-create",
		Method:      http.MethodPost,
		Path:        "/api/records",
		Summary:     "Создать запись (generic)",
		Description: "Создает запись с зашифрованными данными. Для типизированного создания используйте специализированные эндпоинты.",
		Tags:        []string{"records"},
		Security:    []map[string][]string{{"bearer": {}}},
		Middlewares: h.middleware,
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
		Middlewares: h.middleware,
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
		Middlewares: h.middleware,
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
		Middlewares: h.middleware,
	}
}

// ==================== Typed Create Operations ====================

func (h *Handler) createLoginOp() huma.Operation {
	return huma.Operation{
		OperationID: "records-create-login",
		Method:      http.MethodPost,
		Path:        "/api/records/login",
		Summary:     "Создать запись логина",
		Description: "Создает запись с учетными данными (логин/пароль) для веб-сайта или сервиса.",
		Tags:        []string{"records", "login"},
		Security:    []map[string][]string{{"bearer": {}}},
		Middlewares: h.middleware,
	}
}

func (h *Handler) createTextOp() huma.Operation {
	return huma.Operation{
		OperationID: "records-create-text",
		Method:      http.MethodPost,
		Path:        "/api/records/text",
		Summary:     "Создать текстовую запись",
		Description: "Создает запись с текстовым содержимым (заметки, секреты, конфигурации и т.д.).",
		Tags:        []string{"records", "text"},
		Security:    []map[string][]string{{"bearer": {}}},
		Middlewares: h.middleware,
	}
}

func (h *Handler) createCardOp() huma.Operation {
	return huma.Operation{
		OperationID: "records-create-card",
		Method:      http.MethodPost,
		Path:        "/api/records/card",
		Summary:     "Создать запись банковской карты",
		Description: "Создает запись с данными банковской карты (номер, CVV, срок действия и т.д.).",
		Tags:        []string{"records", "card"},
		Security:    []map[string][]string{{"bearer": {}}},
		Middlewares: h.middleware,
	}
}

func (h *Handler) createBinaryOp() huma.Operation {
	return huma.Operation{
		OperationID: "records-create-binary",
		Method:      http.MethodPost,
		Path:        "/api/records/binary",
		Summary:     "Создать бинарную запись",
		Description: "Создает запись с бинарными данными (файлы, изображения, документы). Данные передаются в base64.",
		Tags:        []string{"records", "binary"},
		Security:    []map[string][]string{{"bearer": {}}},
		Middlewares: h.middleware,
	}
}
