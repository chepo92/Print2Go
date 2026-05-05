package gcode

import (
	"bufio"
	"io"
	"strconv"
	"strings"
	"unicode"
)

type Line struct {
	Raw        string           // The raw gcode line as read from the file
	Command    string           // The G-code command (e.g., "G1", "M104")
	HasCommand bool             // If the line has a valid cmd
	Params     map[rune]float64 // Extra params associated with the command: eg X10 Y20
}

// GcodeFilter filters out comments from gcode. repeated stripComments ?
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

func stripComments(s string) string {
	if idx := strings.Index(s, ";"); idx >= 0 {
		s = s[:idx]
	}

	for {
		start := strings.Index(s, "(")
		end := strings.Index(s, ")")
		if start >= 0 && end > start {
			s = s[:start] + s[end+1:]
		} else {
			break
		}
	}
	return strings.TrimSpace(s)
}

func ParseLine(raw string) Line {
	clean := stripComments(raw)

	l := Line{
		Raw:    raw,
		Params: make(map[rune]float64),
	}

	if clean == "" {
		return l
	}

	parts := strings.Fields(clean)

	for _, p := range parts {
		r := rune(p[0])

		if unicode.IsLetter(r) {
			if len(p) == 1 {
				continue
			}
			val, err := strconv.ParseFloat(p[1:], 64)
			if err != nil {
				continue
			}

			if unicode.IsLetter(r) && unicode.IsUpper(r) && l.Command == "" {
				l.Command = p
				l.HasCommand = true
			} else {
				l.Params[r] = val
			}
		}
	}

	return l
}

func serializeParams(p map[rune]float64) string {
	var b strings.Builder
	for k, v := range p {
		b.WriteByte(' ')
		b.WriteRune(k)
		b.WriteString(strings.TrimRight(strings.TrimRight(
			strconv.FormatFloat(v, 'f', 4, 64), "0"), "."))
	}
	return b.String()
}

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
