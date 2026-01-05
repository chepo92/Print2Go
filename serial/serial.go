package serial

import (
	"fmt"
	"io"

	goserial "go.bug.st/serial"
)

type SerialConfig struct {
	Port     string
	BaudRate int
}

// RunPipe opens the supplied tty and pipes data between it and stdin/stdout.
// In other words it is serial monitor (send/receive) for the terminal
// func RunPipe(tty string, baud int) {
// 	serialPort, err := goserial.OpenPort(&goserial.Config{
// 		Name:     tty,
// 		Baud:     baud,
// 		Size:     8,
// 		StopBits: 1,
// 	})
// 	if err != nil {
// 		fmt.Fprintf(os.Stderr, "Failed to open serial port %s: %v\n", tty, err)
// 		os.Exit(1)
// 	}

// 	setupSignals(serialPort) // <-- reemplaza la inicialización de señales

// 	done := make(chan error)
// 	go func() { _, err := io.Copy(serialPort, os.Stdin); done <- err }()
// 	go func() { _, err := io.Copy(os.Stdout, serialPort); done <- err }()
// 	if err := <-done; err != nil {
// 		fmt.Fprintf(os.Stderr, "Pipe failed: %v\n", err)
// 		os.Exit(2)
// 	}
// 	os.Exit(0)
// }

// type SerialPort struct {
// 	stdout  io.ReadCloser
// 	stdin   io.WriteCloser
// 	cleanup func()
// }

// NewSerialPortFunc returns a function that when called opens a serial port connection to the specified tty at the specified baud rate.
// The returned function returns an io.ReadWriteCloser that can be used to read from and write to the serial port.
// It does this by spawning a new instance of the current executable with the -serial-pipe flag, which in turn runs the RunPipe function.
// This allows the serial port to be accessed in a separate process, which can be useful for isolating it from the main application.
// func NewSerialPortFunc(tty string, baud int) func() (io.ReadWriteCloser, error) {
// 	return func() (io.ReadWriteCloser, error) {
// 		ctx, cancel := context.WithCancel(context.Background())
// 		cmd := exec.CommandContext(ctx, os.Args[0], "-tty", tty,
// 			"-baud", fmt.Sprintf("%d", baud), ":serial-pipe")

// 		stdout, err := cmd.StdoutPipe()
// 		if err != nil {
// 			cancel()
// 			return nil, err
// 		}
// 		stdin, err := cmd.StdinPipe()
// 		if err != nil {
// 			cancel()
// 			return nil, err
// 		}

//			return &SerialPort{
//				stdin:  stdin,
//				stdout: stdout,
//				cleanup: func() {
//					cancel()
//					stdin.Close()
//					stdout.Close()
//					cmd.Wait()
//				},
//			}, cmd.Start()
//		}
//	}

func NewSerialPortFunc() func(SerialConfig) (io.ReadWriteCloser, error) {
	return func(cfg SerialConfig) (io.ReadWriteCloser, error) {

		fmt.Printf("[SERIAL] openFn called: port=%s baud=%d\n",
			cfg.Port, cfg.BaudRate)

		port, err := OpenSerialPort(cfg)

		if err != nil {
			fmt.Printf("[SERIAL] openFn ERROR: %v\n", err)
			return nil, err
		}

		fmt.Printf("[SERIAL] openFn SUCCESS\n")
		return port, nil
	}

}

func ListPorts() ([]string, error) {
	ports, err := goserial.GetPortsList()
	if err != nil {
		return nil, err
	}
	return ports, nil
}

// func Open(cfg SerialConfig) (io.ReadWriteCloser, error) {
// 	if cfg.Port == "" {
// 		return nil, fmt.Errorf("serial: port is empty")
// 	}
// 	if cfg.BaudRate <= 0 {
// 		return nil, fmt.Errorf("serial: invalid baudrate")
// 	}

// 	mode := &goserial.Mode{
// 		BaudRate: cfg.BaudRate,
// 	}

// 	port, err := goserial.Open(cfg.Port, mode)
// 	if err != nil {
// 		return nil, err
// 	}

// 	return port, nil
// }

func OpenSerialPort(cfg SerialConfig) (io.ReadWriteCloser, error) {
	mode := &goserial.Mode{
		BaudRate: cfg.BaudRate,
	}

	fmt.Printf("[SERIAL] Opening port=%s baud=%d\n", cfg.Port, cfg.BaudRate)

	port, err := goserial.Open(cfg.Port, mode)
	if err != nil {
		return nil, err
	}

	return port, nil
}

// func (sp *SerialPort) Close() error {
// 	sp.cleanup()
// 	return nil
// }

// func (sp *SerialPort) Write(b []byte) (int, error) {
// 	return sp.stdin.Write(b)
// }

// func (sp *SerialPort) Read(b []byte) (int, error) {
// 	return sp.stdout.Read(b)
// }
