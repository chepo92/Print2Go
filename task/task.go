package task

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/chepo92/PrintAndGo/lib/printandgo"
	"github.com/chepo92/PrintAndGo/store"
)

type TaskStatus struct {
	// Filename we are currently executing.
	File string
	// Human readable description.
	Text string
	// Percentage printed
	DonePercent float64
	// Whether or not we are actually doing anything.
	Active bool
	// Time we started this print
	Started time.Time
	// Last command we sent.
	LastCommand string
	// Last reply we received.
	LastReply string
	// Error flag
	Error bool
	// Error message
	ErrorMsg string
	// Cancellation flag
	Cancelled bool
}

// Task struct manages the print process, holds the PrintAndGo instance, including its status, context, and subscribers for status updates.
type Task struct {
	sync.RWMutex
	// Internal print context.
	ctx context.Context
	// Function to cancel the print context.
	cancel context.CancelFunc
	// PrintAndGo instance struct reference.
	pagInstance *printandgo.PrintAndGo
	// Gcode input.
	gcodeStream store.Stream
	// Status of the currently running print.
	status TaskStatus
	// clients subscribing to updates
	subscribers map[string]chan TaskStatus
}

// New creates a new instance of the Task struct. It initializes the subscribers field as an empty map, where the keys are strings and the values are channels of type TaskStatus
func New() *Task {
	return &Task{
		subscribers: make(map[string]chan TaskStatus),
	}
}

// Launch is a function of the Task struct that starts a new print. Accepts an io.ReadWriteCloser for the serial port and a store.Stream for the gcode input.
// Creates a new PrintAndGo instance, sets up the context and status,
// starts the print process and a callback to update the status in a separate goroutine
func (task *Task) Launch(port io.ReadWriteCloser, gcs store.Stream) error {
	task.Lock()
	defer task.Unlock()

	if task.gcodeStream != nil {
		return fmt.Errorf("Task already running")
	}

	ctx, cancel := context.WithCancel(context.Background())
	task.pagInstance = printandgo.New(port, gcs)
	task.ctx = ctx
	task.cancel = cancel
	task.gcodeStream = gcs
	task.status.DonePercent = 0
	task.status.Active = true
	task.status.Error = false
	task.status.ErrorMsg = ""
	task.status.Text = "Starting..."
	task.status.Started = time.Now()

	fmt.Println("Gcode: ", gcs.Name())
	fmt.Println("Size: ", gcs.Size())
	fmt.Println("Lines: ", gcs.LineCount())

	// This line of code sets up a callback function for the PrintAndGo instance (task.pagInstance). The callback function is task.callback, which will be called by PrintAndGo to update the task status.
	// allowing PrintAndGo to notify the Task  instance of status updates.
	// horray for circular dependencies!
	// printandgo publish through the callback function, Task subscribes to those updates and broadcasts them to its own subscribers.
	// This line effectively links the PrintAndGo instance with the Task instance, enabling real-time status updates during the printing process.

	printandgo.Callback(task.callback)(task.pagInstance)

	// semi-empty callback to anounce subscribers that we are now active.
	go task.callback(&printandgo.CallbackData{})
	go task.start()
	return nil
}

// Returns true if the task is done.
func (task *Task) Done() bool {
	task.RLock()
	defer task.RUnlock()
	return task.ctx == nil || task.ctx.Err() != nil
}

// Returns true if the task is active.
func (task *Task) IsActive() bool {
	task.RLock()
	defer task.RUnlock()
	return task.ctx != nil && task.ctx.Err() == nil && task.status.Active
}

func (task *Task) WaitDone() <-chan struct{} {
	task.RLock()
	defer task.RUnlock()

	if task.ctx == nil {
		return nil
	}
	return task.ctx.Done()
}

// Cancel is a wrapper for the cancel function. It calls the context cancel function, aborting the task.
func (task *Task) Cancel() {
	task.RLock()
	defer task.RUnlock()
	if task.cancel == nil {
		return
	}
	task.status.Cancelled = true
	task.cancel()
}

