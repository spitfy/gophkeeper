package user

import "gophkeeper/internal/domain/user"

type registerInput struct {
	Body user.BaseRequest
}

type registerOutput struct {
	Body RegisterResponse
}

type RegisterResponse struct {
	ID     int    `json:"user_id"`
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

type loginInput struct {
	Body user.BaseRequest
}

type loginOutput struct {
	Body LoginResponse
}

type LoginResponse struct {
	Token  string `json:"token"`
	Status string `json:"status"`
	Error  string `json:"error"`
}
