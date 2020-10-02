package takoprint

type CallbackData struct {
	LastSent string
	NumSent  int
	Reply    string
}

type CallbackDataFunc func(*CallbackData)

func (gf *Gfeeder) fireCallback(l string) {
	if gf.cb == nil {
		return
	}
	gf.RLock()
	v := &CallbackData{
		LastSent: gf.stats.lastCmd,
		NumSent:  gf.stats.numSent,
		Reply:    l,
	}
	gf.RUnlock()
	go gf.cb(v)
}
