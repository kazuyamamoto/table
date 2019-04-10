// Package table provides functionality to unmarshal table string into slice of
// struct. Table format is like those of lightweight markup languages:
//
//   string  | custom | int   | float | bool  | uint | escape | 文字列
//   ------- | ------ | ----- | ----- | ----- | ---- | ------ | --------
//   abc     | OK     | 302   | 1.234 | true  | 7890 | abc\nd | あいうえお
//           | NG     | -0x20 | -5    | F     | 3333 | \\n\|  | 日本語
//   def     | OK     |       | 5.67  |       | 210  | \\     | いろは  \
//   ghi     |        | 404   |       | false |      | \|     | にほへ
//
// A row filled with '-' is a delimiter. It is ignored.
// First row is header. Following rows are body.
//
// Empty lines and lines filled with white spaces above header are ignored.
// Table ends with an empty line or a line filled with white spaces.
// Once table ends, following lines are ignored.
//
// Escape sequences can be used in values. Those are "\n" (unescaped into LF),
// "\\" (\), and "\|" (|).
//
// A row ends with "\" indicates it continues to the next row.
// In above example 5th row and 6th row are merged when unmarshaling.
// So the value of "string" column is "def ghi".
package table
