package task

import (
	"context"
	"io"
	"os"
	"sync"
	"time"

	"fmt"

	"gitlab.com/adrian_blx/takoprint/lib/takoprint"
)

var (
	// Start calculating an ETA after so many minutes.
	etaAfter = time.Duration(5 * time.Minute)
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
	// Size of input file.
	inputStat os.FileInfo
	// Input filehandle.
	inputFh *os.File
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
func (t *Task) Launch(p io.ReadWriteCloser, fh *os.File) error {
	t.Lock()
	defer t.Unlock()

	if t.inputFh != nil {
		return fmt.Errorf("task already running")
	}

	ctx, cancel := context.WithCancel(context.Background())

	stat, err := fh.Stat()
	if err != nil {
		return fmt.Errorf("stat failed: %v", err)
	}

	t.tp = takoprint.New(p, fh)
	t.ctx = ctx
	t.cancel = cancel
	t.inputFh = fh
	t.inputStat = stat

	t.status.Done = 0
	t.status.Active = true
	t.status.Text = "starting..."

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
	return t.ctx.Done()
}

// Cancel calls the context cancel function, aborting the task.
func (t *Task) Cancel() {
	t.RLock()
	defer t.RUnlock()
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
	t.inputFh = nil
	t.inputStat = nil
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

	// Try to calculate overall percentage.
	pos, _ := t.inputFh.Seek(0, os.SEEK_CUR)
	if sz := t.inputStat.Size(); sz > 0 {
		t.status.Done = float64(pos) / float64(sz) * 100
	}

	// Assemble text and send it to printer on changes.
	txt = fmt.Sprintf("%.1f%% (%s)", t.status.Done, t.inputStat.Name())
}
