package table

import (
	"reflect"
	"strconv"
	"testing"
)

func TestUnmarshalRow(t *testing.T) {
	tests := []struct {
		s    string
		want row
	}{
		{"a", row{"a"}},
		{"a|b", row{"a", "b"}},
		{"|a|b", row{"", "a", "b"}},
		{"a|b|", row{"a", "b", ""}},
		{"\\|", row{"\\|"}},
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

func TestRow_unescape(t *testing.T) {
	sut := row{"a", "\\n", "\\\\"}
	want := row{"a", "\n", "\\"}

	if err := sut.unescape(); err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(sut, want) {
		t.Fatalf("want %v, got %v", want, sut)
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
		{`\|`, "|"},
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
