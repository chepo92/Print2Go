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

	// "github.com/chepo92/PrintAndGo/camera"  // Not compatible in windows

	"github.com/chepo92/PrintAndGo/serial"
	"github.com/chepo92/PrintAndGo/store/localstore"
	"github.com/chepo92/PrintAndGo/task"
	"github.com/chepo92/PrintAndGo/webapi"
)

var (
	defaultTTY      string
	defaultStorage  string
	defaultShutdown string
	defaultMotd     string
	defaultCamera   string
)

var (
	flagTTY      *string
	flagBaud     *int
	flagGcode    *string
	flagListen   *string
	flagStorage  *string
	flagShutdown *string
	flagMotd     *string
	flagCamera   *string
)

// Parametros dependientes del sistema operativo
func init() {
	if runtime.GOOS == "windows" {
		defaultTTY = "COM3"
		defaultStorage = "C:\\PrintAndGo_tmp"
		defaultShutdown = "C:\\PrintAndGo_shutdown.bat"
		defaultMotd = "NUL"
		defaultCamera = "" // No soportado en Windows
	} else {
		defaultTTY = "/dev/ttyUSB0"
		defaultStorage = "/tmp/PrintAndGo"
		defaultShutdown = "/usr/lib/printandgo-shutdown.sh"
		defaultMotd = "/dev/null"
		defaultCamera = "/dev/video0"
	}

}

// Parametros por defecto de la CLI
func setDefaultFlags() {
	flagTTY = flag.String("tty", defaultTTY, "Port/tty to use")
	flagBaud = flag.Int("baud", 115200, "baud rate of -port")
	flagGcode = flag.String("gcode", "", "file containing gcode")
	flagListen = flag.String("listen", "127.0.0.1:5001", "ip:port to bind to")
	flagStorage = flag.String("storage", defaultStorage, "path to store gcode in")
	flagShutdown = flag.String("shutdown-script", defaultShutdown, "script to execute to shutdown the printer")
	flagMotd = flag.String("motd-file", defaultMotd, "Message of the day to display on the UI")
	flagCamera = flag.String("camera", defaultCamera, "Camera device")

}

// Funcion principal
func main() {
	setDefaultFlags()

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

	// Declare the server
	srv := &http.Server{
		Addr: *flagListen,
	}
	// Setup serial port reader
	serialPortReader := serial.NewSerialPortFunc(*flagTTY, *flagBaud)

	//s := webapi.New(*flagCamera, localstore.New(*flagStorage), *flagMotd, spf, shutdownFunc(*flagShutdown))
	// create webapi without camera for windows build
	s := webapi.New(localstore.New(*flagStorage), *flagMotd, serialPortReader, shutdownFunc(*flagShutdown))
	s.SetPortBaudRate(*flagTTY, *flagBaud)
	// Print some info
	log.Printf("Listening on '%s' using serial port '%s' at baud %d", *flagListen, *flagTTY, *flagBaud)
	log.Printf("Storage path is '%s', shutdown script is '%s', motd file is '%s', camera is '%s'", *flagStorage, *flagShutdown, *flagMotd, *flagCamera)

	// Start the server
	if err := s.Run(srv); err != nil {
		xdie("server exited: %v", err)
	}
}

// oneshotPrint just prints the specified gcode file.
func oneshotPrint(tty string, baud int, gcodeFileName string) {
	log.Printf("Printing: '%s' on %s\n", gcodeFileName, tty)
	// Create task
	task := task.New()
	// Open serial port
	p, err := serial.NewSerialPortFunc(tty, baud)()
	if err != nil {
		xdie("Failed to attach serial port: %v", err)
	}
	defer p.Close()
	// Open gcode file
	fh, err := os.Open(gcodeFileName)
	if err != nil {
		xdie("Failed to open gcode: %v", err)
	}
	defer fh.Close()
	// Create gcode file stream object
	gf, err := localstore.FromFilehandle(fh)
	if err != nil {
		xdie("Failed to open stream: %v", err)
	}
	fmt.Printf("Gcode size is: '%d'", gf.Size())

	// Start print task asynchronously
	err = task.Launch(p, gf)
	if err != nil {
		xdie("Task setup failed: %v", err)
	}
	// Wait until done
	for !task.Done() {
		time.Sleep(time.Second)
		log.Printf("Working...\n")
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
