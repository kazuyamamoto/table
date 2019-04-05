package table

import (
	"fmt"
	"reflect"
	"strconv"
	"testing"
)

func TestParseRow(t *testing.T) {
	tests := []struct {
		s         string
		wantRow   row
		wantMerge bool
	}{
		{``, row{}, false},
		{` `, row{}, false},
		{`  `, row{}, false},
		{`a`, row{"a"}, false},
		{`a `, row{"a"}, false},
		{`a  `, row{"a"}, false},
		{` a`, row{"a"}, false},
		{`  a`, row{"a"}, false},
		{` a `, row{"a"}, false},
		{`a|b`, row{"a", "b"}, false},
		{` a|b`, row{"a", "b"}, false},
		{`a|b `, row{"a", "b"}, false},
		{`a |b`, row{"a", "b"}, false},
		{`a| b`, row{"a", "b"}, false},
		{`a | b`, row{"a", "b"}, false},
		{` a | b`, row{"a", "b"}, false},
		{`a | b `, row{"a", "b"}, false},
		{`|a|b`, row{"", "a", "b"}, false},
		{` |a|b`, row{"", "a", "b"}, false},
		{`a|b|`, row{"a", "b", ""}, false},
		{`a|b| `, row{"a", "b", ""}, false},
		{`\|`, row{"|"}, false},
		{` \|`, row{"|"}, false},
		{`\| `, row{"|"}, false},
		{`||\||\||`, row{"", "", "|", "|", ""}, false},
		{`\|\\n\|`, row{"|\\n|"}, false},
		{`\\|`, row{"\\", ""}, false},
		{`\n`, row{"\n"}, false},
		{`a\`, row{"a"}, true},
		{`a\ `, row{"a"}, true},
		{` a\`, row{"a"}, true},
		{`a \`, row{"a"}, true},
		{`\`, row{""}, true},
		{`\ `, row{""}, true},
		{` \`, row{""}, true},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("<%s>", tt.s), func(t *testing.T) {
			got, cont, err := parseRow(tt.s)
			if err != nil {
				t.Fatal(err)
			}

			if !reflect.DeepEqual(got, tt.wantRow) {
				t.Fatalf("row: want %v(len=%d), got %v(len=%d)", tt.wantRow, len(tt.wantRow), got, len(got))
			}

			if cont != tt.wantMerge {
				t.Fatalf("want merge: want %v, got %v", tt.wantMerge, cont)
			}
		})
	}
}

func TestParseRow_error(t *testing.T) {
	tests := []string{"\\a", "\\r"}

	for i, tt := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			got, cont, err := parseRow(tt)
			if err == nil {
				t.Fatalf("should be error: got %v, cont %v", got, cont)
			}
		})
	}
}

func TestRow_isDelim(t *testing.T) {
	tests := []struct {
		row  row
		want bool
	}{
		{row{"-"}, true},
		{row{"--"}, true},
		{row{"-a"}, false},
		{row{"-", "-"}, true},
		{row{" - "}, true},
		{row{"a"}, false},
		{row{"a", "-"}, false},
		{row{""}, true},
		{row{"", "-"}, true},
		{row{"", "a"}, false},
	}

	for i, tt := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			if tt.row.isDelim() != tt.want {
				t.Fatalf("row.isDelim() should be %v", tt.want)
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
