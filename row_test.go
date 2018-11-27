package table

import (
	"reflect"
	"strconv"
	"testing"
)

func TestParseRow(t *testing.T) {
	tests := []struct {
		s    string
		want Row
	}{
		{"a", Row{"a"}},
		{"a|b", Row{"a", "b"}},
		{"|a|b", Row{"", "a", "b"}},
		{"a|b|", Row{"a", "b", ""}},
		{"\\|", Row{"|"}},
		{"||\\||\\||", Row{"", "", "|", "|", ""}},
		{"\\|\\\\n\\|", Row{"|\\n|"}},
		{"\\\\|", Row{"\\", ""}},
	}

	for i, tt := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			got, err := ParseRow(tt.s)
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
			got, err := ParseRow(tt)
			if err == nil {
				t.Errorf("should be error: got %v", got)
			}
		})
	}
}

func TestRow_isDelimiter(t *testing.T) {
	tests := []struct {
		row  Row
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
			if tt.row.IsDelimiter() != tt.want {
				t.Fatalf("Row.IsDelimiter() should be %v", tt.want)
			}
		})
	}
}
