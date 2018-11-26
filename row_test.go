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
		{"\\|", row{"|"}},
		{"||\\||\\||", row{"", "", "|", "|", ""}},
		{"\\|\\\\n\\|", row{"|\\n|"}},
		{"\\\\|", row{"\\", ""}},
	}

	for i, tt := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			got, err := unmarshalRow(tt.s)
			if err != nil {
				t.Error(err)
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("want %v, got %v", tt.want, got)
			}
		})
	}
}

func TestUnmarshalRow_error(t *testing.T) {
	tests := []string{"\\a", "\\r"}

	for i, tt := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			got, err := unmarshalRow(tt)
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
