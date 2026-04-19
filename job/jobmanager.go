package job

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/chepo92/Print2Go/gcode"
	"github.com/chepo92/Print2Go/serial/serialmgr"
	store "github.com/chepo92/Print2Go/storage"
)

// JobStatus mantiene el estado actual de un trabajo.
type JobStatus struct {
	// File
	File        string `json:"file"`
	Description string `json:"description"`
	FileSize    int64  `json:"fileSize"` // bytes
	FileDate    int64  `json:"fileDate"` // unix timestamp

	// Estimations
	EstimatedTime  int     `json:"estimatedTime"`  // s
	FilamentLength float64 `json:"filamentLength"` // mm
	FilamentVolume float64 `json:"filamentVolume"` // cm3

	// Info
	LastCommand string `json:"lastcmd"`
	LastReply   string `json:"lastreply"`
	Active      bool   `json:"active"`
	Error       bool   `json:"error"`
	Paused      bool   `json:"paused"`

	// Job progress
	DisplayStatus string    `json:"displayStatus"`
	StartTime     time.Time `json:"startTime"`
	DonePercent   float64   `json:"donePct"`
	RunDuration   string    `json:"runDuration"`

	PrintReport string `json:"printReport"`

	SerialLog []SerialLogEntry `json:"serial_log"`

	Motd string `json:"motd"`
}

// type jobStatus struct {
// 	File        string        `json:"file"`
// 	Done        float64       `json:"donePct"`
// 	Desc        string        `json:"description"`
// 	Cmd         string        `json:"lastcmd"`
// 	Reply       string        `json:"lastreply"`
// 	Active      bool          `json:"active"`
// 	Hash        uint32        `json:"fp"`
// 	Age         time.Duration `json:"ageSecs"`
// 	RunDuration string        `json:"runDuration"`
// 	Motd        string        `json:"motd"`
// 	Status      string        `json:"status"`
// 	Error       bool          `json:"error"`
// }

type SerialLogEntry struct {
	Type string `json:"type"` // "cmd" | "resp"
	Line string `json:"line"`
	Ts   int64  `json:"ts"`
}

// JobManager gestiona un trabajo activo y notifica a suscriptores.
type JobManager struct {
	sync.RWMutex
	serial *serialmgr.SerialManager

	// gcodeStream store.Stream

	ctx    context.Context
	cancel context.CancelFunc

	mu sync.Mutex

	paused bool

	status      JobStatus
	subscribers map[string]chan JobStatus
}

// New crea un JobManager.
func New(serial *serialmgr.SerialManager) *JobManager {

	jm := &JobManager{
		serial:      serial,
		subscribers: make(map[string]chan JobStatus),
	}

	// Registramos onLine permanente
	serial.SetLineHandler(func(line string) {
		jm.Lock()
		defer jm.Unlock()

		jm.status.LastReply = line

		jm.AddLog(SerialLogEntry{
			Type: "resp",
			Line: line,
			Ts:   time.Now().Unix(),
		})

		jm.broadcast()
	})

	return jm
}

const maxLogLines = 200

func (jm *JobManager) AddLog(entry SerialLogEntry) {
	jm.status.SerialLog = append(jm.status.SerialLog, entry)

	if len(jm.status.SerialLog) > maxLogLines {
		jm.status.SerialLog = jm.status.SerialLog[len(jm.status.SerialLog)-maxLogLines:]
	}
}

// StartPrint inicia la impresión de un archivo gcode.
func (jm *JobManager) StartPrint(gcs store.Stream) error {
	jm.Lock()
	defer jm.Unlock()

	if jm.status.Active {
		return fmt.Errorf("job already running")
	}

	if !jm.serial.IsConnected() {
		return fmt.Errorf("serial not connected")
	}

	ctx, cancel := context.WithCancel(context.Background())
	jm.ctx = ctx
	jm.cancel = cancel

	jm.status = JobStatus{
		File:           gcs.Name(),
		DonePercent:    0,
		Description:    "",
		Active:         true,
		Error:          false,
		RunDuration:    "",
		LastCommand:    "",
		LastReply:      "",
		StartTime:      time.Now(),
		EstimatedTime:  0,
		FilamentLength: 0,
		FilamentVolume: 0,
		Motd:           "",
		FileSize:       0,
		FileDate:       0,
		DisplayStatus:  "Printing...",
	}

	// Broadcast status
	jm.broadcast()

	// Excecute in go routine
	fmt.Printf("calling jm.runPrint from StartPrint\n")
	go jm.runPrint(ctx, gcs)

	return nil
}

