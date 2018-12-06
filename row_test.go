package table

import (
	"reflect"
	"strconv"
	"testing"
)

func TestParseRow(t *testing.T) {
	tests := []struct {
		s    string
		want row
	}{
		{"a", row{"a"}},
		{"a ", row{"a"}},
		{"a|b", row{"a", "b"}},
		{"|a|b", row{"", "a", "b"}},
		{"a|b|", row{"a", "b", ""}},
		{"\\|", row{"|"}},
		{"||\\||\\||", row{"", "", "|", "|", ""}},
		{"\\|\\\\n\\|", row{"|\\n|"}},
		{"\\\\|", row{"\\", ""}},
		{"\\n", row{"\n"}},
	}

	for i, tt := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			got, err := parseRow(tt.s)
			if err != nil {
				t.Error(err)
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("want %v, got %v", tt.want, got)
			}
		})
	}
}

func TestParseRow_error(t *testing.T) {
	tests := []string{"\\a", "\\r"}

	for i, tt := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			got, err := parseRow(tt)
			if err == nil {
				t.Errorf("should be error: got %v", got)
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

func TestRow_merge(t *testing.T) {
	tests := []struct {
		to, from, want row
	}{
		{row{"a"}, row{"b"}, row{"a b"}},
		{row{"a", "c"}, row{"b", "d"}, row{"a b", "c d"}},
		{row{"a"}, row{""}, row{"a"}},
		{row{""}, row{"c"}, row{"c"}},
		{row{""}, row{""}, row{""}},
		{nil, nil, nil},
		{row{}, row{}, row{}},
		// "a " and " a" are not tested.
		// These could not be elements of row.
		// See parseRow.
	}

	for i, tt := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			if err := tt.to.merge(tt.from); err != nil {
				t.Fatal(err)
			}

			if !reflect.DeepEqual(tt.want, tt.to) {
				t.Fatalf("want %q, got %q", tt.want, tt.to)
			}
		})
	}
}

func TestRow_merge_error(t *testing.T) {
	tests := []struct {
		to, from row
	}{
		{row{"a"}, row{}},
		{row{}, row{"a"}},
		{row{"a"}, row{"b", "c"}},
		{nil, row{"a"}},
		{row{"a"}, nil},
		{row{}, row{"a"}},
		{row{"a"}, row{}},
	}

	for i, tt := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			if err := tt.to.merge(tt.from); err == nil {
				t.Fatal("error should be non-nil")
			}
		})
	}
}
