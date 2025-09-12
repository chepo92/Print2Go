package printandgo

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"sync"

	"github.com/chepo92/PrintAndGo/chanreader"
)

type PrintAndGo struct {
	sync.RWMutex
	// logger instance
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
	// Error
	err errorDetails
}

type errorDetails struct {
	// number of commands we sent to the printer.
	errFlag bool
	// last command we sent.
	errMsg string
}

type stats struct {
	// number of commands we sent to the printer.
	numSent int
	// last command we sent.
	lastCmd string
}

// New returns a new PrintAndGo instance.
func New(s io.ReadWriteCloser, f io.Reader, opts ...func(*PrintAndGo)) *PrintAndGo {
	tp := &PrintAndGo{
		serialOut: s,
		serialIn:  chanreader.New(s, chanreader.NopFilter()),
		feedIn:    chanreader.New(f, chanreader.GcodeFilter()),
	}
	for _, opt := range opts {
		opt(tp)
	}

	if tp.log == nil {
		tp.log = log.New(os.Stderr, "PrintAndGo: ", 0)
	}
	return tp
}

// Logger configures a custom logger instance.
func Logger(l *log.Logger) func(*PrintAndGo) {
	return func(tp *PrintAndGo) {
		tp.log = l
	}
}

// Callback configures an event callback consumer. Links the provided function to the PrintAndGo structure.
func Callback(cb CallbackDataFunc) func(*PrintAndGo) {
	return func(tp *PrintAndGo) {
		tp.cb = cb
	}
}

func (tp *PrintAndGo) Err() bool {
	return tp.err.errFlag
}

func (tp *PrintAndGo) ErrMsg() string {
	return tp.err.errMsg
}

// Echo prints a string on the printer screen.
func (tp *PrintAndGo) Echo(str string) {
	// TODO: escape str properly
	tp.injectGcode(fmt.Sprintf("M117 %q", str))
}

// Start feeds input data to the serial output.
func (tp *PrintAndGo) Start(ctx context.Context) (err error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()                          // Call cancel when the function returns
	okChan := make(chan bool, 1)            // written to by readPrinter if it accept more data.
	succeedStreamChan := make(chan bool, 1) // written to by feedPrinter if it finished.
	writeErrorChan := make(chan bool, 1)    //
	readErrorChan := make(chan bool, 1)     //
	//sendChan := make(chan bool, 1) // written to by feedPrinter if it has sent data.

	// Force reset

	// give printer some time to become ready.
	tp.waitReady(okChan)

	// Feeds the printer with input from the specified gcode io stream.
	wctx, wcancel := context.WithCancel(ctx)
	go tp.feedPrinter(wctx, wcancel, okChan, succeedStreamChan, writeErrorChan)

	// Reads back messages from the printer and signaling need for new input on okChan.
	rctx, rcancel := context.WithCancel(ctx)
	go tp.readPrinter(rctx, rcancel, okChan, readErrorChan)

	var done bool
	for !done {
		select {
		case <-ctx.Done(): // main context is done, then the for loop exits
			tp.log.Printf("Main context finished")
			// cancel reader contexts
			rcancel()
			// cancel writer contexts
			wcancel()
			done = true
		case <-rctx.Done(): // reader context is done
			fmt.Println("Reader context finished.")
			select {
			case <-writeErrorChan:
				fmt.Println("Write error occurred, probably serial port disconected while writing.")
				tp.err.errFlag = true
				tp.err.errMsg = "Serial port disconected while writing"
			case <-readErrorChan:
				fmt.Println("Read error occurred, probably serial port disconected while reading.")
				tp.err.errFlag = true
				tp.err.errMsg = "Serial port disconected while reading"
				tp.log.Printf("Canceling writer context")
				wcancel()
			case <-succeedStreamChan:
				fmt.Println("Stream Gcode succeeded. Reader Context finished.")
			default:
				fmt.Println("Neither write/read error nor succeed stream occurred.")
			}

			// if tp.cb != nil {
			// 	tp.log.Printf("cbd is not nil, firing error callback")
			// 	tp.cb(&CallbackData{Error: true, Message: "Serial port disconected"})
			// }
			//tp.log.Printf("Serial port disconected, firing error callback")
			//tp.fireErrorCallback("Serial port disconected")

		case <-wctx.Done(): // writer context is done
			fmt.Println("Write context finished.")
			select {
			case <-writeErrorChan:
				fmt.Println("Write error occurred, probably serial port disconected while writing.")
				tp.err.errFlag = true
				tp.err.errMsg = "Serial port disconected while writing"

			case <-readErrorChan:
				fmt.Println("Read error occurred, probably serial port disconected while reading.")
				tp.err.errFlag = true
				tp.err.errMsg = "Serial port disconected while reading"

			case <-succeedStreamChan:
				fmt.Println("Stream Gcode succeeded. Writer will finish everything.")

			default:
				fmt.Println("Neither write/read error nor succeed stream occurred.")

			}
			// Cancel ctx main context to stop everything
			cancel()
		}
	}

	tp.log.Printf("Task finished, returning.\n")

	if tp.err.errFlag {
		return errors.New("something went wrong")
	}
	return nil

}
