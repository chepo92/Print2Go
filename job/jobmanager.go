package job

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/chepo92/PrintAndGo/lib/printandgo"
	"github.com/chepo92/PrintAndGo/serial/serialmgr"
	"github.com/chepo92/PrintAndGo/store"
)

// JobStatus mantiene el estado actual de un trabajo.
type JobStatus struct {
	File        string  `json:"file"`
	DonePercent float64 `json:"donePct"`
	Description string  `json:"description"`
	LastCommand string  `json:"lastcmd"`
	LastReply   string  `json:"lastreply"`
	Active      bool    `json:"active"`
	RunDuration string  `json:"runDuration"`
	Error       bool    `json:"error"`
	Motd        string  `json:"motd"`
	// Time we started this print
	StarTime time.Time `json:"startTime"`
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
		File:        gcs.Name(),
		DonePercent: 0,
		Description: "Starting...",
		Active:      true,
		Error:       false,
		RunDuration: "",
		LastCommand: "",
		LastReply:   "",
		StarTime:    time.Now(),
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

			jm.broadcast()

		},
	)

	jm.Lock()
	defer jm.Unlock()
	fmt.Printf("jm.Runprint: Set status active \n")
	jm.status.Active = false

	if err != nil {
		jm.status.Error = true
		jm.status.Description = err.Error()
	} else {
		jm.status.DonePercent = 100
		jm.status.Description = "Done"
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
	if jm.cancel != nil && jm.status.Active {
		jm.cancel()
		jm.status.Active = false
		jm.status.Description = "Cancelled"
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
