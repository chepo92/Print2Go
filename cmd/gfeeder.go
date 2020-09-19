package main

import (
	"github.com/tarm/serial"
	"gitlab.com/adrian_blx/gfeeder/lib/gfeeder"

	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
)

var (
	flagPort  = flag.String("port", "", "port to use")
	flagBaud  = flag.Int("baud", 115200, "baud rate of -port")
	flagGcode = flag.String("gcode", "", "file containing gcode")
)

func main() {
	flag.Parse()

	if *flagPort == "" {
		xdie("-port flag must be specified")
	}
	if *flagGcode == "" {
		xdie("-gcode flag must be specified")
	}

	c := &serial.Config{Name: *flagPort, Baud: *flagBaud}
	s, err := serial.OpenPort(c)
	if err != nil {
		log.Fatal(err)
	}

	fh, err := os.Open(*flagGcode)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf(">>> opened port %v, feeding %s\n", s, *flagGcode)

	stat, _ := fh.Stat()
	ctx := context.Background()
	gf := gfeeder.New(s, fh, gfeeder.Logger(log.New(os.Stderr, "gfeeder ", 0)))
	cb := &cbx{
		fh:   fh,
		size: stat.Size(),
		gf:   gf,
	}

	gfeeder.Callback(cb.Callback)(gf)
	gf.Start(ctx)
}

type cbx struct {
	gf   *gfeeder.Gfeeder
	fh   io.ReadSeeker
	size int64
	done float64
}

func (x *cbx) Callback(v *gfeeder.CallbackData) {
	pos, _ := x.fh.Seek(0, os.SEEK_CUR)
	if pos == 0 {
		pos = 1
	}
	pct := (float64)(pos) / (float64)(x.size)
	if pct != x.done {
		x.done = pct
		x.gf.Echo(fmt.Sprintf("Progress: %v", x.done))
	}
	fmt.Printf("Callback received at %d of %d: %+v (%v)\n", pos, x.size, v, pct)

}

func xdie(str string) {
	fmt.Printf("%s\n", str)
	flag.PrintDefaults()
	os.Exit(1)
}
