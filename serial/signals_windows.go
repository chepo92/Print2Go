// +build windows

package serial

import "io"

// En Windows, no hacemos nada con señales.
func setupSignals(w io.Writer) {
    // No-op
}