package serial

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/tarm/serial"
)

// RunPipe opens the supplied tty and pipes data between it and stdin/stdout.
func RunPipe(tty string, baud int) {
	serialPort, err := serial.OpenPort(&serial.Config{
		Name:     tty,
		Baud:     baud,
		Size:     8,
		StopBits: 1,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open serial port %s: %v\n", tty, err)
		os.Exit(1)
	}

	setupSignals(serialPort) // <-- reemplaza la inicialización de señales

	done := make(chan error)
	go func() { _, err := io.Copy(serialPort, os.Stdin); done <- err }()
	go func() { _, err := io.Copy(os.Stdout, serialPort); done <- err }()
	if err := <-done; err != nil {
		fmt.Fprintf(os.Stderr, "Pipe failed: %v\n", err)
		os.Exit(2)
	}
	os.Exit(0)
}

type SerialPort struct {
	stdout  io.ReadCloser
	stdin   io.WriteCloser
	cleanup func()
}

func NewSerialPortFunc(tty string, baud int) func() (io.ReadWriteCloser, error) {
	return func() (io.ReadWriteCloser, error) {
		ctx, cancel := context.WithCancel(context.Background())
		cmd := exec.CommandContext(ctx, os.Args[0], "-tty", tty,
			"-baud", fmt.Sprintf("%d", baud), ":serial-pipe")

		stdout, err := cmd.StdoutPipe()
		if err != nil {
			cancel()
			return nil, err
		}
		stdin, err := cmd.StdinPipe()
		if err != nil {
			cancel()
			return nil, err
		}

		return &SerialPort{
			stdin:  stdin,
			stdout: stdout,
			cleanup: func() {
				cancel()
				stdin.Close()
				stdout.Close()
				cmd.Wait()
			},
		}, cmd.Start()
	}
}

func (sp *SerialPort) Close() error {
	sp.cleanup()
	return nil
}

func (sp *SerialPort) Write(b []byte) (int, error) {
	return sp.stdin.Write(b)
}

func (sp *SerialPort) Read(b []byte) (int, error) {
	return sp.stdout.Read(b)
}
