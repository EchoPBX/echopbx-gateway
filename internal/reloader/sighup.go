package reloader

import (
	"os"
	"os/signal"
	"syscall"
)

func OnSIGHUP(fn func()) {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGHUP)
	go func() {
		for range ch {
			fn()
		}
	}()
}
