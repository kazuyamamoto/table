# table

Package table provides functionality to unmarshal table string into slice of struct.


## Usage

Provided that a table string and its struct are as follows

```
const tableString = `
string value | int value
hello world  | 302
こんにちは   | -0x20

ignored lines...
`

type row struct {
        S string `table:"string value"` 
        I int    `table:"int value"` 
}
```

Unmarshalling code is like as follows

```
var tbl []row
err := table.Unmarshal([]byte(tableString), &tbl)
if err != nil {
        panic(err)
}

fmt.Println(tbl[0].S) // hello world
fmt.Println(tbl[0].I) // 302
fmt.Println(tbl[1].S) // こんにちは
fmt.Println(tbl[1].I) // -32
````


### Delimiter

Delimiter is a row filled with `-` and white spaces.
It is ignored in unmarshalling.

```
string value | int value
------------ | ---------
hello world  | 302
------------ | ---------
こんにちは   | -0x20
```


### Custom Unmarshaler

When `table.Unmarshaler` implementation is a struct field,
it is unmarshalled.

```
type custom string // table.Unmarshaler implementation

func (c *custom) UnmarshalTable(p []byte) error {
	*c = custom(p)
	return nil
}

type row struct {
	C custom `table:"custom value"`
}

const tableString = `
custom value
hello world
`
```

```
var tbl []row
_ = table.Unmarshal([]byte(tableString), &tbl)
fmt.Println(tbl[0].C) // hello world
````

### Escape Sequence

Escape sequences are used to represent special characters in table string.

Escape sequence `\n` represents LF.
`\|` represents `|`.
`\\` represents `\`.

Unmarshalled value of string value below is `"\\\n|"` in Go string. 

```
string value
\\\n\|
```

### Multi-line Row

A row ends with `\` continues to the next row.
Lower row is merged into upper row.
Values in corresponding column are concatenated inserting ` ` between them.  

Unmarshalled value of string value below is `"hello world Go"` in Go string. 
Number of rows is 1.

```
string value
hello \
world     \
Go
```