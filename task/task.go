package task

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/chepo92/PrintAndGo/lib/takoprint"
	"github.com/chepo92/PrintAndGo/store"
)

type TaskStatus struct {
	// Filename we are currently executing.
	File string
	// Human readable description.
	Text string
	// Percentage printed
	DonePercent float64
	// Whether or not we actually do anything.
	Active bool
	// Time we started this print
	Started time.Time
	// Last command we sent.
	LastCommand string
	// Last reply we received.
	LastReply string
}

type Task struct {
	sync.RWMutex
	// Internal print context.
	ctx context.Context
	// Function to cancel the print context.
	cancel context.CancelFunc
	// Takoprint reference.
	tp *takoprint.Takoprint
	// Gcode input.
	gcodeStream store.Stream
	// Status of the currently running print.
	status TaskStatus
	// clients subscribing to updates
	subscribers map[string]chan TaskStatus
}

func New() *Task {
	return &Task{
		subscribers: make(map[string]chan TaskStatus),
	}
}

// Launch starts a new print. Accepts an io.ReadWriteCloser for the serial port and a store.Stream for the gcode input.
func (task *Task) Launch(p io.ReadWriteCloser, gcs store.Stream) error {
	task.Lock()
	defer task.Unlock()

	if task.gcodeStream != nil {
		return fmt.Errorf("Task already running")
	}

	ctx, cancel := context.WithCancel(context.Background())
	task.tp = takoprint.New(p, gcs)
	task.ctx = ctx
	task.cancel = cancel
	task.gcodeStream = gcs
	task.status.DonePercent = 0
	task.status.Active = true
	task.status.Text = "Starting..."
	task.status.Started = time.Now()

	fmt.Println("Gcode: ", gcs.Name())
	fmt.Println("Size: ", gcs.Size())

	// horray for circular dependencies!
	takoprint.Callback(task.callback)(task.tp)

	// semi-empty callback to anounce that we are now active.
	go task.callback(&takoprint.CallbackData{})
	go task.start()
	return nil
}

// Returns true if the task is done.
func (task *Task) Done() bool {
	task.RLock()
	defer task.RUnlock()
	return task.ctx == nil || task.ctx.Err() != nil
}

func (task *Task) WaitDone() <-chan struct{} {
	task.RLock()
	defer task.RUnlock()

	if task.ctx == nil {
		return nil
	}
	return task.ctx.Done()
}

// Cancel calls the context cancel function, aborting the task.
func (task *Task) Cancel() {
	task.RLock()
	defer task.RUnlock()
	if task.cancel == nil {
		return
	}
	task.cancel()
}

// start internally launches the task and sets 'done' once the print finished.
func (task *Task) start() {
	task.tp.Start(task.ctx)
	// mark own context as done.

	// cancel our contex and fire an empty callback
	task.Cancel()
	task.callback(nil)
	task.nullify()
}

// nullify clears most of the struct, allowing for a new task to be started.
func (task *Task) nullify() {
	task.Lock()
	defer task.Unlock()

	// nulls *most* of the struct.
	task.tp = nil
	task.ctx = nil
	task.cancel = nil
	task.gcodeStream = nil
}

// called by takoprint to update the status of the task, which is then broadcasted to subscribers
func (task *Task) callback(d *takoprint.CallbackData) {
	var txt string

	task.Lock()
	// notify subscribers (must happen after unlock popped from stack)
	defer task.broadcast()
	defer task.Unlock()
	defer func() {
		if txt != task.status.Text {
			task.status.Text = txt
			task.tp.Echo(txt)
			fmt.Println("Status: ", txt)
		}
	}()

	if d == nil {
		task.status.Active = false
		txt = "Yay! Done!"
		return
	}

	// Keep a couple of seen replies.
	task.status.LastCommand = d.LastSent
	task.status.LastReply = d.Reply

	if sz := task.gcodeStream.Size(); sz > 0 {
		task.status.DonePercent = float64(task.gcodeStream.Pos()) / float64(sz) * 100
	}

	// Assemble text and send it to printer on changes.
	txt = fmt.Sprintf("%.1f%% (%s)", task.status.DonePercent, task.gcodeStream.Name())
}
