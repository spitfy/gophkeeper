package record

import (
	"encoding/json"
	"time"
)

type RecordItem struct {
	ID           int             `json:"id"`
	Type         RecType         `json:"type"`
	Meta         json.RawMessage `json:"meta"`
	Version      int             `json:"version"`
	LastModified time.Time       `json:"last_modified"`
}

type ListResponse struct {
	Records []RecordItem `json:"records"`
	Total   int          `json:"total"`
}
