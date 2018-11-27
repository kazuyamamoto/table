package table

import (
	"fmt"
	"strings"
)

// Row represents a row of table.
type Row []string

// ParseRow parses row string.
func ParseRow(s string) (Row, error) {
	var row Row
	escaping := false
	cell := strings.Builder{}
	for _, r := range s {
		switch r {
		case '\\':
			if escaping {
				cell.WriteRune(r)
			}
			escaping = !escaping
		case 'n':
			if escaping {
				cell.WriteRune('\n')
				escaping = false
			} else {
				cell.WriteRune('n')
			}
		case '|':
			if escaping {
				cell.WriteRune(r)
				escaping = false
			} else {
				row = append(row, strings.TrimSpace(cell.String()))
				cell.Reset()
			}
		default:
			if escaping {
				return nil, fmt.Errorf("unsupported escaped character %q", r)
			}
			cell.WriteRune(r)
		}
	}

	return append(row, strings.TrimSpace(cell.String())), nil
}

// Index returns index of cell whose value matches v.
// Returns -1 if not found.
func (r Row) Index(v string) int {
	for i, e := range r {
		if e == v {
			return i
		}
	}

	return -1
}

// IsDelimiter returns true if r is a delimiter row.
// Delimiter row is consist of sequence of '-' and whitespaces.
func (r Row) IsDelimiter() bool {
	for _, e := range r {
		if strings.IndexFunc(strings.TrimSpace(e), isNotDelimiter) != -1 {
			return false
		}
	}

	return true
}

func isNotDelimiter(r rune) bool {
	return r != '-'
}
