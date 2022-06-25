package serial

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/tarm/serial"
)

// RunPipe opens the supplied tty and pipes data between it and stdin/stdout.
func RunPipe(tty string, baud int) {
	p, err := serial.OpenPort(&serial.Config{
		Name:     tty,
		Baud:     baud,
		Size:     8,
		StopBits: 1,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to open serial port %s: %v\n", tty, err)
		os.Exit(1)
	}

	sigs := make(chan os.Signal)
	signal.Notify(sigs, syscall.SIGUSR1)
	go handleSignals(sigs, p)

	done := make(chan error)
	go func() { _, err := io.Copy(p, os.Stdin); done <- err }()
	go func() { _, err := io.Copy(os.Stdout, p); done <- err }()
	if err := <-done; err != nil {
		fmt.Fprintf(os.Stderr, "pipe failed: %v\n", err)
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

// Escape hatch for stalled prints: signals can be used to send a command to the printer or pretend that we received an OK.
func handleSignals(s <-chan os.Signal, w io.Writer) {
	for sig := range s {
		switch sig {
		case syscall.SIGUSR1:
			// send 'ok' back to takoprint.
			fmt.Fprintf(os.Stdout, "ok\n")
		default:
			// unhandled.
		}
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
