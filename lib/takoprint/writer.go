package takoprint

import (
	"context"
	"fmt"
)

// feedPrinter writes a new line to the printer after receiving an okChan interrupt.
// calls the passed in cancel func once all data was consumed.
func (tp *Takoprint) feedPrinter(ctx context.Context, cancel context.CancelFunc, okChan <-chan bool) {
	defer cancel()
	for {
		select {
		case <-ctx.Done():
			return
		case _, ok := <-okChan:
			if !ok {
				// ok chan closed.
				return
			}
			next, ok := <-tp.feedIn
			if !ok {
				// feed is empty, no more data to send!
				return
			}
			tp.sendCommand(next)
		}
	}
}

// sendCommand writes a command to the printer.
func (tp *Takoprint) sendCommand(l string) {
	tp.Lock()
	tp.stats.numSent++
	tp.stats.lastCmd = l
	tp.Unlock()

	tp.serialOut.Write([]byte(fmt.Sprintf("%s\n", l)))
}

// injectGcode adds a command to the gcode input feed, eventually sending it to the printer.
func (tp *Takoprint) injectGcode(cmd string) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				tp.log.Println("injectGcode failed")
			}
		}()
		tp.feedIn <- cmd
	}()
}
