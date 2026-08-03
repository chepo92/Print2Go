package serialmgr

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/chepo92/Print2Go/gcode"
	"github.com/chepo92/Print2Go/serial"
	hwserial "go.bug.st/serial"
)

type SerialState int

var (
	ErrNotConnected = errors.New("serial not connected")
)

const (
	Disconnected SerialState = iota
	Connecting
	Initializing
	Connected
	Error
)

func (s SerialState) String() string {
	switch s {
	case Disconnected:
		return "Disconnected"
	case Connecting:
		return "Connecting"
	case Initializing:
		return "Initializing"
	case Connected:
		return "Connected"
	case Error:
		return "Error"
	default:
		return "Unknown"
	}
}

type SerialManager struct {
	mu    sync.Mutex
	state SerialState

	port        hwserial.Port
	cfg         serial.SerialConfig
	autoConnect bool

	openFn func(serial.SerialConfig) (hwserial.Port, error)

	OnConnected func()

	sendQ     chan GcodeCmd
	priorityQ chan GcodeCmd
	inbound   chan string
	onLine    func(string)

	paused bool
	stopCh chan struct{}

	OnAction func(action string)

	temps TempState

	CommandTimeout time.Duration
	initTimer      *time.Timer
	initDone       sync.Once
	// Maximum time to wait after the last startup message before
	// considering the printer ready. Every line received during
	// initialization restarts this timer.
	InitTimeout time.Duration
}

type GcodeCmd struct {
	Line     string
	Priority bool
	RespCh   chan error // nil si no se espera ACK
}

type TempState struct {
	ToolActual float64
	ToolTarget float64
	BedActual  float64
	BedTarget  float64
	Valid      bool
}

func (sm *SerialManager) Temps() TempState {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	return sm.temps
}

func New(
	openFn func(serial.SerialConfig) (hwserial.Port, error),
	cfg serial.SerialConfig,
) *SerialManager {

	return &SerialManager{
		state:          Disconnected,
		cfg:            cfg,
		openFn:         openFn,
		sendQ:          make(chan GcodeCmd, 16),
		priorityQ:      make(chan GcodeCmd, 8),
		inbound:        make(chan string, 16),
		stopCh:         make(chan struct{}),
		CommandTimeout: 5 * time.Second,
		// Maximum silence after startup output before the printer is
		// considered initialized. Each received line resets this timer.
		InitTimeout: 10 * time.Second,
		autoConnect: true,
	}
}

func (sm *SerialManager) SetLineHandler(fn func(string)) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.onLine = fn
}

// The real writer
func (sm *SerialManager) writeLoop() {
	for {
		fmt.Printf("[SERIAL] writeLoop alive\n")

		var cmd GcodeCmd

		// Obtener siguiente comando
		select {
		case cmd = <-sm.priorityQ:
			// prioridad
		default:
			select {
			case cmd = <-sm.priorityQ:
			case cmd = <-sm.sendQ:
			case <-sm.stopCh:
				return
			}
		}

		_, err := sm.port.Write([]byte(cmd.Line + "\n"))
		if err != nil {
			if cmd.RespCh != nil {
				cmd.RespCh <- err
			}
			continue
		}

		cmdTimeout := sm.commandTimeout(cmd.Line)
		fmt.Printf("[SERIAL] >> %s (timeout=%s)\n", cmd.Line, cmdTimeout)
		timeout := time.NewTimer(cmdTimeout)
		waiting := true

		for waiting {

			select {

			case line := <-sm.inbound:

				// Reiniciar timeout mientras la impresora siga respondiendo
				if !timeout.Stop() {
					select {
					case <-timeout.C:
					default:
					}
				}
				timeout.Reset(cmdTimeout)

				fmt.Printf("[SERIAL] Reply: %s\n", line)

				if temp, ok := parseTemps(line); ok {
					sm.mu.Lock()
					sm.temps = temp
					sm.mu.Unlock()
				}

				switch {

				case strings.HasPrefix(line, "ok"):
					if cmd.RespCh != nil {
						cmd.RespCh <- nil
					}
					waiting = false

				case strings.HasPrefix(line, "error"):
					if cmd.RespCh != nil {
						cmd.RespCh <- fmt.Errorf(line)
					}
					waiting = false

				case strings.HasPrefix(line, "Unknown"):
					if cmd.RespCh != nil {
						cmd.RespCh <- nil
					}
					waiting = false

				case strings.HasPrefix(line, "echo"):
					fmt.Printf("[SERIAL] Printer echo: %s\n", line)

				case strings.HasPrefix(line, "T:"):
					fmt.Printf("[SERIAL] Printer temps: %s\n", line)

				default:
					fmt.Printf("[SERIAL] Unhandled reply: %s\n", line)
				}

			case <-timeout.C:

				fmt.Printf("[SERIAL] Timeout waiting for reply to '%s'\n", cmd.Line)

				sm.flushInbound()

				if cmd.RespCh != nil {
					cmd.RespCh <- fmt.Errorf("timeout")
				}

				waiting = false

			case <-sm.stopCh:
				timeout.Stop()
				fmt.Printf("[SERIAL] writeLoop stopped\n")
				return
			}
		}

		timeout.Stop()
	}
}

