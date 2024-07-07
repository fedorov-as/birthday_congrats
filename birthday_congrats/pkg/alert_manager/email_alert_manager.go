package alertmanager

import (
	"net/smtp"

	"go.uber.org/zap"
)

type EmailAlertManager struct {
	auth     smtp.Auth
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
	auth := smtp.PlainAuth("", from, password, smtpHost)

	return &EmailAlertManager{
		auth:     auth,
		smtpHost: smtpHost,
		smtpPort: smtpPort,
		from:     from,
		logger:   logger,
	}
}

func (am *EmailAlertManager) Send(to []string, message string) {
	// defer wg.Done()

	// Сообщение.
	msg := []byte(message)

	// Отправка почты.
	err := smtp.SendMail(am.smtpHost+":"+am.smtpPort, am.auth, am.from, to, msg)
	if err != nil {
		am.logger.Warnf("Error while sending emails: %v", err)
		return
	}

	am.logger.Infof("Emails were sent to: %v", to)
}
