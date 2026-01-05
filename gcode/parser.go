package gcode

import (
	"strconv"
	"strings"
	"unicode"
)

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
