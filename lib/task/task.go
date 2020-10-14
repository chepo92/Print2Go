package task

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	"gitlab.com/adrian_blx/takoprint/lib/store"
	"gitlab.com/adrian_blx/takoprint/lib/takoprint"
)

type TaskStatus struct {
	// Filename we are currently executing.
	File string
	// Human readable description.
	Text string
	// Percentage printend.
	Done float64
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

func (t *Task) Launch(p io.ReadWriteCloser, gcs store.Stream) error {
	t.Lock()
	defer t.Unlock()

	if t.gcodeStream != nil {
		return fmt.Errorf("task already running")
	}

	ctx, cancel := context.WithCancel(context.Background())
	t.tp = takoprint.New(p, gcs)
	t.ctx = ctx
	t.cancel = cancel
	t.gcodeStream = gcs
	t.status.Done = 0
	t.status.Active = true
	t.status.Text = "starting..."
	t.status.Started = time.Now()

	// horray for circular dependencies!
	takoprint.Callback(t.callback)(t.tp)

	// semi-empty callback to anounce that we are now active.
	go t.callback(&takoprint.CallbackData{})
	go t.start()
	return nil
}

// Returns true if the task is done.
func (t *Task) Done() bool {
	t.RLock()
	defer t.RUnlock()
	return t.ctx == nil || t.ctx.Err() != nil
}

func (t *Task) WaitDone() <-chan struct{} {
	t.RLock()
	defer t.RUnlock()

	if t.ctx == nil {
		return nil
	}
	return t.ctx.Done()
}

// Cancel calls the context cancel function, aborting the task.
func (t *Task) Cancel() {
	t.RLock()
	defer t.RUnlock()
	if t.cancel == nil {
		return
	}
	t.cancel()
}

// start internally launches the task and sets 'done' once the print finished.
func (t *Task) start() {
	t.tp.Start(t.ctx)
	// mark own context as done.

	// cancel our contex and fire an empty callback
	t.Cancel()
	t.callback(nil)
	t.nullify()
}

func (t *Task) nullify() {
	t.Lock()
	defer t.Unlock()

	// nulls *most* of the struct.
	t.tp = nil
	t.ctx = nil
	t.cancel = nil
	t.gcodeStream = nil
}

// called by takoprint to update the status.
func (t *Task) callback(d *takoprint.CallbackData) {
	var txt string

	t.Lock()
	// notify subscribers (must happen after unlock popped from stack)
	defer t.broadcast()
	defer t.Unlock()
	defer func() {
		if txt != t.status.Text {
			t.status.Text = txt
			t.tp.Echo(txt)
		}
	}()

	if d == nil {
		t.status.Active = false
		txt = "(finished)"
		return
	}

	// Keep a couple of seen replies.
	t.status.LastCommand = d.LastSent
	t.status.LastReply = d.Reply

	if sz := t.gcodeStream.Size(); sz > 0 {
		t.status.Done = float64(t.gcodeStream.Pos()) / float64(sz) * 100
	}

	// Assemble text and send it to printer on changes.
	txt = fmt.Sprintf("%.1f%% (%s)", t.status.Done, t.gcodeStream.Name())
}
