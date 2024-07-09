package subscription

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	sqlmock "gopkg.in/DATA-DOG/go-sqlmock.v1"
)

type customErrorResult struct {
	errLastID   error
	errAffected error
}

var _ sql.Result = &customErrorResult{}

func (res *customErrorResult) LastInsertId() (int64, error) {
	return int64(0), res.errLastID
}

func (res *customErrorResult) RowsAffected() (int64, error) {
	return int64(1), res.errAffected
}

func TestGetAllSubscriptions(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("cant create mock: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	testRepo := NewSubscriptionsMySQLRepo(db, zap.NewNop().Sugar())

	// данные для теста
	subsExpected := []*Subscription{
		{
			Subscriber:   uint32(0),
			Subscription: uint32(1),
			DaysAlert:    42,
		},
		{
			Subscriber:   uint32(0),
			Subscription: uint32(2),
			DaysAlert:    10,
		},
		{
			Subscriber:   uint32(2),
			Subscription: uint32(1),
			DaysAlert:    4,
		},
	}

	// нормальная работа
	rows := sqlmock.NewRows([]string{"subscriber_id", "subscription_id", "days_alert"})
	for _, s := range subsExpected {
		rows = rows.AddRow(
			s.Subscriber,
			s.Subscription,
			s.DaysAlert,
		)
	}

	mock.
		ExpectQuery("SELECT subscriber_id, subscription_id, days_alert FROM subscriptions").
		WillReturnRows(rows)

	subsRecv, err := testRepo.GetAllSubscriptions(ctx)

	assert.NoError(t, err)
	assert.EqualValues(t, subsExpected, subsRecv)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)

	// ответ с ошибкой
	mock.
		ExpectQuery("SELECT subscriber_id, subscription_id, days_alert FROM subscriptions").
		WillReturnError(fmt.Errorf("db error"))

	_, err = testRepo.GetAllSubscriptions(ctx)

	assert.Error(t, err)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)

	// ошибка scan
	rows = sqlmock.NewRows([]string{""})
	rows = rows.AddRow("")

	mock.
		ExpectQuery("SELECT subscriber_id, subscription_id, days_alert FROM subscriptions").
		WillReturnRows(rows)

	_, err = testRepo.GetAllSubscriptions(ctx)

	assert.Error(t, err)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestGetSubscriptionsByUser(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("cant create mock: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	testRepo := NewSubscriptionsMySQLRepo(db, zap.NewNop().Sugar())

	// данные для теста
	subscriberID := uint32(42)
	subsExpected := []*Subscription{
		{
			Subscriber:   subscriberID,
			Subscription: uint32(1),
			DaysAlert:    42,
		},
		{
			Subscriber:   subscriberID,
			Subscription: uint32(2),
			DaysAlert:    10,
		},
	}

	// нормальная работа
	rows := sqlmock.NewRows([]string{"subscription_id", "days_alert"})
	for _, s := range subsExpected {
		rows = rows.AddRow(
			s.Subscription,
			s.DaysAlert,
		)
	}

	mock.
		ExpectQuery("SELECT subscription_id, days_alert FROM subscriptions WHERE").
		WithArgs(subscriberID).
		WillReturnRows(rows)

	subsRecv, err := testRepo.GetSubscriptionsByUser(ctx, subscriberID)

	assert.NoError(t, err)
	assert.EqualValues(t, subsExpected, subsRecv)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)

	// ответ с ошибкой
	mock.
		ExpectQuery("SELECT subscription_id, days_alert FROM subscriptions WHERE").
		WithArgs(subscriberID).
		WillReturnError(fmt.Errorf("db error"))

	_, err = testRepo.GetSubscriptionsByUser(ctx, subscriberID)

	assert.Error(t, err)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)

	// ошибка scan
	rows = sqlmock.NewRows([]string{""})
	rows = rows.AddRow("")

	mock.
		ExpectQuery("SELECT subscription_id, days_alert FROM subscriptions WHERE").
		WithArgs(subscriberID).
		WillReturnRows(rows)

	_, err = testRepo.GetSubscriptionsByUser(ctx, subscriberID)

	assert.Error(t, err)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestAddSubscription(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("cant create mock: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	testRepo := NewSubscriptionsMySQLRepo(db, zap.NewNop().Sugar())

	// данные для теста
	subExpeceted := &Subscription{
		Subscriber:   uint32(10),
		Subscription: uint32(42),
		DaysAlert:    5,
	}

	// нормальная работа
	mock.
		ExpectExec("INSERT INTO subscriptions").
		WithArgs(subExpeceted.Subscriber, subExpeceted.Subscription, subExpeceted.DaysAlert).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = testRepo.AddSubscription(ctx, subExpeceted.Subscriber, subExpeceted.Subscription, subExpeceted.DaysAlert)

	assert.NoError(t, err)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)

	// ответ с ошибкой
	mock.
		ExpectExec("INSERT INTO subscriptions").
		WithArgs(subExpeceted.Subscriber, subExpeceted.Subscription, subExpeceted.DaysAlert).
		WillReturnError(fmt.Errorf("db error"))

	err = testRepo.AddSubscription(ctx, subExpeceted.Subscriber, subExpeceted.Subscription, subExpeceted.DaysAlert)

	assert.Error(t, err)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)

	// rows affected = 0
	mock.
		ExpectExec("INSERT INTO subscriptions").
		WithArgs(subExpeceted.Subscriber, subExpeceted.Subscription, subExpeceted.DaysAlert).
		WillReturnResult(sqlmock.NewResult(int64(0), 0))

	err = testRepo.AddSubscription(ctx, subExpeceted.Subscriber, subExpeceted.Subscription, subExpeceted.DaysAlert)

	assert.ErrorIs(t, err, ErrAddSubscription)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)

	// ошибка rowsAffected()
	mock.
		ExpectExec("INSERT INTO subscriptions").
		WithArgs(subExpeceted.Subscriber, subExpeceted.Subscription, subExpeceted.DaysAlert).
		WillReturnResult(&customErrorResult{errAffected: fmt.Errorf("affected error")})

	err = testRepo.AddSubscription(ctx, subExpeceted.Subscriber, subExpeceted.Subscription, subExpeceted.DaysAlert)

	assert.Error(t, err)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestRemoveSubscription(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("cant create mock: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	testRepo := NewSubscriptionsMySQLRepo(db, zap.NewNop().Sugar())

	// данные для теста
	subExpeceted := &Subscription{
		Subscriber:   uint32(10),
		Subscription: uint32(42),
	}

	// нормальная работа
	mock.
		ExpectExec("DELETE FROM subscriptions WHERE").
		WithArgs(subExpeceted.Subscriber, subExpeceted.Subscription).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = testRepo.RemoveSubscription(ctx, subExpeceted.Subscriber, subExpeceted.Subscription)

	assert.NoError(t, err)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)

	// ответ с ошибкой
	mock.
		ExpectExec("DELETE FROM subscriptions WHERE").
		WithArgs(subExpeceted.Subscriber, subExpeceted.Subscription).
		WillReturnError(fmt.Errorf("db error"))

	err = testRepo.RemoveSubscription(ctx, subExpeceted.Subscriber, subExpeceted.Subscription)

	assert.Error(t, err)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)

	// rows affected = 0
	mock.
		ExpectExec("DELETE FROM subscriptions WHERE").
		WithArgs(subExpeceted.Subscriber, subExpeceted.Subscription).
		WillReturnResult(sqlmock.NewResult(int64(0), 0))

	err = testRepo.RemoveSubscription(ctx, subExpeceted.Subscriber, subExpeceted.Subscription)

	assert.ErrorIs(t, err, ErrRemoveSubscription)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)

	// ошибка rowsAffected()
	mock.
		ExpectExec("DELETE FROM subscriptions WHERE").
		WithArgs(subExpeceted.Subscriber, subExpeceted.Subscription).
		WillReturnResult(&customErrorResult{errAffected: fmt.Errorf("affected error")})

	err = testRepo.RemoveSubscription(ctx, subExpeceted.Subscriber, subExpeceted.Subscription)

	assert.Error(t, err)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}
