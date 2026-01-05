package record

import (
	"fmt"
	"github.com/danielgtaylor/huma/v2"
)

type RecType string

const (
	RecTypeLogin  RecType = "login"
	RecTypeText   RecType = "text"
	RecTypeBinary RecType = "binary"
	RecTypeCard   RecType = "card"
)

func (RecType) Schema() huma.Schema {
	return huma.Schema{
		Type: "string",
		Enum: []any{
			string(RecTypeLogin),
			string(RecTypeText),
			string(RecTypeBinary),
			string(RecTypeCard),
		},
		Description: "Тип хранимой записи",
		Examples:    []any{RecTypeLogin},
	}
}

// Validate реализует интерфейс huma.Validatable.
func (t RecType) Validate() error {
	switch t {
	case RecTypeLogin, RecTypeText, RecTypeBinary, RecTypeCard:
		return nil
	}
	return fmt.Errorf("неверный тип записи: %s", t)
}

// String возвращает строковое представление типа.
func (t RecType) String() string {
	return string(t)
}

// DisplayName возвращает человекочитаемое название типа.
func (t RecType) DisplayName() string {
	switch t {
	case RecTypeLogin:
		return "Логин/Пароль"
	case RecTypeText:
		return "Текстовые данные"
	case RecTypeBinary:
		return "Бинарные данные"
	case RecTypeCard:
		return "Банковская карта"
	default:
		return "Неизвестный тип"
	}
}
