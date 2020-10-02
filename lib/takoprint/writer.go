package takoprint

import (
	"context"
	"fmt"
)

// feedPrinter writes a new line to the printer after receiving an okChan interrupt.
// calls the passed in cancel func once all data was consumed.
func (gf *Gfeeder) feedPrinter(ctx context.Context, cancel context.CancelFunc, okChan <-chan bool) {
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
			next, ok := <-gf.feedIn
			if !ok {
				// feed is empty, no more data to send!
				return
			}
			gf.sendCommand(next)
		}
	}
}

// sendCommand writes a command to the printer.
func (gf *Gfeeder) sendCommand(l string) {
	gf.Lock()
	gf.stats.numSent++
	gf.stats.lastCmd = l
	gf.Unlock()

	gf.serialOut.Write([]byte(fmt.Sprintf("%s\n", l)))
}

// injectGcode adds a command to the gcode input feed, eventually sending it to the printer.
func (gf *Gfeeder) injectGcode(cmd string) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				gf.log.Println("injectGcode failed")
			}
		}()
		gf.feedIn <- cmd
	}()
}
