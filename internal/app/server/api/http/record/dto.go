package record

import (
	"encoding/json"
	"gophkeeper/internal/domain/record"
)

type listOutput struct {
	Body record.ListResponse
}

type createInput struct {
	Body request
}

type output struct {
	Body response
}

type findOutput struct {
	Body findResponse
}

type findInput struct {
	ID int `path:"id" example:"1" doc:"ID записи"`
}

type updateInput struct {
	ID   int `path:"id" example:"1" doc:"ID записи"`
	Body request
}

type request struct {
	Type          record.RecType  `json:"type" doc:"Тип записи, одно из login, text, binary, card"`
	EncryptedData string          `json:"data"` // base64
	Meta          json.RawMessage `json:"meta"`
}

type response struct {
	ID      int    `json:"id,omitempty"`
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
}

type findResponse struct {
	Status string         `json:"status"`
	Record *record.Record `json:"record"`
	Error  string         `json:"error,omitempty"`
}

// ==================== Login ====================

type createLoginInput struct {
	Body createLoginRequest
}

type createLoginRequest struct {
	// Data fields
	Username string `json:"username" doc:"Имя пользователя" minLength:"1"`
	Password string `json:"password" doc:"Пароль" minLength:"1"`
	Notes    string `json:"notes,omitempty" doc:"Заметки"`

	// Meta fields
	Title     string   `json:"title" doc:"Название записи" minLength:"1"`
	Resource  string   `json:"resource" doc:"URL или название ресурса" minLength:"1"`
	Category  string   `json:"category,omitempty" doc:"Категория"`
	Tags      []string `json:"tags,omitempty" doc:"Теги"`
	TwoFA     bool     `json:"two_fa,omitempty" doc:"Включена ли двухфакторная аутентификация"`
	TwoFAType string   `json:"two_fa_type,omitempty" doc:"Тип 2FA: totp, sms, email"`

	// Common fields
	DeviceID string `json:"device_id,omitempty" doc:"ID устройства"`
}

// ==================== Text ====================

type createTextInput struct {
	Body createTextRequest
}

type createTextRequest struct {
	// Data fields
	Content string `json:"content" doc:"Текстовое содержимое" minLength:"1"`

	// Meta fields
	Title       string   `json:"title" doc:"Название записи" minLength:"1"`
	Category    string   `json:"category,omitempty" doc:"Категория"`
	Tags        []string `json:"tags,omitempty" doc:"Теги"`
	Format      string   `json:"format,omitempty" doc:"Формат: plain, markdown, html, json, xml, yaml"`
	Language    string   `json:"language,omitempty" doc:"Язык: ru, en и т.д."`
	IsSensitive bool     `json:"is_sensitive,omitempty" doc:"Содержит чувствительные данные"`

	// Common fields
	DeviceID string `json:"device_id,omitempty" doc:"ID устройства"`
}

// ==================== Card ====================

type createCardInput struct {
	Body createCardRequest
}

type createCardRequest struct {
	// Data fields
	CardNumber     string `json:"card_number" doc:"Номер карты" minLength:"13" maxLength:"19"`
	CardHolder     string `json:"card_holder" doc:"Имя держателя карты" minLength:"1"`
	ExpiryMonth    string `json:"expiry_month" doc:"Месяц истечения (01-12)" pattern:"^(0[1-9]|1[0-2])$"`
	ExpiryYear     string `json:"expiry_year" doc:"Год истечения (20XX)" pattern:"^20\\d{2}$"`
	CVV            string `json:"cvv" doc:"CVV код" minLength:"3" maxLength:"4"`
	PIN            string `json:"pin,omitempty" doc:"PIN код"`
	BillingAddress string `json:"billing_address,omitempty" doc:"Адрес для выставления счетов"`

	// Meta fields
	Title         string   `json:"title" doc:"Название записи" minLength:"1"`
	BankName      string   `json:"bank_name,omitempty" doc:"Название банка"`
	PaymentSystem string   `json:"payment_system,omitempty" doc:"Платежная система: visa, mastercard, mir, unionpay, amex, jcb"`
	Category      string   `json:"category,omitempty" doc:"Категория"`
	Tags          []string `json:"tags,omitempty" doc:"Теги"`
	Notes         string   `json:"notes,omitempty" doc:"Заметки"`
	IsVirtual     bool     `json:"is_virtual,omitempty" doc:"Виртуальная карта"`
	IsActive      bool     `json:"is_active,omitempty" doc:"Карта активна"`
	DailyLimit    *float64 `json:"daily_limit,omitempty" doc:"Дневной лимит"`
	PhoneNumber   string   `json:"phone_number,omitempty" doc:"Привязанный телефон"`

	// Common fields
	DeviceID string `json:"device_id,omitempty" doc:"ID устройства"`
}

// ==================== Binary ====================

type createBinaryInput struct {
	Body createBinaryRequest
}

type createBinaryRequest struct {
	// Data fields
	Data        string `json:"data" doc:"Base64-encoded бинарные данные" minLength:"1"`
	Filename    string `json:"filename" doc:"Имя файла" minLength:"1"`
	ContentType string `json:"content_type,omitempty" doc:"MIME тип файла"`

	// Meta fields
	Title        string                  `json:"title" doc:"Название записи" minLength:"1"`
	Category     string                  `json:"category,omitempty" doc:"Категория"`
	Tags         []string                `json:"tags,omitempty" doc:"Теги"`
	Description  string                  `json:"description,omitempty" doc:"Описание файла"`
	OriginalHash string                  `json:"original_hash,omitempty" doc:"SHA256 хеш оригинального файла"`
	Compression  *record.CompressionInfo `json:"compression,omitempty" doc:"Информация о сжатии"`
	Encryption   *record.EncryptionInfo  `json:"encryption,omitempty" doc:"Информация о шифровании"`
	Dimensions   *record.ImageDimensions `json:"dimensions,omitempty" doc:"Размеры изображения (для картинок)"`
	Duration     *float64                `json:"duration,omitempty" doc:"Длительность в секундах (для аудио/видео)"`

	// Common fields
	DeviceID string `json:"device_id,omitempty" doc:"ID устройства"`
}
