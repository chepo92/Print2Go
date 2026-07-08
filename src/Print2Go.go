package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"

	// "github.com/chepo92/Print2Go/camera"  // Not compatible in windows

	"github.com/chepo92/Print2Go/serial"
	"github.com/chepo92/Print2Go/storage/localstore"
	"github.com/chepo92/Print2Go/webapi"
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
	flagIP       *string
	flagPort     *int
	flagStorage  *string
	flagShutdown *string
	flagMotd     *string
	flagCamera   *string
)

// Parámetros dependientes del sistema operativo
func init() {
	if runtime.GOOS == "windows" {
		defaultTTY = "COM3"
		defaultStorage = "C:\\Print2Go_tmp"
		defaultShutdown = "C:\\Print2Go_shutdown.bat"
		defaultMotd = "NUL"
		defaultCamera = "" // No soportado en Windows
	} else {
		defaultTTY = "/dev/ttyUSB0"
		defaultStorage = "/tmp/Print2Go"
		defaultShutdown = "/usr/lib/print2go-shutdown.sh"
		defaultMotd = "/dev/null"
		defaultCamera = "/dev/video0"
	}

}

// Parametros por defecto de la CLI
func setDefaultFlags() {
	flagTTY = flag.String("tty", defaultTTY, "Port/tty to use")
	flagBaud = flag.Int("baud", 115200, "baud rate of -tty port")
	flagGcode = flag.String("gcode", "", "file containing gcode")
	flagListen = flag.String("listen", "127.0.0.1:5001", "ip:port to bind to")
	flagIP = flag.String("ip", "", "auto | local | <ipv4>. If empty, defaults to local (127.0.0.1)")
	flagPort = flag.Int("port", 5001, "Port to listen on (ignored if -listen is used)")
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
		xdie("-tty Port cannot be blank \n")
	}
	if *flagGcode != "" {
		//oneshotPrint(*flagTTY, *flagBaud, *flagGcode)

		return
	}

	serialCfg := serial.SerialConfig{
		Port:     *flagTTY,
		BaudRate: *flagBaud,
	}

	listenAddr := *flagListen

	// Caso 1: Si el usuario especificó -listen, dejamos eso tal cual
	if listenAddr == "127.0.0.1:5001" && *flagIP != "" {
		// Caso 2: El usuario usa -ip / -port
		var ip string
		switch *flagIP {
		case "":
			ip = "127.0.0.1" // por defecto
		case "local":
			ip = "127.0.0.1"
		case "auto":
			ip = getAutoIP()
		default:
			ip = *flagIP
		}
		listenAddr = fmt.Sprintf("%s:%d", ip, *flagPort)
	}

	// if os.Args[len(os.Args)-1] == ":serial-pipe" {
	// 	serial.RunPipe(*flagTTY, *flagBaud)
	// 	return
	// }
	// if os.Args[len(os.Args)-1] == ":camera-pipe" {
	// 	// camera.RunPipe(*flagCamera, 640, 480) // will not call this func/lib in win
	// 	return
	// }

	// Declare the server
	srv := &http.Server{
		Addr: listenAddr,
	}
	// Setup serial port reader
	//serialPortReader := serial.NewSerialPortFunc()

	//s := webapi.New(*flagCamera, localstore.New(*flagStorage), *flagMotd, spf, shutdownFunc(*flagShutdown))
	// create webapi without camera for windows build
	s := webapi.New(listenAddr, localstore.New(*flagStorage), *flagMotd, serial.OpenSerialPort, serialCfg, shutdownFunc(*flagShutdown))
	//s.SetPortBaudRate(*flagTTY, *flagBaud)
	// Print some info
	log.Printf("Host and API Listening on '%s', using serial port: '%s' at baud: %d", listenAddr, *flagTTY, *flagBaud)
	log.Printf("Storage path is '%s', shutdown script is '%s', motd file is '%s', camera is '%s'", *flagStorage, *flagShutdown, *flagMotd, *flagCamera)

	// Start the server
	if err := s.Run(srv); err != nil {
		xdie("server exited: %v", err)
	}
}

// oneshotPrint just prints the specified gcode file.
// func oneshotPrint(tty string, baud int, gcodeFileName string) {
// 	log.Printf("Printing: '%s' on %s\n", gcodeFileName, tty)
// 	// Create task
// 	task := task.New()
// 	// Open serial port
// 	workingPort, err := serial.NewSerialPortFunc()
// 	if err != nil {
// 		xdie("Failed to attach serial port: %v", err)
// 	}
// 	defer workingPort.Close()
// 	// Open gcode file
// 	fh, err := os.Open(gcodeFileName)
// 	if err != nil {
// 		xdie("Failed to open gcode: %v", err)
// 	}
// 	defer fh.Close()
// 	// Create gcode file stream object
// 	gf, err := localstore.FromFilehandle(fh)
// 	if err != nil {
// 		xdie("Failed to open stream: %v", err)
// 	}
// 	fmt.Printf("Gcode size is: '%d'", gf.Size())

// 	// Start print task asynchronously
// 	err = task.Launch(workingPort, gf)
// 	if err != nil {
// 		xdie("Task setup failed: %v", err)
// 	}
// 	// Wait until done
// 	for !task.Done() {
// 		time.Sleep(time.Second)
// 		log.Printf("Working...\n")
// 	}
// }

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

func getAutoIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "127.0.0.1"
	}
	defer conn.Close()
	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String()
}
