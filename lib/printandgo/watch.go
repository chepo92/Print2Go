package printandgo

import (
	"context"
	"strings"
	"time"
)

// waitReady waits up to 5 seconds for data to appear on the serial port.
func (tp *PrintAndGo) waitReady(okChan chan bool) {
	select {
	case <-time.After(5 * time.Second):
		tp.log.Printf("Timeout waiting for initial line, forcing start...")
		// desbloquear writer aunque no haya ok
		okChan <- true
	case line, statusOk := <-tp.serialIn:
		if !statusOk {
			// serial console closed
			tp.log.Printf("Serial closed before first input")
			return
		}
		tp.log.Printf("Printer sent first input")
		tp.log.Printf("Printer says: %q", line)
		// limpiar el buffer durante 10s, renovables si es que se recibe algo
		idle := time.NewTimer(10 * time.Second)
		for {
			select {
			case line, statusOk := <-tp.serialIn:
				if !statusOk {
					// serial console closed
					tp.log.Printf("Serial port not available")
					return
				}

				idle.Reset(10 * time.Second)
				tp.log.Printf("Printer says: %q", line)
			case <-idle.C:
				tp.log.Printf("Finished draining startup messages")
				// desbloquear writer al terminar la limpieza
				okChan <- true
				return
			}
		}
	}
}

// readPrinter reads data from the printer.
func (tp *PrintAndGo) readPrinter(ctx context.Context, cancel context.CancelFunc, okChan chan bool) {
	defer cancel()
	// start := time.Now()
	// inLoop := false
	for {
		select {
		case <-ctx.Done():
			// context finished
			return
		case line, statusOk := <-tp.serialIn:
			// inLoop := false
			if !statusOk {
				// serial console closed
				return
			}
			if line == "ok" {
				// signal that the printer can accept more data
				okChan <- true
			} else { // else the message was not "ok", but something else

				parts := strings.Split(line, " ")
				if parts[0] == "ok" {
					okChan <- true
					tp.log.Printf("Printer says ok and more data: %q", line)
				} else {
					// Don't unlock the writer, log printer message only
					tp.log.Printf("Printer says: %q", line)
				}

			}
			tp.fireCallback(line)

			// default:
			// 	if !inLoop {
			// 		inLoop = true
			// 		tp.log.Println("No se recibió ninguna señal en watch, iniciando timeout")
			// 		start = time.Now()
			// 	} else {
			// 		tp.log.Println("No se recibió ninguna señal en watch, esperando todavia, timeout activado")
			// 		time.Sleep(1000 * time.Millisecond) // pausa de 1000ms
			// 		// ...
			// 	}

		}

		// // Agregar una condición para que el programa en caso de estar en un loop termine después de un cierto período de tiempo
		// if inLoop && (time.Since(start) > 20*time.Second) {
		// 	tp.log.Println("Se alcanzó el tiempo máximo de ejecución para watch")
		// 	return
		// }
	}
}
