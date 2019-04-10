package table

import (
	"bufio"
	"bytes"
	"fmt"
	"strings"
	"unicode"
)

// row represents a row in table.
type row []string

// parseRow parses s into a row object.
// Returned bool indicates that the row expects to continue to the next one.
// Returned row and error are nil if s is empty or white spaces.
func parseRow(s string) (row, bool, error) {
	rs := newRowScanner(s)
	var row row
	var cont bool
	var b strings.Builder
	for {
		t := rs.scan()
		switch t.typ {
		case illegal:
			return nil, false, fmt.Errorf("scanned token %v", t)
		case eof:
			tr := trim(b.String())
			if tr == "" && row == nil {
				return nil, cont, nil
			}
			return append(row, tr), cont, nil
		case text:
			b.WriteString(t.value)
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
			return nil, false, fmt.Errorf("scanned token %v", t)
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
	escEOF       // \<EOF>
)

func (tt tokenType) String() string {
	switch tt {
	case illegal:
		return "ILLEGAL"
	case eof:
		return "EOF"
	case text:
		return "TEXT"
	case pipe:
		return "PIPE"
	case escBackslash:
		return "ESCAPE_BACKSLASH"
	case escNewline:
		return "ESCAPE_NEWLINE"
	case escPipe:
		return "ESCAPE_PIPE"
	case escEOF:
		return "ESCAPE_EOF"
	default:
		return "UNKNOWN"
	}
}

// token is a token in row string.
type token struct {
	typ   tokenType
	value string
}

func (t *token) String() string {
	return fmt.Sprintf("%v(%v)", t.typ, t.value)
}

// rowScanner scans tokens in row string.
type rowScanner struct {
	reader *bufio.Reader
}

func newRowScanner(s string) *rowScanner {
	return &rowScanner{
		reader: bufio.NewReader(bytes.NewBufferString(s)),
	}
}

// scan returns a token in row string.
func (s *rowScanner) scan() *token {
	r, _, err := s.reader.ReadRune()
	if err != nil {
		return &token{eof, ""}
	}

	if r == '|' {
		return &token{pipe, "|"}
	}

	if r == '\\' {
		r2, _, err := s.reader.ReadRune()
		if err != nil {
			_ = s.reader.UnreadRune()
			return &token{escEOF, "\\"}
		}

		switch r2 {
		case '\\':
			return &token{escBackslash, "\\\\"}
		case '|':
			return &token{escPipe, "\\|"}
		case 'n':
			return &token{escNewline, "\\n"}
		default:
			return &token{illegal, "\\" + string(r2)}
		}
	}

	_ = s.reader.UnreadRune()
	buf := &bytes.Buffer{}
	for {
		r, _, err = s.reader.ReadRune()
		if err != nil {
			return &token{text, buf.String()}
		}

		if r == '|' || r == '\\' {
			_ = s.reader.UnreadRune()
			return &token{text, buf.String()}
		}

		buf.WriteRune(r)
	}
}

// index returns index of column whose value equals v.
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
// Delimiter row is consist of sequence of '-' and white spaces.
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

func (r row) String() string {
	return strings.Join(r, "|")
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
