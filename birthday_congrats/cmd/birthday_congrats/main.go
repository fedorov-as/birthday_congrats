package main

import (
	alertmanager "birthday_congrats/internal/pkg/alert_manager"
	"birthday_congrats/internal/pkg/handlers"
	"birthday_congrats/internal/pkg/middlware"
	"birthday_congrats/internal/pkg/session"
	"birthday_congrats/internal/pkg/subscription"
	"birthday_congrats/internal/pkg/user"
	service "birthday_congrats/internal/services/congrats_service"
	"context"
	"database/sql"
	"fmt"
	"html/template"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

const (
	port                     = ":8080" // порт
	minutesBeforeStartAlerts = 5       // время до запуска сервиса оповещений с момента старта программы в минутах
	logoutTimeoutMinutes     = 60      // время жизни сессии в минутах
	alertPeriodHours         = 24      // период отправки почтовых сообщений в часах
)

func main() {
	templates := template.Must(template.ParseGlob("./templates/*"))

	// логгер
	zapLogger, err := zap.NewProduction()
	if err != nil {
		fmt.Printf("Error while creating zap logger: %v", err)
	}
	logger := zapLogger.Sugar()

	// база данных
	dsn := "root:root@tcp(localhost:3306)/golang?" +
		"&charset=utf8" +
		"&interpolateParams=true"

	dbMySQL, err := sql.Open("mysql", dsn)
	if err != nil {
		logger.Errorf("Cant open connection to usersDB: %v", err)
		return
	}
	defer dbMySQL.Close()
	logger.Infow("Connected to MySQL database")

	dbMySQL.SetMaxOpenConns(10)

	err = dbMySQL.Ping()
	if err != nil {
		logger.Errorf("No connection to dbMySQL: %v", err)
		return
	}

	// репозитории
	usersRepo := user.NewUsersMySQLRepo(
		dbMySQL,
		logger,
	)

	subscriptionsRepo := subscription.NewSubscriptionsMySQLRepo(
		dbMySQL,
		logger,
	)

	// менеджер сессий
	sm := session.NewMySQLSessionsManager(
		dbMySQL,
		logger,
		int64(time.Minute*logoutTimeoutMinutes/time.Second),
		16,
	)

	// менеджер отправки писем
	am := alertmanager.NewEmailAlertManager(
		// Информация об отправителе (в продакшене я бы закинул это в credentials на github/gitlab)
		"birthday.congratulations@yandex.ru",
		"ucgcgejoiguychfa",

		// smtp сервер конфигурация
		"smtp.yandex.ru",
		"587",
		logger,
	)

	// сам сервис
	service := service.NewCongratulationsServiceImpl(
		usersRepo,
		subscriptionsRepo,
		sm,
		am,
		logger,
	)

	// запускаем сервис оповещений
	ctx, cancel := context.WithCancel(context.Background())
	wg := &sync.WaitGroup{}
	defer func(wg *sync.WaitGroup) {
		cancel()
		wg.Wait()
	}(wg)

	wg.Add(1)
	go service.StartAlert(ctx, time.Now().Add(time.Minute*minutesBeforeStartAlerts), time.Hour*alertPeriodHours, wg)

	// хендлеры
	serviceHandler := handlers.NewServiceHandler(
		templates,
		service,
		sm,
		logger,
	)

	// роутер
	router := mux.NewRouter()
	router.HandleFunc("/", serviceHandler.Index).Methods("GET")
	router.HandleFunc("/register", serviceHandler.Register).Methods("POST")
	router.HandleFunc("/login", serviceHandler.Login).Methods("POST")
	router.HandleFunc("/error", serviceHandler.ErrorPage).Methods("GET")

	// хендлеры, требующие авторизации
	router.Handle("/users",
		middlware.Auth(sm, logger, http.HandlerFunc(serviceHandler.Users))).Methods("GET")
	router.Handle("/subscribe/{user_id}",
		middlware.Auth(sm, logger, http.HandlerFunc(serviceHandler.Subscribe))).Methods("POST")
	router.Handle("/unsubscribe/{user_id}",
		middlware.Auth(sm, logger, http.HandlerFunc(serviceHandler.Unsubscribe))).Methods("POST")
	router.Handle("/logout",
		middlware.Auth(sm, logger, http.HandlerFunc(serviceHandler.Logout))).Methods("GET")

	// добавляем миддлверы
	mux := middlware.Logger(logger, router)
	mux = middlware.Panic(logger, mux)

	// сервер
	server := &http.Server{
		Addr:    port,
		Handler: mux,
	}

	ctx, stopServer := context.WithCancel(context.Background())
	wg = &sync.WaitGroup{}
	defer func(wg *sync.WaitGroup) {
		stopServer()
		wg.Wait()
	}(wg)

	wg.Add(1)
	go startServer(ctx, server, logger, wg)

	fmt.Scanln()
	stopServer()
}

func startServer(ctx context.Context, server *http.Server, logger *zap.SugaredLogger, wg *sync.WaitGroup) {
	defer wg.Done()

	// горутина, которая остановит сервер
	go func() {
		<-ctx.Done()
		err := server.Shutdown(ctx)
		if err != nil {
			logger.Errorw("Error while shutting down server",
				"type", "ERROR",
				"addr", port)
		}
	}()

	// запуск сервера
	logger.Infow("Starting server",
		"type", "START",
		"addr", port)
	err := server.ListenAndServe()
	if err == http.ErrServerClosed {
		logger.Infow("Shutting down server",
			"type", "STOP",
			"addr", port)
		return
	}
	if err != nil {
		logger.Errorw("Error while starting server",
			"type", "ERROR",
			"addr", port,
			"error", err)
	}
}
