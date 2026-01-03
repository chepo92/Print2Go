package webapi

import (
	"fmt"
	"time"

	"github.com/chepo92/PrintAndGo/config"
	"github.com/chepo92/PrintAndGo/store"
)

func (wapi *WebApi) enqueuePrint(instr store.Stream, shutdown bool) error {
	s, err := wapi.serialPort()
	if err != nil {
		instr.Close()
		return fmt.Errorf("Failed to open serial port: %v", err)
	}

	if err := wapi.task.Launch(s, instr); err != nil {
		instr.Close()
		s.Close()
		return fmt.Errorf("Failed to launch new task: %v", err)
	}

	//go wapi.emergencyWatchdog()

	go func() {
		started := time.Now()

		wapi.log("Task enqueued")
		<-wapi.task.WaitDone()
		instr.Close()
		if err := s.Close(); err != nil {
			panic(err)
		}
		wapi.log("Task done!")

		if shutdown {
			if time.Now().Sub(started) > config.MinRunTimeForShutdown {
				// give printer some time to cool down.
				wapi.log("shutting printer down in 20 sec...")
				time.Sleep(time.Second * 20)
				wapi.shutdown()
			} else {
				wapi.log("ignoring shutdown as we didn't run for at least %s", config.MinRunTimeForShutdown)
			}
		}
	}()
	return nil
}

// emergencyWatchdog subscribes to the printer feed and initiates a shutdown if the print appears to be stalled.
func (wapi *WebApi) emergencyWatchdog() {
	sub := wapi.task.Subscribe()
	defer wapi.task.Unsubscribe(sub)

	for {
		select {
		case <-sub:
			// still alive.
		case <-wapi.task.WaitDone():
			// print finished.
			return
		case <-time.After(config.EmergencyStallTimeout):
			wapi.log("*** EMERGENCY SHUTDOWN ***: printer froze for %s", config.EmergencyStallTimeout)
			wapi.shutdown()
			return
		}
	}
}
