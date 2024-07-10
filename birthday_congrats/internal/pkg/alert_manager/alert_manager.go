package alertmanager

type AlertManager interface {
	Send(to []string, subject, message string)
}
