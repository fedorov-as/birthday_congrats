package handlers

import (
	"encoding/json"
	"io"
	"net/http"

	"go.uber.org/zap"
)

type Message struct {
	Message string `json:"message"`
}

func send(w http.ResponseWriter, statusCode int, resp interface{}, logger *zap.SugaredLogger) {
	respStr, err := json.Marshal(resp)
	if err != nil {
		logger.Errorf("json.Marshal error",
			"error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_, err = io.WriteString(w, string(respStr))
	if err != nil {
		logger.Errorf("io.WriteString error",
			"error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}
