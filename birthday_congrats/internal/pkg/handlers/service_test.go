package handlers

import (
	"birthday_congrats/internal/pkg/session"
	"birthday_congrats/internal/pkg/user"
	"birthday_congrats/internal/services/congrats_service"
	"fmt"
	"html/template"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

const (
	templatesPath = "./../../../templates/*"
)

type errorResponseWriter struct {
	err  error
	Code int
}

var _ http.ResponseWriter = &errorResponseWriter{}

func (w *errorResponseWriter) Header() http.Header {
	return http.Header{}
}

func (w *errorResponseWriter) Write([]byte) (int, error) {
	return 0, w.err
}

func (w *errorResponseWriter) WriteHeader(statusCode int) {
	w.Code = statusCode
}

func TestExecErrorTemplate(t *testing.T) {
	tmpl := template.Must(template.ParseGlob(templatesPath))

	testHandler := NewServiceHandler(
		tmpl,
		nil, nil,
		zap.NewNop().Sugar(),
	)

	// данные для теста
	status := 201

	// нормальная работа
	w := httptest.NewRecorder()

	testHandler.execErrorTemplate(w, "", status)

	assert.EqualValues(t, status, w.Code)

	// ошибка шаблона
	wErr := &errorResponseWriter{err: fmt.Errorf("error")}

	testHandler.execErrorTemplate(wErr, "", status)

	assert.EqualValues(t, http.StatusInternalServerError, wErr.Code)
}

func TestErrorPage(t *testing.T) {
	tmpl := template.Must(template.ParseGlob(templatesPath))

	testHandler := NewServiceHandler(
		tmpl,
		nil, nil,
		zap.NewNop().Sugar(),
	)

	// нормальная работа
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/error", nil)

	testHandler.ErrorPage(w, r)

	assert.EqualValues(t, http.StatusInternalServerError, w.Code)
}

func TestUsers(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	service := congrats_service.NewMockCongratulationsService(ctrl)

	tmpl := template.Must(template.ParseGlob(templatesPath))

	testHandler := NewServiceHandler(
		tmpl,
		service,
		nil,
		zap.NewNop().Sugar(),
	)

	// данные для теста
	usersSent := make([]*user.User, 10)

	// нормальная работа
	statusExpected := http.StatusOK
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/users", nil)

	service.EXPECT().GetSubscriptionsByUser(r.Context()).Return(usersSent, nil)

	testHandler.Users(w, r)

	assert.EqualValues(t, statusExpected, w.Code)

	// ошибка сервиса -> редирект на /error
	statusExpected = http.StatusFound
	w = httptest.NewRecorder()
	r = httptest.NewRequest(http.MethodGet, "/users", nil)

	service.EXPECT().GetSubscriptionsByUser(r.Context()).Return(nil, fmt.Errorf("service error"))

	testHandler.Users(w, r)

	assert.EqualValues(t, statusExpected, w.Code)

	// ошибка шаблона -> редирект на /error
	statusExpected = http.StatusFound
	wErr := &errorResponseWriter{err: fmt.Errorf("error")}
	r = httptest.NewRequest(http.MethodGet, "/users", nil)

	service.EXPECT().GetSubscriptionsByUser(r.Context()).Return(usersSent, nil)

	testHandler.Users(wErr, r)

	assert.EqualValues(t, statusExpected, wErr.Code)
}

func TestIndex(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	sessManager := session.NewMockSessionsManager(ctrl)

	tmpl := template.Must(template.ParseGlob(templatesPath))

	testHandler := NewServiceHandler(
		tmpl,
		nil,
		sessManager,
		zap.NewNop().Sugar(),
	)

	// данные для теста
	sessSent := &session.Session{
		SessID:  "some_sess_id",
		UserID:  42,
		Expires: time.Now().Unix() + 60,
	}

	// нормальная работа
	statusExpected := http.StatusFound
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)

	sessManager.EXPECT().Check(r).Return(sessSent, nil)

	testHandler.Index(w, r)

	assert.EqualValues(t, statusExpected, w.Code)

	// нет сессии
	statusExpected = http.StatusOK
	w = httptest.NewRecorder()
	r = httptest.NewRequest(http.MethodGet, "/", nil)

	sessManager.EXPECT().Check(r).Return(nil, session.ErrNoSession)

	testHandler.Index(w, r)

	assert.EqualValues(t, statusExpected, w.Code)

	// нет сессии + ошибка шаблона
	statusExpected = http.StatusFound
	wErr := &errorResponseWriter{err: fmt.Errorf("error")}
	r = httptest.NewRequest(http.MethodGet, "/", nil)

	sessManager.EXPECT().Check(r).Return(nil, session.ErrNoSession)

	testHandler.Index(wErr, r)

	assert.EqualValues(t, statusExpected, wErr.Code)
}

