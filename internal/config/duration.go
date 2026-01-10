package config

import (
	"fmt"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"strconv"
)

// parseDurationExtended parses Go-style duration strings and adds support for:
// - d (days) where 1d = 24h
// - w (weeks) where 1w = 7d
//
// Examples: "168h", "7d", "1w2d", "1.5d", "-2w".
func parseDurationExtended(raw string) (time.Duration, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, fmt.Errorf("duration is required")
	}

	// If there are no day/week units, defer entirely to Go.
	if !strings.ContainsAny(raw, "dw") {
		return time.ParseDuration(raw)
	}

	expanded, err := expandDaysWeeksToHours(raw)
	if err != nil {
		return 0, err
	}
	return time.ParseDuration(expanded)
}

func expandDaysWeeksToHours(raw string) (string, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return "", fmt.Errorf("duration is required")
	}

	var b strings.Builder
	// Keep the leading sign, if any.
	if strings.HasPrefix(s, "+") || strings.HasPrefix(s, "-") {
		b.WriteByte(s[0])
		s = s[1:]
		if s == "" {
			return "", fmt.Errorf("invalid duration %q", raw)
		}
	}

	for len(s) > 0 {
		// Number: [0-9]+(\.[0-9]+)?
		i := 0
		dotSeen := false
		for i < len(s) {
			c := s[i]
			if c >= '0' && c <= '9' {
				i++
				continue
			}
			if c == '.' && !dotSeen {
				dotSeen = true
				i++
				continue
			}
			break
		}
		if i == 0 {
			return "", fmt.Errorf("invalid duration %q", raw)
		}
		numStr := s[:i]
		num, err := strconv.ParseFloat(numStr, 64)
		if err != nil {
			return "", fmt.Errorf("invalid duration %q", raw)
		}
		s = s[i:]

		// Unit: one or more letters (incl. µ)
		if s == "" {
			return "", fmt.Errorf("invalid duration %q", raw)
		}
		j := 0
		for j < len(s) {
			r, size := utf8.DecodeRuneInString(s[j:])
			if r == utf8.RuneError && size == 1 {
				return "", fmt.Errorf("invalid duration %q", raw)
			}
			if r == 'µ' || unicode.IsLetter(r) {
				j += size
				continue
			}
			break
		}
		if j == 0 {
			return "", fmt.Errorf("invalid duration %q", raw)
		}
		unit := s[:j]
		s = s[j:]

		switch unit {
		case "d":
			hours := num * 24
			b.WriteString(strconv.FormatFloat(hours, 'f', -1, 64))
			b.WriteByte('h')
		case "w":
			hours := num * 7 * 24
			b.WriteString(strconv.FormatFloat(hours, 'f', -1, 64))
			b.WriteByte('h')
		default:
			// Keep other units unchanged; Go will validate them.
			b.WriteString(numStr)
			b.WriteString(unit)
		}
	}

	return b.String(), nil
}
