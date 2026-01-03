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
				"[HEARTBEAT] writer alive | okChan=%d prioQ=%d, feedIn=%d ",
				len(okChan),
				len(tp.priorityQueue),
				len(tp.feedIn),
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
					// get next line from feedIn channel
					nextCmd, ok := <-tp.feedIn

					if !ok {
						// feed is empty, no more data to send!
						tp.log.Printf("Feed is empty, no more data to send!")
						// signal that the stream succeeded
						succeedStreamChan <- true

					nextCmd := "G1 X94.197 Y110.368 E5.934947" // TEMPORARY HARDCODED COMMAND FOR TESTING
					// send next line to printer
					tp.log.Printf("Sending job gcode: %s", nextCmd)
					err := tp.sendJob(nextCmd)
					if err != nil {
						tp.log.Printf("Error sending command to printer: %v", err)
						writeErrorChan <- true
					}
				} // select priority/default

			} // else: ok

		} // select Done

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
