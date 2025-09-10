//go:build !windows
// +build !windows

package serial

import (
	"fmt"
	"io"
	"os"
	"syscall"
)

// Escape hatch for stalled prints: signals can be used to send a command to the printer or pretend that we received an OK.
func handleSignals(s <-chan os.Signal, w io.Writer) {
	for sig := range s {
		switch sig {
		case syscall.SIGUSR1:
			// send 'ok' back to PrintAndGo.
			fmt.Fprintf(os.Stdout, "ok\r\n")
		default:
			// unhandled.
		}
	}
}
