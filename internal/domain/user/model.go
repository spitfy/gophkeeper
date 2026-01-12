package user

import "time"

type User struct {
	ID        int
	Login     string
	Password  string // хэш
	CreatedAt time.Time
}

type BaseRequest struct {
	Login    string `json:"login" validate:"required,min=3,max=20"`
	Password string `json:"password" validate:"required,min=4,max=20"`
}

type ChangePasswordRequest struct {
	CurrentPassword string `json:"password" validate:"required,min=4,max=20"`
	NewPassword     string `json:"password" validate:"required,min=4,max=20"`
	MasterPassword  string `json:"password" validate:"required,min=4,max=20"`
}
