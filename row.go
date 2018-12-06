package table

import (
	"fmt"
	"strings"
)

// row represents a row of table.
type row []string

// parseRow parses row string.
func parseRow(s string) (row, error) {
	var row row
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
				row = append(row, trim(cell.String()))
				cell.Reset()
			}
		default:
			if escaping {
				return nil, fmt.Errorf("unsupported escaped character %q", r)
			}
			cell.WriteRune(r)
		}
	}

	return append(row, trim(cell.String())), nil
}

// index returns index of cell whose value equals v.
// Returns -1 if not found.
func (r row) index(v string) int {
	for i, e := range r {
		if e == v {
			return i
		}
	}

	return -1
}

// isDelimiter returns true if r is a delimiter row.
// Delimiter row is consist of sequence of '-' and whitespaces.
func (r row) isDelimiter() bool {
	for _, e := range r {
		if strings.IndexFunc(strings.TrimSpace(e), isNotDelimiter) != -1 {
			return false
		}
	}

	return true
}

func (r row) numColumn() int {
	return len(r)
}

// merge concatinates two values in same columns of r and other
// with whitespace between them. Returns non-nil error
// if number of columns of r and other are different.
func (r row) merge(other row) error {
	if r.numColumn() != other.numColumn() {
		return fmt.Errorf("number of header columns is different")
	}

	for i := 0; i < other.numColumn(); i++ {
		if r[i] == "" {
			r[i] = other[i]
		} else if other[i] != "" {
			r[i] = r[i] + " " + other[i]
		}
	}

	return nil
}

func isNotDelimiter(r rune) bool {
	return r != '-'
}

func trim(s string) string {
	return strings.Trim(s, " \t")
}
