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
	ID       uint32 `sql:"AUTO_INCREMENT"`
	Username string
	Password string
	Email    string
	Year     int
	Month    int
	Day      int

	// вспомогательные поле (подписка какого-то пользователя на текущего)
	Subscription bool
	DaysAlert    int
}

type Subscription struct {
	Subscriber   uint32
	Subscription uint32
	DaysAlert    int
}

type UsersRepo interface {
	Create(ctx context.Context, username, password, email string, year, month, day int) (*User, error)
	Login(ctx context.Context, username, password string) (*User, error)
	GetAll(ctx context.Context) ([]*User, error)
	GetByID(ctx context.Context, userID uint32) (*User, error)
	GetSubscriptionsByUser(ctx context.Context, userID uint32) ([]Subscription, error)
	// GetSubscribers(ctx context.Context, userID uint32) ([]uint32, error)
	AddSubscription(ctx context.Context, subscriberID, subscriptionID uint32, daysAlert int) error
	RemoveSubscription(ctx context.Context, subscriberID, subscriptionID uint32) error
	// GetSubscribedEmailsByDate(ctx context.Context, month, day int) (map[string][]string, error)
	GetAllSubscriptions(ctx context.Context) ([]Subscription, error)
}
