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
	ID     int    `json:"id"`
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

type findResponse struct {
	Status string         `json:"status"`
	Record *record.Record `json:"record"`
	Error  string         `json:"error,omitempty"`
}
