package printer

import (
	"strings"
	"sync"
)

type PrinterProtoState int

const (
	ProtoIdle PrinterProtoState = iota
	ProtoPrinting
	ProtoPaused
	ProtoError
)

type PrinterProtocol struct {
	link *PrinterLink

	state PrinterProtoState
	mu    sync.Mutex

	waitOK chan struct{}

	// observabilidad
	lastReply string
}

func NewPrinterProtocol(link *PrinterLink) *PrinterProtocol {
	return &PrinterProtocol{
		link:   link,
		state:  ProtoIdle,
		waitOK: make(chan struct{}, 1),
	}
}

func (pp *PrinterProtocol) Start() {
	go pp.recvLoop()
}

func (pp *PrinterProtocol) recvLoop() {
	for line := range pp.link.Recv() {
		pp.handleLine(line)
	}
}

func (pp *PrinterProtocol) handleLine(line string) {
	pp.mu.Lock()
	pp.lastReply = line
	pp.mu.Unlock()

	l := strings.ToLower(line)

	switch {
	case strings.HasPrefix(l, "ok"):
		select {
		case pp.waitOK <- struct{}{}:
		default:
		}

	case strings.Contains(l, "error"):
		pp.setState(ProtoError)

	case strings.Contains(l, "busy"):
		// ignoramos, el printer sigue vivo
	}
}

func (pp *PrinterProtocol) setState(s PrinterProtoState) {
	pp.mu.Lock()
	defer pp.mu.Unlock()
	pp.state = s
}

func (pp *PrinterProtocol) SendAndWait(cmd string) error {
	pp.link.Send(cmd)

	select {
	case <-pp.waitOK:
		return nil
	}
}

// Exposed functions

func (pp *PrinterProtocol) Inject(cmd string) {
	pp.link.SendPriority(cmd)
}

func (pp *PrinterProtocol) State() PrinterProtoState {
	pp.mu.Lock()
	defer pp.mu.Unlock()
	return pp.state
}

func (pp *PrinterProtocol) LastReply() string {
	pp.mu.Lock()
	defer pp.mu.Unlock()
	return pp.lastReply
}
