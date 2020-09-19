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

type Task struct {
	sync.RWMutex
	// True if the print task is done.
	done bool
	// Function to cancel the print context.
	cancel context.CancelFunc
	// Gfeeder reference.
	gf *gfeeder.Gfeeder
	// Size of input file.
	inputSize int64
	// Input filehandle.
	inputFh *os.File
	// When the print started
	epoch time.Time
	// Description of the print progress
	txt string
}

func New(p *serial.Port, fh *os.File) *Task {
	ctx, cancel := context.WithCancel(context.Background())

	stat, _ := fh.Stat()
	gf := gfeeder.New(p, fh)
	t := &Task{
		gf:        gf,
		cancel:    cancel,
		inputSize: stat.Size(),
		inputFh:   fh,
		epoch:     time.Now(),
		txt:       "<no progress>",
	}
	// horray for circular dependencies!
	gfeeder.Callback(t.callback)(gf)
	go t.start(ctx)
	return t
}

// Returns true if the task is done.
func (t *Task) Done() bool {
	t.RLock()
	defer t.RUnlock()
	return t.done
}

// Describe describes the status of the task.
func (t *Task) Describe() string {
	t.RLock()
	defer t.RUnlock()
	return t.txt
}

// Cancel calls the context cancel function, aborting the task.
func (t *Task) Cancel() {
	t.cancel()
}

// start internally launches the task and sets 'done' once the print finished.
func (t *Task) start(ctx context.Context) {
	t.gf.Start(ctx)
	t.Lock()
	defer t.Unlock()
	t.done = true
}

func (t *Task) callback(d *gfeeder.CallbackData) {
	t.Lock()
	defer t.Unlock()

	pos, _ := t.inputFh.Seek(0, os.SEEK_CUR)
	var pct float64
	if t.inputSize > 0 {
		pct = float64(pos) / float64(t.inputSize) * 100
	}
	runtime := time.Now().Sub(t.epoch)

	var eta time.Duration
	if runtime > time.Duration(10*time.Second) && pct > 0.1 {
		x := float64(runtime.Nanoseconds()) / pct
		eta = time.Duration(x * (100 - pct))
	}
	dbg := fmt.Sprintf("pos=%d, sz=%d", pos, t.inputSize)
	t.txt = fmt.Sprintf("Done=%v, runtime=%v, eta=%v (%s)\n", pct, runtime, eta, dbg)
}
