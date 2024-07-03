package user

import (
	"context"

	"github.com/pkg/errors"
)

var (
	ErrUserExists     = errors.New("username already exists")
	ErrUserNotCreated = errors.New("user was not created")
	ErrNoUser         = errors.New("no such user")
	ErrBadPassword    = errors.New("bad password")

	ErrAddSubscription    = errors.New("subscription was not added")
	ErrRemoveSubscription = errors.New("no subscription to remove")
)

type User struct {
	ID       uint32 `json:"id" sql:"AUTO_INCREMENT"`
	Username string `json:"username"`
	Password string
	Email    string `json:"email"`
	Birthday int64  `json:"birthday"`
}

type UsersRepo interface {
	Create(ctx context.Context, username, password, email string, birth int64) (*User, error)
	Login(ctx context.Context, username, password string) (*User, error)
	GetAll(ctx context.Context) ([]*User, error)
	GetSubscribtions(ctx context.Context, userID uint32) ([]uint32, error)
	AddSubscribtion(ctx context.Context, subscriberID, subscriptionID uint32) error
	RemoveSubscription(ctx context.Context, subscriberID, subscriptionID uint32) error
}
