package printandgo

import (
	"context"
	"strings"
	"time"
)

// feedPrinter writes a new line to the printer after receiving an okChan interrupt.
// calls the passed in cancel func once all data was consumed.
func (tp *PrintAndGo) feedPrinter(ctx context.Context, cancel context.CancelFunc, okChan <-chan bool, succeedStreamChan chan bool, writeErrorChan chan bool) {
	defer cancel()

	lastLog := time.Now()

	for {
		if time.Since(lastLog) > 2*time.Second {
			tp.log.Printf(
				"[HEARTBEAT] writer alive | okChan=%d prioQ=%d",
				len(okChan),
				len(tp.priorityQueue),
			)
			lastLog = time.Now()
		}
		select {
		case <-ctx.Done():
			tp.log.Printf("Writer context cancelled, returning")
			return
		case okValue, okChanStatus := <-okChan:
			// inLoop = false
			if !okChanStatus {
				// ok channel is closed, return
				tp.log.Printf("ok channel is closed, returning")
				// call the error channel
				writeErrorChan <- true
				return
			} // else, we can send more data.
			if !okValue {
				// printer sent something other than "ok", do not send more data now.
				tp.log.Printf("Printer busy, waiting and skipping...")
				//time.Sleep(1000 * time.Millisecond) // pausa de 1000ms
				continue
			} else {
				tp.log.Printf("Ok received in channel, reading next line to be sent")
				var cmd OutCmd

				select {
				case cmd = <-tp.priorityQueue:
					tp.log.Printf("[PRIO] Sending injected cmd: %s", cmd.Cmd)
					tp.sendCommand(cmd.Cmd)

				default:
					// if no priority, continue normally
					//cmd = <-tp.outQueue
					// get next line from feedIn channel
					nextCmd, ok := <-tp.feedIn
					// tp.outQueue <- OutCmd{Cmd: line}
					// tp.log.Printf(
					// 	"[DEBUG] feedIn → outQueue | %q | outQueue=%d/%d",
					// 	line, len(tp.outQueue), cap(tp.outQueue),
					// )
					if !ok {
						// feed is empty, no more data to send!
						tp.log.Printf("Feed is empty, no more data to send!")
						// signal that the stream succeeded
						succeedStreamChan <- true

						return
					}
					// send next line to printer
					tp.log.Printf("Sending job gcode: %s", nextCmd)
					err := tp.sendJob(nextCmd)
					if err != nil {
						tp.log.Printf("Error sending command to printer: %v", err)
						writeErrorChan <- true
					}
				} // select priority/default
				// cmd := <-tp.outQueue // BLOQUEA el turno
				// if cmd.Injected {
				// 	tp.log.Printf("Injecting external command: %q", cmd.Cmd)
				// 	tp.sendCommand(cmd.Cmd)
				// } else {
				// 	tp.log.Printf("Sending next line to printer: %q", cmd.Cmd)
				// 	err := tp.sendJob(cmd.Cmd)
				// 	//sendJob(next)
				// 	if err != nil {
				// 		tp.log.Printf("Error sending command to printer: %v", err)
				// 		writeErrorChan <- true
				// 	}
				// }
				// first check if there's any injection pending
				// select {
				// case cmd := <-tp.injectChan:
				// 	tp.log.Printf("Injecting external command: %q", cmd)
				// 	tp.sendCommand(cmd)
				// 	continue // IMPORTANT: continue and wait for next ok signal
				// default:
				// 	// no injection pending
				// }

				// get next line from feedIn channel
				// next, ok := <-tp.feedIn
				// //
				// if !ok {
				// 	// feed is empty, no more data to send!
				// 	tp.log.Printf("Feed is closed, no more data to send!")
				// 	succeedStreamChan <- true
				// 	// close the feedIn channel to avoid further writes
				// 	//close(tp.feedIn)
				// 	// should close the ok channel too?
				// 	// close(okChan)
				// 	return
				// }
				// send next line to printer
				// tp.log.Printf("Sending next line to printer: %q", next)
				// err := tp.sendJob(next)
				// //sendJob(next)
				// if err != nil {
				// 	tp.log.Printf("Error sending command to printer: %v", err)
				// 	writeErrorChan <- true
				// }

			} // else: ok
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

		} // select Done

		// // Agregar una condición para que el programa termine después de un cierto período de tiempo in case is in a loop
		// if inLoop && (time.Since(start) > 20*time.Second) {
		// 	tp.log.Println("Se alcanzó el tiempo máximo de ejecución para writer")
		// 	return
		// }
	} // for
}

// sendCommand writes a command to the printer.
func (tp *PrintAndGo) sendCommand(l string) (err error) {
	tp.Lock()
	tp.stats.lastCmd = l
	tp.Unlock()
	command := strings.Split(l, ";") // remove Gcode comments if any
	tp.log.Println("Sending Cmd:", command[0])
	_, err = tp.serialOut.Write(append([]byte(l), '\r', '\n'))
	if err != nil {
		tp.log.Printf("Error writing to serial port: %v", err)
	}
	return err
}

// sendJob writes a command to the printer and adds to the stats.
func (tp *PrintAndGo) sendJob(l string) (err error) {
	tp.Lock()
	tp.stats.numSent++
	tp.stats.lastCmd = l
	tp.Unlock()
	command := strings.Split(l, ";") // remove Gcode comments if any
	tp.log.Println("serialOut Job Gcode:", command[0])
	_, err = tp.serialOut.Write(append([]byte(l), '\r', '\n'))
	if err != nil {
		tp.log.Printf("Error writing to serial port: %v", err)
	}
	return err
}

// injectGcode adds a command to the gcode input feed, eventually sending it to the printer.
// func (tp *PrintAndGo) injectGcode(cmd string) {
// 	go func() {
// 		defer func() {
// 			if r := recover(); r != nil {
// 				tp.log.Println("injectGcode failed")
// 			}
// 		}()
// 		tp.feedIn <- cmd
// 		fmt.Printf("Writer: cmd added to feedIn: %s\n", cmd)
// 	}()
// }

// InjectGcode exports a safe injector to other packages.
// func (tp *PrintAndGo) InjectGcode(cmd string) {
// 	fmt.Printf("Writer: InjectGcode: %s\n", cmd)
// 	tp.injectGcode(cmd)
// }

// func (tp *PrintAndGo) InjectGcode(cmd string) error {
// 	fmt.Printf("Writer: cmd added to injectChan: %s\n", cmd)
// 	tp.injectChan <- cmd
// 	return nil
// }
// func (tp *PrintAndGo) InjectGcode(cmd string) {
//     outQueue <- OutCmd{Cmd: cmd, Injected: true}
// }
