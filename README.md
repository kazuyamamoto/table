# table

Package table provides functionality to unmarshal table string into slice of struct.


## Usage

When table and its struct are as follows:

```
const tableString = `
string value | int value
hello world  | 302
こんにちは   | -0x20
`

type row struct {
        S string `table:"string value"` 
        I int    `table:"int value"` 
}
```

Unmarshalling code is like as follows:

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
It is ignored in unmarshaling.

```
string value | int value
------------ | ---------
hello world  | 302
------------ | ---------
こんにちは   | -0x20
```


### Custom Unmarshaler

TBD.


### Escape Sequence

LF is represented by `\n` in table string.
`|` is by `\|`.
`\` is by `\\`. 

```
string value | int value
\\\n\|       | 302
```

String value above is `"\\\n|"` in Go string. 


### Multi-line Rows

TBD
