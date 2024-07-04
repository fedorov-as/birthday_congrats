package middlware

import (
	"net/http"

	"go.uber.org/zap"
)

func Panic(logger *zap.SugaredLogger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				logger.Errorf("Panic recovered")
				http.Redirect(w, r, "/error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}