// runPrint se ejecuta en goroutine y actualiza estado.
func (jm *JobManager) runPrint(ctx context.Context, stream store.Stream) {
	defer stream.Close()

	// totalLines := stream.LineCount()
	//sent := 0
	// jm.serial.SetLineHandler(func(line string) {
	// 	jm.Lock()
	// 	defer jm.Unlock()

	// 	jm.status.LastReply = line
	// 	jm.broadcast()
	// })
	fmt.Printf("calling serial.SendGcodeFileWithContext from jm.runPrint \n")
	err := jm.serial.SendGcodeFileWithContext(
		ctx,
		stream,
		func(sent, total int, parsed gcode.Line) {

			jm.Lock()
			defer jm.Unlock()

			// fmt.Printf("[onLine func] Post Lock \n")
			jm.status.LastCommand = parsed.Raw

			jm.AddLog(SerialLogEntry{
				Type: "cmd",
				Line: parsed.Raw,
				Ts:   time.Now().Unix(),
			})

			jm.HandleGcode(parsed.Command)

			if total > 0 {
				jm.status.DonePercent = float64(sent) / float64(total) * 100
			}
			jm.status.PrintReport = fmt.Sprintf("Progress: %.1f%% | Line %d of %d", jm.status.DonePercent, sent, total)

			jm.status.RunDuration = time.Since(jm.status.StartTime).Truncate(time.Second).String()

			jm.broadcast()

			// fmt.Printf("[onLine func] Pre pause \n")

			// for jm.IsPaused() {
			// 	time.Sleep(100 * time.Millisecond)
			// 	fmt.Printf("[onLine func] Pause, last cmd %s\n", parsed.Command)
			// }

			// fmt.Printf("[onLine func] Post pause \n")

		},
	)

	jm.Lock()
	defer jm.Unlock()
	fmt.Printf("jm.Runprint: Set status active = false \n")
	jm.status.Active = false

	switch {
	case err == nil:
		jm.status.DonePercent = 100
		jm.status.Description = "Done"
		jm.status.DisplayStatus = "Printer Finished"
		jm.status.Error = false

	case errors.Is(err, context.Canceled):
		jm.status.Description = "Cancelled"
		jm.status.DisplayStatus = "Cancelled"
		jm.status.Error = false

	default:
		jm.status.Error = true
		jm.status.Description = err.Error()
		jm.status.DisplayStatus = "Error: " + err.Error()
	}

	//jm.serial.SetLineHandler(nil)

	fmt.Printf("jm.Runprint: broadcast \n")
	jm.broadcast()
}

// Active devuelve true si hay un trabajo activo
func (jm *JobManager) Active() bool {
	jm.RLock()
	defer jm.RUnlock()
	return jm.status.Active
}

// Cancel cancela el trabajo actual
func (jm *JobManager) Cancel() {
	jm.Lock()
	defer jm.Unlock()
	fmt.Printf("[jm.Cancel]  \n")
	if jm.cancel != nil && jm.status.Active {
		fmt.Printf("[jm.Cancel] Cancelling...   \n")
		jm.cancel()
		jm.status.Active = false
		jm.status.Description = "Cancelled"
		jm.status.DisplayStatus = "Cancelled"
		jm.broadcast()
		jm.cleanup()
	}
}

func (jm *JobManager) SendPriorityCommand(cmd string) error {
	jm.Lock()
	defer jm.Unlock()

	if !jm.serial.IsConnected() {
		return fmt.Errorf("serial not connected")
	}

	jm.AddLog(SerialLogEntry{
		Type: "cmd",
		Line: cmd,
		Ts:   time.Now().Unix(),
	})

	// Actualizar estado para la UI
	jm.status.LastCommand = cmd
	jm.broadcast()

	return jm.serial.SendGcodePriority(cmd, false)
}

// Subscribe devuelve un canal para recibir actualizaciones de estado
func (jm *JobManager) Subscribe() chan JobStatus {
	jm.Lock()
	defer jm.Unlock()
	c := make(chan JobStatus, 1)
	c <- jm.status
	jm.subscribers[fmt.Sprintf("%p", c)] = c
	return c
}

// Unsubscribe elimina un canal previamente suscrito
func (jm *JobManager) Unsubscribe(c chan JobStatus) {
	jm.Lock()
	defer jm.Unlock()
	k := fmt.Sprintf("%p", c)
	if _, ok := jm.subscribers[k]; ok {
		delete(jm.subscribers, k)
		close(c)
	}
}

// broadcast notifica a todos los suscriptores
func (jm *JobManager) broadcast() {
	for _, c := range jm.subscribers {
		c := c
		go func() {
			defer func() { recover() }()
			c <- jm.status
		}()
	}
}

// Snapshot devuelve el estado actual del trabajo
func (jm *JobManager) Snapshot() JobStatus {
	jm.RLock()
	defer jm.RUnlock()
	return jm.status
}

// cleanup libera recursos internos después de terminar
func (jm *JobManager) cleanup() {
	// jm.pagInstance = nil
	jm.ctx = nil
	jm.cancel = nil
	// jm.gcodeStream = nil
}

func (jm *JobManager) Pause() {
	jm.mu.Lock()
	jm.paused = true
	jm.status.Paused = true
	jm.serial.SetPaused(true)
	jm.status.DisplayStatus = "Paused"
	jm.mu.Unlock()
}

func (jm *JobManager) Resume() {
	jm.mu.Lock()
	jm.paused = false
	jm.status.Paused = false
	jm.serial.SetPaused(false)
	jm.status.DisplayStatus = "Printing"
	jm.mu.Unlock()
}

func (jm *JobManager) IsPaused() bool {
	jm.mu.Lock()
	defer jm.mu.Unlock()
	return jm.paused
}

func (jm *JobManager) HandleGcode(cmd string) {
	fmt.Println("[HandleGcode]: ", cmd)
	switch cmd {
	case "M0", "M1", "M600", "M25":
		fmt.Println("[JobManager] Pause requested via G-code:", cmd)
		jm.Pause()
	case "M24": // Resume SD print
		fmt.Println("[JobManager] Resume requested via G-code:", cmd)
		jm.Resume()
	}
}
