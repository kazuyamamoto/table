package table

import (
	"fmt"
	"strings"
)

// row represents a row of table.
type row []string

func unmarshalRow(s string) (row, error) {
	var r row
	remainder := s
	for {
		i := columnDelimiterIndex(remainder)
		if i == -1 {
			return append(r, strings.TrimSpace(remainder)), nil
		}

		r = append(r, strings.TrimSpace(remainder[:i]))
		remainder = remainder[i+1:]
	}
}

func columnDelimiterIndex(s string) int {
	start := 0
	for {
		if start > len(s) {
			return -1
		}

		target := s[start:]
		i := strings.Index(target, "|")
		if i == -1 {
			return -1
		}

		if i >= 1 && target[i-1] == '\\' {
			start = start + i + 1
			continue
		}

		return start + i
	}
}

func (r row) index(v string) int {
	for i, e := range r {
		if e == v {
			return i
		}
	}

	return -1
}

func (r row) isDelimiter() bool {
	for _, e := range r {
		if strings.IndexFunc(strings.TrimSpace(e), isNotDelimiter) != -1 {
			return false
		}
	}

	return true
}

func (r *row) unescape() error {
	for i := 0; i < len(*r); i++ {
		unescaped, err := unescape((*r)[i])
		if err != nil {
			return fmt.Errorf("unescape value %q: %v", (*r)[i], err)
		}

		(*r)[i] = unescaped
	}
	return nil
}

func isNotDelimiter(r rune) bool {
	return r != '-'
}

// unescape unescapes escape-sequence.
func unescape(s string) (string, error) {
	return unescapeTailRec("", s)
}

func unescapeTailRec(unescaped string, escaped string) (string, error) {
	i := strings.Index(escaped, "\\")
	if i == -1 {
		return unescaped + escaped, nil
	}

	if i+1 == len(escaped) {
		return "", fmt.Errorf("contains invalid escape seuqence %q", escaped)
	}

	var lit string
	switch escaped[i+1] {
	case 'n':
		lit = "\n"
	case '\\':
		lit = "\\"
	case '|':
		lit = "|"
	default:
		return "", fmt.Errorf("contains unsupported escape sequece %q", escaped)
	}

	return unescapeTailRec(unescaped+escaped[:i]+lit, escaped[i+2:])
}
