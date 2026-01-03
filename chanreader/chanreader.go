package chanreader

import (
	"fmt"
	"io"
	"strings"
	"time"
)

// chanreader.New wraps an io.Reader into a channel, closes it on error.
func New(r io.Reader, filter func(string) bool) chan string {
	ch := make(chan string)
	go func() {
		// rs := bufio.NewScanner(r)
		// for rs.Scan() {
		// 	if filter(rs.Text()) {
		// 		continue
		// 	}
		// 	fmt.Printf("Read: %s \n", rs.Text())
		// 	ch <- rs.Text()
		// }
		// fmt.Println("chanreader: closed, closing channel")
		// close(ch)
		time.Sleep(10 * time.Millisecond)
		fmt.Println("chanreader: closed, closing channel")

	}()
	return ch
}

// GcodeFilter filters out comments from gcode.
func GcodeFilter() func(string) bool {
	return func(s string) bool {
		if len(strings.Trim(s, " ")) == 0 {
			return true
		}
		if strings.HasPrefix(s, ";") {
			return true
		}
		return false
	}
}

// NopFilter never filters.
func NopFilter() func(string) bool {
	return func(_ string) bool {
		return false
	}
}