func (sm *SerialManager) readLoop() {
	scanner := bufio.NewScanner(sm.port)

	for {
		select {
		case <-sm.stopCh:
			fmt.Printf("[SERIAL] readLoop stopped\n")
			return

		default:
			if !scanner.Scan() {
				if err := scanner.Err(); err != nil {
					fmt.Printf("[SERIAL] Read error: %v\n", err)
				} else {
					fmt.Printf("[SERIAL] Port closed\n")
				}
				return
			}

			line := strings.TrimSpace(scanner.Text())
			if line == "" {
				continue
			}

			fmt.Printf("[SERIAL] << %s\n", line)

			// Leer el estado una sola vez
			sm.mu.Lock()
			state := sm.state
			handler := sm.onLine
			sm.mu.Unlock()

			// Mostrar SIEMPRE al frontend
			if handler != nil {
				handler(line)
			}

			// Procesar acciones espontáneas SIEMPRE
			if strings.HasPrefix(line, "M118") {
				if strings.Contains(line, "//action:pause") {
					sm.handleAction("pause")
				} else if strings.Contains(line, "//action:resume") {
					sm.handleAction("resume")
				} else if strings.Contains(line, "//action:cancel") {
					sm.handleAction("cancel")
				}
			}

			// Mientras inicializa, NO enviar al writeLoop
			if state == Initializing {

				sm.resetInitTimer()

				if strings.HasPrefix(line, "ok") {
					sm.finishInitialization()
				}

				continue
			}

			// Sólo cuando ya está conectado,
			// las respuestas pertenecen a comandos.
			select {
			case sm.inbound <- line:
			default:
				fmt.Printf("[SERIAL] inbound buffer full, dropping line\n")
			}
		}
	}
}

func (sm *SerialManager) FilterPorts(ports []string) []string {
	var out []string

	for _, p := range ports {
		if isValidPort(p) {
			out = append(out, p)

		} else {
			fmt.Printf("[SERIAL] Ignoring port: %s\n", p)
		}
	}

	return out
}

func isValidPort(p string) bool {
	switch runtime.GOOS {

	case "windows":
		// Solo COMx (COM1, COM3, etc.)
		return strings.HasPrefix(strings.ToUpper(p), "COM")

	case "linux":
		// Solo dispositivos USB reales
		return strings.HasPrefix(p, "/dev/ttyUSB") ||
			strings.HasPrefix(p, "/dev/ttyACM")

	default:
		return true // fallback (macOS, etc.)
	}
}

