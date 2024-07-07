package alertmanager

import "sync"

type AlertManager interface {
	Send(to []string, message string, wg *sync.WaitGroup)
}
