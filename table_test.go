package table

import (
	"fmt"
	"reflect"
	"testing"
)

type testRow struct {
	Bool      bool    `table:"bool value"`
	Int       int     `table:"int value"`
	Uint      uint    `table:"uint value"`
	Float     float32 `table:"float value"`
	String    string  `table:"string value"`
	Mojiretsu string  `table:"文字列 の 値"`
	Custom    okNg    `table:"custom value"`
	Escaped   string  `table:"escaped value"`
}

// okNg is a custom Unmarshaler.
type okNg bool

func (o *okNg) UnmarshalTable(p []byte) error {
	switch string(p) {
	case "OK", "|":
		*o = true
		return nil
	case "NG":
		*o = false
		return nil
	}

	return fmt.Errorf("neither OK nor NG: %q", string(p))
}

func TestUnmarshal(t *testing.T) {
	tests := []struct {
		name string
		s    string
		want []testRow
	}{
		{
			"common usage",
			`
string value | custom value || int value | float value | bool value | uint value | escaped value | 文字列 の 値
------------ | ------------ || --------- | ----------- | ---------- | ---------- | ------------- | ------------
abc          | OK           || 302       | 1.234       | true       | 7890       | abc\nd        | あいうえお
             | NG           || -0x20     | -5          | F          | 3333       | \|\\n\|       | 日本語

ignored lines...
`,
			[]testRow{
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
			`string value | custom value || int value | float value | bool value | uint value | escaped value | 文字列 の 値`,
			nil,
		},
		{
			"header row and delimiter row",
			`
string value | custom value || int value | float value | bool value | uint value | escaped value | 文字列 の 値
------------ | ------------ || --------- | ----------- | ---------- | ---------- | ------------- | ------------
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
		{
			"multi-line header",
			`
string | custom ||       | float | bool  |       | escaped | 文字列
value  |        || int   | value |       | uint  | value   | の
       | value  || value |       | value | value |         | 値
`,
			nil,
		},
		{
			"escaping for custom Unmarshaler",
			`
string value | custom value || int value | float value | bool value | uint value | escaped value | 文字列 の 値
------------ | ------------ || --------- | ----------- | ---------- | ---------- | ------------- | ------------
abc          | \|           || 302       | 1.234       | true       | 7890       | abc\nd        | あいうえお
`,
			[]testRow{
				{true, 302, 7890, 1.234, "abc", "あいうえお", true, "abc\nd"},
			},
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

func TestUnmarshal_error(t *testing.T) {
	tests := []struct {
		name  string
		s     string
		table interface{}
	}{
		{
			"different column number",
			`
string value | custom value || int value | float value | bool value | uint value | escaped value | 文字列 の 値
------------ | ------------ || --------- | ----------- | ---------- | ---------- | ------------- | ------------
abc          | OK           || 302       | 1.234       | true       | 7890       | \|\\n\|         あいうえお
`,
			&[]testRow{},
		},
		{
			"required column",
			`
string value | custom value || int value | float value | bool value | uint value | escaped value
------------ | ------------ || --------- | ----------- | ---------- | ---------- | ------------- 
abc          | OK           || 302       | 1.234       | true       | 7890       | abc\nd
`,
			&[]testRow{},
		},
		{
			"table:nil",
			`
string value | custom value || int value | float value | bool value | uint value | escaped value | 文字列 の 値
------------ | ------------ || --------- | ----------- | ---------- | ---------- | ------------- | ------------
abc          | OK           || 302       | 1.234       | true       | 7890       | abc\nd        | あいうえお
`,
			nil,
		},
		{
			"table:int",
			`
string value | custom value || int value | float value | bool value | uint value | escaped value | 文字列 の 値
------------ | ------------ || --------- | ----------- | ---------- | ---------- | ------------- | ------------
abc          | OK           || 302       | 1.234       | true       | 7890       | abc\nd        | あいうえお
`,
			123,
		},
		{
			"table:string",
			`
string value | custom value || int value | float value | bool value | uint value | escaped value | 文字列 の 値
------------ | ------------ || --------- | ----------- | ---------- | ---------- | ------------- | ------------
abc          | OK           || 302       | 1.234       | true       | 7890       | abc\nd        | あいうえお
`,
			"abc",
		},
		{
			"table:row",
			`
string value | custom value || int value | float value | bool value | uint value | escaped value | 文字列 の 値
------------ | ------------ || --------- | ----------- | ---------- | ---------- | ------------- | ------------
abc          | OK           || 302       | 1.234       | true       | 7890       | abc\nd        | あいうえお
`,
			testRow{},
		},
		{
			"table:pointer to row",
			`
string value | custom value || int value | float value | bool value | uint value | escaped value | 文字列 の 値
------------ | ------------ || --------- | ----------- | ---------- | ---------- | ------------- | ------------
abc          | OK           || 302       | 1.234       | true       | 7890       | abc\nd        | あいうえお
`,
			&testRow{},
		},
		{
			"table:slice of row",
			`
string value | custom value || int value | float value | bool value | uint value | escaped value | 文字列 の 値
------------ | ------------ || --------- | ----------- | ---------- | ---------- | ------------- | ------------
abc          | OK           || 302       | 1.234       | true       | 7890       | abc\nd        | あいうえお
`,
			[]testRow{},
		},
		{
			"table:slice of pointer to row",
			`
string value | custom value || int value | float value | bool value | uint value | escaped value | 文字列 の 値
------------ | ------------ || --------- | ----------- | ---------- | ---------- | ------------- | ------------
abc          | OK           || 302       | 1.234       | true       | 7890       | abc\nd        | あいうえお
`,
			[]*testRow{},
		},
		{
			"table:pointer to slice of pointer to row",
			`
string value | custom value || int value | float value | bool value | uint value | escaped value | 文字列 の 値
------------ | ------------ || --------- | ----------- | ---------- | ---------- | ------------- | ------------
abc          | OK           || 302       | 1.234       | true       | 7890       | abc\nd        | あいうえお
`,
			&[]*testRow{},
		},
		{
			"table:pointer to slice of non-struct",
			`
string value | custom value || int value | float value | bool value | uint value | escaped value | 文字列 の 値
------------ | ------------ || --------- | ----------- | ---------- | ---------- | ------------- | ------------
abc          | OK           || 302       | 1.234       | true       | 7890       | abc\nd        | あいうえお
`,
			&[][]string{},
		},
		{
			"different number of columns in header",
			`
string | custom ||       | float | bool  |       | escaped | 文字列
value  |        || int   | value |       | uint  | value   | の
       | value  || value |       | value | value |           値
------ | ------ || ----- | ----- | ----- | ----- | ------- | ------------
abc    | OK     || 302   | 1.234 | true  | 7890  | abc\nd  | あいうえお
`,
			&[]testRow{},
		},
		{
			"parse int",
			`
string value | custom value || int value | float value | bool value | uint value | escaped value | 文字列 の 値
------------ | ------------ || --------- | ----------- | ---------- | ---------- | ------------- | ------------
abc          | OK           || ?         | 1.234       | true       | 7890       | abc\nd        | あいうえお
`,
			&[]testRow{},
		},
		{
			"parse uint",
			`
string value | custom value || int value | float value | bool value | uint value | escaped value | 文字列 の 値
------------ | ------------ || --------- | ----------- | ---------- | ---------- | ------------- | ------------
abc          | OK           || 302       | 1.234       | true       | ?          | abc\nd        | あいうえお
`,
			&[]testRow{},
		},
		{
			"parse float",
			`
string value | custom value || int value | float value | bool value | uint value | escaped value | 文字列 の 値
------------ | ------------ || --------- | ----------- | ---------- | ---------- | ------------- | ------------
abc          | OK           || 302       | ?           | true       | 7890       | abc\nd        | あいうえお
`,
			&[]testRow{},
		},
		{
			"parse bool",
			`
string value | custom value || int value | float value | bool value | uint value | escaped value | 文字列 の 値
------------ | ------------ || --------- | ----------- | ---------- | ---------- | ------------- | ------------
abc          | OK           || 302       | 1.234       | x          | 7890       | abc\nd        | あいうえお
`,
			&[]testRow{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Unmarshal([]byte(tt.s), tt.table)

			// t.Log(err)

			if err == nil {
				t.Fatal("error should be non-nil")
			}
		})
	}
}

func BenchmarkUnmarshal(b *testing.B) {
	s := []byte(`
string value | custom value || int value  | float value | bool value | uint value | escaped value | 文字列 の 値
------------ | ------------ || ---------- | ----------- | ---------- | ---------- | ------------- | ------------
abc          | OK           || 302        | 1.234       | true       | 7890       | abc\nd        | あいうえお
             | NG           || -0x20      | -5          | F          | 3333       | \|\\n\|       | 日本語
`)

	for n := 0; n < b.N; n++ {
		var tbl []testRow
		_ = Unmarshal(s, &tbl)
	}
}
