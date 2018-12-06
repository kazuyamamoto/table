// Package table provides functionality to unmarshal table string into slice of
// struct. Table format is like those of lightweight markup languages:
//
//   string  | custom | int   | float | bool  | uint | escape | 文字列
//   ------- | ------ | ----- | ----- | ----- | ---- | ------ | --------
//   abc     | OK     | 302   | 1.234 | true  | 7890 | abc\nd | あいうえお
//           | NG     | -0x20 | -5    | F     | 3333 | \\n\|  | 日本語
//
// A row filled with '-' is assumed as delimiter.
// Header is rows above the first delimiter and body is below that delimiter.
// Delimiters in body are ignored. Empty lines above header are ignored.
// Table ends with an empty line. Following lines are ignored.
// Escape sequences can be used in values. Those are "\n" (unescaped into LF),
// "\\" (\), and "\|" (|).
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

// Unmarshal parses r as table string then sets parsed objects to v.
// If v is not a pointer to slice of struct, Unmarshal returns non-nil error.
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

// UnmarshalReader is like Unmarshal except for parsing data from io.Reader
// instead of []byte.
func UnmarshalReader(r io.Reader, v interface{}) error {
	vPointer := reflect.ValueOf(v)
	if vPointer.Kind() != reflect.Ptr {
		return errors.New("value of interface{} is not a pointer")
	}

	tSlice := vPointer.Type().Elem()
	if tSlice.Kind() != reflect.Slice {
		return errors.New("value of interface{} is not a pointer of slice")
	}

	tStruct := tSlice.Elem()
	if tStruct.Kind() != reflect.Struct {
		return errors.New("value of interface{} is not a pointer of slice of struct")
	}

	sc := scanner{bufio.NewScanner(r)}
	hdr, err := parseHeader(sc)
	if err != nil {
		return fmt.Errorf("parse header: %v", err)
	}

	// table body
	vSlice := vPointer.Elem()
	for sc.Scan() {
		t := sc.Text()
		if t == "" {
			return nil
		}

		r, err := parseRow(t)
		if err != nil {
			return fmt.Errorf("parsing table body: %v", err)
		}

		if r.numColumn() != hdr.numColumn() {
			return fmt.Errorf("number of columns: header=%v body=%v", hdr.numColumn(), r.numColumn())
		}

		if r.isDelimiter() {
			continue
		}

		vStruct, err := unmarshalStruct(tStruct, hdr, r)
		if err != nil {
			return err
		}

		vSlice.Set(reflect.Append(vSlice, vStruct.Elem()))
	}

	return nil
}

func parseHeader(sc scanner) (row, error) {
	enterHeader := false
	var hdr row
	for sc.Scan() {
		t := sc.Text()
		if t == "" {
			if enterHeader {
				return hdr, nil // table end
			}
		} else {
			enterHeader = true
			r, err := parseRow(t)
			if err != nil {
				return nil, fmt.Errorf("parsing header row: %v", err)
			}

			if r.isDelimiter() {
				return hdr, nil
			}

			if hdr == nil {
				hdr = r
			} else {
				if err = hdr.merge(r); err != nil {
					return nil, fmt.Errorf("merging header: %v", err)
				}
			}
		}
	}

	return hdr, nil
}

type scanner struct{ *bufio.Scanner }

func (sc scanner) Text() string {
	return strings.TrimSpace(sc.Scanner.Text())
}

// unmarshalerType is an object represents type of Unmarshaler.
var unmarshalerType = reflect.TypeOf(new(Unmarshaler)).Elem()

// unmarshalStruct unmarshals r into value of tStruct type.
// When successful, this returns pointer to the value and nil.
// When failure, this returns zero-value of reflect.Value and non-nil error.
func unmarshalStruct(tStruct reflect.Type, hdr, r row) (reflect.Value, error) {
	// Not using reflect.Zero for settability.
	// See https://blog.golang.org/laws-of-reflection
	vPointer := reflect.New(tStruct)
	for fi := 0; fi < vPointer.Elem().NumField(); fi++ {
		vField := vPointer.Elem().Field(fi)
		tField := tStruct.Field(fi)
		tag := tField.Tag.Get("table")
		if tag == "" {
			continue
		}

		ti := hdr.index(tag)
		if ti == -1 {
			continue
		}

		s := strings.TrimSpace(r[ti])

		// Unmarshal Unmarshaler implementation
		if reflect.PtrTo(tField.Type).Implements(unmarshalerType) {
			// calls Addr() for pointer receiver
			m := vField.Addr().MethodByName("UnmarshalTable")
			vReturns := m.Call([]reflect.Value{reflect.ValueOf([]byte(s))})
			if len(vReturns) > 0 && !vReturns[0].IsNil() {
				return reflect.Value{}, vReturns[0].Interface().(error)
			}

			continue
		}

		// Unmarshal basic type values
		t := vField.Kind()
		switch t {
		case reflect.String:
			vField.SetString(s)
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			i, err := strconv.ParseInt(s, 0, 64)
			if err != nil {
				return reflect.Value{}, parseBasicError{t, err}
			}
			vField.SetInt(i)
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			u, err := strconv.ParseUint(s, 10, 64)
			if err != nil {
				return reflect.Value{}, parseBasicError{t, err}
			}
			vField.SetUint(u)
		case reflect.Bool:
			b, err := strconv.ParseBool(s)
			if err != nil {
				return reflect.Value{}, parseBasicError{t, err}
			}
			vField.SetBool(b)
		case reflect.Float32, reflect.Float64:
			f, err := strconv.ParseFloat(s, 64)
			if err != nil {
				return reflect.Value{}, parseBasicError{t, err}
			}
			vField.SetFloat(f)
		}
	}

	return vPointer, nil
}

type parseBasicError struct {
	t     reflect.Kind
	cause error
}

func (e parseBasicError) Error() string {
	return fmt.Sprintf("parsing %s: %v", e.t, e.cause)
}
