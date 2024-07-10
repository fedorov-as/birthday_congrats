package congrats_service

import (
	alertmanager "birthday_congrats/internal/pkg/alert_manager"
	"birthday_congrats/internal/pkg/session"
	"birthday_congrats/internal/pkg/subscription"
	"birthday_congrats/internal/pkg/user"
	"context"
	"fmt"
	"slices"
	"sync"
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

type CongratulationsServiceImpl struct {
	usersRepo         user.UsersRepo
	subscriptionsRepo subscription.SubscriptionsRepo
	sm                session.SessionsManager
	alerts            alertmanager.AlertManager
	logger            *zap.SugaredLogger
}

var _ CongratulationsService = &CongratulationsServiceImpl{}

func NewCongratulationsServiceImpl(
	usersRepo user.UsersRepo,
	subscriptionsRepo subscription.SubscriptionsRepo,
	sm session.SessionsManager,
	alerts alertmanager.AlertManager,
	logger *zap.SugaredLogger,
) *CongratulationsServiceImpl {
	return &CongratulationsServiceImpl{
		usersRepo:         usersRepo,
		subscriptionsRepo: subscriptionsRepo,
		sm:                sm,
		alerts:            alerts,
		logger:            logger,
	}
}

func (cs *CongratulationsServiceImpl) Register(ctx context.Context, username, password, email, birth string) (*session.Session, error) {
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

func (cs *CongratulationsServiceImpl) Login(ctx context.Context, username, password string) (*session.Session, error) {
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

func (cs *CongratulationsServiceImpl) Subscribe(ctx context.Context, subscriptionID uint32, daysAlert int) error {
	sess, err := session.SessionFromContext(ctx)
	if err != nil {
		cs.logger.Errorf("Error getting session from context: %v", err)
		return err
	}

	err = cs.subscriptionsRepo.AddSubscription(ctx, sess.UserID, subscriptionID, daysAlert)
	if err != nil && err != subscription.ErrAddSubscription {
		cs.logger.Errorf("Error adding subscription: %v", err)
		return fmt.Errorf("Internal error")
	}
	if err == subscription.ErrAddSubscription {
		cs.logger.Warnf("Subscription was not added")
		return err
	}

	return nil
}

func (cs *CongratulationsServiceImpl) Unsubscribe(ctx context.Context, subscriptionID uint32) error {
	sess, err := session.SessionFromContext(ctx)
	if err != nil {
		cs.logger.Errorf("Error getting session from context: %v", err)
		return err
	}

	err = cs.subscriptionsRepo.RemoveSubscription(ctx, sess.UserID, subscriptionID)
	if err != nil && err != subscription.ErrRemoveSubscription {
		cs.logger.Errorf("Error removing subscription: %v", err)
		return fmt.Errorf("Internal error")
	}
	if err == subscription.ErrRemoveSubscription {
		cs.logger.Warnf("Subscription was not removed")
		return err
	}

	return nil
}

func (cs *CongratulationsServiceImpl) Logout(ctx context.Context) error {
	err := cs.sm.Destroy(ctx)
	if err != nil && err != session.ErrNotDestroyed && err != session.ErrNoSession {
		cs.logger.Errorf("Error while destroying session")
		return fmt.Errorf("internal error")
	}
	if err == session.ErrNoSession || err == session.ErrNotDestroyed {
		cs.logger.Warnf("Session was not destroyed")
		return session.ErrNotDestroyed
	}

	return nil
}

func (cs *CongratulationsServiceImpl) GetSubscriptionsByUser(ctx context.Context) ([]*user.User, error) {
	sess, err := session.SessionFromContext(ctx)
	if err != nil {
		cs.logger.Errorf("Error getting session from context: %v", err)
		return nil, session.ErrNoSession
	}

	users, err := cs.usersRepo.GetAll(ctx)
	if err != nil {
		cs.logger.Errorf("Error while getting all users: %v", err)
		return nil, fmt.Errorf("internal error")
	}

	subscriptions, err := cs.subscriptionsRepo.GetSubscriptionsByUser(ctx, sess.UserID)
	if err != nil {
		cs.logger.Errorf("Error getting subscriptions: %v", err)
		return nil, fmt.Errorf("internal error")
	}

	slices.SortFunc(subscriptions, func(a, b *subscription.Subscription) int { return int(a.Subscription) - int(b.Subscription) })

	i := 0
	for _, u := range users {
		if i >= len(subscriptions) {
			break
		}

		if u.ID == subscriptions[i].Subscription {
			u.Subscription = true
			u.DaysAlert = subscriptions[i].DaysAlert
			i++
		}
	}

	return users, nil
}

func (cs *CongratulationsServiceImpl) StartAlert(ctx context.Context, timeStart time.Time, period time.Duration, wg *sync.WaitGroup) {
	if timeStart.After(time.Now()) {
		cs.logger.Infof("Alert service will start at %v", timeStart)
		time.Sleep(time.Until(timeStart))
	}

	cs.logger.Infof("Starting alert service")
	go cs.alert(ctx, period, wg)
}

func (cs *CongratulationsServiceImpl) alert(ctx context.Context, period time.Duration, wg *sync.WaitGroup) {
	defer wg.Done()

	cs.logger.Infof("Alert service started")

	ticker := time.NewTicker(period)
	defer ticker.Stop()

	// первый раз делаем отправку сразу, затем по тикеру
	messages, recipients, err := cs.makeMessages(ctx)
	if err != nil {
		cs.logger.Errorf("Error while making messages: %v", err)
		return
	}

	cs.logger.Infof("Sending %d different messages today", len(messages))

	for i := range messages {
		cs.alerts.Send(recipients[i], "Напоминание о дне рождения!", messages[i])
	}

	for {
		select {
		case <-ctx.Done():
			cs.logger.Infof("Alert service was stopped")
			return
		case <-ticker.C:
			messages, recipients, err := cs.makeMessages(ctx)
			if err != nil {
				cs.logger.Errorf("Error while making messages: %v", err)
				continue
			}

			cs.logger.Infof("Sending %d different messages today", len(messages))

			for i := range messages {
				cs.alerts.Send(recipients[i], "Напоминание о дне рождения!", messages[i])
			}
		}
	}
}

func (cs *CongratulationsServiceImpl) makeMessages(ctx context.Context) ([]string, [][]string, error) {
	subscriptions, err := cs.subscriptionsRepo.GetAllSubscriptions(ctx)
	if err != nil {
		cs.logger.Errorf("Error getting all subscriptions: %v", err)
		return nil, nil, fmt.Errorf("error getting subscriptions from repo")
	}

	if len(subscriptions) == 0 {
		return nil, nil, nil
	}

	slices.SortStableFunc(subscriptions, func(a, b *subscription.Subscription) int { return int(a.Subscription) - int(b.Subscription) })

	messages := make([]string, 0)
	recipients := make([][]string, 0)

	var subID uint32
	var daysBefore int
	to := make([]string, 0)

	for i, sub := range subscriptions {
		if sub.Subscription != subID || i == 0 {
			subID = sub.Subscription
			us, err := cs.usersRepo.GetByID(ctx, subID)
			if err != nil {
				cs.logger.Errorf("Error getting user by id: %v", err)
				return nil, nil, fmt.Errorf("repo error: %v", err)
			}

			birthday := time.Date(
				time.Now().Year(),
				time.Month(us.Month),
				us.Day,
				0, 0, 0, 0,
				time.UTC,
			)
			if birthday.Before(time.Now()) {
				birthday = birthday.AddDate(1, 0, 0)
			}

			if len(to) > 0 {
				recipients = append(recipients, to)
			} else if len(messages) > 0 {
				messages = messages[:len(messages)-1]
			}

			daysBefore = int(time.Until(birthday).Hours())/24 + 1
			messages = append(messages, fmt.Sprintf("%s празднует свой день рождения через %d дней!", us.Username, daysBefore))

			to = make([]string, 0)
		}

		if daysBefore == sub.DaysAlert {
			subscriber, err := cs.usersRepo.GetByID(ctx, sub.Subscriber)
			if err != nil {
				cs.logger.Errorf("Error getting user by id: %v", err)
				return nil, nil, fmt.Errorf("repo error: %v", err)
			}

			to = append(to, subscriber.Email)
		}
	}

	if len(to) > 0 {
		recipients = append(recipients, to)
	} else if len(messages) > 0 {
		messages = messages[:len(messages)-1]
	}

	return messages, recipients, nil
}
