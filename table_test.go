package table

import (
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

string  | custom || int   | float | bool     | uint | escape | 文字列    
------- | ------ || ----- | ----- | -------- | ---- | ------ | --------
abc     | OK     || 302   | 1.234 | true     | 7890 | abc\nd | あいうえお  
        | NG     || -0x20 | -5    | non-bool | 3333 | abc\\n | 日本語    

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
		Escape:    "abc\\n",
	},
}

// Unmarshaler implementation.
type okng bool

func (o *okng) UnmarshalTable(p []byte) error {
	*o = string(p) == "OK"
	return nil
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

	wantNRows := 2
	if len(tbl) != wantNRows {
		t.Fatalf("#rows: want %v, got %v, table %v", wantNRows, len(tbl), tbl)
	}

	if !reflect.DeepEqual(wantTable, tbl) {
		t.Fatalf("want %v, got %v", wantTable, tbl)
	}
}

func TestUnmarshal_error(t *testing.T) {
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

func TestUnmarshaler(t *testing.T) {
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

func TestUnmarshal_unescapeCustomString(t *testing.T) {
	var tbl []testRowCustomString
	if err := Unmarshal(table, &tbl); err != nil {
		t.Fatal(err)
	}

	if tbl[0].CustomString != customString("abc\nd") {
		t.Fatalf("want %q, got %q", tbl[0].CustomString, "abc\nd")
	}

	if tbl[1].CustomString != customString("abc\\n") {
		t.Fatalf("want %q, got %q", tbl[1].CustomString, "abc\\n")
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
