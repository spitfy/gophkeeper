package user

type registerInput struct {
	Body baseRequest
}

type registerOutput struct {
	Body registerResponse
}

type baseRequest struct {
	Login    string `json:"login" validate:"required,min=3,max=20"`
	Password string `json:"password" validate:"required,min=4,max=20"`
}

type registerResponse struct {
	ID     int    `json:"user_id"`
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

type loginInput struct {
	Body baseRequest
}

type loginOutput struct {
	Body loginResponse
}

type loginResponse struct {
	Token  string `json:"token"`
	Status string `json:"status"`
	Error  string `json:"error"`
}
