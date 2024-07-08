package middlware

import (
	"net/http"
	"time"

	"go.uber.org/zap"
)

func Logger(logger *zap.SugaredLogger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)

		logger.Infow("Request",
			"method", r.Method,
			"remote_address", r.RemoteAddr,
			"url", r.URL.Path,
			"duration", time.Since(start))
	})
}
