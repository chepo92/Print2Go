package task

import (
	"context"
	"os"
	"sync"
	"time"

	"fmt"

	"github.com/tarm/serial"
	"gitlab.com/adrian_blx/gfeeder/lib/gfeeder"
)

var (
	// Start calculating an ETA after so many minutes.
	etaAfter = time.Duration(5 * time.Minute)
)

type Task struct {
	sync.RWMutex
	// Internal print context.
	ctx context.Context
	// Function to cancel the print context.
	cancel context.CancelFunc
	// Gfeeder reference.
	gf *gfeeder.Gfeeder
	// Size of input file.
	inputStat os.FileInfo
	// Input filehandle.
	inputFh *os.File
	// When the print started
	epoch time.Time
	// Description of the print progress
	txt string
	// percentage done
	donePercent float64
	// logbuf
	logbuf []*gfeeder.CallbackData
}

func New(p *serial.Port, fh *os.File) (*Task, error) {
	ctx, cancel := context.WithCancel(context.Background())

	stat, err := fh.Stat()
	if err != nil {
		return nil, fmt.Errorf("stat failed: %v", err)
	}

	gf := gfeeder.New(p, fh)
	t := &Task{
		gf:        gf,
		ctx:       ctx,
		cancel:    cancel,
		inputStat: stat,
		inputFh:   fh,
		epoch:     time.Now(),
		txt:       fmt.Sprintf("<%s>", stat.Name()),
		logbuf:    make([]*gfeeder.CallbackData, 20),
	}
	// horray for circular dependencies!
	gfeeder.Callback(t.callback)(gf)
	go t.start()
	return t, nil
}

// Returns true if the task is done.
func (t *Task) Done() bool {
	return t.ctx.Err() != nil
}

func (t *Task) WaitDone() <-chan struct{} {
	return t.ctx.Done()
}

// Describe describes the status of the task.
func (t *Task) Describe() string {
	t.RLock()
	defer t.RUnlock()
	return t.txt
}

func (t *Task) LogBuffer() []*gfeeder.CallbackData {
	t.RLock()
	defer t.RUnlock()
	return t.logbuf
}

// Cancel calls the context cancel function, aborting the task.
func (t *Task) Cancel() {
	t.cancel()
}

// start internally launches the task and sets 'done' once the print finished.
func (t *Task) start() {
	t.gf.Start(t.ctx)
	// mark own context as done.
	t.Cancel()
}

// called by gfeeder to update the status.
func (t *Task) callback(d *gfeeder.CallbackData) {
	t.Lock()
	defer t.Unlock()

	// Keep a couple of seen replies.
	t.logbuf = append(t.logbuf, d)
	t.logbuf = t.logbuf[1:len(t.logbuf)]

	// Try to calculate overall percentage.
	pos, _ := t.inputFh.Seek(0, os.SEEK_CUR)
	if sz := t.inputStat.Size(); sz > 0 {
		t.donePercent = float64(pos) / float64(sz) * 100
	}

	// Assemble text and send it to printer on changes.
	txt := fmt.Sprintf("%.1f%% (%s)", t.donePercent, t.inputStat.Name())
	if txt != t.txt {
		t.txt = txt
		t.gf.Echo(txt)
	}
}
