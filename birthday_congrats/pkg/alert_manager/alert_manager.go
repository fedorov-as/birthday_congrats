package alertmanager

type AlertManager interface {
	Send(to []string, message string)
}
