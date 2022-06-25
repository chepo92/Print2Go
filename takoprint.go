package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"time"

	"git.sr.ht/~adrian-blx/takoprint/lib/camera"
	"git.sr.ht/~adrian-blx/takoprint/lib/serial"
	"git.sr.ht/~adrian-blx/takoprint/lib/store/localstore"
	"git.sr.ht/~adrian-blx/takoprint/lib/task"
	"git.sr.ht/~adrian-blx/takoprint/webapi"
)

var (
	flagTTY      = flag.String("tty", "/dev/ttyUSB0", "tty to use")
	flagBaud     = flag.Int("baud", 115200, "baud rate of -port")
	flagGcode    = flag.String("gcode", "", "file containing gcode")
	flagListen   = flag.String("listen", "127.0.0.1:5001", "ip:port to bind to")
	flagStorage  = flag.String("storage", "/tmp/takoprint", "path to store gcode in")
	flagShutdown = flag.String("shutdown-script", "/usr/lib/takoprint-shutdown.sh", "script to execute to shutdown the printer")
)

func main() {
	flag.Parse()

	if *flagTTY == "" {
		xdie("-tty must be specified")
	}
	if *flagGcode != "" {
		oneshotPrint(*flagTTY, *flagBaud, *flagGcode)
		return
	}

	if os.Args[len(os.Args)-1] == ":serial-pipe" {
		serial.RunPipe(*flagTTY, *flagBaud)
		return
	}
	if os.Args[len(os.Args)-1] == ":camera-pipe" {
		camera.RunPipe("/dev/video0", 640, 480)
		return
	}

	srv := &http.Server{
		Addr: *flagListen,
	}

	spf := serial.NewSerialPortFunc(*flagTTY, *flagBaud)
	s := webapi.New(localstore.New(*flagStorage), spf, shutdownFunc(*flagShutdown))
	log.Printf("Listeing on '%s' using serial port '%s'", *flagListen, *flagTTY)
	if err := s.Run(srv); err != nil {
		xdie("server exited: %v", err)
	}
}

// oneshotPrint just prints the specified gcode file.
func oneshotPrint(tty string, baud int, gcode string) {
	log.Printf("Printing '%s' on %s\n", gcode, tty)

	t := task.New()
	p, err := serial.NewSerialPortFunc(tty, baud)()
	if err != nil {
		xdie("failed to attach serial port: %v", err)
	}
	defer p.Close()

	fh, err := os.Open(gcode)
	if err != nil {
		xdie("failed to open gcode: %v", err)
	}
	defer fh.Close()

	gf, err := localstore.FromFilehandle(fh)
	if err != nil {
		xdie("failed to open stream: %v", err)
	}

	err = t.Launch(p, gf)
	if err != nil {
		xdie("task setup failed: %v", err)
	}

	for !t.Done() {
		time.Sleep(time.Second)
		log.Printf("working...\n")
	}
}

func shutdownFunc(script string) func() {
	return func() {
		if len(script) == 0 {
			return
		}
		fmt.Printf("# shutdown: executing '%s'\n", script)
		cmd := exec.Command(script)
		cmd.Stderr = os.Stderr
		cmd.Stdout = os.Stdout
		rv := cmd.Run()
		fmt.Printf("# exited with: %v\n", rv)
	}
}

func xdie(f string, args ...interface{}) {
	fmt.Printf(f, args...)
	flag.PrintDefaults()
	os.Exit(1)
}
