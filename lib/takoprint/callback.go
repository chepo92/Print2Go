package takoprint

type CallbackData struct {
	LastSent string
	NumSent  int
	Reply    string
}

type CallbackDataFunc func(*CallbackData)

func (tp *Takoprint) fireCallback(l string) {
	if tp.cb == nil {
		return
	}
	tp.RLock()
	cbd := &CallbackData{
		LastSent: tp.stats.lastCmd,
		NumSent:  tp.stats.numSent,
		Reply:    l,
	}
	tp.RUnlock()
	go tp.cb(cbd)
}