func TestRegister(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	service := congrats_service.NewMockCongratulationsService(ctrl)

	tmpl := template.Must(template.ParseGlob(templatesPath))

	testHandler := NewServiceHandler(
		tmpl,
		service,
		nil,
		zap.NewNop().Sugar(),
	)

	// данные для теста
	username := "some_user"
	password := "some_password"
	email := "some@email.com"
	birth := "2006-01-02"

	sessExpected := &session.Session{
		SessID:  "some_sess_id",
		UserID:  42,
		Expires: time.Now().Unix() + 60,
	}

	cookieExpected := &http.Cookie{
		Name:    "session_id",
		Value:   sessExpected.SessID,
		Expires: time.Unix(sessExpected.Expires, 0).UTC(),
	}

	// нормальная работа
	statusExpected := http.StatusFound
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/register", nil)

	err := r.ParseForm()
	if err != nil {
		t.Fatalf(err.Error())
	}

	r.Form.Set("username", username)
	r.Form.Set("password", password)
	r.Form.Set("email", email)
	r.Form.Set("birth", birth)

	service.EXPECT().Register(
		r.Context(),
		username,
		password,
		email,
		birth,
	).Return(sessExpected, nil)

	testHandler.Register(w, r)

	assert.EqualValues(t, statusExpected, w.Code)

	result := w.Result()

	assert.EqualValues(t, 1, len(result.Cookies()))
	assert.EqualValues(t, cookieExpected.Name, result.Cookies()[0].Name)
	assert.EqualValues(t, cookieExpected.Value, result.Cookies()[0].Value)
	assert.EqualValues(t, cookieExpected.Expires, result.Cookies()[0].Expires)

	err = result.Body.Close()
	if err != nil {
		t.Fatalf("error closing body: %v", err)
	}

	// ошибка сервиса
	statusExpected = http.StatusFound
	w = httptest.NewRecorder()
	r = httptest.NewRequest(http.MethodPost, "/register", nil)

	err = r.ParseForm()
	if err != nil {
		t.Fatalf(err.Error())
	}

	r.Form.Set("username", username)
	r.Form.Set("password", password)
	r.Form.Set("email", email)
	r.Form.Set("birth", birth)

	service.EXPECT().Register(
		r.Context(),
		username,
		password,
		email,
		birth,
	).Return(nil, fmt.Errorf("service error"))

	testHandler.Register(w, r)

	result = w.Result()

	assert.EqualValues(t, statusExpected, w.Code)
	assert.EqualValues(t, 0, len(result.Cookies()))

	err = result.Body.Close()
	if err != nil {
		t.Fatalf("error closing body: %v", err)
	}

	// пользователь уже существует
	statusExpected = http.StatusForbidden
	w = httptest.NewRecorder()
	r = httptest.NewRequest(http.MethodPost, "/register", nil)

	err = r.ParseForm()
	if err != nil {
		t.Fatalf(err.Error())
	}

	r.Form.Set("username", username)
	r.Form.Set("password", password)
	r.Form.Set("email", email)
	r.Form.Set("birth", birth)

	service.EXPECT().Register(
		r.Context(),
		username,
		password,
		email,
		birth,
	).Return(nil, user.ErrUserExists)

	testHandler.Register(w, r)

	result = w.Result()

	assert.EqualValues(t, statusExpected, w.Code)
	assert.EqualValues(t, 0, len(result.Cookies()))

	err = result.Body.Close()
	if err != nil {
		t.Fatalf("error closing body: %v", err)
	}
}

