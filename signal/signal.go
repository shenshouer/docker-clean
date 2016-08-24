package signal

import (
	"os"
	"os/signal"
	"syscall"
)

// CloseFunc before close will do CloseFunc
type CloseFunc func()

// HandleSignal handler close signal
func HandleSignal(closeF CloseFunc) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)

	for sig := range c {
		switch sig {
		case syscall.SIGINT, syscall.SIGTERM:
			closeF()
			return
		}
	}
}
