package serialmgr

import (
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/chepo92/PrintAndGo/serial"
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
	port  io.ReadWriteCloser
	cfg   serial.SerialConfig

	openFn func(serial.SerialConfig) (io.ReadWriteCloser, error)
}

func New(openFn func(serial.SerialConfig) (io.ReadWriteCloser, error)) *SerialManager {
	return &SerialManager{
		state:  Disconnected,
		openFn: openFn,
	}
}

func (sm *SerialManager) Connect(cfg serial.SerialConfig) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// if sm.state == SerialConnected || sm.state == SerialConnecting {
	// 	return nil
	// }

	fmt.Printf("[SERIAL] Connect requested: port=%s baud=%d state=%v\n", cfg.Port, cfg.BaudRate, sm.state)
	if sm.state == Connected || sm.state == Connecting {
		fmt.Printf("[SERIAL] Connect ignored (already connected/connecting)\n")
		return nil
	}

	sm.state = Connecting
	sm.cfg = cfg
	fmt.Printf("[SERIAL] State -> CONNECTING\n")

	port, err := sm.openFn(cfg)
	if err != nil {
		sm.state = Error
		fmt.Printf("[SERIAL] Open FAILED: %v\n", err)
		return err
	}

	sm.port = port
	sm.state = Connected

	fmt.Printf("[SERIAL] Open OK, state -> CONNECTED\n")

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
