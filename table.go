package table

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strconv"
)

// Unmarshal parses s as table string then sets parsed objects to t.
// t should be a pointer to slice of struct.
//
// Headers are bound to struct field tags.
// Tag format is as follows:
//    `table:"column name"`
// When header corresponds to "column name" is found,
// element of the column is parsed and the value is set to a struct field of the tag.
func Unmarshal(s []byte, t interface{}) error {
	return UnmarshalReader(bytes.NewReader(s), t)
}

// Unmarshaler provides custom unmarshalling method.
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
		return errors.New("table: value of interface{} is not a pointer")
	}

	tSlice := vPointer.Type().Elem()
	if tSlice.Kind() != reflect.Slice {
		return errors.New("table: value of interface{} is not a pointer of slice")
	}

	tStruct := tSlice.Elem()
	if tStruct.Kind() != reflect.Struct {
		return errors.New("table: value of interface{} is not a pointer of slice of struct")
	}

	ts := newTableScanner(s)
	header, err := parseHeader(ts)
	if err != nil {
		return fmt.Errorf("table: failed to parse header: %v", err)
	}

	if header.cols() == 0 {
		return nil
	}

	indices, err := indexFieldToColumn(tStruct, header)
	if err != nil {
		return fmt.Errorf("table: check header: %v", err)
	}

	// table body
	vSlice := vPointer.Elem()
	for {
		r, err := ts.mergedRow()
		if err == io.EOF {
			return nil
		}

		if err != nil {
			return fmt.Errorf("table: failed to parse table body: %v", err)
		}

		if r == nil {
			return nil
		}

		if r.cols() != header.cols() {
			return fmt.Errorf("table: number of columns: header=%v body=%v", header.cols(), r.cols())
		}

		vStruct, err := unmarshalStruct(tStruct, r, indices)
		if err != nil {
			return fmt.Errorf("table: failed to unmarshal row: %v", err)
		}

		vSlice.Set(reflect.Append(vSlice, vStruct.Elem()))
	}
}

func parseHeader(ts *tableScanner) (row, error) {
	for {
		header, err := ts.mergedRow()
		if err == io.EOF {
			return nil, nil
		}

		if err != nil {
			return nil, fmt.Errorf("get header: %v", err)
		}

		if header != nil {
			return header, nil
		}
	}
}

// tableScanner is a bufio.Scanner for table string.
type tableScanner struct {
	scanner *bufio.Scanner
}

func newTableScanner(r io.Reader) *tableScanner {
	return &tableScanner{bufio.NewScanner(r)}
}

// mergedRow returns a row. If the row consists of multiple rows, they are merged.
func (ts *tableScanner) mergedRow() (row, error) {
	var row row
	var cont bool
	for {
		if !ts.scan() {
			if cont {
				return nil, fmt.Errorf("row continues but the file ended")
			}
			return row, io.EOF
		}

		r, c, err := ts.row()
		if err != nil {
			return nil, fmt.Errorf("get row: %v", err)
		}

		if r == nil {
			if cont {
				return nil, fmt.Errorf("row continues but the table ended ")
			}
			return row, nil
		}

		cont = c
		if r.isDelim() {
			continue
		}

		if row == nil {
			row = r
		} else {
			if err := row.merge(r); err != nil {
				return nil, fmt.Errorf("merging: %v", err)
			}
		}

		if !c {
			return row, nil
		}
	}
}

func (ts *tableScanner) scan() bool {
	return ts.scanner.Scan()
}

func (ts *tableScanner) row() (row, bool, error) {
	return parseRow(ts.scanner.Text())
}

func indexFieldToColumn(tStruct reflect.Type, header row) ([]int, error) {
	ret := make([]int, tStruct.NumField())
	for i := 0; i < tStruct.NumField(); i++ {
		tag := tStruct.Field(i).Tag.Get("table")
		if tag == "" {
			continue
		}

		index := header.index(tag)
		if index == -1 {
			return nil, fmt.Errorf("column '%s' not found in table", tag)
		}

		ret[i] = index
	}
	return ret, nil
}

// unmarshalStruct unmarshals r into value of tStruct type.
// When successful, this returns pointer to the value and nil.
// When failure, this returns zero-value of reflect.Value and non-nil error.
func unmarshalStruct(tStruct reflect.Type, row row, indices []int) (reflect.Value, error) {
	// Not using reflect.Zero because of "settability".
	// See https://blog.golang.org/laws-of-reflection
	vPointer := reflect.New(tStruct)
	for fi := 0; fi < vPointer.Elem().NumField(); fi++ {
		vField := vPointer.Elem().Field(fi)
		tField := tStruct.Field(fi)
		s := row[indices[fi]]
		if reflect.PtrTo(tField.Type).Implements(unmarshalerType) {
			if err := unmarshalUnmarshalerType(vField, s); err != nil {
				return reflect.Value{}, fmt.Errorf("unmarshaling Unmarshaler: %v", err)
			}
			continue
		}

		if err := unmarshalBasicType(vField, s); err != nil {
			return reflect.Value{}, fmt.Errorf("unmarshaling basic type: %v", err)
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
