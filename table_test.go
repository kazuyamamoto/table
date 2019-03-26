package table

import (
	"fmt"
	"reflect"
	"testing"
)

func TestUnmarshal(t *testing.T) {
	tests := []struct {
		name string
		s    string
		want []testRow
	}{
		{
			"common usage",
			`

string  | custom || int   | float | bool  | uint | escape  | 文字列
------- | ------ || ----- | ----- | ----- | ---- | ------- | --------
abc     | OK     || 302   | 1.234 | true  | 7890 | abc\nd  | あいうえお
        | NG     || -0x20 | -5    | F     | 3333 | \|\\n\| | 日本語

ignored lines...

`,
			[]testRow{
				// bool, int, uint, float, string, 文字列, custom, escape
				{true, 302, 7890, 1.234, "abc", "あいうえお", true, "abc\nd"},
				{false, -0x20, 3333, -5, "", "日本語", false, "|\\n|"},
			},
		},
		{
			"empty",
			``,
			nil,
		},
		{
			"ignored lines only",
			`
    

`,
			nil,
		},
		{
			"header row only",
			`string  | custom || int   | float | bool  | uint | escape  | 文字列
`,
			nil,
		},
		{
			"header row and delimiter row",
			`
string  | custom || int   | float | bool  | uint | escape  | 文字列
------- | ------ || ----- | ----- | ----- | ---- | ------- | --------
`,
			nil,
		},
		{
			"delimiter row only",
			`
------- | ------ || ----- | ----- | ----- | ---- | ------- | --------
`,
			nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var table []testRow
			if err := Unmarshal([]byte(tt.s), &table); err != nil {
				t.Fatal(err)
			}

			if !reflect.DeepEqual(table, tt.want) {
				t.Fatalf("want %v, got %v", tt.want, table)
			}
		})
	}
}

func TestUnmarshal_error_string(t *testing.T) {
	tests := []struct {
		name string
		s    string
	}{
		{
			"different column number",
			`
string  | custom || int   | float | bool  | uint | escape  | 文字列
------- | ------ || ----- | ----- | ----- | ---- | ------- | --------
        | NG     || -0x20 | -5    | F     | 3333 | \|\\n\|   日本語
`,
		},
		{
			"required",
			`
string  | custom || int   | float | bool  | uint | escape  
------- | ------ || ----- | ----- | ----- | ---- | ------- 
        | NG     || -0x20 | -5    | F     | 3333 | \|\\n\| 
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var table []testRow
			err := Unmarshal([]byte(tt.s), &table)
			t.Log(err)
			if err == nil {
				t.Fatal("error should be non-nil")
			}
		})
	}
}

func TestUnmarshal_error_table(t *testing.T) {
	const s = `
string  | custom || int   | float | bool  | uint | escape  | 文字列
------- | ------ || ----- | ----- | ----- | ---- | ------- | --------
abc     | OK     || 302   | 1.234 | true  | 7890 | abc\nd  | あいうえお
`

	tests := []interface{}{
		nil,
		123,
		"abc",
		struct{}{},
		testRow{},
		&testRow{},
		[]*testRow{},
		&[]*testRow{},
		&[][]string{},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%T", tt), func(t *testing.T) {
			if err := Unmarshal([]byte(s), tt); err == nil {
				t.Fatal("err should be non-nil")
			}
		})
	}
}

type testRow struct {
	Bool      bool    `table:"bool"`
	Int       int     `table:"int"`
	Uint      uint    `table:"uint"`
	Float     float32 `table:"float"`
	String    string  `table:"string"`
	Mojiretsu string  `table:"文字列"`
	Custom    okNg    `table:"custom"`
	Escape    string  `table:"escape"`
}

// okNg is a sample Unmarshaler.
type okNg bool

func (o *okNg) UnmarshalTable(p []byte) error {
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

func TestUnmarshal_error_parseBasicType(t *testing.T) {
	const s = `value
--
?`

	tests := []struct {
		name  string
		table interface{}
	}{
		{"int", &[]intRow{}},
		{"uint", &[]uintRow{}},
		{"bool", &[]boolRow{}},
		{"float", &[]floatRow{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := Unmarshal([]byte(s), tt.table); err == nil {
				t.Fatal("error should be non-nil")
			}
		})
	}
}

type intRow struct {
	Value int `table:"value"`
}

type uintRow struct {
	Value uint `table:"value"`
}

type boolRow struct {
	Value bool `table:"value"`
}

type floatRow struct {
	Value float32 `table:"value"`
}

func TestUnmarshal_unescapeCustomString(t *testing.T) {
	const s = `
escape 
------- 
abc\nd  
\|\\n\| 
`

	var table []testRowCustomString
	err := Unmarshal([]byte(s), &table)
	if err != nil {
		t.Fatal(err)
	}

	want := customString("abc\nd")
	if table[0].CustomString != want {
		t.Fatalf("want %q, got %q", want, table[0].CustomString)
	}

	want = customString("|\\n|")
	if table[1].CustomString != want {
		t.Fatalf("want %q, got %q", want, table[1].CustomString)
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
	tests := []struct {
		name string
		s    string
		want []multilineHeaderRow
	}{
		{
			"golden path",
			`
                   | dual line | 三行     | skipped
                   | header    | の       |
single line header |           | ヘッダー | header
------------------ | --------- | -------- | --------
1                  | 2	       | 3        | 4
`,
			[]multilineHeaderRow{{1, 2, 3, 4}},
		},
		{
			"no body",
			`
                   | dual line | 三行     | skipped
                   | header    | の       |
single line header |           | ヘッダー | header

`,
			nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var table []multilineHeaderRow
			if err := Unmarshal([]byte(tt.s), &table); err != nil {
				t.Fatal(err)
			}

			if !reflect.DeepEqual(table, tt.want) {
				t.Fatalf("want %q, got %q", tt.want, table)
			}
		})
	}
}

func TestUnmarshal_multilineHeader_error(t *testing.T) {
	tests := []struct {
		name string
		s    string
	}{
		{
			"different column number",
			`
                   | dual line | 三行     | skipped
                   | header    | の       |
single line header |           | ヘッダー   header
------------------ | --------- | -------- | --------
1                  | 2	       | 3        | 4

`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var table []multilineHeaderRow
			if err := Unmarshal([]byte(tt.s), &table); err == nil {
				t.Fatal("error should be non-nil")
			}
		})
	}
}

func BenchmarkUnmarshal(b *testing.B) {
	s := []byte(`
string  | custom || int   | float | bool  | uint | escape  | 文字列
------- | ------ || ----- | ----- | ----- | ---- | ------- | --------
abc     | OK     || 302   | 1.234 | true  | 7890 | abc\nd  | あいうえお
        | NG     || -0x20 | -5    | F     | 3333 | \|\\n\| | 日本語
`)

	for n := 0; n < b.N; n++ {
		var tbl []testRow
		_ = Unmarshal(s, &tbl)
	}
}
