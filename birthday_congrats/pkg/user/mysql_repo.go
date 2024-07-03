package user

import (
	"context"
	"database/sql"
	"fmt"
	"sync"

	_ "github.com/go-sql-driver/mysql"
	"go.uber.org/zap"
)

type UsersMySQLRepo struct {
	mu     *sync.RWMutex
	db     *sql.DB
	logger *zap.SugaredLogger
}

var _ UsersRepo = &UsersMySQLRepo{}

func NewUsersMySQLRepo(db *sql.DB, logger *zap.SugaredLogger) *UsersMySQLRepo {
	return &UsersMySQLRepo{
		mu:     &sync.RWMutex{},
		db:     db,
		logger: logger,
	}
}

func (repo UsersMySQLRepo) Create(ctx context.Context, username, password, email string, birth int64) (*User, error) {
	// проверка, что пользователя с таким юзернэймом нет
	// сразу залочимся, чтобы никто не влез между запросами и не создал пользователя с таким же именем
	repo.mu.Lock()
	var id uint32
	err := repo.db.QueryRowContext(
		ctx,
		"SELECT id from users WHERE username = ?",
		username,
	).Scan(&id)
	if err != nil && err != sql.ErrNoRows {
		repo.mu.Unlock()
		repo.logger.Errorf("Error while SELECT from db: %v", err)
		return nil, fmt.Errorf("db error: %v", err)
	}
	if err == nil {
		repo.mu.Unlock()
		return nil, ErrUserExists
	}

	result, err := repo.db.ExecContext(
		ctx,
		"INSERT INTO users (`username`, `password`, `email`, `birthday`) VALUES (?, ?, ?, ?)",
		username, password, email, birth,
	)
	repo.mu.Unlock()

	if err != nil {
		repo.logger.Errorf("Error while INSERT into db: %v", err)
		return nil, fmt.Errorf("db error: %v", err)
	}

	// проверка, что запись добавлена
	affected, err := result.RowsAffected()
	if err != nil {
		repo.logger.Errorf("Error in RowsAffected(): %v", err)
		return nil, fmt.Errorf("db error: %v", err)
	}
	if affected == 0 {
		repo.logger.Errorf("User with username `%d` was not created", username)
		return nil, ErrUserNotCreated
	}

	lastID, err := result.LastInsertId()
	if err != nil {
		repo.logger.Errorf("Error in LastInsertedId(): %v", err)
		return nil, fmt.Errorf("db error: %v", err)
	}

	newUser := &User{
		ID:       uint32(lastID),
		Username: username,
		Email:    email,
		Birthday: birth,
	}

	return newUser, nil
}

func (repo *UsersMySQLRepo) Login(ctx context.Context, username, password string) (*User, error) {
	user := &User{}
	var passInDB string

	err := repo.db.QueryRowContext(
		ctx,
		"SELECT id, username, password, email, birthday FROM users WHERE username = ?",
		username,
	).Scan(
		&user.ID,
		&user.Username,
		&passInDB,
		&user.Email,
		&user.Birthday,
	)
	if err != nil && err != sql.ErrNoRows {
		repo.logger.Errorf("Error while SELECT from db: %v", err)
		return nil, fmt.Errorf("db error: %v", err)
	}
	if err == sql.ErrNoRows {
		return nil, ErrNoUser
	}

	if password != passInDB {
		return nil, ErrBadPassword
	}

	return user, nil
}

func (repo *UsersMySQLRepo) GetAll(ctx context.Context) ([]*User, error) {
	users := make([]*User, 0, 10)

	rows, err := repo.db.QueryContext(
		ctx,
		"SELECT id, username, birthday FROM users",
	)
	if err != nil {
		repo.logger.Errorf("Error while SELECT from db: %v", err)
		return nil, fmt.Errorf("db error: %v", err)
	}

	for rows.Next() {
		user := &User{}
		err = rows.Scan(
			&user.ID,
			&user.Username,
			&user.Birthday,
		)
		if err != nil {
			repo.logger.Errorf("Error while scanning from sql row: %v", err)
			return nil, fmt.Errorf("db error: %v", err)
		}

		users = append(users, user)
	}

	err = rows.Close()
	if err != nil {
		repo.logger.Errorf("Error while closing sql rows: %v", err)
		return nil, fmt.Errorf("db error: %v", err)
	}

	return users, nil
}

func (repo *UsersMySQLRepo) GetSubscribtions(ctx context.Context, userID uint32) ([]uint32, error) {
	subscriptions := make([]uint32, 0, 10)

	rows, err := repo.db.QueryContext(
		ctx,
		"SELECT subscription_id FROM subscriptions WHERE subscriber_id = ?",
		userID,
	)
	if err != nil {
		repo.logger.Errorf("Error while SELECT from db: %v", err)
		return nil, fmt.Errorf("db error: %v", err)
	}

	for rows.Next() {
		var id uint32
		err = rows.Scan(&id)
		if err != nil {
			repo.logger.Errorf("Error while scanning from sql row: %v", err)
			return nil, fmt.Errorf("db error: %v", err)
		}

		subscriptions = append(subscriptions, id)
	}

	err = rows.Close()
	if err != nil {
		repo.logger.Errorf("Error while closing sql rows: %v", err)
		return nil, fmt.Errorf("db error: %v", err)
	}

	return subscriptions, nil
}

func (repo *UsersMySQLRepo) AddSubscribtion(ctx context.Context, subscriberID, subscriptionID uint32) error {
	result, err := repo.db.ExecContext(
		ctx,
		"INSERT INTO subscriptions (`subscriber_id`, `subscription_id`) VALUES (?, ?)",
		subscriberID,
		subscriptionID,
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

func (repo *UsersMySQLRepo) RemoveSubscription(ctx context.Context, subscriberID, subscriptionID uint32) error {
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
		repo.logger.Errorf("Subscription was not removed")
		return ErrRemoveSubscription
	}

	return nil
}
