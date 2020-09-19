package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/tarm/serial"
	"gitlab.com/adrian_blx/gfeeder/lib/task"
)

var (
	flagTTY   = flag.String("tty", "/dev/ttyUSB0", "tty to use")
	flagBaud  = flag.Int("baud", 115200, "baud rate of -port")
	flagGcode = flag.String("gcode", "", "file containing gcode")
)

func main() {
	flag.Parse()

	if *flagTTY == "" {
		xdie("-tty must be specified")
	}
	if *flagGcode == "" {
		xdie("-gcode must be specified")
	}

	oneshotPrint()
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

func serialPort() (*serial.Port, error) {
	p, err := serial.OpenPort(&serial.Config{Name: *flagTTY, Baud: *flagBaud})
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
