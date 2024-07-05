package alertmanager

import (
	"fmt"
	"net/smtp"

	"go.uber.org/zap"
)

type EmailAlertManager struct {
	auth   smtp.Auth
	logger *zap.SugaredLogger
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
		auth:   auth,
		logger: logger,
	}
}

const (
	// Информация об отправителе (в продакшене я бы закинул это в credentials на github/gitlab)
	from     = "birthday.congratulations@yandex.ru"
	password = "ucgcgejoiguychfa"

	// smtp сервер конфигурация
	smtpHost = "smtp.yandex.ru"
	smtpPort = "587"
)

func (am *EmailAlertManager) Send(to []string, message string) error {
	// Сообщение.
	msg := []byte(message)

	// Отправка почты.
	err := smtp.SendMail(smtpHost+":"+smtpPort, am.auth, from, to, msg)
	if err != nil {
		am.logger.Warnf("Email was mot sent: %v", err)
		return fmt.Errorf("error sending email: %v", err)
	}

	am.logger.Infof("Emails were sent to: %v", to)
	return nil
}
