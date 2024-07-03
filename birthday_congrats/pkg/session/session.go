package session

import (
	"context"
	"math/rand"

	"github.com/pkg/errors"
)

var (
	ErrSessionNotCreated = errors.New("session was not created")
	ErrNoSession         = errors.New("no session")
	ErrSessionExpired    = errors.New("session expired")
	ErrNotDestroyed      = errors.New("session was not destroyed")
)

type Session struct {
	SessID  string
	UserID  uint32
	Expires int64
}

type SessionsManager interface {
	Create(ctx context.Context, userID uint32) (Session, error)
	Check(ctx context.Context, sess Session) error
	Destroy(ctx context.Context, sess Session) error
}

func newSession(sessIDLength int, userID uint32, expires int64) Session {
	return Session{
		SessID:  RandStringRunes(sessIDLength),
		UserID:  userID,
		Expires: expires,
	}
}

var (
	letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
)

func RandStringRunes(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}
