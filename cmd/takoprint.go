package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/jacobsa/go-serial/serial"
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

	task, err := task.New(p, fh)
	if err != nil {
		xdie("task setup failed: %v", err)
	}

	for !task.Done() {
		time.Sleep(time.Second)
		log.Printf("%s", task.Describe())
	}
}

func serialPort() (io.ReadWriteCloser, error) {
	fmt.Printf("++++ opening serial port\n")
	p, err := serial.Open(serial.OpenOptions{
		PortName:        *flagTTY,
		BaudRate:        uint(*flagBaud),
		StopBits:        1,
		DataBits:        8,
		MinimumReadSize: 1,
	})
	if err != nil {
		return nil, err
	}
	return p, nil
}

func xdie(f string, args ...interface{}) {
	fmt.Printf(f, args...)
	flag.PrintDefaults()
	os.Exit(1)
}