func (sm *SerialManager) Connect(cfg serial.SerialConfig) error {
	sm.mu.Lock()

	fmt.Printf("[SERIAL] Connect requested: port=%s baud=%d state=%v\n", cfg.Port, cfg.BaudRate, sm.state)
	if sm.state == Connected || sm.state == Connecting {
		fmt.Printf("[SERIAL] Connect ignored (already connected/connecting)\n")
		sm.mu.Unlock()
		return nil
	}

	sm.state = Connecting
	sm.cfg = cfg
	fmt.Printf("[SERIAL] State -> CONNECTING\n")

	// Resolver el puerto antes de abrirlo
	sm.resolvePortLocked()

	// Recuperar la configuración posiblemente modificada
	cfg = sm.cfg
	sm.mu.Unlock()

	fmt.Printf("[SERIAL] Try opening port=%s baud=%d\n", cfg.Port, cfg.BaudRate)

	// --- heavy IO out from lock ---
	port, err := sm.openFn(cfg)
	if err != nil {
		sm.mu.Lock()
		sm.state = Error
		sm.mu.Unlock()

		fmt.Printf("[SERIAL] Open FAILED: %v\n", err)
		return err
	}

	sm.mu.Lock()
	sm.port = port
	sm.state = Initializing

	// init channels
	sm.sendQ = make(chan GcodeCmd, 32)
	sm.priorityQ = make(chan GcodeCmd, 8)
	sm.inbound = make(chan string, 128)
	sm.stopCh = make(chan struct{})

	fmt.Printf("[SERIAL] Open OK, state -> INITIALIZING\n")
	sm.mu.Unlock()

	// Init write and read Loops
	go sm.writeLoop()
	go sm.readLoop()
	go sm.tempLoop()

	// Start listening for initial messages
	sm.initDone = sync.Once{}

	sm.mu.Lock()
	sm.initTimer = time.AfterFunc(5*time.Second, sm.finishInitialization)
	sm.mu.Unlock()

	if sm.OnConnected != nil {
		sm.OnConnected()
	}

	return nil

}

func (sm *SerialManager) Disconnect() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	fmt.Printf("[SERIAL] Disconnect requested, state=%v\n", sm.state)
	if sm.state == Disconnected {
		return nil
	}

	if sm.state != Connected || sm.port == nil {
		fmt.Printf("[SERIAL] Disconnect ignored (not connected)\n")
		return nil
	}

	if sm.port != nil {
		sm.stopCh <- struct{}{}
		err := sm.port.Close()
		sm.port = nil
		if err != nil {
			fmt.Printf("[SERIAL] Close error: %v\n", err)
		}
	}

	sm.state = Disconnected
	fmt.Printf("[SERIAL] State -> DISCONNECTED\n")

	// return err

	// sm.state = Disconnected
	return nil
}

func (sm *SerialManager) GetPort() (io.ReadWriteCloser, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.state != Connected || sm.port == nil {
		return nil, fmt.Errorf("serial not connected")
	}

	return sm.port, nil
}

func (sm *SerialManager) GetState() SerialState {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	return sm.state
}

func (sm *SerialManager) GetConfig() serial.SerialConfig {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	return sm.cfg
}

func (sm *SerialManager) SendLine(line string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.state != Connected {
		return fmt.Errorf("serial not connected")
	}
	fmt.Printf("[SendLine] Sending line:  %s\n", line)
	_, err := sm.port.Write([]byte(line + "\n"))
	if err != nil {
		sm.state = Error
		return err
	}

	return nil
}

func (sm *SerialManager) SendGcode(line string, wait bool) error {
	if err := sm.WaitReady(10 * time.Second); err != nil {
		return err
	}
	sm.mu.Lock()
	if sm.state != Connected {
		sm.mu.Unlock()
		return fmt.Errorf("Serial not connected")
	}
	sm.mu.Unlock()

	var resp chan error
	if wait {
		resp = make(chan error, 1)
	}

	sm.sendQ <- GcodeCmd{
		Line:   strings.TrimSpace(line),
		RespCh: resp,
	}
	fmt.Printf("[SendGcode] Added G-code to Queue: %s\n", line)

	if wait {
		return <-resp
	}
	return nil
}

func (sm *SerialManager) SendGcodeLines(lines []string) error {
	for _, line := range lines {
		if err := sm.SendGcode(line, true); err != nil {
			return err
		}
	}
	return nil
}

