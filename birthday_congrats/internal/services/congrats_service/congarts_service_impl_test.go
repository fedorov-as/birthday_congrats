package congrats_service

import (
	alertmanager "birthday_congrats/internal/pkg/alert_manager"
	"birthday_congrats/internal/pkg/session"
	"birthday_congrats/internal/pkg/subscription"
	"birthday_congrats/internal/pkg/user"
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestRegister(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	usersRepo := user.NewMockUsersRepo(ctrl)
	subscriptionsRepo := subscription.NewMockSubscriptionsRepo(ctrl)
	sessManager := session.NewMockSessionsManager(ctrl)
	alertManager := alertmanager.NewMockAlertManager(ctrl)

	testService := NewCongratulationsServiceImpl(
		usersRepo,
		subscriptionsRepo,
		sessManager,
		alertManager,
		zap.NewNop().Sugar(),
	)

	// данные для теста
	password := "some_pass"
	userExpected := &user.User{
		ID:       0,
		Username: "some_user",
		Password: "", // Пароль не возвращается
		Email:    "some@email.net",
		Year:     2000,
		Month:    1,
		Day:      2,
	}

	sessExpected := &session.Session{
		SessID:  "some_sess_id",
		UserID:  userExpected.ID,
		Expires: time.Now().Unix() + 60,
	}

	// нормальная работа
	birth := fmt.Sprintf("%04d-%02d-%02d", userExpected.Year, userExpected.Month, userExpected.Day)

	usersRepo.EXPECT().Create(
		context.Background(),
		userExpected.Username,
		password,
		userExpected.Email,
		userExpected.Year,
		userExpected.Month,
		userExpected.Day,
	).Return(userExpected, nil)

	sessManager.EXPECT().Create(
		context.Background(),
		userExpected.ID,
	).Return(sessExpected, nil)

	sessRecv, err := testService.Register(
		context.Background(),
		userExpected.Username,
		password,
		userExpected.Email,
		birth,
	)

	assert.NoError(t, err)
	assert.EqualValues(t, sessExpected, sessRecv)

	// ошибка в формате даты
	birth = fmt.Sprintf("%04d-%02d%02d", userExpected.Year, userExpected.Month, userExpected.Day)

	_, err = testService.Register(
		context.Background(),
		userExpected.Username,
		password,
		userExpected.Email,
		birth,
	)

	assert.ErrorIs(t, err, ErrBadDateFormat)

	// ошибка хранилища
	birth = fmt.Sprintf("%04d-%02d-%02d", userExpected.Year, userExpected.Month, userExpected.Day)

	usersRepo.EXPECT().Create(
		context.Background(),
		userExpected.Username,
		password,
		userExpected.Email,
		userExpected.Year,
		userExpected.Month,
		userExpected.Day,
	).Return(nil, fmt.Errorf("repo error"))

	_, err = testService.Register(
		context.Background(),
		userExpected.Username,
		password,
		userExpected.Email,
		birth,
	)

	assert.Error(t, err)

	// пользователь уже существует
	birth = fmt.Sprintf("%04d-%02d-%02d", userExpected.Year, userExpected.Month, userExpected.Day)

	usersRepo.EXPECT().Create(
		context.Background(),
		userExpected.Username,
		password,
		userExpected.Email,
		userExpected.Year,
		userExpected.Month,
		userExpected.Day,
	).Return(nil, user.ErrUserExists)

	_, err = testService.Register(
		context.Background(),
		userExpected.Username,
		password,
		userExpected.Email,
		birth,
	)

	assert.ErrorIs(t, err, user.ErrUserExists)

	// ошибка менеджера сессий
	birth = fmt.Sprintf("%04d-%02d-%02d", userExpected.Year, userExpected.Month, userExpected.Day)

	usersRepo.EXPECT().Create(
		context.Background(),
		userExpected.Username,
		password,
		userExpected.Email,
		userExpected.Year,
		userExpected.Month,
		userExpected.Day,
	).Return(userExpected, nil)

	sessManager.EXPECT().Create(
		context.Background(),
		userExpected.ID,
	).Return(nil, fmt.Errorf("sessions error"))

	_, err = testService.Register(
		context.Background(),
		userExpected.Username,
		password,
		userExpected.Email,
		birth,
	)

	assert.Error(t, err)
}

func TestLogin(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	usersRepo := user.NewMockUsersRepo(ctrl)
	subscriptionsRepo := subscription.NewMockSubscriptionsRepo(ctrl)
	sessManager := session.NewMockSessionsManager(ctrl)
	alertManager := alertmanager.NewMockAlertManager(ctrl)

	testService := NewCongratulationsServiceImpl(
		usersRepo,
		subscriptionsRepo,
		sessManager,
		alertManager,
		zap.NewNop().Sugar(),
	)

	// данные для теста
	password := "some_pass"
	userExpected := &user.User{
		ID:       0,
		Username: "some_user",
		Password: "", // Пароль не возвращается
		Email:    "some@email.net",
		Year:     2000,
		Month:    1,
		Day:      2,
	}

	sessExpected := &session.Session{
		SessID:  "some_sess_id",
		UserID:  userExpected.ID,
		Expires: time.Now().Unix() + 60,
	}

	// нормальные данные
	usersRepo.EXPECT().Login(
		context.Background(),
		userExpected.Username,
		password,
	).Return(userExpected, nil)

	sessManager.EXPECT().Create(
		context.Background(),
		userExpected.ID,
	).Return(sessExpected, nil)

	sessRecv, err := testService.Login(
		context.Background(),
		userExpected.Username,
		password,
	)

	assert.NoError(t, err)
	assert.EqualValues(t, sessExpected, sessRecv)

	// ошибка хранилища
	usersRepo.EXPECT().Login(
		context.Background(),
		userExpected.Username,
		password,
	).Return(nil, fmt.Errorf("repo error"))

	_, err = testService.Login(
		context.Background(),
		userExpected.Username,
		password,
	)

	assert.Error(t, err)

	// пользователь не найден или неверный пароль
	usersRepo.EXPECT().Login(
		context.Background(),
		userExpected.Username,
		password,
	).Return(nil, user.ErrBadPassword)

	_, err = testService.Login(
		context.Background(),
		userExpected.Username,
		password,
	)

	assert.ErrorIs(t, err, user.ErrNoUser)

	// ошибка менеджера сессий
	usersRepo.EXPECT().Login(
		context.Background(),
		userExpected.Username,
		password,
	).Return(userExpected, nil)

	sessManager.EXPECT().Create(
		context.Background(),
		userExpected.ID,
	).Return(nil, fmt.Errorf("sessions error"))

	_, err = testService.Login(
		context.Background(),
		userExpected.Username,
		password,
	)

	assert.Error(t, err)
}

func TestSubscribe(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	usersRepo := user.NewMockUsersRepo(ctrl)
	subscriptionsRepo := subscription.NewMockSubscriptionsRepo(ctrl)
	sessManager := session.NewMockSessionsManager(ctrl)
	alertManager := alertmanager.NewMockAlertManager(ctrl)

	testService := NewCongratulationsServiceImpl(
		usersRepo,
		subscriptionsRepo,
		sessManager,
		alertManager,
		zap.NewNop().Sugar(),
	)

	// данные для теста
	subscriberID := uint32(10)
	subscriptionID := uint32(42)
	daysAlert := 5

	sessExpected := &session.Session{
		SessID:  "some_sess_id",
		UserID:  subscriberID,
		Expires: time.Now().Unix() + 60,
	}

	// нормальная работа
	ctx := session.ContextWithSession(context.Background(), sessExpected)

	subscriptionsRepo.EXPECT().AddSubscription(
		ctx,
		subscriberID,
		subscriptionID,
		daysAlert,
	).Return(nil)

	err := testService.Subscribe(
		ctx,
		subscriptionID,
		daysAlert,
	)

	assert.NoError(t, err)

	// нет сессии
	ctx = context.Background()

	err = testService.Subscribe(
		ctx,
		subscriptionID,
		daysAlert,
	)

	assert.ErrorIs(t, err, session.ErrNoSession)

	// ошибка хранилища
	ctx = session.ContextWithSession(context.Background(), sessExpected)

	subscriptionsRepo.EXPECT().AddSubscription(
		ctx,
		subscriberID,
		subscriptionID,
		daysAlert,
	).Return(fmt.Errorf("repo error"))

	err = testService.Subscribe(
		ctx,
		subscriptionID,
		daysAlert,
	)

	assert.Error(t, err)

	// подписка не добавлена
	ctx = session.ContextWithSession(context.Background(), sessExpected)

	subscriptionsRepo.EXPECT().AddSubscription(
		ctx,
		subscriberID,
		subscriptionID,
		daysAlert,
	).Return(subscription.ErrAddSubscription)

	err = testService.Subscribe(
		ctx,
		subscriptionID,
		daysAlert,
	)

	assert.ErrorIs(t, err, subscription.ErrAddSubscription)
}

func TestUnsubscribe(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	usersRepo := user.NewMockUsersRepo(ctrl)
	subscriptionsRepo := subscription.NewMockSubscriptionsRepo(ctrl)
	sessManager := session.NewMockSessionsManager(ctrl)
	alertManager := alertmanager.NewMockAlertManager(ctrl)

	testService := NewCongratulationsServiceImpl(
		usersRepo,
		subscriptionsRepo,
		sessManager,
		alertManager,
		zap.NewNop().Sugar(),
	)

	// данные для теста
	subscriberID := uint32(10)
	subscriptionID := uint32(42)

	sessExpected := &session.Session{
		SessID:  "some_sess_id",
		UserID:  subscriberID,
		Expires: time.Now().Unix() + 60,
	}

	// нормальная работа
	ctx := session.ContextWithSession(context.Background(), sessExpected)

	subscriptionsRepo.EXPECT().RemoveSubscription(
		ctx,
		subscriberID,
		subscriptionID,
	).Return(nil)

	err := testService.Unsubscribe(
		ctx,
		subscriptionID,
	)

	assert.NoError(t, err)

	// нет сессии
	ctx = context.Background()

	err = testService.Unsubscribe(
		ctx,
		subscriptionID,
	)

	assert.ErrorIs(t, err, session.ErrNoSession)

	// ошибка хранилища
	ctx = session.ContextWithSession(context.Background(), sessExpected)

	subscriptionsRepo.EXPECT().RemoveSubscription(
		ctx,
		subscriberID,
		subscriptionID,
	).Return(fmt.Errorf("repo error"))

	err = testService.Unsubscribe(
		ctx,
		subscriptionID,
	)

	assert.Error(t, err)

	// подписка не добавлена
	ctx = session.ContextWithSession(context.Background(), sessExpected)

	subscriptionsRepo.EXPECT().RemoveSubscription(
		ctx,
		subscriberID,
		subscriptionID,
	).Return(subscription.ErrRemoveSubscription)

	err = testService.Unsubscribe(
		ctx,
		subscriptionID,
	)

	assert.ErrorIs(t, err, subscription.ErrRemoveSubscription)
}

func TestLogout(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	usersRepo := user.NewMockUsersRepo(ctrl)
	subscriptionsRepo := subscription.NewMockSubscriptionsRepo(ctrl)
	sessManager := session.NewMockSessionsManager(ctrl)
	alertManager := alertmanager.NewMockAlertManager(ctrl)

	testService := NewCongratulationsServiceImpl(
		usersRepo,
		subscriptionsRepo,
		sessManager,
		alertManager,
		zap.NewNop().Sugar(),
	)

	// данные для теста
	userID := uint32(42)

	sessExpected := &session.Session{
		SessID:  "some_sess_id",
		UserID:  userID,
		Expires: time.Now().Unix() + 60,
	}

	// нормальная работа
	ctx := session.ContextWithSession(context.Background(), sessExpected)

	sessManager.EXPECT().Destroy(ctx).Return(nil)

	err := testService.Logout(ctx)

	assert.NoError(t, err)

	// ошибка менеджера сессий
	ctx = session.ContextWithSession(context.Background(), sessExpected)

	sessManager.EXPECT().Destroy(ctx).Return(fmt.Errorf("sessions error"))

	err = testService.Logout(ctx)

	assert.Error(t, err)

	// нет сессии или она не уничтожилась
	ctx = session.ContextWithSession(context.Background(), sessExpected)

	sessManager.EXPECT().Destroy(ctx).Return(session.ErrNoSession)

	err = testService.Logout(ctx)

	assert.ErrorIs(t, err, session.ErrNotDestroyed)
}

func TestGetSubscriptionsByUser(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	usersRepo := user.NewMockUsersRepo(ctrl)
	subscriptionsRepo := subscription.NewMockSubscriptionsRepo(ctrl)
	sessManager := session.NewMockSessionsManager(ctrl)
	alertManager := alertmanager.NewMockAlertManager(ctrl)

	testService := NewCongratulationsServiceImpl(
		usersRepo,
		subscriptionsRepo,
		sessManager,
		alertManager,
		zap.NewNop().Sugar(),
	)

	// данные для теста
	userID := uint32(42)

	sess := &session.Session{
		SessID:  "some_sess_id",
		UserID:  userID,
		Expires: time.Now().Unix() + 60,
	}

	subsSent := []*subscription.Subscription{
		{
			Subscriber:   userID,
			Subscription: 10,
			DaysAlert:    5,
		},
		{
			Subscriber:   userID,
			Subscription: 4,
			DaysAlert:    1,
		},
	}

	users := []user.User{
		{ID: 4},
		{ID: 5},
		{ID: 10},
		{ID: 16},
		{ID: userID},
	}

	usersRet := []user.User{
		{
			ID:           4,
			Subscription: true,
			DaysAlert:    1,
		},
		{ID: 5},
		{
			ID:           10,
			Subscription: true,
			DaysAlert:    5,
		},
		{ID: 16},
		{ID: userID},
	}

	usersSent := make([]*user.User, 0)
	for i := range users {
		usersSent = append(usersSent, &users[i])
	}

	usersExpected := make([]*user.User, 0)
	for i := range usersRet {
		usersExpected = append(usersExpected, &usersRet[i])
	}

	// нормальная работа
	ctx := session.ContextWithSession(context.Background(), sess)

	usersRepo.EXPECT().GetAll(ctx).Return(usersSent, nil)
	subscriptionsRepo.EXPECT().GetSubscriptionsByUser(ctx, userID).Return(subsSent, nil)

	usersRecv, err := testService.GetSubscriptionsByUser(ctx)

	assert.NoError(t, err)
	assert.EqualValues(t, usersExpected, usersRecv)

	// ошибка хранилища пользователей
	ctx = session.ContextWithSession(context.Background(), sess)

	usersRepo.EXPECT().GetAll(ctx).Return(nil, fmt.Errorf("repo error"))

	_, err = testService.GetSubscriptionsByUser(ctx)

	assert.Error(t, err)

	// нет сессии
	ctx = context.Background()

	_, err = testService.GetSubscriptionsByUser(ctx)

	assert.ErrorIs(t, err, session.ErrNoSession)

	// ошибка хранилища подписок
	ctx = session.ContextWithSession(context.Background(), sess)

	usersRepo.EXPECT().GetAll(ctx).Return(usersSent, nil)
	subscriptionsRepo.EXPECT().GetSubscriptionsByUser(ctx, userID).Return(nil, fmt.Errorf("repo error"))

	_, err = testService.GetSubscriptionsByUser(ctx)

	assert.Error(t, err)
}

func TestMakeMessages(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	usersRepo := user.NewMockUsersRepo(ctrl)
	subscriptionsRepo := subscription.NewMockSubscriptionsRepo(ctrl)
	sessManager := session.NewMockSessionsManager(ctrl)
	alertManager := alertmanager.NewMockAlertManager(ctrl)

	testService := NewCongratulationsServiceImpl(
		usersRepo,
		subscriptionsRepo,
		sessManager,
		alertManager,
		zap.NewNop().Sugar(),
	)

	// данные для теста
	subsSent := []*subscription.Subscription{
		{
			Subscriber:   0,
			Subscription: 1,
			DaysAlert:    2,
		},
		{
			Subscriber:   2,
			Subscription: 1,
			DaysAlert:    1,
		},
		{
			Subscriber:   3,
			Subscription: 1,
			DaysAlert:    1,
		},
		{
			Subscriber:   2,
			Subscription: 0,
			DaysAlert:    5,
		},
		{
			Subscriber:   3,
			Subscription: 2,
			DaysAlert:    10,
		},
		{
			Subscriber:   2,
			Subscription: 3,
			DaysAlert:    1,
		},
	}

	usersSent := []*user.User{
		{
			ID:       0,
			Username: "zero",
			Email:    "zero@zero.net",
			Month:    int(time.Now().AddDate(0, 0, 5).Month()),
			Day:      time.Now().AddDate(0, 0, 5).Day(),
		},
		{
			ID:       1,
			Username: "one",
			Email:    "one@one.net",
			Month:    int(time.Now().AddDate(0, 0, 1).Month()),
			Day:      time.Now().AddDate(0, 0, 1).Day(),
		},
		{
			ID:       2,
			Username: "two",
			Email:    "two@two.net",
			Month:    int(time.Now().AddDate(0, 0, 9).Month()),
			Day:      time.Now().AddDate(0, 0, 9).Day(),
		},
		{
			ID:       3,
			Username: "three",
			Email:    "three@three.net",
			Month:    int(time.Now().Month()),
			Day:      time.Now().Day(),
		},
	}

	messagesExpected := []string{
		"zero празднует свой день рождения через 5 дней!",
		"one празднует свой день рождения через 1 дней!",
	}

	recipientsExpected := [][]string{
		{
			"two@two.net",
		},
		{
			"two@two.net",
			"three@three.net",
		},
	}

	// нормальная работа
	subscriptionsRepo.EXPECT().GetAllSubscriptions(context.Background()).Return(subsSent, nil)

	usersRepo.EXPECT().GetByID(context.Background(), uint32(0)).Return(usersSent[0], nil)
	usersRepo.EXPECT().GetByID(context.Background(), uint32(2)).Return(usersSent[2], nil)
	usersRepo.EXPECT().GetByID(context.Background(), uint32(1)).Return(usersSent[1], nil)
	usersRepo.EXPECT().GetByID(context.Background(), uint32(2)).Return(usersSent[2], nil)
	usersRepo.EXPECT().GetByID(context.Background(), uint32(3)).Return(usersSent[3], nil)
	usersRepo.EXPECT().GetByID(context.Background(), uint32(2)).Return(usersSent[2], nil)
	usersRepo.EXPECT().GetByID(context.Background(), uint32(3)).Return(usersSent[3], nil)

	messagesRecv, recipientsRecv, err := testService.makeMessages(context.Background())

	assert.NoError(t, err)
	assert.EqualValues(t, messagesExpected, messagesRecv)
	assert.EqualValues(t, recipientsExpected, recipientsRecv)

	// ошибка хранилища подписок
	subscriptionsRepo.EXPECT().GetAllSubscriptions(context.Background()).Return(nil, fmt.Errorf("repo error"))

	_, _, err = testService.makeMessages(context.Background())

	assert.Error(t, err)

	// подписок нет
	subscriptionsRepo.EXPECT().GetAllSubscriptions(context.Background()).Return([]*subscription.Subscription{}, nil)

	messagesRecv, recipientsRecv, err = testService.makeMessages(context.Background())

	assert.NoError(t, err)
	assert.Nil(t, messagesRecv)
	assert.Nil(t, recipientsRecv)

	// ошибка в хранилище пользователей
	subscriptionsRepo.EXPECT().GetAllSubscriptions(context.Background()).Return(subsSent, nil)

	usersRepo.EXPECT().GetByID(context.Background(), uint32(0)).Return(nil, fmt.Errorf("repo error"))

	_, _, err = testService.makeMessages(context.Background())

	assert.Error(t, err)

	// ошибка в хранилище пользователей в блоке сравнения daysAlert
	subscriptionsRepo.EXPECT().GetAllSubscriptions(context.Background()).Return(subsSent, nil)

	usersRepo.EXPECT().GetByID(context.Background(), uint32(0)).Return(usersSent[0], nil)
	usersRepo.EXPECT().GetByID(context.Background(), uint32(2)).Return(nil, fmt.Errorf("repo error"))

	_, _, err = testService.makeMessages(context.Background())

	assert.Error(t, err)
}

func TestAlert(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	usersRepo := user.NewMockUsersRepo(ctrl)
	subscriptionsRepo := subscription.NewMockSubscriptionsRepo(ctrl)
	sessManager := session.NewMockSessionsManager(ctrl)
	alertManager := alertmanager.NewMockAlertManager(ctrl)

	testService := NewCongratulationsServiceImpl(
		usersRepo,
		subscriptionsRepo,
		sessManager,
		alertManager,
		zap.NewNop().Sugar(),
	)

	// данные для теста
	subsSent := []*subscription.Subscription{
		{
			Subscriber:   0,
			Subscription: 1,
			DaysAlert:    2,
		},
		{
			Subscriber:   2,
			Subscription: 1,
			DaysAlert:    1,
		},
		{
			Subscriber:   3,
			Subscription: 1,
			DaysAlert:    1,
		},
		{
			Subscriber:   2,
			Subscription: 0,
			DaysAlert:    5,
		},
		{
			Subscriber:   3,
			Subscription: 2,
			DaysAlert:    10,
		},
		{
			Subscriber:   2,
			Subscription: 3,
			DaysAlert:    1,
		},
	}

	usersSent := []*user.User{
		{
			ID:       0,
			Username: "zero",
			Email:    "zero@zero.net",
			Month:    int(time.Now().AddDate(0, 0, 5).Month()),
			Day:      time.Now().AddDate(0, 0, 5).Day(),
		},
		{
			ID:       1,
			Username: "one",
			Email:    "one@one.net",
			Month:    int(time.Now().AddDate(0, 0, 1).Month()),
			Day:      time.Now().AddDate(0, 0, 1).Day(),
		},
		{
			ID:       2,
			Username: "two",
			Email:    "two@two.net",
			Month:    int(time.Now().AddDate(0, 0, 9).Month()),
			Day:      time.Now().AddDate(0, 0, 9).Day(),
		},
		{
			ID:       3,
			Username: "three",
			Email:    "three@three.net",
			Month:    int(time.Now().Month()),
			Day:      time.Now().Day(),
		},
	}

	messagesExpected := []string{
		"zero празднует свой день рождения через 5 дней!",
		"one празднует свой день рождения через 1 дней!",
	}

	recipientsExpected := [][]string{
		{
			"two@two.net",
		},
		{
			"two@two.net",
			"three@three.net",
		},
	}

	// нормальная работа
	wg := &sync.WaitGroup{}
	ctx, cancel := context.WithCancel(context.Background())

	for i := 0; i < 2; i++ {
		subscriptionsRepo.EXPECT().GetAllSubscriptions(ctx).Return(subsSent, nil)

		usersRepo.EXPECT().GetByID(ctx, uint32(0)).Return(usersSent[0], nil)
		usersRepo.EXPECT().GetByID(ctx, uint32(2)).Return(usersSent[2], nil)
		usersRepo.EXPECT().GetByID(ctx, uint32(1)).Return(usersSent[1], nil)
		usersRepo.EXPECT().GetByID(ctx, uint32(2)).Return(usersSent[2], nil)
		usersRepo.EXPECT().GetByID(ctx, uint32(3)).Return(usersSent[3], nil)
		usersRepo.EXPECT().GetByID(ctx, uint32(2)).Return(usersSent[2], nil)
		usersRepo.EXPECT().GetByID(ctx, uint32(3)).Return(usersSent[3], nil)

		alertManager.EXPECT().Send(recipientsExpected[0], "Напоминание о дне рождения!", messagesExpected[0])
		alertManager.EXPECT().Send(recipientsExpected[1], "Напоминание о дне рождения!", messagesExpected[1])
	}

	wg.Add(1)
	go testService.alert(ctx, time.Second, wg)

	time.Sleep(time.Millisecond * 1500)
	cancel()
	wg.Wait()

	// ошибка при первом создании сообщений
	wg = &sync.WaitGroup{}
	ctx, cancel = context.WithCancel(context.Background())

	subscriptionsRepo.EXPECT().GetAllSubscriptions(ctx).Return(nil, fmt.Errorf("repo error"))

	wg.Add(1)
	go testService.alert(ctx, time.Second, wg)

	time.Sleep(time.Millisecond * 1500)
	cancel()
	wg.Wait()

	// ошибка при создании сообщений по тикеру
	wg = &sync.WaitGroup{}
	ctx, cancel = context.WithCancel(context.Background())

	subscriptionsRepo.EXPECT().GetAllSubscriptions(ctx).Return(subsSent, nil)

	usersRepo.EXPECT().GetByID(ctx, uint32(0)).Return(usersSent[0], nil)
	usersRepo.EXPECT().GetByID(ctx, uint32(2)).Return(usersSent[2], nil)
	usersRepo.EXPECT().GetByID(ctx, uint32(1)).Return(usersSent[1], nil)
	usersRepo.EXPECT().GetByID(ctx, uint32(2)).Return(usersSent[2], nil)
	usersRepo.EXPECT().GetByID(ctx, uint32(3)).Return(usersSent[3], nil)
	usersRepo.EXPECT().GetByID(ctx, uint32(2)).Return(usersSent[2], nil)
	usersRepo.EXPECT().GetByID(ctx, uint32(3)).Return(usersSent[3], nil)

	alertManager.EXPECT().Send(recipientsExpected[0], "Напоминание о дне рождения!", messagesExpected[0])
	alertManager.EXPECT().Send(recipientsExpected[1], "Напоминание о дне рождения!", messagesExpected[1])

	subscriptionsRepo.EXPECT().GetAllSubscriptions(ctx).Return(nil, fmt.Errorf("repo error"))

	wg.Add(1)
	go testService.alert(ctx, time.Second, wg)

	time.Sleep(time.Millisecond * 1500)
	cancel()
	wg.Wait()
}
