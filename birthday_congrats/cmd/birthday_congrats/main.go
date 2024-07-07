package main

import (
	alertmanager "birthday_congrats/pkg/alert_manager"
	"birthday_congrats/pkg/handlers"
	"birthday_congrats/pkg/middlware"
	"birthday_congrats/pkg/service"
	"birthday_congrats/pkg/session"
	"birthday_congrats/pkg/user"
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

// const debugPathToTemplates = "C:\\Users\\sasha\\Desktop\\job_search\\RuTube\\test_task\\birthday_congrats_repository\\birthday_congrats\\templates\\*"

func main() {
	templates := template.Must(template.ParseGlob("./templates/*"))

	// логгер
	zapLogger, err := zap.NewProduction()
	if err != nil {
		fmt.Printf("Error while creating zap logger: %v", err)
	}
	defer func() {
		err = zapLogger.Sync()
		if err != nil {
			fmt.Printf("Error while zapLogger.Symc(): %v", err)
		}
	}()
	logger := zapLogger.Sugar()

	// база данных
	// основные настройки к базе
	dsn := "root:root@tcp(localhost:3306)/golang?"
	// указываем кодировку
	dsn += "&charset=utf8"
	// отказываемся от prapared statements
	// параметры подставляются сразу
	dsn += "&interpolateParams=true"

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

	// репозиторий
	usersRepo := user.NewUsersMySQLRepo(
		dbMySQL,
		logger,
	)

	// менеджер сессий
	sm := session.NewMySQLSessionsManager(
		dbMySQL,
		logger,
		int64(time.Minute/time.Second),
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
	service := service.NewCongratulationsService(
		usersRepo,
		sm,
		am,
		logger,
	)

	// запускаем сервис оповещений
	ctx, cancel := context.WithCancel(context.Background())
	wg := &sync.WaitGroup{}
	defer func() {
		cancel()
		wg.Wait()
	}()

	service.StartAlert(ctx, time.Now().Add(time.Minute*1), wg)

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
	router.HandleFunc("/error", serviceHandler.Error).Methods("GET")

	// хендлеры, требующие авторизации
	router.Handle("/users",
		middlware.Auth(sm, logger, http.HandlerFunc(serviceHandler.Users)))
	router.Handle("/subscribe/{user_id}",
		middlware.Auth(sm, logger, http.HandlerFunc(serviceHandler.Subscribe)))
	router.Handle("/unsubscribe/{user_id}",
		middlware.Auth(sm, logger, http.HandlerFunc(serviceHandler.Unsubscribe)))
	router.Handle("/logout",
		middlware.Auth(sm, logger, http.HandlerFunc(serviceHandler.Logout)))

	// добавляем миддлверы
	mux := middlware.Logger(logger, router)
	mux = middlware.Panic(logger, mux)

	port := ":8080"
	logger.Infow("Starting server",
		"type", "START",
		"addr", port)
	err = http.ListenAndServe(port, mux)
	if err != nil {
		logger.Errorw("Error while starting server",
			"type", "ERROR",
			"addr", port,
			"error", err)
	}
}
