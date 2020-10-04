package svc

import (
	"fmt"

	"gitlab.com/adrian_blx/takoprint/lib/store"
)

func (svc *Svc) enqueuePrint(instr store.Stream) error {
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
		svc.log("task enqueued")
		<-svc.task.WaitDone()
		instr.Close()
		if err := s.Close(); err != nil {
			panic(err)
		}
		svc.log("task done!")
	}()
	return nil
}
