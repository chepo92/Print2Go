package svc

import (
	"fmt"
	"time"

	"git.sr.ht/~adrian-blx/takoprint/lib/config"
	"git.sr.ht/~adrian-blx/takoprint/lib/store"
)

func (svc *Svc) enqueuePrint(instr store.Stream, shutdown bool) error {
	s, err := svc.serialPort()
	if err != nil {
		instr.Close()
		return fmt.Errorf("failed to open serial port: %v", err)
	}

	if err := svc.task.Launch(s, instr); err != nil {
		instr.Close()
		s.Close()
		return fmt.Errorf("failed to launch new task: %v", err)
	}

	go svc.emergencyWatchdog()

	go func() {
		started := time.Now()

		svc.log("task enqueued")
		<-svc.task.WaitDone()
		instr.Close()
		if err := s.Close(); err != nil {
			panic(err)
		}
		svc.log("task done!")

		if shutdown {
			if time.Now().Sub(started) > config.MinRunTimeForShutdown {
				// give printer some time to cool down.
				svc.log("shutting printer down in 20 sec...")
				time.Sleep(time.Second * 20)
				svc.shutdown()
			} else {
				svc.log("ignoring shutdown as we didn't run for at least %s", config.MinRunTimeForShutdown)
			}
		}
	}()
	return nil
}

// emergencyWatchdog subscribes to the printer feed and initiates a shutdown if the print appears to be stalled.
func (svc *Svc) emergencyWatchdog() {
	sub := svc.task.Subscribe()
	defer svc.task.Unsubscribe(sub)

	for {
		select {
		case <-sub:
			// still alive.
		case <-svc.task.WaitDone():
			// print finished.
			return
		case <-time.After(config.EmergencyStallTimeout):
			svc.log("*** EMERGENCY SHUTDOWN ***: printer forze for %s", config.EmergencyStallTimeout)
			svc.shutdown()
			return
		}
	}
}
