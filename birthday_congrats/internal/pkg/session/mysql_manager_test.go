package session

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	sqlmock "gopkg.in/DATA-DOG/go-sqlmock.v1"
)

type customErrorResult struct {
	errLastID   error
	errAffected error
}

var _ sql.Result = &customErrorResult{}

func (res *customErrorResult) LastInsertId() (int64, error) {
	return int64(0), res.errLastID
}

func (res *customErrorResult) RowsAffected() (int64, error) {
	return int64(1), res.errAffected
}

func TestCreate(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("cant create mock: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	expirationTime := 60
	sessIDlength := 16
	testManager := NewMySQLSessionsManager(
		db,
		zap.NewNop().Sugar(),
		int64(expirationTime),
		sessIDlength,
	)

	// данные для теста
	userID := uint32(42)
	sessExpected := &Session{
		UserID: userID,
	}

	// нормальная работа
	mock.
		ExpectExec("INSERT INTO sessions").
		WithArgs(sqlmock.AnyArg(), userID, sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))

	sessRecv, err := testManager.Create(ctx, userID)

	assert.NoError(t, err)
	assert.EqualValues(t, sessExpected.UserID, sessRecv.UserID)
	assert.EqualValues(t, sessIDlength, utf8.RuneCountInString(sessRecv.SessID))

	actualExpirationTime := sessRecv.Expires - time.Now().Unix()
	assert.True(t, float64(actualExpirationTime) > 0.5*float64(expirationTime) &&
		float64(actualExpirationTime) < 1.5*float64(expirationTime))

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)

	// ответ с ошибкой
	mock.
		ExpectExec("INSERT INTO sessions").
		WithArgs(sqlmock.AnyArg(), userID, sqlmock.AnyArg()).
		WillReturnError(fmt.Errorf("db error"))

	_, err = testManager.Create(ctx, userID)

	assert.Error(t, err)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)

	// rows affected = 0
	mock.
		ExpectExec("INSERT INTO sessions").
		WithArgs(sqlmock.AnyArg(), userID, sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 0))

	_, err = testManager.Create(ctx, userID)

	assert.ErrorIs(t, err, ErrSessionNotCreated)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)

	// ошибка rowsAffected()
	mock.
		ExpectExec("INSERT INTO sessions").
		WithArgs(sqlmock.AnyArg(), userID, sqlmock.AnyArg()).
		WillReturnResult(&customErrorResult{errAffected: fmt.Errorf("affected error")})

	_, err = testManager.Create(ctx, userID)

	assert.Error(t, err)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestDestroy(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("cant create mock: %v", err)
	}
	defer db.Close()

	testManager := NewMySQLSessionsManager(
		db,
		zap.NewNop().Sugar(),
		int64(0),
		0,
	)

	// данные для теста
	sessExpected := &Session{
		SessID:  "some_sess_id",
		UserID:  uint32(42),
		Expires: 0,
	}

	// нормальная работа
	ctx := ContextWithSession(context.Background(), sessExpected)

	mock.
		ExpectExec("DELETE FROM sessions WHERE").
		WithArgs(sessExpected.SessID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = testManager.Destroy(ctx)

	assert.NoError(t, err)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)

	// в контексте нет сессии
	ctx = context.Background()

	err = testManager.Destroy(ctx)
	assert.ErrorIs(t, err, ErrNoSession)

	// ответ с ошибкой
	ctx = ContextWithSession(context.Background(), sessExpected)

	mock.
		ExpectExec("DELETE FROM sessions WHERE").
		WithArgs(sessExpected.SessID).
		WillReturnError(fmt.Errorf("db error"))

	err = testManager.Destroy(ctx)

	assert.Error(t, err)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)

	// rows affected = 0
	mock.
		ExpectExec("DELETE FROM sessions WHERE").
		WithArgs(sessExpected.SessID).
		WillReturnResult(sqlmock.NewResult(0, 0))

	err = testManager.Destroy(ctx)

	assert.ErrorIs(t, err, ErrNotDestroyed)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)

	// ошибка rowsAffected()
	mock.
		ExpectExec("DELETE FROM sessions WHERE").
		WithArgs(sessExpected.SessID).
		WillReturnResult(&customErrorResult{errAffected: fmt.Errorf("affected error")})

	err = testManager.Destroy(ctx)

	assert.Error(t, err)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestCheck(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("cant create mock: %v", err)
	}
	defer db.Close()

	testManager := NewMySQLSessionsManager(
		db,
		zap.NewNop().Sugar(),
		int64(0),
		0,
	)

	// данные для теста
	sessExpected := &Session{
		SessID:  "some_sess_id",
		UserID:  uint32(42),
		Expires: time.Now().Unix() + 60,
	}

	cookie := &http.Cookie{
		Name:    "session_id",
		Value:   sessExpected.SessID,
		Expires: time.Unix(sessExpected.Expires, 0),
	}

	// нормальная работа
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(cookie)

	rows := sqlmock.NewRows([]string{"user_id", "expires"})
	rows = rows.AddRow(sessExpected.UserID, sessExpected.Expires)

	mock.
		ExpectQuery("SELECT user_id, expires FROM sessions WHERE").
		WithArgs(sessExpected.SessID).
		WillReturnRows(rows)

	sessRecv, err := testManager.Check(req)

	assert.NoError(t, err)
	assert.EqualValues(t, sessExpected, sessRecv)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)

	// нет куки
	req = httptest.NewRequest(http.MethodGet, "/", nil)

	_, err = testManager.Check(req)

	assert.ErrorIs(t, err, ErrNoSession)

	// ответ с ошибкой
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(cookie)

	mock.
		ExpectQuery("SELECT user_id, expires FROM sessions WHERE").
		WithArgs(sessExpected.SessID).
		WillReturnError(fmt.Errorf("db error"))

	_, err = testManager.Check(req)

	assert.Error(t, err)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)

	// ошибка scan
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(cookie)

	rows = sqlmock.NewRows([]string{""})
	rows = rows.AddRow("")

	mock.
		ExpectQuery("SELECT user_id, expires FROM sessions WHERE").
		WithArgs(sessExpected.SessID).
		WillReturnRows(rows)

	_, err = testManager.Check(req)

	assert.Error(t, err)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)

	// сессия не найдена
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(cookie)

	rows = sqlmock.NewRows([]string{"user_id", "expires"})

	mock.
		ExpectQuery("SELECT user_id, expires FROM sessions WHERE").
		WithArgs(sessExpected.SessID).
		WillReturnRows(rows)

	_, err = testManager.Check(req)

	assert.ErrorIs(t, err, ErrNoSession)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)

	// сессия истекла
	sessExpected.Expires = 0
	cookie.Expires = time.Unix(0, 0)

	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(cookie)

	ctx := ContextWithSession(req.Context(), sessExpected)
	req = req.WithContext(ctx)

	rows = sqlmock.NewRows([]string{"user_id", "expires"})
	rows = rows.AddRow(sessExpected.UserID, sessExpected.Expires)

	mock.
		ExpectQuery("SELECT user_id, expires FROM sessions WHERE").
		WithArgs(sessExpected.SessID).
		WillReturnRows(rows)

	mock.
		ExpectExec("DELETE FROM sessions WHERE").
		WithArgs(sessExpected.SessID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	_, err = testManager.Check(req)

	assert.ErrorIs(t, err, ErrSessionExpired)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)

	// сессия истекла + ошибка
	sessExpected.Expires = 0
	cookie.Expires = time.Unix(0, 0)

	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(cookie)

	rows = sqlmock.NewRows([]string{"user_id", "expires"})
	rows = rows.AddRow(sessExpected.UserID, sessExpected.Expires)

	mock.
		ExpectQuery("SELECT user_id, expires FROM sessions WHERE").
		WithArgs(sessExpected.SessID).
		WillReturnRows(rows)

	_, err = testManager.Check(req)

	assert.Error(t, err)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}
