package service

import (
	alertmanager "birthday_congrats/pkg/alert_manager"
	"birthday_congrats/pkg/session"
	"birthday_congrats/pkg/user"
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/zap"
)

const (
	dateLayout = "2006-01-02"
)

var (
	ErrBadDateFormat = errors.New("bad date format")
)

type CongratulationsService struct {
	usersRepo user.UsersRepo
	alerts    alertmanager.AlertManager
	logger    *zap.SugaredLogger
}

func NewCongratulationsService(
	usersRepo user.UsersRepo,
	alerts alertmanager.AlertManager,
	logger *zap.SugaredLogger,
) *CongratulationsService {
	return &CongratulationsService{
		usersRepo: usersRepo,
		alerts:    alerts,
		logger:    logger,
	}
}

func (cs *CongratulationsService) GetAll(ctx context.Context) ([]*user.User, error) {
	users, err := cs.usersRepo.GetAll(ctx)
	if err != nil {
		cs.logger.Errorf("Error while getting all users: %v", err)
		return nil, fmt.Errorf("internal error")
	}

	return users, nil
}

func (cs *CongratulationsService) Register(ctx context.Context, username, password, email, birth string) (*user.User, error) {
	birthday, err := time.Parse(dateLayout, birth)
	if err != nil {
		cs.logger.Errorf("Error while parsing date: %v", err)
		return nil, ErrBadDateFormat
	}

	newUser, err := cs.usersRepo.Create(
		ctx,
		username,
		password,
		email,
		birthday.Year(),
		int(birthday.Month()),
		birthday.Day(),
	)
	if err != nil && err != user.ErrUserExists {
		cs.logger.Errorf("Error while creating user: %v", err)
		return nil, fmt.Errorf("internal error")
	}
	if err == user.ErrUserExists {
		cs.logger.Warnf("User already exists")
		return nil, err
	}

	return newUser, nil
}

func (cs *CongratulationsService) Login(ctx context.Context, username, password string) (*user.User, error) {
	us, err := cs.usersRepo.Login(ctx, username, password)
	if err != nil && err != user.ErrNoUser && err != user.ErrBadPassword {
		cs.logger.Errorf("Error while logging in: %v", err)
		return nil, fmt.Errorf("internal error")
	}
	if err == user.ErrNoUser || err == user.ErrBadPassword {
		cs.logger.Warnf("User not exist: %v", err)
		return nil, user.ErrNoUser // оставляем только один тип ошибки, чтобы мошеннику было сложнее подобрать пароль
	}

	return us, nil
}

func (cs *CongratulationsService) Subscribe(ctx context.Context, subscriptionID uint32) error {
	sess, err := session.SessionFromContext(ctx)
	if err != nil {
		cs.logger.Errorf("Error getting session from context: %v", err)
		return err
	}

	err = cs.usersRepo.AddSubscription(ctx, sess.UserID, subscriptionID)
	if err != nil {
		cs.logger.Errorf("Error adding subscription: %v", err)
		return fmt.Errorf("Internal error")
	}

	return nil
}

func (cs *CongratulationsService) Unsubscribe(ctx context.Context, subscriptionID uint32) error {
	sess, err := session.SessionFromContext(ctx)
	if err != nil {
		cs.logger.Errorf("Error getting session from context: %v", err)
		return err
	}

	err = cs.usersRepo.RemoveSubscription(ctx, sess.UserID, subscriptionID)
	if err != nil {
		cs.logger.Errorf("Error removing subscription: %v", err)
		return fmt.Errorf("Internal error")
	}

	return nil
}
