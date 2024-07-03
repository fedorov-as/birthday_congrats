package alertmanager

type AlertManager interface {
	Send(to, message string) error
}
