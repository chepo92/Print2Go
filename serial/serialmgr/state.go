package serialmgr

import "github.com/chepo92/PrintAndGo/serial"

func (sm *SerialManager) State() SerialState {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	return sm.state
}

func (sm *SerialManager) Config() serial.SerialConfig {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	return sm.cfg
}
