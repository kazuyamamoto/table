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

// Unmarshal parses s as table string then sets parsed objects to t.
// If t is not a pointer to slice of struct, Unmarshal returns non-nil error.
// If parsing an element in s is failed, returns non-nil error.
// Headers are bound to struct field tags.
// Tag format is as follows:
//    `table:"column name"`
// When header corresponds to "column name" is found,
// element of the column is parsed and the value is set to a struct field of the tag.
func Unmarshal(s []byte, t interface{}) error {
	return UnmarshalReader(bytes.NewReader(s), t)
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
func UnmarshalReader(s io.Reader, t interface{}) error {
	// vXxx represents a value. tXxx represents a type.
	vPointer := reflect.ValueOf(t)
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

	sc := scanner{bufio.NewScanner(s)}
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

func fieldToHeader(tStruct reflect.Type, hdr row) ([]int, error) {
	var field []int
	for fi := 0; fi < tStruct.NumField(); fi++ {
		t := tStruct.Field(fi).Tag.Get("table")
		if t == "" {
			continue
		}
		ti := hdr.index(t)
		if ti == -1 {
			return nil, fmt.Errorf("field tag not found in table header: %s", t)
		}
		field = append(field, ti)
	}

	return field, nil
}

// unmarshalStruct unmarshals r into value of tStruct type.
// When successful, this returns pointer to the value and nil.
// When failure, this returns zero-value of reflect.Value and non-nil error.
func unmarshalStruct(tStruct reflect.Type, hdr, r row) (reflect.Value, error) {
	// Not using reflect.Zero for "settability".
	// See https://blog.golang.org/laws-of-reflection
	vPointer := reflect.New(tStruct)
	for fi := 0; fi < vPointer.Elem().NumField(); fi++ {
		vField := vPointer.Elem().Field(fi)
		tField := tStruct.Field(fi)
		tag := tField.Tag.Get("table")
		if tag == "" { // TODO check can be once
			continue
		}

		ti := hdr.index(tag)
		if ti == -1 {
			return reflect.Value{}, fmt.Errorf("found no tag named %s", tag)
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
				return reflect.Value{}, parseBasicTypeError{t, err}
			}
			vField.SetInt(i)
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			u, err := strconv.ParseUint(s, 10, 64)
			if err != nil {
				return reflect.Value{}, parseBasicTypeError{t, err}
			}
			vField.SetUint(u)
		case reflect.Bool:
			b, err := strconv.ParseBool(s)
			if err != nil {
				return reflect.Value{}, parseBasicTypeError{t, err}
			}
			vField.SetBool(b)
		case reflect.Float32, reflect.Float64:
			f, err := strconv.ParseFloat(s, 64)
			if err != nil {
				return reflect.Value{}, parseBasicTypeError{t, err}
			}
			vField.SetFloat(f)
		}
	}

	return vPointer, nil
}

// parseBasicTypeError is an error represents failure for parsing basic types string.
type parseBasicTypeError struct {
	t     reflect.Kind
	cause error
}

func (e parseBasicTypeError) Error() string {
	return fmt.Sprintf("parsing %s: %v", e.t, e.cause)
}
