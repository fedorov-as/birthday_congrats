package handlers

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

type testCase struct {
	sendingObject      any
	respBody           string
	statucCodeSent     int
	statucCodeExpected int
}

// Тест работы на корректных и некорректных объектах для отправки
func TestSend(t *testing.T) {
	objectOK := struct {
		Name string `json:"name"`
		ID   int    `json:"id"`
	}{
		Name: "Alex",
		ID:   2,
	}
	var objectFail complex64

	cases := []testCase{
		{
			sendingObject:      objectOK,
			respBody:           `{"name":"Alex","id":2}`,
			statucCodeSent:     http.StatusOK,
			statucCodeExpected: http.StatusOK,
		},
		{
			sendingObject:      objectFail,
			respBody:           "",
			statucCodeSent:     http.StatusOK,
			statucCodeExpected: http.StatusInternalServerError,
		},
	}

	for caseNum, item := range cases {
		w := httptest.NewRecorder()

		send(w, item.statucCodeSent, item.sendingObject, zap.S())

		resp := w.Result()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Errorf("[%d] failed to read body: %v", caseNum, err)
		}

		err = resp.Body.Close()
		if err != nil {
			t.Errorf("[%d] failed to close body: %v", caseNum, err)
		}

		assert.Equal(t, item.statucCodeExpected, w.Code,
			"[%d] expected status code %d, got %d", caseNum, item.statucCodeExpected, w.Code)

		assert.Equal(t, item.respBody, string(body),
			"[%d] wrong response body\n\texpected: %s\n\tgot: %s", caseNum, item.respBody, string(body))
	}
}

// кастомный ResponseWriter, который будет возвращать ошибку
type errorResponseWriter struct {
	Code int
}

func (w errorResponseWriter) Header() http.Header {
	return http.Header{"code": []string{strconv.Itoa(w.Code)}}
}

func (w *errorResponseWriter) Write([]byte) (int, error) {
	return -1, fmt.Errorf("error")
}

func (w *errorResponseWriter) WriteHeader(statusCode int) {
	w.Code = statusCode
}

func TestIOError(t *testing.T) {
	item := testCase{
		sendingObject: struct {
			Name string `json:"name"`
			ID   int    `json:"id"`
		}{
			Name: "Alex",
			ID:   2,
		},
		statucCodeSent:     http.StatusOK,
		statucCodeExpected: http.StatusInternalServerError,
	}

	w := &errorResponseWriter{}

	send(w, item.statucCodeSent, item.sendingObject, zap.S())

	assert.Equal(t, item.statucCodeExpected, w.Code,
		"expected status code %d, got %d", item.statucCodeExpected, w.Code)
}
