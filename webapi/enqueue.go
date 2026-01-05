package webapi

import (
	"fmt"
	"time"

	"github.com/chepo92/PrintAndGo/config"
	"github.com/chepo92/PrintAndGo/store"
)

func (wapi *WebApi) enqueuePrint(instr store.Stream, shutdown bool) error {
	fmt.Printf("In Enqueue print,\n")

	fmt.Printf("Get serial \n")

	//_, err := wapi.getSerial()
	fmt.Printf("Open serial port w err''\n")
	// port, err := wapi.serial.Port()
	// if err != nil {
	// 	instr.Close()
	// 	return fmt.Errorf("serial not connected")
	// }
	// if err != nil {
	// 	instr.Close()
	// 	return fmt.Errorf("Failed to open serial port: %v", err)
	// }

	fmt.Printf("Launch task '' \n")
	// if err := wapi.task.Launch(s, instr); err != nil {
	// 	instr.Close()
	// 	s.Close()
	// 	return fmt.Errorf("Failed to launch new task: %v", err)
	// }
	// if err := wapi.task.Launch(port, instr); err != nil {
	// 	instr.Close()
	// 	return err
	// }

	//go wapi.emergencyWatchdog()

	// go func() {
	// 	started := time.Now()

	// 	wapi.log("Task enqueued")
	// 	fmt.Printf("Waiting for task done\n")
	// 	<-wapi.task.WaitDone()
	// 	instr.Close()
	// 	if err := s.Close(); err != nil {
	// 		panic(err)
	// 	}
	// 	wapi.log("Task done!")

	// 	if shutdown {
	// 		if time.Now().Sub(started) > config.MinRunTimeForShutdown {
	// 			// give printer some time to cool down.
	// 			wapi.log("shutting printer down in 20 sec...")
	// 			time.Sleep(time.Second * 20)
	// 			wapi.shutdown()
	// 		} else {
	// 			wapi.log("ignoring shutdown as we didn't run for at least %s", config.MinRunTimeForShutdown)
	// 		}
	// 	}
	// }()
	fmt.Printf("Returning '' \n")
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
