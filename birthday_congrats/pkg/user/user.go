package user

import (
	"context"
)

type User struct {
	ID         uint32   `json:"id" sql:"AUTO_INCREMENT"`
	Username   string   `json:"username"`
	password   string   `json:"password"`
	Birthday   int64    `json:"birthday"`
	Email      string   `json:"email"`
	Subscribes []uint32 `json:"subscribes"`
}

type UsersRepo interface {
	Create(ctx context.Context, username, password string) (*User, error)
	Login(ctx context.Context, username, password string) (*User, error)
	GetAll(ctx context.Context) ([]*User, error)
}
