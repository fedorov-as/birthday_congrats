package main

import (
	"birthday_congrats/pkg/handlers"
	"birthday_congrats/pkg/middlware"
	"birthday_congrats/pkg/session"
	"fmt"
	"html/template"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

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

	// менеджер сессий
	sm := session.NewMySQLSessionsManager(
		nil,
		logger,
		int64(time.Minute),
		16,
	)

	// хендлеры
	serviceHandler := handlers.NewServiceHandler(
		templates,
		nil,
		nil,
		logger,
	)

	// роутер
	router := mux.NewRouter()
	router.HandleFunc("/", serviceHandler.Index).Methods("GET")
	router.HandleFunc("/error", serviceHandler.Error).Methods("GET")

	// хендлеры, требующие авторизации
	router.Handle("/users",
		middlware.Auth(sm, logger, http.HandlerFunc(serviceHandler.Index)))
	router.Handle("/subscribe/{user_id}",
		middlware.Auth(sm, logger, http.HandlerFunc(serviceHandler.Index)))
	router.Handle("/unsubscribe/{user_id}",
		middlware.Auth(sm, logger, http.HandlerFunc(serviceHandler.Index)))

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