func (sm *SerialManager) SendGcodePriority(line string, wait bool) error {
	if err := sm.WaitReady(10 * time.Second); err != nil {
		return err
	}
	sm.mu.Lock()
	if sm.state != Connected {
		sm.mu.Unlock()
		return ErrNotConnected
	}
	sm.mu.Unlock()

	var resp chan error
	if wait {
		resp = make(chan error, 1)
	}

	cmd := GcodeCmd{
		Line:   strings.TrimSpace(line),
		RespCh: resp,
	}

	select {
	case sm.priorityQ <- cmd:
		// ok
	default:
		// queue full → non blocking
		if wait {
			return fmt.Errorf("priority queue full")
		}
		fmt.Printf("[SERIAL] priority queue full, dropping command: %s ", cmd.Line)
		return nil
	}

	if wait {
		return <-resp
	}
	return nil
}

func (sm *SerialManager) SendGcodeFile(r io.Reader) error {
	scanner := bufio.NewScanner(r)

	for scanner.Scan() {
		raw := scanner.Text()
		parsed := gcode.ParseLine(raw)
		fmt.Printf("[SendGcodeFile] Sending G-code %s\n", parsed.Raw)
		if !parsed.HasCommand {
			continue
		}

		if err := sm.SendGcode(parsed.Raw, true); err != nil {
			return err
		}
	}

	return scanner.Err()
}

func (sm *SerialManager) SendGcodeFileWithContext(
	ctx context.Context,
	r io.Reader,
	onLine func(sent int, total int, parsed gcode.Line),
) error {

	scanner := bufio.NewScanner(r)

	// Count Lines
	lines := 0
	buf, _ := io.ReadAll(r)
	for _, b := range bytes.Split(buf, []byte("\n")) {
		if len(bytes.TrimSpace(b)) > 0 {
			lines++
		}
	}

	// Reader
	scanner = bufio.NewScanner(bytes.NewReader(buf))

	sent := 0

	for {

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// If paused, wait, and continue the loop from start
		if sm.paused {
			time.Sleep(100 * time.Millisecond)
			fmt.Printf("[SendGcodeFileWithContext] Pause 1 \n")
			continue
		}

		// Read line
		if !scanner.Scan() {
			break
		}

		raw := scanner.Text()
		parsed := gcode.ParseLine(raw)

		if !parsed.HasCommand {
			fmt.Printf("[SendGcodeFileWCtx] Skipping non G-code: %s\n", parsed.Raw)
			continue
		}

		// fmt.Printf("[SendGcodeFileWCtx] Adding G-code to Q %s\n", parsed.Raw)

		// // Notify gcode to JobManager
		// if onGcode != nil {
		// 	onGcode(parsed.Raw)
		// }

		// Check pause again
		for sm.paused {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
				time.Sleep(100 * time.Millisecond)
				fmt.Printf("[SendGcodeFileWithContext] Pause 2 \n")
			}
		}

		if err := sm.SendGcode(parsed.Raw, true); err != nil {
			return err
		}

		sent++
		// fmt.Printf("[SendGcodeFileWCtx] Calling onLine \n")

		if onLine != nil {
			onLine(sent, lines, parsed)
		}

		// fmt.Printf("[SendGcodeFileWCtx] Post onLine \n")

	}

	return scanner.Err()
}

func (sm *SerialManager) IsConnected() bool {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	return sm.state == Connected
}

func (sm *SerialManager) SetPaused(p bool) {
	sm.paused = p
}

func (sm *SerialManager) IsPaused() bool {
	return sm.paused
}

func (sm *SerialManager) handleAction(action string) {
	fmt.Printf("[SERIAL] Action received: %s\n", action)

	if sm.OnAction != nil {
		sm.OnAction(action)
	}
}

func parseTemps(line string) (TempState, bool) {
	var t TempState

	_, err := fmt.Sscanf(
		line,
		"ok T:%f /%f B:%f /%f",
		&t.ToolActual,
		&t.ToolTarget,
		&t.BedActual,
		&t.BedTarget,
	)

	if err == nil {
		t.Valid = true
		return t, true
	}

	_, err = fmt.Sscanf(
		line,
		"T:%f /%f B:%f /%f",
		&t.ToolActual,
		&t.ToolTarget,
		&t.BedActual,
		&t.BedTarget,
	)

	if err == nil {
		t.Valid = true
		return t, true
	}

	return t, false
}

