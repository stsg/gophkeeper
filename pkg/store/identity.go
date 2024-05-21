package postgres

import (
	"fmt"
)

var (
	ErrUserNotFound      = fmt.Errorf("user not found")
	ErrUserExists        = fmt.Errorf("user exists")
	ErrUserUnauthorized  = fmt.Errorf("user unauthorized")
	ErrUserWrong         = fmt.Errorf("user wrong")
	ErrUserWrongPassword = fmt.Errorf("user password wrong")
)

type User struct {
	Login string `json:"login,omitempty" db:"id"`
	Passw string `json:"passw,omitempty" db:"passw"`
}
