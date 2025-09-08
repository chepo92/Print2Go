package takoprint

import (
	"context"
	"strings"
	"time"
)

// feedPrinter writes a new line to the printer after receiving an okChan interrupt.
// calls the passed in cancel func once all data was consumed.
func (tp *Takoprint) feedPrinter(ctx context.Context, cancel context.CancelFunc, okChan <-chan bool) {
	defer cancel()
	// start := time.Now()
	// inLoop := false
	for {
		select {
		case <-ctx.Done():
			return
		case okValue, okChanStatus := <-okChan:
			// inLoop = false
			if !okChanStatus {
				// ok channel is closed, return
				tp.log.Printf("ok channel is closed, returning")
				return
			} // else, we can send more data.
			if !okValue {
				// printer sent something other than "ok", do not send more data now.
				tp.log.Printf("Printer busy, waiting and skipping...")
				time.Sleep(1000 * time.Millisecond) // pausa de 1000ms

			} else {
				tp.log.Printf("Ok received in channel, reading next line to be sent")
				// get next line from feedIn channel
				next, ok := <-tp.feedIn
				if !ok {
					// feed is empty, no more data to send!
					tp.log.Printf("Feed is empty, no more data to send!")
					return
				}
				// send next line to printer
				tp.log.Printf("Sending next line to printer: %q", next)
				tp.sendCommand(next)
			}
			// default:
			// 	if !inLoop {
			// 		inLoop = true
			// 		tp.log.Println("No se recibió ninguna señal en writer, iniciando timeout")
			// 		start = time.Now()
			// 	} else {
			// 		tp.log.Println("No se recibió ninguna señal en writer, esperando todavia, timeout activado")
			// 		time.Sleep(1000 * time.Millisecond) // pausa de 1000ms
			// 		// ...
			// 	}

		}

		// // Agregar una condición para que el programa termine después de un cierto período de tiempo in case is in a loop
		// if inLoop && (time.Since(start) > 20*time.Second) {
		// 	tp.log.Println("Se alcanzó el tiempo máximo de ejecución para writer")
		// 	return
		// }
	}
}

// sendCommand writes a command to the printer.
func (tp *Takoprint) sendCommand(l string) {
	tp.Lock()
	tp.stats.numSent++
	tp.stats.lastCmd = l
	tp.Unlock()
	command := strings.Split(l, ";") // remove Gcode comments if any
	tp.log.Println("Sending: ", command[0])
	tp.serialOut.Write(append([]byte(l), '\r', '\n'))
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
