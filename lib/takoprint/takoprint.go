package takoprint

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"sync"

	"github.com/chepo92/PrintAndGo/chanreader"
)

type Takoprint struct {
	sync.RWMutex
	log *log.Logger
	// input from serial port
	serialIn <-chan string
	// writer to send data to serial port
	serialOut io.Writer
	// gcode input feed
	feedIn chan string
	// runtime statistics.
	stats stats
	// callback function, may be nil.
	cb CallbackDataFunc
}

type stats struct {
	// number of commands we sent to the printer.
	numSent int
	// last command we sent.
	lastCmd string
}

// New returns a new takoprint instance.
func New(s io.ReadWriteCloser, f io.Reader, opts ...func(*Takoprint)) *Takoprint {
	tp := &Takoprint{
		serialOut: s,
		serialIn:  chanreader.New(s, chanreader.NopFilter()),
		feedIn:    chanreader.New(f, chanreader.GcodeFilter()),
	}
	for _, opt := range opts {
		opt(tp)
	}

	if tp.log == nil {
		tp.log = log.New(os.Stderr, "takoprint ", 0)
	}
	return tp
}

// Logger configures a custom logger instance.
func Logger(l *log.Logger) func(*Takoprint) {
	return func(tp *Takoprint) {
		tp.log = l
	}
}

// Callback configures an event callback consumer
func Callback(cb CallbackDataFunc) func(*Takoprint) {
	return func(tp *Takoprint) {
		tp.cb = cb
	}
}

// Echo prints a string on the printer screen.
func (tp *Takoprint) Echo(str string) {
	// TODO: escape.
	tp.injectGcode(fmt.Sprintf("M117 %s", str))
}

// Start feeds input data to the serial output.
func (tp *Takoprint) Start(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	okChan := make(chan bool, 1) // written to by readPrinter if it accept more data.

	// give printer some time to become ready.
	tp.waitReady()

	// mark printer as ready for first command.
	okChan <- true

	// Feeds the printer with input from the specified gcode io stream.
	wctx, wcancel := context.WithCancel(ctx)
	go tp.feedPrinter(wctx, wcancel, okChan)

	// Reads back messages from the printer and signaling need for new input on okChan.
	rctx, rcancel := context.WithCancel(ctx)
	go tp.readPrinter(rctx, rcancel, okChan)

	var done bool
	for !done {
		select {
		case <-ctx.Done():
			tp.log.Printf("main context finished")
			rcancel()
			wcancel()
			done = true
		case <-rctx.Done():
			tp.log.Printf("serial port vanished.")
			wcancel()
		case <-wctx.Done():
			tp.log.Printf("gcode input stream is done")
			cancel()
		}
	}

	tp.log.Printf("Print is done, returning.\n")
}
