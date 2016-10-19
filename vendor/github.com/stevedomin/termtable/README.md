termtable
==========

A Go library to easily generate table in you CLI

## Install

```bash
$ go get github.com/stevedomin/termtable
```

## Usage

Print a simple table without separators:

```go
package main

import(
    "fmt"
    "github.com/stevedomin/termtable"
)

func main() {
    t := termtable.NewTable(nil, nil)
    t.SetHeader([]string{"LOWERCASE", "UPPERCASE", "NUMBERS"})
    t.AddRow([]string{"abc", "ABCD", "12345"})
    t.AddRow([]string{"defg", "EFGHI", "678"})
    t.AddRow([]string{"hijkl", "JKL", "9000"})
    fmt.Println(t.Render())

    // Output :
    // LOWERCASE UPPERCASE NUMBERS
    // abc       ABCD      12345
    // defg      EFGHI     678
    // hijkl     JKL       9000
}
```

Print a simple table with separators and custom padding (alt syntax):

```go
package main

import(
    "github.com/stevedomin/termtable"
)

func main() {
    rows := [][]string{
      []string{"abc", "ABCD", "12345"},
      []string{"defg", "EFGHI", "678"},
      []string{"hijkl", "JKL", "9000"},
    }
    t := termtable.NewTable(rows, &termtable.TableOptions{
     Padding: 3,
     UseSeparator: true,
    })
    fmt.Println(t.Render())

    // Output (vertical bars look better in terminal):
    // +---------------+---------------+-------------+
    // |   LOWERCASE   |   UPPERCASE   |   NUMBERS   |
    // +---------------+---------------+-------------+
    // |   abc         |   ABCD        |   12345     |
    // |   defg        |   EFGHI       |   678       |
    // |   hijkl       |   JKL         |   9000      |
    // +---------------+---------------+-------------+
}
```

## Development

To run test:

```bash
$ go test
```

To run benchmarks:

```bash
$ go test -bench . -benchmem
```

## Todo

* Column align
* Cell align
* Support more types: int, float, bool
* Rows separator
* Multiline cell
* Custom column width
