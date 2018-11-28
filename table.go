// Package table provides functionality to unmarshal table string into slice of
// struct. Table format is like those of lightweight markup languages:
//
//   string  | custom | int   | float | bool     | uint | escape | 文字列
//   ------- | ------ | ----- | ----- | -------- | ---- | ------ | --------
//   abc     | OK     | 302   | 1.234 | true     | 7890 | abc\nd | あいうえお
//           | NG     | -0x20 | -5    | non-bool | 3333 | \\n\|  | 日本語
//
// First row is header. A row filled with '-' is assumed as delimiter.
// It is ignored. Empty lines before header are ignored.
// Table ends with an empty line. Its following lines are ignored.
// Values in table body are unescaped while unmarshaling.
// Escape sequences are "\n" (unescaped into LF), "\\"(\), and "\|"(|).
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

	scanner := bufio.NewScanner(r)
	hdr, err := parseHeader(scanner)
	if err != nil {
		return fmt.Errorf("read header: %v", err)
	}

	// table body
	vSlice := vPointer.Elem()
	for scanner.Scan() {
		t := strings.TrimSpace(scanner.Text())
		if t == "" {
			return nil
		}

		r, err := parseRow(t)
		if err != nil {
			return fmt.Errorf("parse table body: %v", err)
		}

		if len(r) != len(hdr) {
			return fmt.Errorf("#columns: header is %v but table body is %v", len(hdr), len(r))
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

// unmarshalerType is an object of type of Unmarshaler.
var unmarshalerType = reflect.TypeOf(new(Unmarshaler)).Elem()

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

		// Unmarshal basic type values. Ignore unknown type values
		switch vField.Kind() {
		case reflect.String:
			vField.SetString(s)
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			if i, err := strconv.ParseInt(s, 0, 64); err == nil {
				vField.SetInt(i)
			}
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			if i, err := strconv.ParseUint(s, 10, 64); err == nil {
				vField.SetUint(i)
			}
		case reflect.Bool:
			if b, err := strconv.ParseBool(s); err == nil {
				vField.SetBool(b)
			}
		case reflect.Float32, reflect.Float64:
			if f, err := strconv.ParseFloat(s, 64); err == nil {
				vField.SetFloat(f)
			}
		}
	}

	return vPointer, nil
}

func parseHeader(scanner *bufio.Scanner) (row, error) {
	// ignore empty lines
	s := ""
	for scanner.Scan() {
		if s = strings.TrimSpace(scanner.Text()); s != "" {
			break
		}
	}

	if s == "" {
		return nil, errors.New("no header")
	}

	hdr, err := parseRow(s)
	if err != nil {
		return nil, fmt.Errorf("parse header row: %v", err)
	}

	return hdr, nil
}
