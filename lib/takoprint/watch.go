package takoprint

import (
	"context"
	"time"
)

// waitReady waits up to 5 seconds for data to appear on the serial port.
func (tp *Takoprint) waitReady() {
	select {
	case <-time.After(5 * time.Second):
		tp.log.Printf("timeout waiting for initial line, trying anyway...")
	case <-tp.serialIn:
		tp.log.Printf("printer sent first input")
	}
}

// readPrinter reads data from the printer.
func (tp *Takoprint) readPrinter(ctx context.Context, cancel context.CancelFunc, okChan chan bool) {
	defer cancel()
	for {
		select {
		case <-ctx.Done():
			return
		case line, ok := <-tp.serialIn:
			if !ok {
				// serial console closed.
				return
			}
			if line == "ok" {
				// signal that the printer can accept more data.
				okChan <- true
			}
			tp.fireCallback(line)
		}
	}
}