func (sm *SerialManager) tempLoop() {
	fmt.Println("[SERIAL] tempLoop started")

	for {
		select {
		case <-sm.stopCh:
			fmt.Println("[SERIAL] tempLoop stopped")
			return

		default:
			if sm.IsConnected() {
				sm.SendGcodePriority("M105", false)
			}

			time.Sleep(sm.tempPollDelay())
		}
	}
}

func (sm *SerialManager) tempPollDelay() time.Duration {
	if sm.IsPrinting() {
		return 2 * time.Second
	}

	return 5 * time.Second
}

func (sm *SerialManager) IsPrinting() bool {
	//job := sm.jobManager.Snapshot()
	//return job.Active && !job.Paused
	return !sm.paused
}

func (sm *SerialManager) resolvePortLocked() {

	ports := sm.AvailablePorts()

	if len(ports) == 0 {
		return
	}

	for _, p := range ports {
		if p == sm.cfg.Port {
			return
		}
	}

	log.Printf("[SERIAL] Port %q not found, using %q",
		sm.cfg.Port,
		ports[0],
	)

	sm.cfg.Port = ports[0]
}

func (sm *SerialManager) AvailablePorts() []string {
	ports, err := serial.ListPorts()
	if err != nil {
		log.Printf("[SERIAL] Failed to enumerate ports: %v", err)
		return nil
	}

	return sm.FilterPorts(ports)
}

func (sm *SerialManager) flushInbound() {
	for {
		select {
		case <-sm.inbound:
		default:
			return
		}
	}
}

func drainQueue(ch chan GcodeCmd) {
	for len(ch) > 0 {
		<-ch
	}
}

func (sm *SerialManager) resetQueues() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	fmt.Println("[SERIAL] Resetting command queues")

	drainQueue(sm.sendQ)
	drainQueue(sm.priorityQ)
}

func (sm *SerialManager) ResetQueues() {
	sm.resetQueues()
}

func (sm *SerialManager) WaitReady(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	for {
		sm.mu.Lock()
		state := sm.state
		sm.mu.Unlock()

		switch state {
		case Connected:
			return nil

		case Initializing:
			if time.Now().After(deadline) {
				return fmt.Errorf("initialization timeout")
			}
			time.Sleep(100 * time.Millisecond)

		default:
			return ErrNotConnected
		}
	}
}

func (sm *SerialManager) finishInitialization() {

	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.state != Initializing {
		return
	}

	sm.initDone.Do(func() {
		sm.state = Connected
		fmt.Printf("[SERIAL] Initialization complete, state -> CONNECTED\n")
	})
}

// Some firmwares emit startup messages for several seconds after the
// serial port opens. The printer is considered ready only after no
// new startup output has been received for this duration.
func (sm *SerialManager) resetInitTimer() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.initTimer != nil {
		sm.initTimer.Stop()
	}

	sm.initTimer = time.AfterFunc(sm.InitTimeout, func() {
		sm.finishInitialization()
	})
}

func (sm *SerialManager) commandTimeout(cmd string) time.Duration {

	cmd = strings.TrimSpace(strings.ToUpper(cmd))

	switch {

	// Homing
	case strings.HasPrefix(cmd, "G28"):
		return 2 * time.Minute

	// Auto Bed Leveling
	case strings.HasPrefix(cmd, "G29"):
		return 5 * time.Minute

	// Esperar temperatura
	case strings.HasPrefix(cmd, "M109"),
		strings.HasPrefix(cmd, "M190"):
		return 15 * time.Minute

	default:
		return sm.CommandTimeout
	}
}

func (sm *SerialManager) AutoConnect() bool {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	return sm.autoConnect
}

func (sm *SerialManager) SetAutoConnect(enabled bool) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.autoConnect = enabled
}
