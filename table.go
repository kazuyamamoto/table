// Package table provides functionality to parse table string into slice of struct.
// Table format is like that of lightweight markup language.
//
//   string  | custom | int   | float | bool     | uint | escape | 文字列
//   ------- | ------ | ----- | ----- | -------- | ---- | ------ | --------
//   abc     | OK     | 302   | 1.234 | true     | 7890 | abc\nd | あいうえお
//           | NG     | -0x20 | -5    | non-bool | 3333 | abc\\n | 日本語
package table

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strconv"
	"strings"
)

// Unmarshal parses r as table string then sets parsed values to v.
// If v is not a pointer to slice of struct, Unmarshal returns non-nil error.
// First row is header. Delimiter rows are ignored.
// When parsing an element in table string is failed, its value in v is zero.
// Headers are bound to struct field tags.
// Tag format is as follows:
//    `table:"column name"`
// When header corresponds to "column name", element of the column is parsed and
// set to struct field with the tag.
func Unmarshal(r []byte, v interface{}) error {
	return UnmarshalReader(bytes.NewReader(r), v)
}

// Unmarshaler provides customized unmarshaling method.
// An implementation is assumed to be fields of struct which is underlying
// type of Unmarshal's second parameter.
// Unmarshal calls implementation's UnmarshalTable.
// An implementation object is assumed to be filled with a parsed value.
// So the receiver type should be pointer.
type Unmarshaler interface {
	UnmarshalTable([]byte) error
}

// UnmarshalReader is like Unmarshal except for parsing data from io.Reader instead of []byte.
func UnmarshalReader(r io.Reader, v interface{}) error {
	vptr := reflect.ValueOf(v)
	if vptr.Kind() != reflect.Ptr {
		return errors.New("value of interface{} is not a pointer")
	}

	tslc := vptr.Type().Elem()
	if tslc.Kind() != reflect.Slice {
		return errors.New("value of interface{} is not a pointer of slice")
	}

	tstr := tslc.Elem()
	if tstr.Kind() != reflect.Struct {
		return errors.New("value of interface{} is not a pointer of slice of struct")
	}

	scanner := bufio.NewScanner(r)
	hdr, err := readHeader(scanner)
	if err != nil {
		return fmt.Errorf("read header: %v", err)
	}

	vslc := vptr.Elem()
	for scanner.Scan() {
		r, err := parseRow(scanner.Text())
		if err != nil {
			return fmt.Errorf("parse row: %v", err)
		}

		if len(r) != len(hdr) {
			return fmt.Errorf("number of columns: header %v, row %v", len(hdr), len(r))
		}

		if r.isDelimiter() {
			continue
		}

		vstr, err := unmarshalRow(tstr, hdr, r)
		if err != nil {
			return err
		}

		vslc.Set(reflect.Append(vslc, vstr.Elem()))
	}

	return nil
}

// unmarshalerType is an object of type of Unmarshaler.
var unmarshalerType = reflect.TypeOf(new(Unmarshaler)).Elem()

func unmarshalRow(tstr reflect.Type, hdr row, r row) (reflect.Value, error) {
	// Not using reflect.Zero for settability.
	// See https://blog.golang.org/laws-of-reflection
	vstr := reflect.New(tstr)
	for fidx := 0; fidx < vstr.Elem().NumField(); fidx++ {
		vfld := vstr.Elem().Field(fidx)
		tfld := tstr.Field(fidx)
		tag := tfld.Tag.Get("table")
		if tag == "" {
			continue
		}

		tidx := hdr.index(tag)
		if tidx == -1 {
			continue
		}

		s := strings.TrimSpace(r[tidx])

		// Unmarshal Unmarshaler implementation
		if reflect.PtrTo(tfld.Type).Implements(unmarshalerType) {
			// calls Addr() for pointer receiver
			m := vfld.Addr().MethodByName("UnmarshalTable")
			verr := m.Call([]reflect.Value{reflect.ValueOf([]byte(s))})
			if len(verr) > 0 && !verr[0].IsNil() {
				return reflect.Value{}, verr[0].Interface().(error)
			}

			continue
		}

		// Unmarshal basic type values. Ignore unknown type values
		switch vfld.Kind() {
		case reflect.String:
			if s, err := unescape(s); err == nil {
				vfld.SetString(s)
			}
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			if i, err := strconv.ParseInt(s, 0, 64); err == nil {
				vfld.SetInt(i)
			}
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			if i, err := strconv.ParseUint(s, 10, 64); err == nil {
				vfld.SetUint(i)
			}
		case reflect.Bool:
			if b, err := strconv.ParseBool(s); err == nil {
				vfld.SetBool(b)
			}
		case reflect.Float32, reflect.Float64:
			if f, err := strconv.ParseFloat(s, 64); err == nil {
				vfld.SetFloat(f)
			}
		}
	}

	return vstr, nil
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

func readHeader(s *bufio.Scanner) (row, error) {
	// ignore empty lines
	r := ""
	for s.Scan() {
		if r = strings.TrimSpace(s.Text()); r != "" {
			break
		}
	}

	if r == "" {
		return nil, errors.New("no header")
	}

	hdr, err := parseRow(r)
	if err != nil {
		return nil, fmt.Errorf("parse header row: %v", err)
	}

	for i := 0; i < len(hdr); i++ {
		t := strings.TrimSpace(hdr[i])
		if t == "" {
			return nil, errors.New("contains empty header name")
		}
		hdr[i] = t
	}

	return hdr, nil
}

type row []string

func parseRow(s string) (row, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}

	// TODO literal "\|"
	sp := strings.Split(s, "|")
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
		if strings.IndexFunc(strings.TrimSpace(e), func(r rune) bool { return r != '-' }) != -1 {
			return false
		}
	}

	return true
}
