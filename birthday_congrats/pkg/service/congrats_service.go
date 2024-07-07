package service

import (
	alertmanager "birthday_congrats/pkg/alert_manager"
	"birthday_congrats/pkg/session"
	"birthday_congrats/pkg/user"
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

// func (cs *CongratulationsService) GetAll(ctx context.Context) ([]*user.User, error) {
// 	users, err := cs.usersRepo.GetAll(ctx)
// 	if err != nil {
// 		cs.logger.Errorf("Error while getting all users: %v", err)
// 		return nil, fmt.Errorf("internal error")
// 	}

// 	return users, nil
// }

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

func (cs *CongratulationsService) Subscribe(ctx context.Context, subscriptionID uint32, daysAlert int) error {
	sess, err := session.SessionFromContext(ctx)
	if err != nil {
		cs.logger.Errorf("Error getting session from context: %v", err)
		return err
	}

	err = cs.usersRepo.AddSubscription(ctx, sess.UserID, subscriptionID, daysAlert)
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

	subscriptions, err := cs.usersRepo.GetSubscriptionsByUser(ctx, sess.UserID)
	if err != nil {
		cs.logger.Errorf("Error getting subscriptions: %v", err)
		return nil, fmt.Errorf("internal error")
	}

	slices.SortFunc(subscriptions, func(a, b user.Subscription) int { return int(a.Subscription) - int(b.Subscription) })

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

func (cs *CongratulationsService) Logout(ctx context.Context) error {
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

func (cs *CongratulationsService) StartAlert(ctx context.Context, timeStart time.Time, wg *sync.WaitGroup) {
	if timeStart.Before(time.Now()) {
		time.Sleep(time.Until(timeStart))
	}

	go cs.alert(ctx, wg)
}

func (cs *CongratulationsService) alert(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	wg1 := &sync.WaitGroup{}
	ticker := time.NewTicker(time.Hour * 24)
	for {
		select {
		case <-ctx.Done():
			wg1.Wait()
			return
		case <-ticker.C:
			messages, recipients, err := cs.makeMessages(ctx)
			if err != nil {
				cs.logger.Errorf("Error while making messages: %v", err)
			}

			for i := range messages {
				wg1.Add(1)
				go cs.alerts.Send(recipients[i], messages[i], wg1)
			}
		}
	}
}

func (cs *CongratulationsService) makeMessages(ctx context.Context) ([]string, [][]string, error) {
	subscriptions, err := cs.usersRepo.GetAllSubscriptions(ctx)
	if err != nil {
		cs.logger.Errorf("Error getting all subscriptions: %v", err)
	}

	if len(subscriptions) == 0 {
		return nil, nil, nil
	}

	slices.SortFunc(subscriptions, func(a, b user.Subscription) int { return int(a.Subscription) - int(b.Subscription) })

	subID := subscriptions[0].Subscription
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

	daysBefore := int(time.Until(birthday).Hours())/24 + 1

	messages := make([]string, 0)
	messages = append(messages, us.Username+" празднует свой день рождения через "+string(daysBefore)+" дней")

	allRecipients := make([][]string, 0)
	to := make([]string, 0)

	for _, sub := range subscriptions {
		if sub.Subscription != subID {
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

			daysBefore := int(time.Until(birthday).Hours())/24 + 1
			messages = append(messages, us.Username+" празднует свой день рождения через "+string(daysBefore)+" дней")

			allRecipients = append(allRecipients, to)
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

	allRecipients = append(allRecipients, to)

	return messages, allRecipients, nil
}
