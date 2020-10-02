package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"time"

	"gitlab.com/adrian_blx/takoprint/lib/store/localstore"
	"gitlab.com/adrian_blx/takoprint/lib/svc"
	"gitlab.com/adrian_blx/takoprint/lib/task"
)

var (
	flagTTY     = flag.String("tty", "/dev/ttyUSB0", "tty to use")
	flagBaud    = flag.Int("baud", 115200, "baud rate of -port")
	flagGcode   = flag.String("gcode", "", "file containing gcode")
	flagListen  = flag.String("listen", "127.0.0.1:5001", "ip:port to bind to")
	flagStorage = flag.String("storage", "/tmp/takoprint", "path to store gcode in")
)

func main() {
	flag.Parse()

	if *flagTTY == "" {
		xdie("-tty must be specified")
	}
	if *flagGcode != "" {
		oneshotPrint()
		return
	}

	srv := &http.Server{
		Addr: *flagListen,
	}

	s := svc.New(srv, localstore.New(*flagStorage), serialPort)
	if err := s.Run(); err != nil {
		xdie("server exited: %v", err)
	}
}

func oneshotPrint() {
	log.Printf("Printing '%s' on %s\n", *flagGcode, *flagTTY)
	t := task.New()
	p, err := serialPort()
	if err != nil {
		xdie("failed to attach serial port: %v", err)
	}
	defer p.Close()

	fh, err := os.Open(*flagGcode)
	if err != nil {
		xdie("failed to open gcode: %v", err)
	}
	defer fh.Close()

	err = t.Launch(p, fh)
	if err != nil {
		xdie("task setup failed: %v", err)
	}

	for !t.Done() {
		time.Sleep(time.Second)
		log.Printf("working...\n")
	}
}

type SerialPort struct {
	context context.Context
	cancel  context.CancelFunc
	stdout  io.ReadCloser
	stdin   io.WriteCloser
}

func serialPort() (io.ReadWriteCloser, error) {
	ctx, cancel := context.WithCancel(context.Background())
	fmt.Printf("> FIXME: BAUD RATE\n")
	cmd := exec.CommandContext(ctx, "socat", *flagTTY, "-")

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}

	return &SerialPort{
		context: ctx,
		cancel:  cancel,
		stdin:   stdin,
		stdout:  stdout,
	}, cmd.Start()
}

func (sp *SerialPort) Close() error {
	sp.cancel()
	sp.stdin.Close()
	sp.stdout.Close()
	return nil
}

func (sp *SerialPort) Write(b []byte) (int, error) {
	return sp.stdin.Write(b)
}

func (sp *SerialPort) Read(b []byte) (int, error) {
	return sp.stdout.Read(b)
}

func xdie(f string, args ...interface{}) {
	fmt.Printf(f, args...)
	flag.PrintDefaults()
	os.Exit(1)
}
