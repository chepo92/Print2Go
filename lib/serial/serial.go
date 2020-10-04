package serial

import (
	"fmt"
	"io"
	"os"

	"context"
	"github.com/jacobsa/go-serial/serial"
	"os/exec"
)

// RunPipe opens the supplied tty and pipes data between it and stdin/stdout.
func RunPipe(tty string, baud uint) {
	p, err := serial.Open(serial.OpenOptions{
		PortName:        tty,
		BaudRate:        baud,
		StopBits:        1,
		DataBits:        8,
		MinimumReadSize: 1,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to open serial port: %v\n", err)
		os.Exit(1)
	}

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
	context context.Context
	cancel  context.CancelFunc
	stdout  io.ReadCloser
	stdin   io.WriteCloser
}

func NewSerialPortFunc(tty string, baud int) func() (io.ReadWriteCloser, error) {
	return func() (io.ReadWriteCloser, error) {
		ctx, cancel := context.WithCancel(context.Background())
		cmd := exec.CommandContext(ctx, os.Args[0], "-tty", tty,
			"-baud", fmt.Sprintf("%d", baud), ":serial-pipe")

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
