package record

import (
	"encoding/json"
	"time"
)

type Type string

const (
	TypePassword Type = "password"
	TypeFile     Type = "file"
	TypeCard     Type = "card"
	TypeNote     Type = "note"
)

type Record struct {
	ID            int             `json:"id"`
	UserID        int             `json:"-"`
	Type          Type            `json:"type"`
	EncryptedData string          `json:"-"`
	Meta          json.RawMessage `json:"meta"`
	Version       int             `json:"version"`
	LastModified  time.Time       `json:"last_modified"`
}
