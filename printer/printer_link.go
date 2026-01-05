package printer

import (
	"bufio"
	"fmt"
	"io"
	"sync"
)

type PrinterLink struct {
	port io.ReadWriteCloser

	// canales de entrada
	sendNormal   chan string
	sendPriority chan string

	// canal de salida
	recv chan string

	// control
	done chan struct{}
	wg   sync.WaitGroup
}

func NewPrinterLink(port io.ReadWriteCloser) *PrinterLink {
	return &PrinterLink{
		port:         port,
		sendNormal:   make(chan string, 64),
		sendPriority: make(chan string, 16),
		recv:         make(chan string, 64),
		done:         make(chan struct{}),
	}
}

func (pl *PrinterLink) Start() {
	pl.wg.Add(2)
	go pl.readerLoop()
	go pl.writerLoop()
}

func (pl *PrinterLink) Close() {
	close(pl.done)
	pl.wg.Wait()
	pl.port.Close()
}

func (pl *PrinterLink) writerLoop() {
	defer pl.wg.Done()

	for {
		select {
		case <-pl.done:
			return

		// PRIORIDAD SIEMPRE PRIMERO
		case cmd := <-pl.sendPriority:
			pl.writeLine(cmd)

		default:
			select {
			case <-pl.done:
				return
			case cmd := <-pl.sendPriority:
				pl.writeLine(cmd)
			case cmd := <-pl.sendNormal:
				pl.writeLine(cmd)
			}
		}
	}
}

func (pl *PrinterLink) writeLine(cmd string) {
	if len(cmd) == 0 {
		return
	}
	fmt.Fprintf(pl.port, "%s\n", cmd)
}

func (pl *PrinterLink) readerLoop() {
	defer pl.wg.Done()

	scanner := bufio.NewScanner(pl.port)

	for scanner.Scan() {
		select {
		case pl.recv <- scanner.Text():
		case <-pl.done:
			return
		}
	}

	// error o EOF
	close(pl.recv)
}

// / API pública mínima
func (pl *PrinterLink) Send(cmd string) {
	pl.sendNormal <- cmd
}

func (pl *PrinterLink) SendPriority(cmd string) {
	pl.sendPriority <- cmd
}

func (pl *PrinterLink) Recv() <-chan string {
	return pl.recv
}
