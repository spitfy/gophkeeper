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

type listResponse struct {
	Records []RecordItem `json:"records"`
	Total   int          `json:"total"`
}

type listOutput struct {
	Body listResponse
}

type createInput struct {
	Body request
}

type Output struct {
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
	Type          RecType         `json:"type" doc:"Тип записи, одно из login, text, binary, card"`
	EncryptedData string          `json:"data"` // base64
	Meta          json.RawMessage `json:"meta"`
}

type response struct {
	ID     int    `json:"id"`
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

type findResponse struct {
	Status string  `json:"status"`
	Record *Record `json:"record"`
	Error  string  `json:"error,omitempty"`
}
