package job

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/chepo92/PrintAndGo/lib/printandgo"
	"github.com/chepo92/PrintAndGo/serial/serialmgr"
	store "github.com/chepo92/PrintAndGo/storage"
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

	// Job progress
	DisplayStatus string    `json:"displayStatus"`
	StartTime     time.Time `json:"startTime"`
	DonePercent   float64   `json:"donePct"`
	RunDuration   string    `json:"runDuration"`

	PrintReport string `json:"printReport"`

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

// JobManager gestiona un trabajo activo y notifica a suscriptores.
type JobManager struct {
	sync.RWMutex
	serial *serialmgr.SerialManager

	// gcodeStream store.Stream

	ctx    context.Context
	cancel context.CancelFunc

	status      JobStatus
	subscribers map[string]chan JobStatus
}

// New crea un JobManager.
func New(serial *serialmgr.SerialManager) *JobManager {
	return &JobManager{
		serial:      serial,
		subscribers: make(map[string]chan JobStatus),
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
	// jm.gcodeStream = gcs

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

	// jm.pagInstance = printandgo.New(nil, gcs) // puerto ya manejado dentro de PrintAndGo

	// Configuramos callback para actualizar estado
	// printandgo.Callback(jm.callback)(jm.pagInstance)
	jm.broadcast()

	fmt.Printf("calling jm.runPrint from StartPrint\n")
	go jm.runPrint(ctx, gcs)

	return nil
}

// runPrint se ejecuta en goroutine y actualiza estado.
func (jm *JobManager) runPrint(ctx context.Context, stream store.Stream) {
	defer stream.Close()

	// totalLines := stream.LineCount()
	//sent := 0
	fmt.Printf("calling serial.SendGcodeFileWithContext from jm.runPrint \n")
	err := jm.serial.SendGcodeFileWithContext(
		ctx,
		stream,
		func(sent, total int, cmd, reply string) {
			jm.Lock()
			defer jm.Unlock()

			jm.status.LastCommand = cmd
			jm.status.LastReply = reply

			if total > 0 {
				jm.status.DonePercent = float64(sent) / float64(total) * 100
			}
			jm.status.PrintReport = fmt.Sprintf("Progress: %.1f%% | Line %d of %d", jm.status.DonePercent, sent, total)

			jm.status.RunDuration = time.Since(jm.status.StartTime).Truncate(time.Second).String()

			jm.broadcast()

		},
	)

	jm.Lock()
	defer jm.Unlock()
	fmt.Printf("jm.Runprint: Set status active = false \n")
	jm.status.Active = false

	if err != nil {
		jm.status.Error = true
		jm.status.Description = err.Error()
		jm.status.DisplayStatus = "Error: " + err.Error()
	} else {
		jm.status.DonePercent = 100
		jm.status.Description = "Done"
		jm.status.DisplayStatus = "Printer Finished"
	}
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

// InjectGcode envía un comando gcode al trabajo activo
func (jm *JobManager) InjectGcode(cmd string) error {
	jm.RLock()
	defer jm.RUnlock()
	fmt.Println("TBI: InjectGcode called in JobManager. ")
	// if !jm.Active() || jm.pagInstance == nil {
	// 	return fmt.Errorf("no hay trabajo activo")
	// }
	// jm.pagInstance.InjectGcode(cmd)
	return nil
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

// callback usado por PrintAndGo para actualizar estado en tiempo real
func (jm *JobManager) callback(cbd *printandgo.CallbackData) {
	jm.Lock()
	defer jm.Unlock()

	if cbd != nil {
		jm.status.LastCommand = cbd.LastSent
		jm.status.LastReply = cbd.Reply

		// lines := jm.gcodeStream.LineCount()
		// if lines > 0 {
		// 	jm.status.DonePercent = float64(cbd.NumSent) / float64(lines) * 100
		// }
		// jm.status.Description = fmt.Sprintf("%s | %.1f%% (%d/%d)", jm.gcodeStream.Name(), jm.status.DonePercent, cbd.NumSent, lines)
	}

	jm.broadcast()
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
