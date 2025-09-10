//go:build !windows
// +build !windows

package serial

import (
	"io"
	"os"
	"os/signal"
	"syscall"
)

// Solo en Unix: inicializa señales y maneja SIGUSR1,
// Note that PrintAndGo has an escape hatch for stalled prints: Sending a USR1 signal to PrintAndGo's TTY subprocess should resume the print in most cases, see readme for how to use it.
func setupSignals(w io.Writer) {
	sigs := make(chan os.Signal)
	signal.Notify(sigs, syscall.SIGUSR1)
	go handleSignals(sigs, w)
}

// func handleSignals(sigs <-chan os.Signal, w io.Writer) {
//     for sig := range sigs {
//         if sig == syscall.SIGUSR1 {
//             fmt.Fprintf(w, "ok\r\n")
//         }
//     }
// }
