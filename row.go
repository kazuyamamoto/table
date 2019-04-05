package table

import (
	"fmt"
	"strings"
)

// row represents a row of table.
type row []string

// parseRow parses s into row.
// Bool return value indicates that the row wants to be merged with the next row.
func parseRow(s string) (row, bool, error) {
	var row row
	escaping := false
	b := strings.Builder{}
	for _, rn := range s {
		switch rn {
		case '\\':
			if escaping {
				b.WriteRune(rn)
			}
			escaping = !escaping
		case 'n':
			if escaping {
				b.WriteRune('\n')
				escaping = false
			} else {
				b.WriteRune('n')
			}
		case '|':
			if escaping {
				b.WriteRune(rn)
				escaping = false
			} else {
				row = append(row, trim(b.String()))
				b.Reset()
			}
		default:
			if escaping {
				return nil, false, fmt.Errorf("unsupported escaped character %q", rn)
			}
			b.WriteRune(rn)
		}
	}

	return append(row, trim(b.String())), escaping, nil
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

// isDelim returns true if r is a delimiter row.
// Delimiter row is consist of sequence of '-' and whitespaces.
func (r row) isDelim() bool {
	for _, e := range r {
		if strings.IndexFunc(strings.TrimSpace(e), isNotDelim) != -1 {
			return false
		}
	}

	return true
}

// cols returns number of columns of r.
func (r row) cols() int {
	return len(r)
}

// merge merges r and other.
// Values in corresponding column of two rows are merged
// inserting whitespace between them. Returns non-nil error
// if number of columns of r and other are different.
func (r row) merge(other row) error {
	if r.cols() != other.cols() {
		return fmt.Errorf("number of header columns is different")
	}

	for i := 0; i < other.cols(); i++ {
		if r[i] == "" {
			r[i] = other[i]
		} else if other[i] != "" {
			r[i] = r[i] + " " + other[i]
		}
	}

	return nil
}

func isNotDelim(rn rune) bool {
	return rn != '-'
}

func trim(s string) string {
	return strings.Trim(s, " \t")
}
