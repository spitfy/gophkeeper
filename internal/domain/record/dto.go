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

type listOutput struct {
	Body ListResponse
}

type createInput struct {
	Body Request
}

type Output struct {
	Body Response
}

type findOutput struct {
	Body FindResponse
}

type findInput struct {
	ID int `path:"id" example:"1" doc:"ID записи"`
}

type updateInput struct {
	ID   int `path:"id" example:"1" doc:"ID записи"`
	Body Request
}

type Request struct {
	Type          RecType         `json:"type" doc:"Тип записи, одно из login, text, binary, card"`
	EncryptedData string          `json:"data"` // base64
	Meta          json.RawMessage `json:"meta"`
}

type Response struct {
	ID     int    `json:"id"`
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

type FindResponse struct {
	Status string  `json:"status"`
	Record *Record `json:"record"`
	Error  string  `json:"error,omitempty"`
}
