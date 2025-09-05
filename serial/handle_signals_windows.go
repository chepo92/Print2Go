// +build windows

package serial

import "io"

// En Windows, no hacemos nada con señales.
func handleSignals(s <-chan interface{}, w io.Writer) {
    // No-op
}