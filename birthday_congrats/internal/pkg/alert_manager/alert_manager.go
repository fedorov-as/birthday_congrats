package alertmanager

import "sync"

type AlertManager interface {
	Send(to []string, subject, message string, wg *sync.WaitGroup)
}
