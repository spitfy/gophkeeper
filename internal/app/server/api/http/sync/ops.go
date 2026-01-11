package sync

import (
	"net/http"

	"github.com/danielgtaylor/huma/v2"
)

func (h *Handler) getChangesOp() huma.Operation {
	return huma.Operation{
		OperationID: "sync-get-changes",
		Method:      http.MethodPost,
		Path:        "/api/sync/changes",
		Summary:     "Получить изменения для синхронизации",
		Description: "Возвращает записи, измененные после указанного времени",
		Tags:        []string{"sync"},
		Middlewares: h.middleware,
	}
}

func (h *Handler) batchSyncOp() huma.Operation {
	return huma.Operation{
		OperationID: "sync-batch",
		Method:      http.MethodPost,
		Path:        "/api/sync/batch",
		Summary:     "Пакетная синхронизация записей",
		Description: "Принимает пакет записей для синхронизации с сервером",
		Tags:        []string{"sync"},
		Middlewares: h.middleware,
	}
}

func (h *Handler) getStatusOp() huma.Operation {
	return huma.Operation{
		OperationID: "sync-get-status",
		Method:      http.MethodGet,
		Path:        "/api/sync/status",
		Summary:     "Получить статус синхронизации",
		Description: "Возвращает текущий статус синхронизации пользователя",
		Tags:        []string{"sync"},
		Middlewares: h.middleware,
	}
}

func (h *Handler) getConflictsOp() huma.Operation {
	return huma.Operation{
		OperationID: "sync-get-conflicts",
		Method:      http.MethodGet,
		Path:        "/api/sync/conflicts",
		Summary:     "Получить конфликты синхронизации",
		Description: "Возвращает список неразрешенных конфликтов",
		Tags:        []string{"sync"},
		Middlewares: h.middleware,
	}
}

func (h *Handler) resolveConflictOp() huma.Operation {
	return huma.Operation{
		OperationID: "sync-resolve-conflict",
		Method:      http.MethodPost,
		Path:        "/api/sync/conflicts/{id}/resolve",
		Summary:     "Разрешить конфликт синхронизации",
		Description: "Разрешает указанный конфликт",
		Tags:        []string{"sync"},
		Middlewares: h.middleware,
	}
}

func (h *Handler) getDevicesOp() huma.Operation {
	return huma.Operation{
		OperationID: "sync-get-devices",
		Method:      http.MethodGet,
		Path:        "/api/sync/devices",
		Summary:     "Получить список устройств",
		Description: "Возвращает список всех устройств пользователя",
		Tags:        []string{"sync"},
		Middlewares: h.middleware,
	}
}

func (h *Handler) removeDeviceOp() huma.Operation {
	return huma.Operation{
		OperationID: "sync-remove-device",
		Method:      http.MethodDelete,
		Path:        "/api/sync/devices/{id}",
		Summary:     "Удалить устройство",
		Description: "Удаляет устройство из списка синхронизации",
		Tags:        []string{"sync"},
		Middlewares: h.middleware,
	}
}
