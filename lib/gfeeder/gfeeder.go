package gfeeder

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"sync"

	"gitlab.com/adrian_blx/gfeeder/lib/chanreader"
)

type Gfeeder struct {
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

// New returns a new gfeeder instance.
func New(s io.ReadWriteCloser, f io.Reader, opts ...func(*Gfeeder)) *Gfeeder {
	gf := &Gfeeder{
		serialOut: s,
		serialIn:  chanreader.New(s, chanreader.NopFilter()),
		feedIn:    chanreader.New(f, chanreader.GcodeFilter()),
	}
	for _, opt := range opts {
		opt(gf)
	}

	if gf.log == nil {
		gf.log = log.New(os.Stderr, "gfeeder ", 0)
	}
	return gf
}

// Logger configures a custom logger instance.
func Logger(l *log.Logger) func(*Gfeeder) {
	return func(g *Gfeeder) {
		g.log = l
	}
}

// Callback configures an event callback consumer
func Callback(cb CallbackDataFunc) func(*Gfeeder) {
	return func(g *Gfeeder) {
		g.cb = cb
	}
}

// Echo prints a string on the printer screen.
func (gf *Gfeeder) Echo(str string) {
	// TODO: escape.
	gf.injectGcode(fmt.Sprintf("M117 %s", str))
}

// Start feeds input data to the serial output.
func (gf *Gfeeder) Start(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	okChan := make(chan bool, 1) // written to by readPrinter if it accept more data.

	// give printer some time to become ready.
	gf.waitReady()

	// mark printer as ready for first command.
	okChan <- true

	// Feeds the printer with input from the specified gcode io stream.
	wctx, wcancel := context.WithCancel(ctx)
	go gf.feedPrinter(wctx, wcancel, okChan)

	// Reads back messages from the printer and signaling need for new input on okChan.
	rctx, rcancel := context.WithCancel(ctx)
	go gf.readPrinter(rctx, rcancel, okChan)

	var done bool
	for !done {
		select {
		case <-ctx.Done():
			gf.log.Printf("main context finished")
			rcancel()
			wcancel()
			done = true
		case <-rctx.Done():
			gf.log.Printf("serial port vanished.")
			wcancel()
		case <-wctx.Done():
			gf.log.Printf("gcode input stream is done")
			cancel()
		}
	}

	gf.log.Printf("Print is done, returning.\n")
}
