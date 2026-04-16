package serialmgr

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
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
	Connected
	Error
)

func (s SerialState) String() string {
	switch s {
	case Disconnected:
		return "Disconnected"
	case Connecting:
		return "Connecting"
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

	port hwserial.Port
	cfg  serial.SerialConfig

	openFn func(serial.SerialConfig) (hwserial.Port, error)

	OnConnected func()

	sendQ     chan GcodeCmd
	priorityQ chan GcodeCmd
	inbound   chan string
	onLine    func(string)

	paused bool
	stopCh chan struct{}
}

type GcodeCmd struct {
	Line     string
	Priority bool
	RespCh   chan error // nil si no se espera ACK
}

func New(openFn func(serial.SerialConfig) (hwserial.Port, error)) *SerialManager {
	return &SerialManager{
		state:     Disconnected,
		openFn:    openFn,
		sendQ:     make(chan GcodeCmd, 16),
		priorityQ: make(chan GcodeCmd, 8),
		inbound:   make(chan string, 16),
		stopCh:    make(chan struct{}),
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
		fmt.Printf("[SERIAL] writeLoop alive \n")
		var cmd GcodeCmd

		select {
		case cmd = <-sm.priorityQ:
			// prioridad absoluta
		default:
			select {
			case cmd = <-sm.priorityQ:
			case cmd = <-sm.sendQ:
			case <-sm.stopCh:
				return
			}
		}

		fmt.Printf("[SERIAL] >> %s\n", cmd.Line)

		_, err := sm.port.Write([]byte(cmd.Line + "\n"))
		if err != nil {
			if cmd.RespCh != nil {
				cmd.RespCh <- err
			}
			continue
		}

		for {
			select {
			case line := <-sm.inbound:
				fmt.Printf("[SERIAL] Reply: %s\n", line)
				if strings.HasPrefix(line, "ok") {
					if cmd.RespCh != nil {
						cmd.RespCh <- nil
					}
					goto nextCmd
				}

				if strings.HasPrefix(line, "error") {
					if cmd.RespCh != nil {
						cmd.RespCh <- fmt.Errorf(line)
					}
					goto nextCmd
				}

				if strings.HasPrefix(line, "Unknown") {
					if cmd.RespCh != nil {
						cmd.RespCh <- nil
					}
					goto nextCmd
				}

				if strings.HasPrefix(line, "echo") {
					fmt.Printf("[SERIAL] Printer echo: %s\n", line)
				}
				if strings.HasPrefix(line, "T") {
					fmt.Printf("[SERIAL] Printer temps: %s\n", line)
				} else {
					fmt.Printf("[SERIAL] Unhandled reply: %s\n", line)
				}

			case <-sm.stopCh:
				fmt.Printf("[SERIAL] writeLoop stopped\n")
				return
			}
		}

	nextCmd:
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

			// Enviar línea cruda al sistema
			select {
			case sm.inbound <- line:
			default:
				fmt.Printf("[SERIAL] inbound buffer full, dropping line\n")
			}

			sm.mu.Lock()
			handler := sm.onLine
			sm.mu.Unlock()

			if handler != nil {
				handler(line)
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

	sm.mu.Unlock()

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
	sm.state = Connected

	// init channels
	sm.sendQ = make(chan GcodeCmd, 32)
	sm.priorityQ = make(chan GcodeCmd, 8)
	sm.inbound = make(chan string, 128)
	sm.stopCh = make(chan struct{})

	fmt.Printf("[SERIAL] Open OK, state -> CONNECTED\n")
	sm.mu.Unlock()

	// Init write and read Loops
	go sm.writeLoop()
	go sm.readLoop()

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
	fmt.Printf("[SendGcode] Added G-code to Q %s\n", line)

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
		// queue full → non blocling
		if wait {
			return fmt.Errorf("priority queue full")
		}
		fmt.Printf("[SERIAL] priority queue full, dropping command: %s", cmd.Line)
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