func TestLogin(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	service := congrats_service.NewMockCongratulationsService(ctrl)

	tmpl := template.Must(template.ParseGlob(templatesPath))

	testHandler := NewServiceHandler(
		tmpl,
		service,
		nil,
		zap.NewNop().Sugar(),
	)

	// данные для теста
	username := "some_user"
	password := "some_password"

	sessExpected := &session.Session{
		SessID:  "some_sess_id",
		UserID:  42,
		Expires: time.Now().Unix() + 60,
	}

	cookieExpected := &http.Cookie{
		Name:    "session_id",
		Value:   sessExpected.SessID,
		Expires: time.Unix(sessExpected.Expires, 0).UTC(),
	}

	// нормальная работа
	statusExpected := http.StatusFound
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/login", nil)

	err := r.ParseForm()
	if err != nil {
		t.Fatalf(err.Error())
	}

	r.Form.Set("username", username)
	r.Form.Set("password", password)

	service.EXPECT().Login(
		r.Context(),
		username,
		password,
	).Return(sessExpected, nil)

	testHandler.Login(w, r)

	result := w.Result()

	assert.EqualValues(t, statusExpected, w.Code)
	assert.EqualValues(t, 1, len(result.Cookies()))
	assert.EqualValues(t, cookieExpected.Name, result.Cookies()[0].Name)
	assert.EqualValues(t, cookieExpected.Value, result.Cookies()[0].Value)
	assert.EqualValues(t, cookieExpected.Expires, result.Cookies()[0].Expires)

	err = result.Body.Close()
	if err != nil {
		t.Fatalf("error closing body: %v", err)
	}

	// ошибка сервиса
	statusExpected = http.StatusFound
	w = httptest.NewRecorder()
	r = httptest.NewRequest(http.MethodPost, "/login", nil)

	err = r.ParseForm()
	if err != nil {
		t.Fatalf(err.Error())
	}

	r.Form.Set("username", username)
	r.Form.Set("password", password)

	service.EXPECT().Login(
		r.Context(),
		username,
		password,
	).Return(nil, fmt.Errorf("service error"))

	testHandler.Login(w, r)

	result = w.Result()

	assert.EqualValues(t, statusExpected, w.Code)
	assert.EqualValues(t, 0, len(result.Cookies()))

	err = result.Body.Close()
	if err != nil {
		t.Fatalf("error closing body: %v", err)
	}

	// пользователь уже существует
	statusExpected = http.StatusForbidden
	w = httptest.NewRecorder()
	r = httptest.NewRequest(http.MethodPost, "/login", nil)

	err = r.ParseForm()
	if err != nil {
		t.Fatalf(err.Error())
	}

	r.Form.Set("username", username)
	r.Form.Set("password", password)

	service.EXPECT().Login(
		r.Context(),
		username,
		password,
	).Return(nil, user.ErrNoUser)

	testHandler.Login(w, r)

	result = w.Result()

	assert.EqualValues(t, statusExpected, w.Code)
	assert.EqualValues(t, 0, len(result.Cookies()))

	err = result.Body.Close()
	if err != nil {
		t.Fatalf("error closing body: %v", err)
	}
}

