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

var tableString = []byte(`

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
	err := Unmarshal(tableString, &tbl)
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

func TestUnmarshal_invalidRowStruct(t *testing.T) {
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
			err := Unmarshal(tableString, tt)
			if err == nil {
				t.Fatal("err should be non-nil")
			}
		})
	}
}

func TestUnmarshal_empty(t *testing.T) {
	s := []byte("")

	var tbl []testRow
	err := Unmarshal(s, &tbl)
	if err != nil {
		t.Fatal(err)
	}

	if len(tbl) > 0 {
		t.Fatalf("table size should be 0: %d", len(tbl))
	}
}

func TestUnmarshal_ignoredLinesOnly(t *testing.T) {
	s := []byte(`
    

`)

	var tbl []testRow
	err := Unmarshal(s, &tbl)
	if err != nil {
		t.Fatal(err)
	}

	if len(tbl) > 0 {
		t.Fatalf("table size should be 0: %d", len(tbl))
	}
}

func TestUnmarshal_headerOnly(t *testing.T) {
	s := []byte(`string  | custom || int   | float | bool  | uint | escape  | 文字列
`)

	var tbl []testRow
	err := Unmarshal(s, &tbl)
	if err != nil {
		t.Fatal(err)
	}

	if len(tbl) > 0 {
		t.Fatalf("table size should be 0: %d", len(tbl))
	}
}

func TestUnmarshal_headerAndDelimiter(t *testing.T) {
	s := []byte(`
string  | custom || int   | float | bool  | uint | escape  | 文字列
------- | ------ || ----- | ----- | ----- | ---- | ------- | --------
`)

	var tbl []testRow
	err := Unmarshal(s, &tbl)
	if err != nil {
		t.Fatal(err)
	}

	if len(tbl) > 0 {
		t.Fatalf("table size should be 0: %d", len(tbl))
	}
}

func TestUnmarshal_delimiterOnly(t *testing.T) {
	s := []byte(`
------- | ------ || ----- | ----- | ----- | ---- | ------- | --------
`)

	var tbl []testRow
	err := Unmarshal(s, &tbl)
	if err != nil {
		t.Fatal(err)
	}

	if len(tbl) > 0 {
		t.Fatalf("table size should be 0: %d", len(tbl))
	}
}

func TestUnmarshal_numberOfColumnDifferent(t *testing.T) {
	s := []byte(`

string  | custom || int   | float | bool  | uint | escape  | 文字列
------- | ------ || ----- | ----- | ----- | ---- | ------- | --------
abc     | OK     || 302   | 1.234 | true  | 7890 | abc\nd  | あいうえお
        | NG     || -0x20 | -5    | F     | 3333 | \|\\n\|   日本語
`)
	var tbl []testRow
	err := Unmarshal(s, &tbl)
	if err == nil {
		t.Fatal("error should be non-nil")
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

func TestUnmarshal_invalidInt(t *testing.T) {
	s := []byte(`intValue
---------
x`)
	var intTable []intTableRow
	err := Unmarshal(s, &intTable)
	if err == nil {
		t.Fatal("error should be non-nil")
	}
}

type uintTableRow struct {
	Value uint `table:"uintValue"`
}

func TestUnmarshal_invalidUint(t *testing.T) {
	s := []byte(`uintValue
---------
x`)
	var uintTable []uintTableRow
	err := Unmarshal(s, &uintTable)
	if err == nil {
		t.Fatal("error should be non-nil")
	}
}

type boolTableRow struct {
	Value bool `table:"boolValue"`
}

func TestUnmarshal_invalidBool(t *testing.T) {
	s := []byte(`boolValue
---------
x`)
	var boolTable []boolTableRow
	err := Unmarshal(s, &boolTable)
	if err == nil {
		t.Fatal("error should be non-nil")
	}
}

type floatTableRow struct {
	Value float32 `table:"floatValue"`
}

func TestUnmarshal_invalidFloat(t *testing.T) {
	s := []byte(`floatValue
---------
x`)
	var floatTable []floatTableRow
	err := Unmarshal(s, &floatTable)
	if err == nil {
		t.Fatal("error should be non-nil")
	}
}

func TestUnmarshal_unescapeCustomString(t *testing.T) {
	var tbl []testRowCustomString
	err := Unmarshal(tableString, &tbl)
	if err != nil {
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

type multilineHeaderRow struct {
	Single  int `table:"single line header"`
	Dual    int `table:"dual line header"`
	Triple  int `table:"三行 の ヘッダー"`
	Skipped int `table:"skipped header"`
}

func TestUnmarshal_multilineHeader(t *testing.T) {
	var s = []byte(`
                   | dual line | 三行     | skipped
                   | header    | の       |
single line header |           | ヘッダー | header
------------------ | --------- | -------- | --------
1                  | 2	       | 3        | 4
`)

	var tbl []multilineHeaderRow
	err := Unmarshal(s, &tbl)
	if err != nil {
		t.Fatal(err)
	}

	want := []multilineHeaderRow{
		{1, 2, 3, 4},
	}
	if !reflect.DeepEqual(tbl, want) {
		t.Fatalf("want %q, got %q", want, tbl)
	}
}

func TestUnmarshal_multilineHeader_noBody(t *testing.T) {
	var s = []byte(`
                   | dual line | 三行     | skipped
                   | header    | の       |
single line header |           | ヘッダー | header

`)

	var tbl []multilineHeaderRow
	err := Unmarshal(s, &tbl)
	if err != nil {
		t.Fatal(err)
	}

	var want []multilineHeaderRow
	if !reflect.DeepEqual(tbl, want) {
		t.Fatalf("want %q, got %q", want, tbl)
	}
}

func TestUnmarshal_multilineHeader_differentColumns(t *testing.T) {
	var s = []byte(`
                   | dual line | 三行     | skipped
                   | header    | の       |
single line header |           | ヘッダー   header
------------------ | --------- | -------- | --------
1                  | 2	       | 3        | 4

`)

	var tbl []multilineHeaderRow
	err := Unmarshal(s, &tbl)
	if err == nil {
		t.Fatal("error should be non-nil")
	}
}
