# Tree

[![Tests](https://github.com/mojatter/tree/actions/workflows/tests.yaml/badge.svg?branch=main)](https://github.com/mojatter/tree/actions/workflows/tests.yaml)
[![govulncheck](https://github.com/mojatter/tree/actions/workflows/govulncheck.yaml/badge.svg?branch=main)](https://github.com/mojatter/tree/actions/workflows/govulncheck.yaml)
[![PkgGoDev](https://pkg.go.dev/badge/github.com/mojatter/tree)](https://pkg.go.dev/github.com/mojatter/tree)
[![Report Card](https://goreportcard.com/badge/github.com/mojatter/tree)](https://goreportcard.com/report/github.com/mojatter/tree)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

Tree is a simple structure for dealing with dynamic or unknown JSON/YAML in Go.

## Table of Contents

- [Features](#features)
- [Road to 1.0](#road-to-10)
- [Syntax](#syntax)
- [Marshal and Unmarshal](#marshal-and-unmarshal)
- [Get](#get)
- [Find](#find)
  - [Query](#query)
  - [Built-in Methods](#built-in-methods)
- [Edit](#edit)
- [tq](#tq)
- [Contributing](#contributing)
- [Third-party library licenses](#third-party-library-licenses)

## Features

- Parses json/yaml of unknown structure to get to nodes with fluent interface.
- Syntax similar to Go standard and map and slice.
- Find function can be specified the [Query](#query) expression with [built-in methods](#built-in-methods).
- Edit function can be specified the [Edit](#edit) expression.
- Bundled 'tq' that is a portable command-line JSON/YAML processor.

## Road to 1.0

- Placeholders in query.
- Merge support in tq.

## Syntax

### Go

```go
tree.Map{
	"ID":     tree.V(1),
	"Name":   tree.V("Reds"),
	"Colors": tree.A("Crimson", "Red", "Ruby", "Maroon"),
}
```

### JSON

```json
{
	"ID": 1,
	"Name": "Reds",
	"Colors": ["Crimson", "Red", "Ruby", "Maroon"]
}
```

### YAML

```yaml
ID: 1
Name: Reds
Colors:
- Crimson
- Red
- Ruby
- Maroon
```

## Marshal and Unmarshal

```go
func ExampleMarshalJSON() {
	group := tree.Map{
		"ID":     tree.V(1),
		"Name":   tree.V("Reds"),
		"Colors": tree.A("Crimson", "Red", "Ruby", "Maroon"),
	}
	b, err := json.Marshal(group)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(b))

	// Output:
	// {"Colors":["Crimson","Red","Ruby","Maroon"],"ID":1,"Name":"Reds"}
}
```

```go
func ExampleUnmarshalJSON() {
	data := []byte(`[
  {"Name": "Platypus", "Order": "Monotremata"},
  {"Name": "Quoll",    "Order": "Dasyuromorphia"}
]`)

	var animals tree.Array
	err := json.Unmarshal(data, &animals)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%+v\n", animals)

	// Output:
	// [map[Name:Platypus Order:Monotremata] map[Name:Quoll Order:Dasyuromorphia]]
}
```

### Using other parsers

Tree may work on other parsers that are compatible with "encoding/json" or "gopkg.in/yaml.v2". See [examples](examples) directory.

### Alternate json.RawMessage

For example, [Dynamic JSON in Go](https://eagain.net/articles/go-dynamic-json/) shows an example of using json.RawMessage.

It may be simpler to use tree.Map instead of json.RawMessage.

```go
package main

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/mojatter/tree"
)

const input = `
{
	"type": "sound",
	"msg": {
		"description": "dynamite",
		"authority": "the Bruce Dickinson"
	}
}
`

type Envelope struct {
	Type string
	Msg  tree.Map
}

func main() {
	env := Envelope{}
	if err := json.Unmarshal([]byte(input), &env); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%#v\n", env)
	fmt.Printf("%#v\n", env.Msg.Get("description"))

	// Output:
	// main.Envelope{Type:"sound", Msg:tree.Map{"authority":"the Bruce Dickinson", "description":"dynamite"}}
	// "dynamite"
}
```

## Get

```go
func ExampleGet() {
	group := tree.Map{
		"ID":     tree.V(1),
		"Name":   tree.V("Reds"),
		"Colors": tree.A("Crimson", "Red", "Ruby", "Maroon"),
		"Nil":    nil,
	}
	fmt.Println(group.Get("Colors").Get(1))
	fmt.Println(group.Get("Colors", 2))
	fmt.Println(group.Get("Colors").Get(5).IsNil())
	fmt.Println(group.Get("Nil").IsNil())

	// Output:
	// Red
	// Ruby
	// true
	// true
}
```

## Find

```go
func ExampleFind() {
	group := tree.Map{
		"ID":     tree.V(1),
		"Name":   tree.V("Reds"),
		"Colors": tree.A("Crimson", "Red", "Ruby", "Maroon"),
	}

	rs, err := group.Find(".Colors[1:3]")
	if err != nil {
		log.Fatal(err)
	}
	for _, r := range rs {
		fmt.Println(r)
	}

	// Output:
	// Red
	// Ruby
}
```

### Query

For more details on built-in methods, see the [Built-in Methods](#built-in-methods) section.

| Query | Description | Results |
| - | - | - |
| .store.book[0] | The first book | {"category": "reference", "author": "Nigel Rees", "title": "Sayings of the Century", "price": 8.95, "tags": [...]} |
| .store.book[0].price | The price of the first book | 8.95 |
| .store.book.0.price | The price of the first book (using dot) | 8.95 |
| .store.book[:2].price | All prices of books[0:2] (index 2 is exclusive) | 8.95, 12.99 |
| .store.book[].author | All authors of all books | "Nigel Rees", "Evelyn Waugh", "Herman Melville", "J. R. R. Tolkien" |
| ..author | All authors |  "Nigel Rees", "Evelyn Waugh", "Herman Melville", "J. R. R. Tolkien" |
| ..author \| [0] | The first author | "Nigel Rees" |
| .store.book[.tags[.name == "genre" and .value == "fiction"]].title | All titles of books tagged "fiction" | "Sword of Honour", "Moby Dick" |
| .store.book[(.category == "fiction" or .category == "reference") and .price < 10].title | All titles of books that are categorized as "fiction" or "reference" and price < 10 | "Sayings of the Century", "Moby Dick" |
| .store.book[.title ~= "^S"].title | Titles beginning with "S" | "Sayings of the Century", "Sword of Honour" |
| .store.book.count() | Count books | 4 |
| .store.book[0].keys() | Sorted keys of the first book | ["author", "category", "price", "tags", "title"] |
| .store.book[0].values() | Values of the first book | ["Nigel Rees", "reference", 8.95, [tag objects], "Sayings of the Century"] |
| .store.book.last().has("isbn") | Check if last book has ISBN | true |
| .store.book[.author.contains("Tolkien")] | Check if any author contains "Tolkien" | [{"author": "J. R. R. Tolkien", ...}] |
| .store.book[0].category.type() | Get type of category field | "string" |
| .store.book[0].empty() | Check if first book is empty | false |
| .store.book.first().title | Get title of first book | "Sayings of the Century" |
| .store.book.last().author | Get author of last book | "J. R. R. Tolkien" |
| .store.book.sort(".price").first().title | Cheapest book | "Sayings of the Century" |
| .store.book.rsort(".price").first().title | Most expensive book | "The Lord of the Rings" |

#### Illustrative Object

```json
{
  "store": {
    "bicycle": {
      "color": "red",
      "price": 19.95
    },
    "book": [
      {
        "author": "Nigel Rees",
        "category": "reference",
        "price": 8.95,
        "title": "Sayings of the Century",
        "tags": [
          { "name": "genre", "value": "reference" },
          { "name": "era", "value": "20th century" },
          { "name": "theme", "value": "quotations" }
        ]
      },
      {
        "author": "Evelyn Waugh",
        "category": "fiction",
        "price": 12.99,
        "title": "Sword of Honour",
        "tags": [
          { "name": "genre", "value": "fiction" },
          { "name": "era", "value": "20th century" },
          { "name": "theme", "value": "WWII" }
        ]
      },
      {
        "author": "Herman Melville",
        "category": "fiction",
        "isbn": "0-553-21311-3",
        "price": 8.99,
        "title": "Moby Dick",
        "tags": [
          { "name": "genre", "value": "fiction" },
          { "name": "era", "value": "19th century" },
          { "name": "theme", "value": "whale hunting" }
        ]
      },
      {
        "author": "J. R. R. Tolkien",
        "category": "fiction",
        "isbn": "0-395-19395-8",
        "price": 22.99,
        "title": "The Lord of the Rings",
        "tags": [
          { "name": "genre", "value": "fantasy" },
          { "name": "era", "value": "20th century" },
          { "name": "theme", "value": "good vs evil" }
        ]
      }
    ]
  }
}
```

### Built-in Methods

Tree provides several built-in methods for data manipulation and querying:

#### Aggregate Methods
- **`count()`** - Returns the count of elements in arrays or maps
- **`keys()`** - Returns the keys of arrays (as indices) or maps
- **`values()`** - Returns the values of arrays or maps as an array

#### Condition Methods
- **`empty()`** - Checks if the node is empty (empty arrays, maps, null values, or empty strings)
- **`has(key)`** - Checks if the node has the specified key (works with arrays and maps)
- **`contains(value)`** - Checks if the node contains the specified value (arrays, maps, or substring in strings)

#### Data Type Methods
- **`type()`** - Returns the type name of the node ("array", "object", "string", "number", "boolean", "null")

#### Array Methods
- **`first()`** - Returns the first element of an array
- **`last()`** - Returns the last element of an array
- **`sort([expr])`** - Returns the array sorted in ascending order. With an optional query expression, sorts by the value that expression resolves to on each element.
- **`rsort([expr])`** - Same as `sort()` but in descending order.

#### Examples

```go
// Count elements
node.Find(".store.book.count()")  // Returns: 4

// Check if key exists
node.Find(".store.book[0].has(\"title\")")  // Returns: true

// Check if contains value in string
node.Find(".store.book[0].author.contains(\"Nigel\")")  // Returns: true

// Get type
node.Find(".store.book[0].price.type()")  // Returns: "number"

// Array operations
node.Find(".store.book.first().title")  // Returns: "Sayings of the Century"
node.Find(".store.book.last().title")   // Returns: "The Lord of the Rings"

// Sort by a sub-expression
node.Find(".store.book.sort(\".price\")")   // Books ordered by ascending price
node.Find(".store.book.rsort(\".price\")")  // Books ordered by descending price
```


## Edit

```go
func ExampleEdit() {
	var group tree.Node = tree.Map{
		"ID":     tree.V(1),
		"Name":   tree.V("Reds"),
		"Colors": tree.A("Crimson", "Red", "Ruby", "Maroon"),
	}

	if err := tree.Edit(&group, ".Colors += \"Pink\""); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Append Pink to Colors:\n  %+v\n", group)

	if err := tree.Edit(&group, ".Name = \"Blue\""); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Set Blue to Name:\n  %+v\n", group)

	if err := tree.Edit(&group, ".Colors ^?"); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Delete Colors:\n  %+v\n", group)

	// Output:
	// Append Pink to Colors:
	//   map[Colors:[Crimson Red Ruby Maroon Pink] ID:1 Name:Reds]
	// Set Blue to Name:
	//   map[Colors:[Crimson Red Ruby Maroon Pink] ID:1 Name:Blue]
	// Delete Colors:
	//   map[ID:1 Name:Blue]
}
```

## tq

tq is a portable command-line JSON/YAML processor.

### Installation

> Previously published as `jarxorg/tree`. If you installed `tq` from
> the old tap or have the old Go module in your `go.mod`, read
> [docs/MIGRATION.md](docs/MIGRATION.md) before installing the new
> version — the old binary can shadow the new one on `$PATH`.

```sh
go install github.com/mojatter/tree/cmd/tq@latest
```

Using Homebrew

```sh
brew tap mojatter/tree
brew install mojatter/tree/tq
```

Download binary

```sh
# For macOS (Darwin)
VERSION=0.10.0 GOOS=Darwin GOARCH=arm64; curl -fsSL "https://github.com/mojatter/tree/releases/download/v${VERSION}/tree_${VERSION}_${GOOS}_${GOARCH}.tar.gz" | tar xz tq && mv tq /usr/local/bin

# For Linux x64
VERSION=0.10.0 GOOS=Linux GOARCH=amd64; curl -fsSL "https://github.com/mojatter/tree/releases/download/v${VERSION}/tree_${VERSION}_${GOOS}_${GOARCH}.tar.gz" | tar xz tq && mv tq /usr/local/bin

# For Windows x64
VERSION=0.10.0; curl -fsSL "https://github.com/mojatter/tree/releases/download/v${VERSION}/tree_${VERSION}_windows_amd64.zip" -o tq.zip && unzip tq.zip tq.exe
```

### Usage

```sh
tq is a command-line JSON/YAML processor.

Usage:
  tq [flags] [query] ([file...])

Flags:
  -c, --color                  output with colors
  -e, --edit stringArray       edit expression
  -x, --expand                 expand results
  -h, --help                   help for tq
  -U, --inplace                update files, inplace
  -i, --input-format string    input format (json or yaml)
  -j, --input-json             alias --input-format json
  -y, --input-yaml             alias --input-format yaml
  -O, --output string          output file
  -o, --output-format string   output format (json or yaml, default json)
  -J, --output-json            alias --output-format json
  -Y, --output-yaml            alias --output-format yaml
  -r, --raw                    output raw strings
  -s, --slurp                  slurp all results into an array
  -t, --template string        golang text/template string
  -v, --version                print version

Examples:
  % echo '{"colors": ["red", "green", "blue"]}' | tq '.colors[0]'
  "red"

  % echo '{"users":[{"id":1,"name":"one"},{"id":2,"name":"two"}]}' | tq -x -t '{{.id}}: {{.name}}' '.users'
  1: one
  2: two

  % echo '{}' | tq -e '.colors = ["red", "green"]' -e '.colors += "blue"' .
  {
    "colors": [
      "red",
      "green",
      "blue"
    ]
  }

  # Using built-in methods
  % echo '{"books": [{"author": "Tolkien"}, {"author": "Hemingway"}]}' | tq '.books[.author.contains("Tol")]'
  [{"author": "Tolkien"}]

```

### for jq user

| tq | jq |
| - | - |
| tq '.store.book[0]' | jq '.store.book[0]' |
| tq '.store.book[]' | jq '.store.book[]' |
| tq '.store.book[:2].price' | jq '.store.book[:2][] \| .price' |
| tq '.store.book[.category == "fiction" and .price < 10].title' | jq '.store.book[] \| select(.category == "fiction" and .price < 10) \| .title' |


## Contributing

Bug reports, feature requests, and pull requests are welcome. See
[CONTRIBUTING.md](CONTRIBUTING.md) for the local development workflow,
the `make` targets that wrap the test/lint/fuzz/bench commands, and
the project's code conventions.

## Third-party library licenses

- [spf13/pflag](https://github.com/spf13/pflag/blob/master/LICENSE)