func TestSubscribe(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	service := congrats_service.NewMockCongratulationsService(ctrl)

	tmpl := template.Must(template.ParseGlob(templatesPath))

	testHandler := NewServiceHandler(
		tmpl,
		service,
		nil,
		zap.NewNop().Sugar(),
	)

	// данные для теста
	userID := uint32(42)
	daysAlert := 7

	// нормальная работа
	statusExpected := http.StatusFound
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/subscribe/", nil)
	r = mux.SetURLVars(r, map[string]string{"user_id": strconv.Itoa(int(userID))})

	err := r.ParseForm()
	if err != nil {
		t.Fatalf(err.Error())
	}

	r.Form.Set("days_alert", strconv.Itoa(daysAlert))

	service.EXPECT().Subscribe(r.Context(), userID, daysAlert).Return(nil)

	testHandler.Subscribe(w, r)

	assert.EqualValues(t, statusExpected, w.Code)

	// некорректный id
	statusExpected = http.StatusFound
	w = httptest.NewRecorder()
	r = httptest.NewRequest(http.MethodPost, "/subscribe/", nil)
	r = mux.SetURLVars(r, map[string]string{"user_id": "bad_id"})

	err = r.ParseForm()
	if err != nil {
		t.Fatalf(err.Error())
	}

	r.Form.Set("days_alert", strconv.Itoa(daysAlert))

	testHandler.Subscribe(w, r)

	assert.EqualValues(t, statusExpected, w.Code)

	// некорректный daysAlert
	statusExpected = http.StatusFound
	w = httptest.NewRecorder()
	r = httptest.NewRequest(http.MethodPost, "/subscribe/", nil)
	r = mux.SetURLVars(r, map[string]string{"user_id": strconv.Itoa(int(userID))})

	err = r.ParseForm()
	if err != nil {
		t.Fatalf(err.Error())
	}

	r.Form.Set("days_alert", "days_alert")

	testHandler.Subscribe(w, r)

	assert.EqualValues(t, statusExpected, w.Code)

	// ошибка сервиса
	statusExpected = http.StatusFound
	w = httptest.NewRecorder()
	r = httptest.NewRequest(http.MethodPost, "/subscribe/", nil)
	r = mux.SetURLVars(r, map[string]string{"user_id": strconv.Itoa(int(userID))})

	err = r.ParseForm()
	if err != nil {
		t.Fatalf(err.Error())
	}

	r.Form.Set("days_alert", strconv.Itoa(daysAlert))

	service.EXPECT().Subscribe(r.Context(), userID, daysAlert).Return(fmt.Errorf("service error"))

	testHandler.Subscribe(w, r)

	assert.EqualValues(t, statusExpected, w.Code)
}

func TestUnsubscribe(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	service := congrats_service.NewMockCongratulationsService(ctrl)

	tmpl := template.Must(template.ParseGlob(templatesPath))

	testHandler := NewServiceHandler(
		tmpl,
		service,
		nil,
		zap.NewNop().Sugar(),
	)

	// данные для теста
	userID := uint32(42)

	// нормальная работа
	statusExpected := http.StatusFound
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/unsubscribe/", nil)
	r = mux.SetURLVars(r, map[string]string{"user_id": strconv.Itoa(int(userID))})

	err := r.ParseForm()
	if err != nil {
		t.Fatalf(err.Error())
	}

	service.EXPECT().Unsubscribe(r.Context(), userID).Return(nil)

	testHandler.Unsubscribe(w, r)

	assert.EqualValues(t, statusExpected, w.Code)

	// некорректный id
	statusExpected = http.StatusFound
	w = httptest.NewRecorder()
	r = httptest.NewRequest(http.MethodPost, "/unsubscribe/", nil)
	r = mux.SetURLVars(r, map[string]string{"user_id": "bad_id"})

	err = r.ParseForm()
	if err != nil {
		t.Fatalf(err.Error())
	}

	testHandler.Unsubscribe(w, r)

	assert.EqualValues(t, statusExpected, w.Code)

	// ошибка сервиса
	statusExpected = http.StatusFound
	w = httptest.NewRecorder()
	r = httptest.NewRequest(http.MethodPost, "/unsubscribe/", nil)
	r = mux.SetURLVars(r, map[string]string{"user_id": strconv.Itoa(int(userID))})

	err = r.ParseForm()
	if err != nil {
		t.Fatalf(err.Error())
	}

	service.EXPECT().Unsubscribe(r.Context(), userID).Return(fmt.Errorf("service error"))

	testHandler.Unsubscribe(w, r)

	assert.EqualValues(t, statusExpected, w.Code)
}

