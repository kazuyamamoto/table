// Package table provides functionality to unmarshal table string into slice of
// struct. Table format is like those of lightweight markup languages:
//
//   string  | custom | int   | float | bool  | uint | escape | 文字列
//   ------- | ------ | ----- | ----- | ----- | ---- | ------ | --------
//   abc     | OK     | 302   | 1.234 | true  | 7890 | abc\nd | あいうえお
//           | NG     | -0x20 | -5    | F     | 3333 | \\n\|  | 日本語
//
// A row filled with '-' is assumed as delimiter.
// Header is rows above the first delimiter and body is below that delimiter.
// Delimiters in body are ignored. Empty lines above header are ignored.
// Table ends with an empty line. Following lines are ignored.
// Escape sequences can be used in values. Those are "\n" (unescaped into LF),
// "\\" (\), and "\|" (|).
package table
