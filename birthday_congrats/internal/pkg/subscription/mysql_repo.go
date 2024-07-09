package subscription

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/go-sql-driver/mysql"
	"go.uber.org/zap"
)

type SubscriptionsMySQLRepo struct {
	db     *sql.DB
	logger *zap.SugaredLogger
}

var _ SubscriptionsRepo = &SubscriptionsMySQLRepo{}

func NewSubscriptionsMySQLRepo(db *sql.DB, logger *zap.SugaredLogger) *SubscriptionsMySQLRepo {
	return &SubscriptionsMySQLRepo{
		db:     db,
		logger: logger,
	}
}

func (repo *SubscriptionsMySQLRepo) GetAllSubscriptions(ctx context.Context) ([]Subscription, error) {
	subscriptions := make([]Subscription, 0, 10)

	rows, err := repo.db.QueryContext(
		ctx,
		"SELECT subscriber_id, subscription_id, days_alert FROM subscriptions",
	)
	if err != nil {
		repo.logger.Errorf("Error while SELECT from db: %v", err)
		return nil, fmt.Errorf("db error: %v", err)
	}

	for rows.Next() {
		var subscr Subscription
		err = rows.Scan(&subscr.Subscriber, &subscr.Subscription, &subscr.DaysAlert)
		if err != nil {
			repo.logger.Errorf("Error while scanning from sql row: %v", err)
			return nil, fmt.Errorf("db error: %v", err)
		}

		subscriptions = append(subscriptions, subscr)
	}

	err = rows.Close()
	if err != nil {
		repo.logger.Errorf("Error while closing sql rows: %v", err)
		return nil, fmt.Errorf("db error: %v", err)
	}

	return subscriptions, nil
}

func (repo *SubscriptionsMySQLRepo) GetSubscriptionsByUser(ctx context.Context, userID uint32) ([]Subscription, error) {
	subscriptions := make([]Subscription, 0, 10)

	rows, err := repo.db.QueryContext(
		ctx,
		"SELECT subscription_id, days_alert FROM subscriptions WHERE subscriber_id = ?",
		userID,
	)
	if err != nil {
		repo.logger.Errorf("Error while SELECT from db: %v", err)
		return nil, fmt.Errorf("db error: %v", err)
	}

	for rows.Next() {
		var subscr Subscription
		err = rows.Scan(&subscr.Subscription, &subscr.DaysAlert)
		if err != nil {
			repo.logger.Errorf("Error while scanning from sql row: %v", err)
			return nil, fmt.Errorf("db error: %v", err)
		}

		subscriptions = append(subscriptions, subscr)
	}

	err = rows.Close()
	if err != nil {
		repo.logger.Errorf("Error while closing sql rows: %v", err)
		return nil, fmt.Errorf("db error: %v", err)
	}

	return subscriptions, nil
}

func (repo *SubscriptionsMySQLRepo) AddSubscription(ctx context.Context, subscriberID, subscriptionID uint32, daysAlert int) error {
	result, err := repo.db.ExecContext(
		ctx,
		"INSERT INTO subscriptions (`subscriber_id`, `subscription_id`, `days_alert`) VALUES (?, ?, ?)",
		subscriberID,
		subscriptionID,
		daysAlert,
	)
	if err != nil {
		repo.logger.Errorf("Error while INSERT into db: %v", err)
		return fmt.Errorf("db error: %v", err)
	}

	// проверка, что запись добавлена
	affected, err := result.RowsAffected()
	if err != nil {
		repo.logger.Errorf("Error in RowsAffected(): %v", err)
		return fmt.Errorf("db error: %v", err)
	}
	if affected == 0 {
		repo.logger.Errorf("Subscription was not added")
		return ErrAddSubscription
	}

	return nil
}

func (repo *SubscriptionsMySQLRepo) RemoveSubscription(ctx context.Context, subscriberID, subscriptionID uint32) error {
	result, err := repo.db.ExecContext(
		ctx,
		"DELETE FROM subscriptions WHERE subscriber_id = ? AND subscription_id = ?",
		subscriberID,
		subscriptionID,
	)
	if err != nil {
		repo.logger.Errorf("Error while DELETE from db: %v", err)
		return fmt.Errorf("db error: %v", err)
	}

	// проверка, что запись удалена
	affected, err := result.RowsAffected()
	if err != nil {
		repo.logger.Errorf("Error in RowsAffected(): %v", err)
		return fmt.Errorf("db error: %v", err)
	}
	if affected == 0 {
		repo.logger.Warnf("Subscription was not removed")
		return ErrRemoveSubscription
	}

	return nil
}
