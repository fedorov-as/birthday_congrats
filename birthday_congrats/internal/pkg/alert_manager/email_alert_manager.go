package alertmanager

import (
	"strings"
	"sync"

	"github.com/emersion/go-sasl"
	"github.com/emersion/go-smtp"

	"go.uber.org/zap"
)

type EmailAlertManager struct {
	auth     sasl.Client
	smtpHost string
	smtpPort string
	from     string
	logger   *zap.SugaredLogger
}

var _ AlertManager = &EmailAlertManager{}

func NewEmailAlertManager(
	from string,
	password string,
	smtpHost string,
	smtpPort string,
	logger *zap.SugaredLogger,
) *EmailAlertManager {
	auth := sasl.NewPlainClient("", "birthday.congratulations@yandex.ru", "ucgcgejoiguychfa")

	return &EmailAlertManager{
		auth:     auth,
		smtpHost: smtpHost,
		smtpPort: smtpPort,
		from:     from,
		logger:   logger,
	}
}

func (am *EmailAlertManager) Send(to []string, subject, message string, wg *sync.WaitGroup) {
	defer wg.Done()

	// Сообщение.
	msg := strings.NewReader(
		"From: " + am.from + "\r\n" +
			"Subject: " + subject + "\r\n" +
			"\r\n" +
			message + "\r\n",
	)

	// Отправка почты.
	err := smtp.SendMail(am.smtpHost+":"+am.smtpPort, am.auth, am.from, to, msg)
	if err != nil {
		am.logger.Warnf("Error while sending emails: %v", err)
		return
	}

	am.logger.Infof("Emails were sent to: %v", to)
}
