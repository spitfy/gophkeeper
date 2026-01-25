package record

import (
	"encoding/json"
	"time"
)

type Record struct {
	ID            int             `json:"id"`
	UserID        int             `json:"user_id"`
	Type          RecType         `json:"type"`
	EncryptedData string          `json:"encrypted_data,omitempty"`
	Meta          json.RawMessage `json:"meta,omitempty"`
	Version       int             `json:"version"`
	LastModified  time.Time       `json:"last_modified"`
	DeletedAt     *time.Time      `json:"deleted_at,omitempty"`
	Checksum      string          `json:"checksum,omitempty"`
	DeviceID      string          `json:"device_id,omitempty"`
}

type BaseRecord struct {
	ID            int             `json:"id"`
	UserID        int             `json:"user_id"`
	Type          RecType         `json:"type"`
	EncryptedData string          `json:"encrypted_data"`
	Meta          json.RawMessage `json:"meta"`
	Version       int             `json:"version"`
	LastModified  time.Time       `json:"last_modified"`
	DeletedAt     *time.Time      `json:"deleted_at,omitempty"`
	Checksum      string          `json:"checksum,omitempty"`
	DeviceID      string          `json:"device_id,omitempty"`
}

// RecordData - интерфейс для данных записи (до шифрования)
type RecordData interface {
	GetType() RecType
	Validate() error
	ToJSON() ([]byte, error)
	FromJSON(data []byte) error
}

// MetaData - интерфейс для метаданных
type MetaData interface {
	ToJSON() ([]byte, error)
	FromJSON(data []byte) error
	Validate() error
}

type Version struct {
	ID            int             `json:"id"`
	RecordID      int             `json:"record_id"`
	Version       int             `json:"version"`
	EncryptedData string          `json:"encrypted_data"`
	Meta          json.RawMessage `json:"meta"`
	Checksum      string          `json:"checksum"`
	CreatedAt     time.Time       `json:"created_at"`
}

// BatchUpdate представляет пакетное обновление записей
type BatchUpdate struct {
	Records []Record `json:"records"`
}

// SearchCriteria критерии поиска записей
type SearchCriteria struct {
	Type      string
	MetaQuery json.RawMessage
	FromDate  *time.Time
	ToDate    *time.Time
	Limit     int
	Offset    int
}
