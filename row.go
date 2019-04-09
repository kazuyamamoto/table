package table

import (
	"bufio"
	"bytes"
	"fmt"
	"strings"
	"unicode"
)

// row represents a row of table.
type row []string

// parseRow parses s into row.
// Returned bool indicates that the row wants to be merged with the next row.
// Returned row and error are nil if s is empty or spaces.
func parseRow(s string) (row, bool, error) {
	s = trim(s)
	if s == "" {
		return nil, false, nil
	}

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
				return nil, false, fmt.Errorf("unsupported escape character %q", rn)
			}
			b.WriteRune(rn)
		}
	}

	return append(row, trim(b.String())), escaping, nil
}

func parseRow2(s string) (row, bool, error) {
	rs := newRowScanner(s)
	var row row
	var cont bool
	var b strings.Builder
	for {
		t, v := rs.scan()
		switch t {
		case illegal:
			return nil, false, fmt.Errorf("illegal token %q", v)
		case eof:
			row = append(row, trim(b.String()))
			return row, cont, nil
		case text:
			b.WriteString(v)
		case pipe:
			row = append(row, trim(b.String()))
			b.Reset()
		case escBackslash:
			b.WriteString("\\")
		case escNewline:
			b.WriteString("\n")
		case escPipe:
			b.WriteString("|")
		case escEOF:
			cont = true
		default:
			return nil, false, fmt.Errorf("unknown token type %v, value %q", t, v)
		}
	}
}

type tokenType int

const (
	illegal tokenType = iota
	eof
	text
	pipe         // |
	escBackslash // \\
	escNewline   // \n
	escPipe      // \|
	escEOF       // \
)

type rowScanner struct {
	reader *bufio.Reader
}

func newRowScanner(s string) *rowScanner {
	return &rowScanner{
		reader: bufio.NewReader(bytes.NewBufferString(s)),
	}
}

func (s *rowScanner) scan() (tokenType, string) {
	r, _, err := s.reader.ReadRune()
	if err != nil {
		return eof, ""
	}

	if r == '|' {
		return pipe, "|"
	}

	if r == '\\' {
		r2, _, err := s.reader.ReadRune()
		if err != nil {
			_ = s.reader.UnreadRune()
			return escEOF, "\\"
		}

		switch r2 {
		case '\\':
			return escBackslash, "\\\\"
		case '|':
			return escPipe, "\\|"
		case 'n':
			return escNewline, "\\n"
		default:
			return illegal, "\\" + string(r2)
		}
	}

	_ = s.reader.UnreadRune()
	buf := &bytes.Buffer{}
	for {
		r, _, err = s.reader.ReadRune()
		if err != nil {
			return text, buf.String()
		}

		if r == '|' || r == '\\' {
			_ = s.reader.UnreadRune()
			return text, buf.String()
		}

		buf.WriteRune(r)
	}
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
		if strings.IndexFunc(trim(e), notDelim) != -1 {
			return false
		}
	}

	return true
}

// cols returns number of columns.
func (r row) cols() int {
	return len(r)
}

// merge merges o into r.
// Values in corresponding column of two rows are merged
// inserting whitespace between them. Returns non-nil error
// if number of columns of r and o are different.
func (r row) merge(o row) error {
	if r.cols() != o.cols() {
		return fmt.Errorf("number of columns are different")
	}

	for i := 0; i < o.cols(); i++ {
		if r[i] == "" {
			r[i] = o[i]
		} else if o[i] != "" {
			r[i] = r[i] + " " + o[i]
		}
	}

	return nil
}

func notDelim(rn rune) bool {
	return rn != '-'
}

// Not use strings.TrimSpace because '\n' should not be trimmed.
func trim(s string) string {
	return strings.TrimFunc(s, isSpace)
}

func isSpace(r rune) bool {
	if r == '\n' {
		return false
	}

	return unicode.IsSpace(r)
}
