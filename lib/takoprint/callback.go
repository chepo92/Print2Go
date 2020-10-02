package takoprint

type CallbackData struct {
	LastSent string
	NumSent  int
	Reply    string
}

type CallbackDataFunc func(*CallbackData)

func (tf *Takoprint) fireCallback(l string) {
	if tf.cb == nil {
		return
	}
	tf.RLock()
	v := &CallbackData{
		LastSent: tf.stats.lastCmd,
		NumSent:  tf.stats.numSent,
		Reply:    l,
	}
	tf.RUnlock()
	go tf.cb(v)
}
