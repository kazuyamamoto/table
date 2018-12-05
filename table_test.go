package table

import (
	"fmt"
	"reflect"
	"strconv"
	"testing"
)

type testRow struct {
	Bool      bool    `table:"bool"`
	Int       int     `table:"int"`
	Uint      uint    `table:"uint"`
	Float     float32 `table:"float"`
	String    string  `table:"string"`
	Mojiretsu string  `table:"文字列"`
	Custom    okng    `table:"custom"`
	Escape    string  `table:"escape"`
}

var table = []byte(`

string  | custom || int   | float | bool  | uint | escape  | 文字列    
------- | ------ || ----- | ----- | ----- | ---- | ------- | --------
abc     | OK     || 302   | 1.234 | true  | 7890 | abc\nd  | あいうえお  
        | NG     || -0x20 | -5    | F     | 3333 | \|\\n\| | 日本語    

ignored lines...

`)

var wantTable = []testRow{
	{
		Bool:      true,
		Int:       302,
		Uint:      7890,
		Float:     1.234,
		String:    "abc",
		Mojiretsu: "あいうえお",
		Custom:    true,
		Escape:    "abc\nd",
	},
	{
		Bool:      false,
		Int:       -0x20,
		Uint:      3333,
		Float:     -5,
		String:    "",
		Mojiretsu: "日本語",
		Custom:    false,
		Escape:    "|\\n|",
	},
}

// Unmarshaler implementation.
type okng bool

func (o *okng) UnmarshalTable(p []byte) error {
	switch string(p) {
	case "OK":
		*o = true
		return nil
	case "NG":
		*o = false
		return nil
	}

	return fmt.Errorf("neither OK nor NG: %q", string(p))
}

func (o okng) String() string {
	if o {
		return "OK"
	}

	return "NG"
}

func TestUnmarshal(t *testing.T) {
	var tbl []testRow
	err := Unmarshal(table, &tbl)
	if err != nil {
		t.Fatal(err)
	}

	if len(tbl) != len(wantTable) {
		t.Fatalf("#rows: want %v, got %v, table %v", len(wantTable), len(tbl), tbl)
	}

	if !reflect.DeepEqual(wantTable, tbl) {
		t.Fatalf("want %v, got %v", wantTable, tbl)
	}
}

func TestUnmarshal_rowStructParameterError(t *testing.T) {
	tests := []interface{}{
		nil,
		123,
		"abc",
		struct{}{},
		testRow{},
		&testRow{},
		&[][]string{},
	}

	for i, tt := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			err := Unmarshal(table, tt)
			if err == nil {
				t.Fatal("err should be non-nil")
			}
		})
	}
}

func TestUnmarshaler_UnmarshalTable(t *testing.T) {
	var sut okng
	if !reflect.PtrTo(reflect.TypeOf(sut)).Implements(unmarshalerType) {
		t.Fatal()
	}

	err := sut.UnmarshalTable([]byte("OK"))
	if err != nil {
		t.Fatal(err)
	}

	if !sut {
		t.Fatal("UnmarshalTable(OK) should be true")
	}
}

type intTableRow struct {
	Value int `table:"intValue"`
}

func TestUnmarshal_intError(t *testing.T) {
	ts := []byte(`intValue
---------
x`)
	var intTable []intTableRow
	if err := Unmarshal(ts, &intTable); err == nil {
		t.Fatal("error should be non-nil")
	}
}

type uintTableRow struct {
	Value uint `table:"uintValue"`
}

func TestUnmarshal_uintError(t *testing.T) {
	ts := []byte(`uintValue
---------
x`)
	var uintTable []uintTableRow
	if err := Unmarshal(ts, &uintTable); err == nil {
		t.Fatal("error should be non-nil")
	}
}

type boolTableRow struct {
	Value bool `table:"boolValue"`
}

func TestUnmarshal_boolError(t *testing.T) {
	ts := []byte(`boolValue
---------
x`)
	var boolTable []boolTableRow
	if err := Unmarshal(ts, &boolTable); err == nil {
		t.Fatal("error should be non-nil")
	}
}

type floatTableRow struct {
	Value float32 `table:"floatValue"`
}

func TestUnmarshal_floatError(t *testing.T) {
	ts := []byte(`floatValue
---------
x`)
	var floatTable []floatTableRow
	if err := Unmarshal(ts, &floatTable); err == nil {
		t.Fatal("error should be non-nil")
	}
}

func TestUnmarshal_UnmarshalerError(t *testing.T) {
	ts := []byte(`floatValue
---------
x`)
	var okngTable []okng
	if err := Unmarshal(ts, &okngTable); err == nil {
		t.Fatal("error should be non-nil")
	}
}

func TestUnmarshal_unescapeCustomString(t *testing.T) {
	var tbl []testRowCustomString
	if err := Unmarshal(table, &tbl); err != nil {
		t.Fatal(err)
	}

	want := customString("abc\nd")
	if tbl[0].CustomString != want {
		t.Fatalf("want %q, got %q", want, tbl[0].CustomString)
	}

	want = customString("|\\n|")
	if tbl[1].CustomString != want {
		t.Fatalf("want %q, got %q", want, tbl[1].CustomString)
	}
}

type testRowCustomString struct {
	CustomString customString `table:"escape"`
}

type customString string

func (c *customString) UnmarshalTable(p []byte) error {
	*c = customString(p)
	return nil
}
