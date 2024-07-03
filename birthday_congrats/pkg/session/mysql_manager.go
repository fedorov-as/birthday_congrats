package session

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"go.uber.org/zap"
)

type MySQLSessionsManager struct {
	db     *sql.DB
	logger *zap.SugaredLogger

	expiresTime  int64
	sessIDLength int
}

var _ SessionsManager = &MySQLSessionsManager{}

func NewMySQLSessionsManager(db *sql.DB, logger *zap.SugaredLogger, expiresTime int64, sessIDLength int) *MySQLSessionsManager {
	return &MySQLSessionsManager{
		db:           db,
		logger:       logger,
		expiresTime:  expiresTime,
		sessIDLength: sessIDLength,
	}
}

func (sm *MySQLSessionsManager) Create(ctx context.Context, userID uint32) (Session, error) {
	newSession := newSession(sm.sessIDLength, userID, time.Now().Unix()+sm.expiresTime)

	result, err := sm.db.ExecContext(
		ctx,
		"INSERT INTO sessions (`sess_id`, `user_id`, `expires`) VALUES (?, ?, ?)",
		newSession.SessID,
		newSession.UserID,
		newSession.Expires,
	)
	if err != nil {
		sm.logger.Errorf("Error while INSERT into db: %v", err)
		return Session{}, fmt.Errorf("db error: %v", err)
	}

	// проверка, что запись добавлена
	affected, err := result.RowsAffected()
	if err != nil {
		sm.logger.Errorf("Error in RowsAffected(): %v", err)
		return Session{}, fmt.Errorf("db error: %v", err)
	}
	if affected == 0 {
		sm.logger.Errorf("Subscription was not added")
		return Session{}, ErrSessionNotCreated
	}

	return newSession, nil
}

func (sm *MySQLSessionsManager) Check(ctx context.Context, sess Session) error {
	// проверка, что сессия не истекла
	if sess.Expires < time.Now().Unix() {
		err := sm.Destroy(ctx, sess)
		if err != nil {
			sm.logger.Errorf("Error while destroying session: %v", err)
			return fmt.Errorf("destroy session error: %v", err)
		}

		return ErrSessionExpired
	}

	// проверка, что сессия существует
	err := sm.db.QueryRowContext(
		ctx,
		"SELECT FROM sessions WHERE sess_id = ? AND user_id = ? AND expires = ?",
		sess.SessID,
		sess.UserID,
		sess.Expires,
	).Scan()
	if err != nil && err != sql.ErrNoRows {
		sm.logger.Errorf("Error while SELECT from db: %v", err)
		return fmt.Errorf("db error: %v", err)
	}
	if err == sql.ErrNoRows {
		return ErrNoSession
	}

	return nil
}

func (sm *MySQLSessionsManager) Destroy(ctx context.Context, sess Session) error {
	result, err := sm.db.ExecContext(
		ctx,
		"DELETE FROM sessions WHERE sess_id = ?",
		sess.SessID,
	)
	if err != nil {
		sm.logger.Errorf("Error while DELETE from db: %v", err)
		return fmt.Errorf("db error: %v", err)
	}

	// проверка, что запись удалена
	affected, err := result.RowsAffected()
	if err != nil {
		sm.logger.Errorf("Error in RowsAffected(): %v", err)
		return fmt.Errorf("db error: %v", err)
	}
	if affected == 0 {
		sm.logger.Errorf("Subscription was not added")
		return ErrNotDestroyed
	}

	return nil
}
