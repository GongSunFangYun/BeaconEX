//go:build windows

package modules

import (
	"os"
	"time"

	"golang.org/x/term"
)

var resizeCh = make(chan struct{}, 1)

func startResizeWatcher() {
	go func() {
		w, h, _ := term.GetSize(int(os.Stdout.Fd()))
		for {
			time.Sleep(100 * time.Millisecond)
			nw, nh, err := term.GetSize(int(os.Stdout.Fd()))
			if err != nil {
				continue
			}
			if nw != w || nh != h {
				w, h = nw, nh
				select {
				case resizeCh <- struct{}{}:
				default:
				}
			}
		}
	}()
}
