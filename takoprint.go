package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"time"

	// "git.sr.ht/~adrian-blx/takoprint/camera"  // Not compatible in windows

	"git.sr.ht/~adrian-blx/takoprint/serial"
	"git.sr.ht/~adrian-blx/takoprint/store/localstore"
	"git.sr.ht/~adrian-blx/takoprint/task"
	"git.sr.ht/~adrian-blx/takoprint/webapi"
)

var (
	defaultTTY      string
	defaultStorage  string
	defaultShutdown string
	defaultMotd     string
	defaultCamera   string
)

func init() {
	if runtime.GOOS == "windows" {
		defaultTTY = "COM3"
		defaultStorage = "C:\\takoprint_tmp"
		defaultShutdown = "C:\\takoprint_shutdown.bat"
		defaultMotd = "NUL"
		defaultCamera = "" // No soportado en Windows
	} else {
		defaultTTY = "/dev/ttyUSB0"
		defaultStorage = "/tmp/takoprint"
		defaultShutdown = "/usr/lib/takoprint-shutdown.sh"
		defaultMotd = "/dev/null"
		defaultCamera = "/dev/video0"
	}
}

var (
	flagTTY      = flag.String("tty", defaultTTY, "Port/tty to use")
	flagBaud     = flag.Int("baud", 115200, "baud rate of -port")
	flagGcode    = flag.String("gcode", "", "file containing gcode")
	flagListen   = flag.String("listen", "127.0.0.1:5001", "ip:port to bind to")
	flagStorage  = flag.String("storage", defaultStorage, "path to store gcode in")
	flagShutdown = flag.String("shutdown-script", defaultShutdown, "script to execute to shutdown the printer")
	flagMotd     = flag.String("motd-file", defaultMotd, "Message of the day to display on the UI")
	flagCamera   = flag.String("camera", defaultCamera, "Camera device")
)

func main() {
	flag.Parse()

	if *flagTTY == "" {
		xdie("-tty Port must be specified")
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
		// camera.RunPipe(*flagCamera, 640, 480) // will not call this func/lib in win
		return
	}

	srv := &http.Server{
		Addr: *flagListen,
	}

	spf := serial.NewSerialPortFunc(*flagTTY, *flagBaud)
	//s := webapi.New(*flagCamera, localstore.New(*flagStorage), *flagMotd, spf, shutdownFunc(*flagShutdown))
	s := webapi.New(localstore.New(*flagStorage), *flagMotd, spf, shutdownFunc(*flagShutdown))
	log.Printf("Listening on '%s' using serial port '%s'", *flagListen, *flagTTY)
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
