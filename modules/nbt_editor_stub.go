//go:build !windows

package modules

import (
	"os"
	"os/signal"
	"syscall"
)

var resizeCh = make(chan struct{}, 1)

func startResizeWatcher() {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGWINCH)
	go func() {
		for range sigCh {
			select {
			case resizeCh <- struct{}{}:
			default:
			}
		}
	}()
}
