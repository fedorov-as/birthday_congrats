package middlware

import (
	"birthday_congrats/internal/pkg/session"
	"net/http"

	"go.uber.org/zap"
)

func Auth(sm session.SessionsManager, logger *zap.SugaredLogger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sess, err := sm.Check(r)
		if err != nil {
			logger.Warnf("auth error: %v", err)
			http.Redirect(w, r, "/error", http.StatusFound)
			return
		}

		ctx := session.ContextWithSession(r.Context(), sess)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
