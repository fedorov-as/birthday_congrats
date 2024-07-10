package congrats_service

import (
	"birthday_congrats/internal/pkg/session"
	"birthday_congrats/internal/pkg/user"
	"context"
	"sync"
	"time"
)

type CongratulationsService interface {
	Register(ctx context.Context, username, password, email, birth string) (*session.Session, error)
	Login(ctx context.Context, username, password string) (*session.Session, error)
	Subscribe(ctx context.Context, subscriptionID uint32, daysAlert int) error
	Unsubscribe(ctx context.Context, subscriptionID uint32) error
	Logout(ctx context.Context) error
	GetSubscriptionsByUser(ctx context.Context) ([]*user.User, error)
	StartAlert(ctx context.Context, timeStart time.Time, period time.Duration, wg *sync.WaitGroup)
}
