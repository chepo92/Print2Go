package svc

import (
	"fmt"
	"time"

	"gitlab.com/adrian_blx/takoprint/lib/store"
)

var (
	minRuntime = 3 * time.Minute
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
			if time.Now().Sub(started) > minRuntime {
				// give printer some time to cool down.
				svc.log("shutting printer down in 20 sec...")
				time.Sleep(time.Second * 20)
				svc.shutdown()
			} else {
				svc.log("ignoring shutdown as we didn't run for at least %s", minRuntime)
			}
		}
	}()
	return nil
}