// Cancel is a wrapper for the cancel function. It calls the context cancel function, aborting the task.
func (task *Task) NormalExit() {
	task.RLock()
	defer task.RUnlock()
	if task.cancel == nil {
		return
	}
	task.cancel()
}

// start is a wrapper for the PrintAndGo.Start function. It internally launches the task and sets 'done' once the print finished.
func (task *Task) start() {
	// launch the print
	err := task.pagInstance.Start(task.ctx)
	if err != nil {
		fmt.Println("Error running PrintAndGo:", err)
		task.Cancel()
		task.callback(nil)
		task.nullify()
	} else {
		task.NormalExit()
		task.callback(nil)
		task.nullify()
	}

	// cancel our contex and fire an empty callback

}

// nullify clears most of the struct, allowing for a new task to be started.
func (task *Task) nullify() {
	task.Lock()
	defer task.Unlock()

	// nulls *most* of the struct.
	task.pagInstance = nil
	task.ctx = nil
	task.cancel = nil
	task.gcodeStream = nil
}

// InjectGcode injects a single gcode command into a running print. Returns an error if no task is active.
func (task *Task) InjectGcode(cmd string) error {
	task.RLock()
	defer task.RUnlock()
	if task.pagInstance == nil {
		return fmt.Errorf("no active print")
	}
	fmt.Println("Task: Injecting gcode:", cmd)
	task.pagInstance.InjectGcode(cmd)
	return nil
}

// Function to update the status of the task, which is then broadcasted to subscribers
func (task *Task) callback(cbd *printandgo.CallbackData) {
	var txt string

	task.Lock()
	// Broadcast/notify subscribers (must happen after unlock popped from stack)
	// defer task.broadcast() // defer keyword is used to schedule a function call to be executed when the surrounding function returns
	defer task.Unlock()

	defer func() {
		if txt != task.status.Text {
			//fmt.Println("Old Status:", task.status.Text)
			//fmt.Println("New status:", txt)
			//if task.status.Active { // only update text if the status is active
			fmt.Println("Status:", txt)
			task.status.Text = txt
			//	fmt.Println("Updated status:", task.status.Text)
			//}
			// task.tp.Echo(txt) // disable echoing status to printer, it's noisy

		}
	}()

	if cbd == nil && (task.status.Cancelled || task.status.DonePercent < 100) && task.status.Active {
		str := fmt.Sprintf("Cancel triggers: %t %f %t", task.status.Cancelled, task.status.DonePercent, task.status.Active)
		fmt.Println(str)

		if task.pagInstance != nil && task.pagInstance.Err() {
			task.status.Active = false
			txt = fmt.Sprintf("Terminated due to error: %s", task.pagInstance.ErrMsg())
			return
		}
		task.status.Active = false
		txt = "Cancelled"
		return
	}

	if cbd == nil && task.status.DonePercent >= 100 && task.status.Active {
		task.status.Active = false
		txt = "Done!"
		return
	}

	if cbd != nil && cbd.Error && task.status.Active {
		task.status.Active = false
		txt = "Hubo un Error"
		if cbd.Message != "" {
			txt = cbd.Message
		}
		return
	}

	// if cbd is not nil, update status based on callback data
	if cbd != nil {
		// Keep a couple of seen replies.
		task.status.LastCommand = cbd.LastSent
		task.status.LastReply = cbd.Reply

		lines := task.gcodeStream.LineCount()
		if lines > 0 {
			task.status.DonePercent = float64(cbd.NumSent) / float64(lines) * 100
		}

		// Assemble text and send for broadcasting.
		txt = fmt.Sprintf("%s | Progress: %.1f%% | Line %d of %d", task.gcodeStream.Name(), task.status.DonePercent, cbd.NumSent, lines)
	}
}
