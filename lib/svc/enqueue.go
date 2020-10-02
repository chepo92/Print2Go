package svc

import (
	"fmt"
)

func (svc *Svc) enqueuePrint(path, file string) error {
	fh, err := svc.storage.ReadFile(path, file)
	if err != nil {
		return fmt.Errorf("failed to read input file: %v", err)
	}

	s, err := svc.serialPort()
	if err != nil {
		fh.Close()
		return fmt.Errorf("failed to open serial port: %v", err)
	}

	if err := svc.task.Launch(s, fh); err != nil {
		fh.Close()
		s.Close()
		return fmt.Errorf("failed to launch new task: %v", err)
	}

	go func() {
		svc.log("task enqueued")
		<-svc.task.WaitDone()
		fh.Close()
		if err := s.Close(); err != nil {
			panic(err)
		}
		svc.log("task done!")
	}()
	return nil
}
