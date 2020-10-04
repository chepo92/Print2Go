package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"time"
)

var (
	flagTTY = flag.String("tty", "/tmp/virtual-tty-2", "path to tty")
)

func main() {
	flag.Parse()

	fh, err := os.OpenFile(*flagTTY, os.O_RDWR, 0755)
	if err != nil {
		panic(err)
	}

	fh.Write([]byte("printer booting blabla\nok\n"))
	s := bufio.NewScanner(fh)
	for s.Scan() {
		fmt.Printf(">> %s\n", s.Text())
		time.Sleep(time.Second)
		fh.Write([]byte("ok\n"))
	}
}
