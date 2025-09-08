package takoprint

import (
	"context"
	"strings"
	"time"
)

// waitReady waits up to 5 seconds for data to appear on the serial port.
func (tp *Takoprint) waitReady() {
	select {
	case <-time.After(5 * time.Second):
		tp.log.Printf("Timeout waiting for initial line, trying anyway...")
	case <-tp.serialIn:
		tp.log.Printf("Printer sent first input")

		// Lee todo el buffer serial
		for {
			select {
			case line, statusOk := <-tp.serialIn:
				if !statusOk {
					// El canal se cerró, salimos del bucle
					return
				}
				// Procesa la línea
				tp.log.Printf("Received: %s", line)
			case <-time.After(500 * time.Millisecond):
				// Timeout, salimos del bucle
				tp.log.Printf("Serial buffer cleared")
				return
			}
		}

	}
}

// readPrinter reads data from the printer.
func (tp *Takoprint) readPrinter(ctx context.Context, cancel context.CancelFunc, okChan chan bool) {
	defer cancel()
	// start := time.Now()
	// inLoop := false
	for {
		select {
		case <-ctx.Done():
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
					// signal that the printer cannot accept more data now, but we need to do something with it
					//okChan <- false
					okChan <- true
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
