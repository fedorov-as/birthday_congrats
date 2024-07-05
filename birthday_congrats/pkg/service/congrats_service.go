package service

import (
	alertmanager "birthday_congrats/pkg/alert_manager"
	"birthday_congrats/pkg/session"
	"birthday_congrats/pkg/user"
	"context"
	"fmt"
	"slices"
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
	sm        session.SessionsManager
	alerts    alertmanager.AlertManager
	logger    *zap.SugaredLogger
}

func NewCongratulationsService(
	usersRepo user.UsersRepo,
	sm session.SessionsManager,
	alerts alertmanager.AlertManager,
	logger *zap.SugaredLogger,
) *CongratulationsService {
	return &CongratulationsService{
		usersRepo: usersRepo,
		sm:        sm,
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

func (cs *CongratulationsService) Register(ctx context.Context, username, password, email, birth string) (*session.Session, error) {
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

	sess, err := cs.sm.Create(ctx, newUser.ID)
	if err != nil {
		cs.logger.Errorf("Error while creating session")
		return nil, fmt.Errorf("internal error")
	}

	return sess, nil
}

func (cs *CongratulationsService) Login(ctx context.Context, username, password string) (*session.Session, error) {
	us, err := cs.usersRepo.Login(ctx, username, password)
	if err != nil && err != user.ErrNoUser && err != user.ErrBadPassword {
		cs.logger.Errorf("Error while logging in: %v", err)
		return nil, fmt.Errorf("internal error")
	}
	if err == user.ErrNoUser || err == user.ErrBadPassword {
		cs.logger.Warnf("User not exist: %v", err)
		return nil, user.ErrNoUser // оставляем только один тип ошибки, чтобы мошеннику было сложнее подобрать пароль
	}

	sess, err := cs.sm.Create(ctx, us.ID)
	if err != nil {
		cs.logger.Errorf("Error while creating session")
		return nil, fmt.Errorf("internal error")
	}

	return sess, nil
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
	if err != nil && err != user.ErrRemoveSubscription {
		cs.logger.Errorf("Error removing subscription: %v", err)
		return fmt.Errorf("Internal error")
	}
	if err == user.ErrRemoveSubscription {
		cs.logger.Warnf("Subscription was not removed")
	}

	return nil
}

func (cs *CongratulationsService) GetSubscriptions(ctx context.Context) ([]*user.User, error) {
	users, err := cs.usersRepo.GetAll(ctx)
	if err != nil {
		cs.logger.Errorf("Error while getting all users: %v", err)
		return nil, fmt.Errorf("internal error")
	}

	sess, err := session.SessionFromContext(ctx)
	if err != nil {
		cs.logger.Errorf("Error getting session from context: %v", err)
		return nil, session.ErrNoSession
	}

	subscriptions, err := cs.usersRepo.GetSubscriptions(ctx, sess.UserID)
	if err != nil {
		cs.logger.Errorf("Error getting subscriptions: %v", err)
		return nil, fmt.Errorf("internal error")
	}

	slices.Sort(subscriptions)

	i := 0
	for _, u := range users {
		if i >= len(subscriptions) {
			break
		}

		if u.ID == subscriptions[i] {
			u.Subscription = true
			i++
		}
	}

	return users, nil
}
