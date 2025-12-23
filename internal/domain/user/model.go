package user

import "time"

type User struct {
	ID        int
	Login     string
	Password  string // хэш
	CreatedAt time.Time
}
