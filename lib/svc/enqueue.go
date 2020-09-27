package svc

import (
	"fmt"

	"gitlab.com/adrian_blx/gfeeder/lib/task"
)

func (svc *Svc) enqueuePrint(path, file string) error {
	svc.Lock()
	defer svc.Unlock()

	if svc.task != nil {
		return fmt.Errorf("can not enqueue print while a task is still running")
	}

	fh, err := svc.storage.ReadFile(path, file)
	if err != nil {
		return fmt.Errorf("failed to read input file: %v", err)
	}

	s, err := svc.serialPort()
	if err != nil {
		fh.Close()
		return fmt.Errorf("failed to open serial port: %v", err)
	}

	t, err := task.New(s, fh)
	if err != nil {
		fh.Close()
		s.Close()
		return fmt.Errorf("failed to launch new task: %v", err)
	}
	svc.task = t

	go func() {
		svc.log("task enqueued: %p", t)
		<-t.WaitDone()
		fh.Close()
		s.Flush()
		fmt.Printf("++++ closing serial port\n")
		err := s.Close()
		svc.log("task done, cleaning up: %p, serial err = %v", t, err)

		svc.Lock()
		defer svc.Unlock()
		svc.task = nil
	}()
	return nil
}
