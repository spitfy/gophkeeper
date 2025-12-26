package record

import (
	"encoding/json"
	"time"
)

type RecordItem struct {
	ID           int             `json:"id"`
	Type         Type            `json:"type"`
	Meta         json.RawMessage `json:"meta"`
	Version      int             `json:"version"`
	LastModified time.Time       `json:"last_modified"`
}

type listResponse struct {
	Records []RecordItem `json:"records"`
}

type listOutput struct {
	Body listResponse
}

type createInput struct {
	Body createRequest
}

type createOutput struct {
	Body createResponse
}

type createRequest struct {
	Type          string          `json:"type"`
	EncryptedData string          `json:"data" format:"binary"` // base64
	Meta          json.RawMessage `json:"meta"`
}

type createResponse struct {
	ID     int    `json:"id"`
	Status string `json:"status"`
	Error  string `json:"error"`
}
