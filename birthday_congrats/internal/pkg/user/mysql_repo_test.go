package user

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

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

	testRepo := NewUsersMySQLRepo(db, zap.NewNop().Sugar())

	// общие данные для теста
	userID := uint32(0)
	username := "some_user"
	password := "some_pass"
	email := "some@email.net"
	year := 2000
	month := 1
	day := 1
	userExpected := &User{
		ID:       userID,
		Username: username,
		Password: "", // пароль не возвращается
		Email:    email,
		Year:     year,
		Month:    month,
		Day:      day,
	}

	// нормальная работа
	rows := sqlmock.NewRows([]string{"id"})

	mock.
		ExpectQuery("SELECT id from users WHERE").
		WithArgs(username).
		WillReturnRows(rows)

	mock.
		ExpectExec("INSERT INTO users").
		WithArgs(username, password, email, year, month, day).
		WillReturnResult(sqlmock.NewResult(int64(userExpected.ID), 1))

	userRecv, err := testRepo.Create(ctx, username, password, email, year, month, day)

	assert.NoError(t, err)
	assert.EqualValues(t, userExpected, userRecv)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)

	// ответ с ошибкой
	mock.
		ExpectQuery("SELECT id from users WHERE").
		WithArgs(username).
		WillReturnError(fmt.Errorf("db error"))

	_, err = testRepo.Create(ctx, username, password, email, year, month, day)

	assert.Error(t, err)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)

	// ошибка scan
	rows = sqlmock.NewRows([]string{})

	mock.
		ExpectQuery("SELECT id from users WHERE").
		WithArgs(username).
		WillReturnRows(rows)

	_, err = testRepo.Create(ctx, username, password, email, year, month, day)

	assert.Error(t, err)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)

	// найден пользователь с таким же именем
	rows = sqlmock.NewRows([]string{"id"})
	rows = rows.AddRow(uint32(0))

	mock.
		ExpectQuery("SELECT id from users WHERE").
		WithArgs(username).
		WillReturnRows(rows)

	_, err = testRepo.Create(ctx, username, password, email, year, month, day)

	assert.ErrorIs(t, err, ErrUserExists)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)

	// rows affected = 0
	rows = sqlmock.NewRows([]string{"id"})

	mock.
		ExpectQuery("SELECT id from users WHERE").
		WithArgs(username).
		WillReturnRows(rows)

	mock.
		ExpectExec("INSERT INTO users").
		WithArgs(username, password, email, year, month, day).
		WillReturnResult(sqlmock.NewResult(int64(userExpected.ID), 0))

	_, err = testRepo.Create(ctx, username, password, email, year, month, day)

	assert.ErrorIs(t, err, ErrUserNotCreated)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)

	// ошибка rowsAffected()
	rows = sqlmock.NewRows([]string{"id"})

	mock.
		ExpectQuery("SELECT id from users WHERE").
		WithArgs(username).
		WillReturnRows(rows)

	mock.
		ExpectExec("INSERT INTO users").
		WithArgs(username, password, email, year, month, day).
		WillReturnResult(&customErrorResult{errAffected: fmt.Errorf("affected error")})

	_, err = testRepo.Create(ctx, username, password, email, year, month, day)

	assert.Error(t, err)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)

	// ошибка LastInsertedId()
	rows = sqlmock.NewRows([]string{"id"})

	mock.
		ExpectQuery("SELECT id from users WHERE").
		WithArgs(username).
		WillReturnRows(rows)

	mock.
		ExpectExec("INSERT INTO users").
		WithArgs(username, password, email, year, month, day).
		WillReturnResult(&customErrorResult{errLastID: fmt.Errorf("lastID error")})

	_, err = testRepo.Create(ctx, username, password, email, year, month, day)

	assert.Error(t, err)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestLogin(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("cant create mock: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	testRepo := NewUsersMySQLRepo(db, zap.NewNop().Sugar())

	// общие данные для теста
	userID := uint32(0)
	username := "some_user"
	password := "some_pass"
	email := "some@email.net"
	year := 2000
	month := 1
	day := 1
	userExpected := &User{
		ID:       userID,
		Username: username,
		Password: "", // пароль не возвращается
		Email:    email,
		Year:     year,
		Month:    month,
		Day:      day,
	}

	// нормальная работа
	rows := sqlmock.NewRows([]string{"id", "username", "password", "email", "year", "month", "day"})
	rows = rows.AddRow(
		userExpected.ID,
		userExpected.Username,
		password,
		userExpected.Email,
		userExpected.Year,
		userExpected.Month,
		userExpected.Day,
	)

	mock.
		ExpectQuery("SELECT id, username, password, email, year, month, day FROM users WHERE").
		WithArgs(username).
		WillReturnRows(rows)

	userRecv, err := testRepo.Login(ctx, username, password)

	assert.NoError(t, err)
	assert.EqualValues(t, userExpected, userRecv)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)

	// ответ с ошибкой
	mock.
		ExpectQuery("SELECT id, username, password, email, year, month, day FROM users WHERE").
		WithArgs(username).
		WillReturnError(fmt.Errorf("db error"))

	_, err = testRepo.Login(ctx, username, password)

	assert.Error(t, err)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)

	// ошибка scan
	rows = sqlmock.NewRows([]string{""})

	mock.
		ExpectQuery("SELECT id, username, password, email, year, month, day FROM users WHERE").
		WithArgs(username).
		WillReturnRows(rows)

	_, err = testRepo.Login(ctx, username, password)

	assert.Error(t, err)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)

	// не найден пользователь с таким именем
	rows = sqlmock.NewRows([]string{"id", "username", "password", "email", "year", "month", "day"})

	mock.
		ExpectQuery("SELECT id, username, password, email, year, month, day FROM users WHERE").
		WithArgs(username).
		WillReturnRows(rows)

	_, err = testRepo.Login(ctx, username, password)

	assert.ErrorIs(t, err, ErrNoUser)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)

	// неверный пароль
	rows = sqlmock.NewRows([]string{"id", "username", "password", "email", "year", "month", "day"})
	rows = rows.AddRow(
		userExpected.ID,
		userExpected.Username,
		"bad_pass",
		userExpected.Email,
		userExpected.Year,
		userExpected.Month,
		userExpected.Day,
	)

	mock.
		ExpectQuery("SELECT id, username, password, email, year, month, day FROM users WHERE").
		WithArgs(username).
		WillReturnRows(rows)

	_, err = testRepo.Login(ctx, username, password)

	assert.ErrorIs(t, err, ErrBadPassword)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestGetAll(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("cant create mock: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	testRepo := NewUsersMySQLRepo(db, zap.NewNop().Sugar())

	// общие данные для теста
	usersExpected := []*User{
		{
			ID:       uint32(0),
			Username: "first",
			Email:    "first@first.net",
			Year:     2000,
			Month:    1,
			Day:      2,
		},
		{
			ID:       uint32(1),
			Username: "second",
			Email:    "second@second.net",
			Year:     1990,
			Month:    4,
			Day:      3,
		},
		{
			ID:       uint32(2),
			Username: "third",
			Email:    "third@third.net",
			Year:     2010,
			Month:    12,
			Day:      31,
		},
	}

	// нормальная работа
	rows := sqlmock.NewRows([]string{"id", "username", "email", "year", "month", "day"})
	for _, u := range usersExpected {
		rows = rows.AddRow(
			u.ID,
			u.Username,
			u.Email,
			u.Year,
			u.Month,
			u.Day,
		)
	}

	mock.
		ExpectQuery("SELECT id, username, email, year, month, day FROM users").
		WillReturnRows(rows)

	usersRecv, err := testRepo.GetAll(ctx)

	assert.NoError(t, err)
	assert.EqualValues(t, usersExpected, usersRecv)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)

	// ответ с ошибкой
	mock.
		ExpectQuery("SELECT id, username, email, year, month, day FROM users").
		WillReturnError(fmt.Errorf("db error"))

	_, err = testRepo.GetAll(ctx)

	assert.Error(t, err)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)

	// ошибка scan
	rows = sqlmock.NewRows([]string{""})
	rows = rows.AddRow("")

	mock.
		ExpectQuery("SELECT id, username, email, year, month, day FROM users").
		WillReturnRows(rows)

	_, err = testRepo.GetAll(ctx)

	assert.Error(t, err)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestGetByID(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("cant create mock: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	testRepo := NewUsersMySQLRepo(db, zap.NewNop().Sugar())

	// общие данные для теста
	userExpected := &User{
		ID:       uint32(0),
		Username: "some_user",
		Email:    "some@email.net",
		Year:     2000,
		Month:    1,
		Day:      2,
	}

	// нормальная работа
	rows := sqlmock.NewRows([]string{"id", "username", "email", "year", "month", "day"})
	rows = rows.AddRow(
		userExpected.ID,
		userExpected.Username,
		userExpected.Email,
		userExpected.Year,
		userExpected.Month,
		userExpected.Day,
	)

	mock.
		ExpectQuery("SELECT id, username, email, year, month, day FROM users WHERE").
		WithArgs(userExpected.ID).
		WillReturnRows(rows)

	userRecv, err := testRepo.GetByID(ctx, userExpected.ID)

	assert.NoError(t, err)
	assert.EqualValues(t, userExpected, userRecv)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)

	// ответ с ошибкой
	mock.
		ExpectQuery("SELECT id, username, email, year, month, day FROM users WHERE").
		WithArgs(userExpected.ID).
		WillReturnError(fmt.Errorf("db error"))

	_, err = testRepo.GetByID(ctx, userExpected.ID)

	assert.Error(t, err)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)

	// ошибка scan
	rows = sqlmock.NewRows([]string{""})
	rows = rows.AddRow("")

	mock.
		ExpectQuery("SELECT id, username, email, year, month, day FROM users WHERE").
		WithArgs(userExpected.ID).
		WillReturnRows(rows)

	_, err = testRepo.GetByID(ctx, userExpected.ID)

	assert.Error(t, err)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)

	// пользователь не найден
	rows = sqlmock.NewRows([]string{"id", "username", "password", "email", "year", "month", "day"})

	mock.
		ExpectQuery("SELECT id, username, email, year, month, day FROM users WHERE").
		WithArgs(userExpected.ID).
		WillReturnRows(rows)

	_, err = testRepo.GetByID(ctx, userExpected.ID)

	assert.ErrorIs(t, err, ErrNoUser)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}
