package table

import (
	"fmt"
	"strings"
)

// row represents a row of table.
type row []string

func unmarshalRow(s string) (row, error) {
	// TODO literal "\|"
	sp := strings.Split(s, "|")
	for i := 0; i < len(sp); i++ {
		sp[i] = strings.TrimSpace(sp[i])
	}

	return sp, nil
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
	default:
		return "", fmt.Errorf("contains unsupported escape sequece %q", escaped)
	}

	return unescapeTailRec(unescaped+escaped[:i]+lit, escaped[i+2:])
}
