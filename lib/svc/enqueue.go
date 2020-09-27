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
		fmt.Printf("++++ closing serial port\n")
		// triggers a send before close, unblocking the FH.
		s.Write([]byte("M117 done\n"))
		if err := s.Close(); err != nil {
			panic(err)
		}
		svc.log("task done, cleaning up: %p", t)

		svc.Lock()
		defer svc.Unlock()
		svc.task = nil
	}()
	return nil
}
