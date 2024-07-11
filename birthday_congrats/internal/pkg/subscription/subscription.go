package subscription

import (
	"context"

	"github.com/pkg/errors"
)

var (
	ErrAddSubscription    = errors.New("subscription was not added")
	ErrRemoveSubscription = errors.New("no subscription to remove")
)

type Subscription struct {
	Subscriber   uint32
	Subscription uint32
	DaysAlert    int
}

type SubscriptionsRepo interface {
	GetAllSubscriptions(ctx context.Context) ([]*Subscription, error)
	GetSubscriptionsByUser(ctx context.Context, userID uint32) ([]*Subscription, error)
	AddSubscription(ctx context.Context, subscriberID, subscriptionID uint32, daysAlert int) error
	RemoveSubscription(ctx context.Context, subscriberID, subscriptionID uint32) error
}