func TestLogout(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	service := congrats_service.NewMockCongratulationsService(ctrl)

	tmpl := template.Must(template.ParseGlob(templatesPath))

	testHandler := NewServiceHandler(
		tmpl,
		service,
		nil,
		zap.NewNop().Sugar(),
	)

	// данные для теста
	cookieSent := &http.Cookie{
		Name:    "session_id",
		Value:   "some_sess_id",
		Expires: time.Now().Add(time.Minute),
	}

	cookieExpected := &http.Cookie{
		Name:    "session_id",
		Value:   "some_sess_id",
		Expires: time.Now().AddDate(0, 0, -1),
	}

	// нормальная работа
	statusExpected := http.StatusFound
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/logout", nil)

	r.AddCookie(cookieSent)

	service.EXPECT().Logout(r.Context()).Return(nil)

	testHandler.Logout(w, r)

	result := w.Result()

	assert.EqualValues(t, statusExpected, w.Code)
	assert.EqualValues(t, 1, len(result.Cookies()))
	assert.EqualValues(t, cookieExpected.Name, result.Cookies()[0].Name)
	assert.EqualValues(t, cookieExpected.Value, result.Cookies()[0].Value)
	assert.EqualValues(t, cookieExpected.Expires.Local().Day(), result.Cookies()[0].Expires.Local().Day())

	err := result.Body.Close()
	if err != nil {
		t.Fatalf("error closing body: %v", err)
	}

	// ошибка сервиса
	statusExpected = http.StatusFound
	w = httptest.NewRecorder()
	r = httptest.NewRequest(http.MethodGet, "/logout", nil)

	r.AddCookie(cookieSent)

	service.EXPECT().Logout(r.Context()).Return(fmt.Errorf("service error"))

	testHandler.Logout(w, r)

	result = w.Result()

	assert.EqualValues(t, statusExpected, w.Code)
	assert.EqualValues(t, 0, len(result.Cookies()))

	err = result.Body.Close()
	if err != nil {
		t.Fatalf("error closing body: %v", err)
	}

	// сессия не уничтожена на сервере
	statusExpected = http.StatusFound
	w = httptest.NewRecorder()
	r = httptest.NewRequest(http.MethodGet, "/logout", nil)

	r.AddCookie(cookieSent)

	service.EXPECT().Logout(r.Context()).Return(session.ErrNotDestroyed)

	testHandler.Logout(w, r)

	result = w.Result()

	assert.EqualValues(t, statusExpected, w.Code)
	assert.EqualValues(t, 1, len(result.Cookies()))
	assert.EqualValues(t, cookieExpected.Name, result.Cookies()[0].Name)
	assert.EqualValues(t, cookieExpected.Value, result.Cookies()[0].Value)
	assert.EqualValues(t, cookieExpected.Expires.Local().Day(), result.Cookies()[0].Expires.Local().Day())

	err = result.Body.Close()
	if err != nil {
		t.Fatalf("error closing body: %v", err)
	}

	// в запросе нет куки
	statusExpected = http.StatusFound
	w = httptest.NewRecorder()
	r = httptest.NewRequest(http.MethodGet, "/logout", nil)

	service.EXPECT().Logout(r.Context()).Return(nil)

	testHandler.Logout(w, r)

	result = w.Result()

	assert.EqualValues(t, statusExpected, w.Code)
	assert.EqualValues(t, 0, len(result.Cookies()))

	err = result.Body.Close()
	if err != nil {
		t.Fatalf("error closing body: %v", err)
	}
}
