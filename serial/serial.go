package serial

import (
	"fmt"

	hwserial "go.bug.st/serial"
)

type SerialConfig struct {
	Port     string
	BaudRate int
}

// func NewSerialPortFunc() func(SerialConfig) (io.ReadWriteCloser, error) {
// 	return func(cfg SerialConfig) (io.ReadWriteCloser, error) {

// 		fmt.Printf("[SERIAL] openFn called: port=%s baud=%d\n",
// 			cfg.Port, cfg.BaudRate)

// 		port, err := OpenSerialPort(cfg)

// 		if err != nil {
// 			fmt.Printf("[SERIAL] openFn ERROR: %v\n", err)
// 			return nil, err
// 		}

// 		fmt.Printf("[SERIAL] openFn SUCCESS\n")
// 		return port, nil
// 	}

// }

func ListPorts() ([]string, error) {
	ports, err := hwserial.GetPortsList()
	fmt.Printf("[SERIAL] Ports: %v\n", ports)
	if err != nil {
		return nil, err
	}
	return ports, nil
}

func OpenSerialPort(cfg SerialConfig) (hwserial.Port, error) {
	mode := &hwserial.Mode{
		BaudRate: cfg.BaudRate,
	}

	fmt.Printf("[SERIAL] Opening port=%s baud=%d\n", cfg.Port, cfg.BaudRate)

	port, err := hwserial.Open(cfg.Port, mode)
	if err != nil {
		return nil, err
	}

	return port, nil
}
