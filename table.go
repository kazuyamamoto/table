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

// Unmarshaler provides customized unmarshalling method.
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

	scanner := scanner{bufio.NewScanner(s)}
	header, err := parseHeader(scanner)
	if err != nil {
		return fmt.Errorf("parsing header: %v", err)
	}

	if header.columns() == 0 {
		return nil
	}

	if err = checkHeader(tStruct, header); err != nil {
		return fmt.Errorf("checking header: %v", err)
	}

	// table body
	vSlice := vPointer.Elem()
	for scanner.Scan() {
		t := scanner.Text()
		if t == "" {
			return nil
		}

		r, err := parseRow(t)
		if err != nil {
			return fmt.Errorf("parsing table body: %v", err)
		}

		if r.columns() != header.columns() {
			return fmt.Errorf("number of columns: header=%v body=%v", header.columns(), r.columns())
		}

		if r.isDelimiter() {
			continue
		}

		vStruct, err := unmarshalStruct(tStruct, header, r)
		if err != nil {
			return err
		}

		vSlice.Set(reflect.Append(vSlice, vStruct.Elem()))
	}

	return nil
}

func parseHeader(sc scanner) (row, error) {
	enterHeader := false
	var header row
	for sc.Scan() {
		t := sc.Text()
		if t == "" {
			if enterHeader {
				return header, nil // table end
			}
		} else {
			enterHeader = true
			r, err := parseRow(t)
			if err != nil {
				return nil, fmt.Errorf("parsing header row: %v", err)
			}

			if r.isDelimiter() {
				return header, nil
			}

			if header == nil {
				header = r
			} else {
				if err = header.merge(r); err != nil {
					return nil, fmt.Errorf("merging header: %v", err)
				}
			}
		}
	}

	return header, nil
}

type scanner struct{ *bufio.Scanner }

func (sc scanner) Text() string {
	return strings.TrimSpace(sc.Scanner.Text())
}

func checkHeader(tStruct reflect.Type, header row) error {
	for i := 0; i < tStruct.NumField(); i++ {
		tag := tStruct.Field(i).Tag.Get("table")
		if tag == "" {
			continue
		}

		if header.index(tag) == -1 {
			return fmt.Errorf("column named '%s' not found", tag)
		}
	}
	return nil
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

		if reflect.PtrTo(tField.Type).Implements(unmarshalerType) {
			if err := unmarshalUnmarshalerType(vField, s); err != nil {
				return reflect.Value{}, err
			}
			continue
		}

		if err := unmarshalBasicType(vField, s); err != nil {
			return reflect.Value{}, err
		}
	}

	return vPointer, nil
}

// unmarshalerType is an object represents type of Unmarshaler.
var unmarshalerType = reflect.TypeOf(new(Unmarshaler)).Elem()

func unmarshalUnmarshalerType(v reflect.Value, s string) error {
	// calls Addr() for pointer receiver
	m := v.Addr().MethodByName("UnmarshalTable")
	ret := m.Call([]reflect.Value{reflect.ValueOf([]byte(s))})
	if len(ret) > 0 && !ret[0].IsNil() {
		return ret[0].Interface().(error)
	}

	return nil
}

func unmarshalBasicType(v reflect.Value, s string) error {
	switch k := v.Kind(); k {
	case reflect.String:
		v.SetString(s)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		i, err := strconv.ParseInt(s, 0, 64)
		if err != nil {
			return parseBasicTypeError{k, err}
		}
		v.SetInt(i)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		u, err := strconv.ParseUint(s, 10, 64)
		if err != nil {
			return parseBasicTypeError{k, err}
		}
		v.SetUint(u)
	case reflect.Bool:
		b, err := strconv.ParseBool(s)
		if err != nil {
			return parseBasicTypeError{k, err}
		}
		v.SetBool(b)
	case reflect.Float32, reflect.Float64:
		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return parseBasicTypeError{k, err}
		}
		v.SetFloat(f)
	default:
		return parseBasicTypeError{k, fmt.Errorf("unknown type")}
	}

	return nil
}

// parseBasicTypeError is an error represents failure for parsing basic types string.
type parseBasicTypeError struct {
	kind  reflect.Kind
	cause error
}

func (e parseBasicTypeError) Error() string {
	return fmt.Sprintf("parsing %s: %v", e.kind, e.cause)
}
