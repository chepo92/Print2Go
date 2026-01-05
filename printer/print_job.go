package printer

import (
	"bufio"
	"os"
	"sync"
)

type JobState int

const (
	JobIdle JobState = iota
	JobPrinting
	JobPaused
	JobCancelled
	JobDone
	JobError
)

type PrintJob struct {
	proto *PrinterProtocol

	filePath   string
	totalLines int
	sentLines  int

	state JobState
	mu    sync.Mutex

	pauseCh  chan struct{}
	resumeCh chan struct{}
	cancelCh chan struct{}
}

func NewPrintJob(proto *PrinterProtocol, path string) *PrintJob {
	return &PrintJob{
		proto:    proto,
		filePath: path,
		state:    JobIdle,
		pauseCh:  make(chan struct{}),
		resumeCh: make(chan struct{}),
		cancelCh: make(chan struct{}),
	}
}

func countLines(path string) (int, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	n := 0
	for sc.Scan() {
		n++
	}
	return n, sc.Err()
}

func (j *PrintJob) Start() error {
	total, err := countLines(j.filePath)
	if err != nil {
		return err
	}
	j.totalLines = total

	j.setState(JobPrinting)
	go j.run()
	return nil
}

func (j *PrintJob) run() {
	f, err := os.Open(j.filePath)
	if err != nil {
		j.setState(JobError)
		return
	}
	defer f.Close()

	sc := bufio.NewScanner(f)

	for sc.Scan() {

		// --- cancel ---
		select {
		case <-j.cancelCh:
			j.setState(JobCancelled)
			return
		default:
		}

		// --- pause ---
		j.waitIfPaused()

		line := sc.Text()
		if line == "" || line[0] == ';' {
			continue
		}

		j.mu.Lock()
		j.sentLines++
		j.mu.Unlock()
	}

	j.setState(JobDone)
}

func (j *PrintJob) Pause() {
	j.setState(JobPaused)
}

func (j *PrintJob) Resume() {
	j.setState(JobPrinting)
	j.resumeCh <- struct{}{}
}

func (j *PrintJob) waitIfPaused() {
	for {
		j.mu.Lock()
		paused := j.state == JobPaused
		j.mu.Unlock()

		if !paused {
			return
		}
		<-j.resumeCh
	}
}

func (j *PrintJob) Cancel() {
	select {
	case j.cancelCh <- struct{}{}:
	default:
	}
}

func (j *PrintJob) Progress() float64 {
	j.mu.Lock()
	defer j.mu.Unlock()
	if j.totalLines == 0 {
		return 0
	}
	return float64(j.sentLines) / float64(j.totalLines)
}

func (j *PrintJob) State() JobState {
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.state
}

func (j *PrintJob) setState(s JobState) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.state = s
}
