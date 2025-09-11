package printandgo

type CallbackData struct {
	LastSent string
	NumSent  int
	Reply    string
	Error    bool
	Message  string
}

type CallbackDataFunc func(*CallbackData)

// fireCallback fires the callback function in a new goroutine, if set,
// data includes the last sent command, the number of sent commands and the reply, as is normal operation error and message from the printer are false and empty.
func (tp *PrintAndGo) fireCallback(l string) {
	if tp.cb == nil {
		return
	}
	tp.RLock()
	cbd := &CallbackData{
		LastSent: tp.stats.lastCmd,
		NumSent:  tp.stats.numSent,
		Reply:    l,
		Error:    false,
		Message:  "",
	}
	tp.RUnlock()
	go tp.cb(cbd)
}

// fireCallback fires the callback function in a new goroutine, if set,
// data includes the last sent command, the number of sent commands and the reply, error and message from the printer
func (tp *PrintAndGo) fireErrorCallback(er string) {
	if tp.cb == nil {
		return
	}
	tp.RLock()
	cbd := &CallbackData{
		LastSent: tp.stats.lastCmd,
		NumSent:  tp.stats.numSent,
		Reply:    "",
		Error:    true,
		Message:  er,
	}
	tp.RUnlock()
	go tp.cb(cbd)
}
