// Package testdata provides sample JSON fixtures used by tree and
// its subpackages for tests, benchmarks, and godoc examples. It is
// intentionally internal so the fixtures do not surface in the
// public API of the tree module.
package testdata

// StoreJSON is the "Illustrative Object" from tree's README: a
// bookstore document with nested arrays and maps. It is the default
// fixture for query, editing, and validation examples.
const StoreJSON = `{
  "store": {
    "book": [
      {
        "category": "reference",
        "author": "Nigel Rees",
        "authors": ["Nigel Rees"],
        "title": "Sayings of the Century",
        "price": 8.95,
        "tags": [
          { "name": "genre", "value": "reference" },
          { "name": "era", "value": "20th century" },
          { "name": "theme", "value": "quotations" }
        ]
      },
      {
        "category": "fiction",
        "author": "Evelyn Waugh",
        "title": "Sword of Honour",
        "price": 12.99,
        "tags": [
          { "name": "genre", "value": "fiction" },
          { "name": "era", "value": "20th century" },
          { "name": "theme", "value": "WWII" }
        ]
      },
      {
        "category": "fiction",
        "author": "Herman Melville",
        "title": "Moby Dick",
        "isbn": "0-553-21311-3",
        "price": 8.99,
        "tags": [
          { "name": "genre", "value": "fiction" },
          { "name": "era", "value": "19th century" },
          { "name": "theme", "value": "whale hunting" }
        ]
      },
      {
        "category": "fiction",
        "author": "J. R. R. Tolkien",
        "title": "The Lord of the Rings",
        "isbn": "0-395-19395-8",
        "price": 22.99,
        "tags": [
          { "name": "genre", "value": "fantasy" },
          { "name": "era", "value": "20th century" },
          { "name": "theme", "value": "good vs evil" }
        ]
      }
    ],
    "bicycle": {
      "color": "red",
      "price": 19.95
    }
  }
}
`
