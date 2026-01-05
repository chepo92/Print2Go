package serial

import (
	"bufio"
	"io"
	"strings"
)

func ParseGcode(r io.Reader) ([]string, error) {
	var lines []string
	scanner := bufio.NewScanner(r)

	for scanner.Scan() {
		l := strings.TrimSpace(scanner.Text())
		if l == "" || strings.HasPrefix(l, ";") {
			continue
		}
		lines = append(lines, l)
	}
	return lines, scanner.Err()
}
