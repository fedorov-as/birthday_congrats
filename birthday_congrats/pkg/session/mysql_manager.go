package session

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
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

func (sm *MySQLSessionsManager) Create(ctx context.Context, userID uint32) (*Session, error) {
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
		return nil, fmt.Errorf("db error: %v", err)
	}

	// проверка, что запись добавлена
	affected, err := result.RowsAffected()
	if err != nil {
		sm.logger.Errorf("Error in RowsAffected(): %v", err)
		return nil, fmt.Errorf("db error: %v", err)
	}
	if affected == 0 {
		sm.logger.Errorf("Subscription was not added")
		return nil, ErrSessionNotCreated
	}

	return &newSession, nil
}

func (sm *MySQLSessionsManager) Check(r *http.Request) (*Session, error) {
	cookie, err := r.Cookie("session_id")
	if err != nil {
		sm.logger.Warnf("No session cookie found")
		return nil, ErrNoSession
	}

	sessID := cookie.String()

	// проверка, что сессия существует
	sess := &Session{
		SessID: sessID,
	}
	err = sm.db.QueryRowContext(
		r.Context(),
		"SELECT user_id, expires FROM sessions WHERE sess_id = ?",
		sessID,
	).Scan(
		&sess.UserID,
		&sess.Expires,
	)
	if err != nil && err != sql.ErrNoRows {
		sm.logger.Errorf("Error while SELECT from db: %v", err)
		return nil, fmt.Errorf("db error: %v", err)
	}
	if err == sql.ErrNoRows {
		return nil, ErrNoSession
	}

	// проверка, что сессия не истекла
	if sess.Expires < time.Now().Unix() {
		err := sm.Destroy(r.Context())
		if err != nil {
			sm.logger.Errorf("Error while destroying session: %v", err)
			return nil, fmt.Errorf("destroy session error: %v", err)
		}

		return nil, ErrSessionExpired
	}

	return sess, nil
}

func (sm *MySQLSessionsManager) Destroy(ctx context.Context) error {
	sess, err := SessionFromContext(ctx)
	if err != nil {
		sm.logger.Errorf("Error getting session from context: %v", err)
		return err
	}

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
