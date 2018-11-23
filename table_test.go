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

func TestUnescape(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{``, ""},
		{`a`, "a"},
		{` a`, " a"},
		{`\n`, "\n"},
		{`\\`, "\\"},
		{`\n\n`, "\n\n"},
		{`a\nb\nc`, "a\nb\nc"},
	}

	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got, err := unescape(tt.in)
			if err != nil {
				t.Fatal(err)
			}

			if got != tt.want {
				t.Fatalf("want %q, got %q", tt.want, got)
			}
		})
	}
}

func TestUnescape_error(t *testing.T) {
	tests := []string{
		`\r`,
		`\|`,
		`\`,
	}

	for _, tt := range tests {
		t.Run(tt, func(t *testing.T) {
			l, err := unescape(tt)
			if err == nil {
				t.Fatalf("err should be non-nil. unescaped=%q", l)
			}
		})
	}
}

func TestParseRow(t *testing.T) {
	tests := []struct {
		s    string
		want row
	}{
		{"", nil},
		{" ", nil},
		{"a", row{"a"}},
		{"a|b", row{"a", "b"}},
		{"|a|b", row{"", "a", "b"}},
		{"a|b|", row{"a", "b", ""}},
	}

	for i, tt := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			got, err := unmarshalRow(tt.s)
			if err != nil {
				t.Fatal(err)
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("want %v, got %v", tt.want, got)
			}
		})
	}
}

func TestRow_isDelimiter(t *testing.T) {
	tests := []struct {
		row  row
		want bool
	}{
		{[]string{"-"}, true},
		{[]string{"--"}, true},
		{[]string{"-a"}, false},
		{[]string{"-", "-"}, true},
		{[]string{" - "}, true},
		{[]string{"a"}, false},
		{[]string{"a", "-"}, false},
		{[]string{""}, true},
		{[]string{"", "-"}, true},
		{[]string{"", "a"}, false},
	}

	for i, tt := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			if tt.row.isDelimiter() != tt.want {
				t.Fatalf("row.isDelimiter() should be %v", tt.want)
			}
		})
	}
}
