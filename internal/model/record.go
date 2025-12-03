package model

import "time"

type Record struct {
	ID            int       `db:"id"`
	UserID        int       `db:"user_id"`
	Type          string    `db:"type"` // "login", "text", "binary", "card"
	EncryptedData string    `db:"encrypted_data"`
	Meta          string    `db:"meta"` // JSON
	Version       int       `db:"version"`
	LastModified  time.Time `db:"last_modified"`
}

type User struct {
	ID       int    `json:"id"`
	Login    string `json:"login"`
	Password string `json:"password"`
}
