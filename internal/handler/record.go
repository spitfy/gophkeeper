package handler

import (
	"github.com/danielgtaylor/huma/v2"
	"net/http"
)

func (a *API) recordsListOp() huma.Operation {
	return huma.Operation{
		OperationID: "records-list",
		Method:      http.MethodGet,
		Path:        "/api/records",
		Summary:     "Список записей пользователя",
		Tags:        []string{"records"},
		Security:    []map[string][]string{{"bearer": {}}},
	}
}

func (a *API) recordsCreateOp() huma.Operation {
	return huma.Operation{
		OperationID: "records-create",
		Method:      http.MethodPost,
		Path:        "/api/records",
		Summary:     "Создать запись",
		Tags:        []string{"records"},
		Security:    []map[string][]string{{"bearer": {}}},
	}
}

func (a *API) recordsGetOp() huma.Operation {
	return huma.Operation{
		OperationID: "records-get",
		Method:      http.MethodGet,
		Path:        "/api/records/{id}",
		Summary:     "Получить запись",
		Tags:        []string{"records"},
		Security:    []map[string][]string{{"bearer": {}}},
	}
}

func (a *API) recordsUpdateOp() huma.Operation {
	return huma.Operation{
		OperationID: "records-update",
		Method:      http.MethodPut,
		Path:        "/api/records/{id}",
		Summary:     "Обновить запись",
		Tags:        []string{"records"},
		Security:    []map[string][]string{{"bearer": {}}},
	}
}

func (a *API) recordsDeleteOp() huma.Operation {
	return huma.Operation{
		OperationID: "records-delete",
		Method:      http.MethodDelete,
		Path:        "/api/records/{id}",
		Summary:     "Удалить запись",
		Tags:        []string{"records"},
		Security:    []map[string][]string{{"bearer": {}}},
	}
}
