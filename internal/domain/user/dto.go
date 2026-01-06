package user

type registerInput struct {
	Body BaseRequest
}

type registerOutput struct {
	Body RegisterResponse
}

type BaseRequest struct {
	Login    string `json:"login" validate:"required,min=3,max=20"`
	Password string `json:"password" validate:"required,min=4,max=20"`
}

type RegisterResponse struct {
	ID     int    `json:"user_id"`
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

type loginInput struct {
	Body BaseRequest
}

type loginOutput struct {
	Body LoginResponse
}

type LoginResponse struct {
	Token  string `json:"token"`
	Status string `json:"status"`
	Error  string `json:"error"`
}
