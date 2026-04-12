package tree

import (
	"reflect"
	"testing"
)

func Test_Query(t *testing.T) {
	tests := []struct {
		q      Query
		n      Node
		want   []Node
		errstr string
	}{
		{
			q:    NopQuery{},
			n:    Array{},
			want: []Node{Array{}},
		}, {
			q:    MapQuery("key"),
			n:    Map{"key": ToValue("value")},
			want: []Node{ToValue("value")},
		}, {
			q:      MapQuery("key"),
			n:      ToValue("not map"),
			errstr: `cannot index array with "key"`,
		}, {
			q:    ArrayQuery(0),
			n:    Array{ToValue(1)},
			want: []Node{ToValue(1)},
		}, {
			q:      ArrayQuery(0),
			n:      ToValue("not array"),
			errstr: `cannot index array with 0`,
		}, {
			q:    ArrayRangeQuery{0, 2},
			n:    Array{ToValue(0), ToValue(1), ToValue(2)},
			want: []Node{ToValue(0), ToValue(1)},
		}, {
			q:    ArrayRangeQuery{1, -1},
			n:    Array{ToValue(0), ToValue(1), ToValue(2)},
			want: []Node{ToValue(1), ToValue(2)},
		}, {
			q:      ArrayRangeQuery{0, 1, 2},
			n:      Array{},
			errstr: `invalid array range [0:1:2]`,
		}, {
			q:      ArrayRangeQuery{0, 1},
			n:      Map{},
			errstr: `cannot index array with range 0:1`,
		}, {
			q:    FilterQuery{MapQuery("key"), ArrayQuery(0)},
			n:    Map{"key": Array{ToValue(1)}},
			want: []Node{ToValue(1)},
		}, {
			q:      FilterQuery{MapQuery("key"), ArrayQuery(0)},
			n:      Map{"key": ToValue(1)},
			errstr: `cannot index array with 0`,
		}, {
			q: SelectQuery{And{
				Comparator{MapQuery("key"), EQ, ValueQuery{ToValue(1)}},
			}},
			n:    Array{Map{"key": ToValue(1)}, Map{"key": ToValue(2)}},
			want: []Node{Map{"key": ToValue(1)}},
		}, {
			q:    SelectQuery{},
			n:    Array{Map{"key": ToValue(1)}, Map{"key": ToValue(2)}},
			want: []Node{Map{"key": ToValue(1)}, Map{"key": ToValue(2)}},
		}, {
			q: SelectQuery{And{
				Comparator{MapQuery("key"), EQ, ValueQuery{ToValue(1)}},
			}},
			n: Map{},
		}, {
			q: SelectQuery{
				And{
					Or{
						Comparator{MapQuery("key2"), EQ, ValueQuery{ToValue("a")}},
						Comparator{MapQuery("key2"), EQ, ValueQuery{ToValue("b")}},
					},
					Comparator{MapQuery("key1"), LE, ValueQuery{ToValue(1)}},
				},
			},
			n: Array{
				Map{"key1": ToValue(1), "key2": ToValue("a")},
				Map{"key1": ToValue(2), "key2": ToValue("b")},
				Map{"key1": ToValue(3), "key2": ToValue("c")},
			},
			want: []Node{
				Map{"key1": ToValue(1), "key2": ToValue("a")},
			},
		}, {
			q: WalkQuery("key1"),
			n: Array{
				Map{"key1": ToValue(1), "key2": ToValue("a")},
				Map{"key1": ToValue(2), "key2": ToValue("b")},
				Map{"key1": ToValue(3), "key2": ToValue("c")},
			},
			want: []Node{ToValue(1), ToValue(2), ToValue(3)},
		},
	}
	for i, test := range tests {
		got, err := test.q.Exec(test.n)
		if test.errstr != "" {
			if err == nil {
				t.Fatalf("tests[%d] for %v; no error", i, test.q)
			}
			if err.Error() != test.errstr {
				t.Errorf("tests[%d] for %v; got %s; want %s", i, test.q, err.Error(), test.errstr)
			}
			continue
		}
		if err != nil {
			t.Fatalf("tests[%d] %v", i, err)
		}
		if !reflect.DeepEqual(got, test.want) {
			t.Errorf("tests[%d] for %v; got %v; want %v", i, test.q, got, test.want)
		}
	}
}

func Test_Query_String(t *testing.T) {
	tests := []struct {
		q    Query
		want string
	}{
		{
			q:    NopQuery{},
			want: ".",
		}, {
			q:    MapQuery("key"),
			want: ".key",
		}, {
			q:    ArrayQuery(1),
			want: "[1]",
		}, {
			q:    ArrayRangeQuery{0, 2},
			want: "[0:2]",
		}, {
			q:    ArrayRangeQuery{-1, 2},
			want: "[:2]",
		}, {
			q:    SlurpQuery{},
			want: " | ",
		}, {
			q:    FilterQuery{MapQuery("key1"), ArrayQuery(0), MapQuery("key2")},
			want: ".key1[0].key2",
		}, {
			q: SelectQuery{
				And{
					Or{
						Comparator{MapQuery("key2"), EQ, ValueQuery{ToValue("a")}},
						Comparator{MapQuery("key2"), EQ, ValueQuery{ToValue("b")}},
					},
					Comparator{MapQuery("key1"), LE, ValueQuery{ToValue(1)}},
				},
			},
			want: `[((.key2 == "a" or .key2 == "b") and .key1 <= 1)]`,
		}, {
			q:    WalkQuery("key"),
			want: "..key",
		}, {
			q:    FilterQuery{MapQuery("key1"), WalkQuery("key2")},
			want: ".key1..key2",
		},
	}
	for i, test := range tests {
		got := test.q.String()
		if got != test.want {
			t.Errorf("tests[%d] got %v; want %v", i, got, test.want)
		}
	}
}

func Test_ParseQuery(t *testing.T) {
	tests := []struct {
		expr string
		want Query
	}{
		{
			expr: `.`,
			want: NopQuery{},
		}, {
			expr: `[]`,
			want: SelectQuery{},
		}, {
			expr: `.store.book[0]`,
			want: FilterQuery{
				MapQuery("store"),
				MapQuery("book"),
				ArrayQuery(0),
			},
		}, {
			expr: `..book[0]`,
			want: FilterQuery{
				WalkQuery("book"),
				ArrayQuery(0),
			},
		}, {
			expr: `..0..0`,
			want: FilterQuery{
				WalkQuery("0"),
				WalkQuery("0"),
			},
		}, {
			expr: `."store"."book"[0]`,
			want: FilterQuery{
				MapQuery("store"),
				MapQuery("book"),
				ArrayQuery(0),
			},
		}, {
			expr: `.'store'.'book'[0]`,
			want: FilterQuery{
				MapQuery("store"),
				MapQuery("book"),
				ArrayQuery(0),
			},
		}, {
			expr: `.store.book[0:1]`,
			want: FilterQuery{
				MapQuery("store"),
				MapQuery("book"),
				ArrayRangeQuery{0, 1},
			},
		}, {
			expr: `.store.book[.category=="fiction" and .price < 10].title`,
			want: FilterQuery{
				MapQuery("store"),
				MapQuery("book"),
				SelectQuery{
					And{
						Comparator{MapQuery("category"), EQ, ValueQuery{StringValue("fiction")}},
						Comparator{MapQuery("price"), LT, ValueQuery{NumberValue(10)}},
					},
				},
				MapQuery("title"),
			},
		}, {
			expr: `.store.book[.authors[0] == "Nigel Rees"]`,
			want: FilterQuery{
				MapQuery("store"),
				MapQuery("book"),
				SelectQuery{
					And{
						Comparator{FilterQuery{MapQuery("authors"), ArrayQuery(0)}, EQ, ValueQuery{ToValue("Nigel Rees")}},
					},
				},
			},
		}, {
			expr: `.store.book[].author|[0]`,
			want: FilterQuery{
				MapQuery("store"),
				MapQuery("book"),
				SelectQuery{},
				MapQuery("author"),
				SlurpQuery{},
				ArrayQuery(0),
			},
		},
	}

	for i, test := range tests {
		got, err := ParseQuery(test.expr)
		if err != nil {
			t.Fatalf("tests[%d] %v", i, err)
		}
		if !reflect.DeepEqual(got, test.want) {
			t.Errorf("tests[%d] for %v; got %#v; want %#v", i, test.expr, got, test.want)
		}
	}
}

func Test_ParseQuery_Errors(t *testing.T) {
	tests := []struct {
		expr   string
		errstr string
	}{
		{
			expr:   `<`,
			errstr: `syntax error: invalid token <: "<"`,
		}, {
			expr:   `[`,
			errstr: `syntax error: no right brackets: "["`,
		}, {
			expr:   `]`,
			errstr: `syntax error: no left bracket: "]"`,
		}, {
			expr:   `[a]`,
			errstr: `syntax error: invalid array index: "[a]"`,
		}, {
			expr:   `[a:b]`,
			errstr: `syntax error: invalid array range: "[a:b]"`,
		}, {
			expr:   `[0:a]`,
			errstr: `syntax error: invalid array range: "[0:a]"`,
		}, {
			expr:   `[[l] == .r]`,
			errstr: `syntax error: invalid array index: "[[l] == .r]"`,
		}, {
			expr:   `[.l == [r]]`,
			errstr: `syntax error: invalid array index: "[.l == [r]]"`,
		}, {
			expr:   `.a[a]`,
			errstr: `syntax error: invalid array index: ".a[a]"`,
		},
	}
	for i, test := range tests {
		_, err := ParseQuery(test.expr)
		if err == nil {
			t.Fatalf("tests[%d] no error", i)
		}
		if err.Error() != test.errstr {
			t.Errorf("tests[%d] got %s; want %s", i, err.Error(), test.errstr)
		}
	}
}

// NOTE: Copy from https://github.com/stedolan/jq/wiki/For-JSONPath-users#illustrative-object
var testStoreJSON = `{
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

func TestFind(t *testing.T) {
	n, err := UnmarshalJSON([]byte(testStoreJSON))
	if err != nil {
		t.Fatal(err)
	}
	testCases := []struct {
		expr string
		want []Node
	}{
		{
			expr: `.store`,
			want: []Node{n.Get("store")},
		}, {
			expr: `.store[]`,
			want: []Node{
				n.Get("store", "bicycle"),
				n.Get("store", "book"),
			},
		}, {
			expr: `.store.book[0]`,
			want: []Node{n.Get("store", "book", 0)},
		}, {
			expr: `.store.book[]`,
			want: []Node{
				n.Get("store", "book", 0),
				n.Get("store", "book", 1),
				n.Get("store", "book", 2),
				n.Get("store", "book", 3),
			},
		}, {
			expr: `..book[0]`,
			want: []Node{n.Get("store", "book", 0)},
		}, {
			expr: `..book[0:2].title`,
			want: []Node{StringValue("Sayings of the Century"), StringValue("Sword of Honour")},
		}, {
			expr: `..book[0:2] | [0].title`,
			want: []Node{StringValue("Sayings of the Century")},
		}, {
			expr: `.store.book.0`,
			want: []Node{n.Get("store", "book", 0)},
		}, {
			expr: `.store.book[0].price`,
			want: []Node{n.Get("store", "book", 0, "price")},
		}, {
			expr: `.store.book[0:2]`,
			want: []Node{
				n.Get("store", "book", 0),
				n.Get("store", "book", 1),
			},
		}, {
			expr: `.store.book[1:].price`,
			want: ToNodeValues(12.99, 8.99, 22.99),
		}, {
			expr: `.store.book[:1].price`,
			want: ToNodeValues(8.95),
		}, {
			expr: `.store.book[].author`,
			want: ToNodeValues("Nigel Rees", "Evelyn Waugh", "Herman Melville", "J. R. R. Tolkien"),
		}, {
			expr: `.store.book[.tags[.name == "genre" and .value == "fiction"].count() > 0].title`,
			want: ToNodeValues("Sword of Honour", "Moby Dick"),
		}, {
			expr: `.store.book[.tags[.name == "genre" and .value == "fiction"]].title`,
			want: ToNodeValues("Sword of Honour", "Moby Dick"),
		}, {
			expr: `.store.book[.category == "fiction"].title`,
			want: ToNodeValues("Sword of Honour", "Moby Dick", "The Lord of the Rings"),
		}, {
			expr: `.store.book[.category == "fiction" and .price < 10].title`,
			want: ToNodeValues("Moby Dick"),
		}, {
			expr: `.store.book[.authors[0] == "Nigel Rees"].title`,
			want: ToNodeValues("Sayings of the Century"),
		}, {
			expr: `.store.book[(.category == "fiction" or .category == "reference") and .price < 10].title`,
			want: ToNodeValues("Sayings of the Century", "Moby Dick"),
		}, {
			expr: `.store.book[(.category != "reference") and .price >= 10].title`,
			want: ToNodeValues("Sword of Honour", "The Lord of the Rings"),
		}, {
			expr: `.store.book[].author|[0]`,
			want: ToNodeValues("Nigel Rees"),
		}, {
			expr: `.store..book[.category=="fiction"].title`,
			want: ToNodeValues("Sword of Honour", "Moby Dick", "The Lord of the Rings"),
		}, {
			expr: `..book[.category=="fiction"].title`,
			want: ToNodeValues("Sword of Honour", "Moby Dick", "The Lord of the Rings"),
		}, {
			expr: `..0`,
			want: []Node{
				n.Get("store", "book", 0),
				n.Get("store", "book", 0, "authors", 0),
				n.Get("store", "book", 0, "tags", 0),
				n.Get("store", "book", 1, "tags", 0),
				n.Get("store", "book", 2, "tags", 0),
				n.Get("store", "book", 3, "tags", 0),
			},
		}, {
			expr: `.store.book[.title ~= "^S"].title`,
			want: ToNodeValues("Sayings of the Century", "Sword of Honour"),
		}, {
			expr: `.store.book[.author ~= "^(Evelyn Waugh|Herman Melville)$"].title`,
			want: ToNodeValues("Sword of Honour", "Moby Dick"),
		}, {
			expr: `.store.book.count()`,
			want: []Node{NumberValue(4)},
		}, {
			expr: `.store.book[].count()`,
			want: []Node{NumberValue(6), NumberValue(5), NumberValue(6), NumberValue(6)},
		}, {
			expr: `.store.book[0].keys()`,
			want: []Node{ToArrayValues("author", "authors", "category", "price", "tags", "title")},
		}, {
			expr: `.store.book[].keys()`,
			want: []Node{
				ToArrayValues("author", "authors", "category", "price", "tags", "title"),
				ToArrayValues("author", "category", "price", "tags", "title"),
				ToArrayValues("author", "category", "isbn", "price", "tags", "title"),
				ToArrayValues("author", "category", "isbn", "price", "tags", "title"),
			},
		}, {
			expr: `.store.book[0].values()`,
			want: []Node{
				ToArrayValues(
					"Nigel Rees",
					ToArrayValues("Nigel Rees"),
					"reference",
					8.95,
					n.Get("store", "book", 0, "tags"),
					"Sayings of the Century",
				),
			},
		},
	}
	for _, tc := range testCases {
		t.Run("", func(t *testing.T) {
			got, err := Find(n, tc.expr)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("got %#v; want %#v", got, tc.want)
			}
		})
	}
}

func Test_holdArray(t *testing.T) {
	var got Node = Array{
		StringValue("0"),
		Array{StringValue("0-0"), StringValue("0-1")},
		Map{"1": Array{BoolValue(true)}},
	}
	want := &arrayHolder{
		&Array{
			StringValue("0"),
			&arrayHolder{a: &Array{StringValue("0-0"), StringValue("0-1")}},
			Map{"1": &arrayHolder{a: &Array{BoolValue(true)}}},
		},
	}
	holdArray(&got)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %#v; want %#v", got, want)
	}
}

func Test_unholdArray(t *testing.T) {
	var want Node = Array{
		StringValue("0"),
		Array{StringValue("0-0"), StringValue("0-1")},
		Map{"1": Array{BoolValue(true)}},
	}
	var got Node = &arrayHolder{
		&Array{
			StringValue("0"),
			&arrayHolder{a: &Array{StringValue("0-0"), StringValue("0-1")}},
			Map{"1": &arrayHolder{a: &Array{BoolValue(true)}}},
		},
	}
	unholdArray(&got)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %#v; want %#v", got, want)
	}
}

func Test_Edit(t *testing.T) {
	testCases := []struct {
		caseName string
		n        Node
		expr     string
		want     Node
		errstr   string
	}{
		// --- Set operations on Map ---
		{
			caseName: "set empty map on map key",
			n:        Map{},
			expr:     `.store = {}`,
			want:     Map{"store": Map{}},
		}, {
			caseName: "set empty map on map key without spaces",
			n:        Map{},
			expr:     `.store={}`, // NOTE: trim spaces
			want:     Map{"store": Map{}},
		}, {
			caseName: "set nested map key",
			n:        Map{},
			expr:     `.store.book = {}`,
			want:     Map{"store": Map{"book": Map{}}},
		}, {
			caseName: "set array of maps with double quotes",
			n:        Map{},
			expr:     `.store.pen = [{"color":"red"},{"color":"blue"}]`,
			want: Map{
				"store": Map{
					"pen": Array{
						Map{"color": StringValue("red")},
						Map{"color": StringValue("blue")},
					},
				},
			},
		}, {
			caseName: "set array of maps with single quotes",
			n:        Map{},
			expr:     `.store.pen = [{'color':'red'},{'color':'blue'}]`,
			want: Map{
				"store": Map{
					"pen": Array{
						Map{"color": StringValue("red")},
						Map{"color": StringValue("blue")},
					},
				},
			},
		}, {
			caseName: "set on non-map returns error",
			n:        StringValue("str"),
			expr:     `.key = {}`,
			errstr:   `cannot index array with "key"`,
		},

		// --- Append error cases ---
		{
			caseName: "append to map root returns error",
			n:        Map{"key": StringValue("str")},
			expr:     `. += {}`,
			errstr:   "cannot append to .",
		}, {
			caseName: "append to string root returns error",
			n:        StringValue("str"),
			expr:     `. += {}`,
			errstr:   "cannot append to .",
		}, {
			caseName: "append to map string value returns error",
			n:        Map{"key": StringValue("str")},
			expr:     `.key += {}`,
			errstr:   `cannot append to "key"`,
		}, {
			caseName: "append to string value returns error",
			n:        StringValue("str"),
			expr:     `.key += {}`,
			errstr:   `cannot append to "key"`,
		},

		// --- Set operations on Array ---
		{
			caseName: "set on array by index",
			n:        Array{},
			expr:     `[0] = "red"`,
			want:     Array{StringValue("red")},
		}, {
			caseName: "set on nested array by index",
			n:        Array{},
			expr:     `[0][1] = "red"`,
			want:     Array{Array{nil, StringValue("red")}},
		}, {
			caseName: "set on non-array by index returns error",
			n:        StringValue("str"),
			expr:     `[0] = "red"`,
			errstr:   `cannot index array with 0`,
		}, {
			caseName: "set on array by dot-index",
			n:        Array{},
			expr:     `.0 = "red"`,
			want:     Array{StringValue("red")},
		}, {
			caseName: "set on array by dot-index creates map",
			n:        Array{},
			expr:     `.0.1 = "red"`,
			want:     Array{Map{"1": StringValue("red")}},
		}, {
			caseName: "set root to value",
			n:        Array{},
			expr:     `. = "red"`,
			want:     StringValue("red"),
		},

		// --- Append operations ---
		{
			caseName: "append to new map key creates array",
			n:        Map{},
			expr:     `.colors += "red"`,
			want:     Map{"colors": Array{StringValue("red")}},
		}, {
			caseName: "append to new map key without spaces",
			n:        Map{},
			expr:     `.colors+="red"`, // NOTE: trim spaces
			want:     Map{"colors": Array{StringValue("red")}},
		}, {
			caseName: "append to existing array in map",
			n:        Map{"colors": Array{StringValue("red"), StringValue("green")}},
			expr:     `.colors += "blue"`,
			want:     Map{"colors": Array{StringValue("red"), StringValue("green"), StringValue("blue")}},
		}, {
			caseName: "append to nested array by index",
			n:        Array{Array{StringValue("red")}},
			expr:     `[0] += "blue"`,
			want:     Array{Array{StringValue("red"), StringValue("blue")}},
		}, {
			caseName: "append to nested array by index with gap",
			n:        Array{Array{StringValue("red")}},
			expr:     `[2] += "blue"`,
			want:     Array{Array{StringValue("red")}, nil, Array{StringValue("blue")}},
		}, {
			caseName: "append to non-array element returns error",
			n:        Array{StringValue("red")},
			expr:     `[0] += "blue"`,
			errstr:   `cannot append to array with 0`,
		}, {
			caseName: "append to non-array root returns error",
			n:        StringValue("red"),
			expr:     `[0] += "blue"`,
			errstr:   `cannot append to array with 0`,
		}, {
			caseName: "append to array root",
			n:        Array{},
			expr:     `. += "red"`,
			want:     Array{StringValue("red")},
		},

		// --- Delete operations ---
		{
			caseName: "delete map key",
			n:        Map{"key1": StringValue("value1"), "key2": StringValue("value2")},
			expr:     `.key1 ^?`,
			want:     Map{"key2": StringValue("value2")},
		}, {
			caseName: "delete map key without spaces",
			n:        Map{"key1": StringValue("value1"), "key2": StringValue("value2")},
			expr:     `.key1^?`, // NOTE: trim spaces
			want:     Map{"key2": StringValue("value2")},
		}, {
			caseName: "delete array element by bracket-index",
			n:        Array{StringValue("red")},
			expr:     `[0] ^?`,
			want:     Array{},
		}, {
			caseName: "delete array element by dot-index",
			n:        Array{StringValue("red")},
			expr:     `.0 ^?`,
			want:     Array{},
		}, {
			caseName: "delete root returns error",
			n:        Map{},
			expr:     `. ^?`,
			errstr:   "cannot delete .",
		}, {
			caseName: "delete key on non-map returns error",
			n:        StringValue("str"),
			expr:     `.key ^?`,
			errstr:   `cannot delete "key"`,
		}, {
			caseName: "delete index on non-array returns error",
			n:        StringValue("str"),
			expr:     `[0] ^?`,
			errstr:   `cannot delete array with 0`,
		},

		// --- Recursive walk operations ---
		{
			caseName: "walk set on nested name fields",
			n: Map{
				"users": Array{
					Map{"name": StringValue("one"), "class": StringValue("A")},
					Map{"name": StringValue("two"), "job": Map{"name": StringValue("engineer")}},
				},
			},
			expr: `..name = "NAME"`,
			want: Map{
				"users": Array{
					Map{"name": StringValue("NAME"), "class": StringValue("A")},
					Map{"name": StringValue("NAME"), "job": Map{"name": StringValue("NAME")}},
				},
			},
		}, {
			caseName: "walk append to array",
			n: Map{
				"numbers": Array{
					NumberValue(1),
					NumberValue(2),
				},
			},
			expr: `..numbers += 3`,
			want: Map{
				"numbers": Array{
					NumberValue(1),
					NumberValue(2),
					NumberValue(3),
				},
			},
		}, {
			caseName: "walk delete key",
			n: Map{
				"users": Array{
					Map{"name": StringValue("one"), "class": StringValue("A")},
					Map{"name": StringValue("two"), "class": StringValue("B")},
				},
			},
			expr: `..class ^?`,
			want: Map{
				"users": Array{
					Map{"name": StringValue("one")},
					Map{"name": StringValue("two")},
				},
			},
		},

		// --- Filtered edit operations ---
		{
			caseName: "filtered set on all array elements",
			n: Map{
				"users": Array{
					Map{"name": StringValue("one"), "class": StringValue("A")},
					Map{"name": StringValue("two"), "class": StringValue("A")},
				},
			},
			expr: `..users[].class = "B"`,
			want: Map{
				"users": Array{
					Map{"name": StringValue("one"), "class": StringValue("B")},
					Map{"name": StringValue("two"), "class": StringValue("B")},
				},
			},
		}, {
			caseName: "filtered set with condition",
			n: Map{
				"users": Array{
					Map{"name": StringValue("one"), "class": StringValue("A")},
					Map{"name": StringValue("two"), "class": StringValue("A")},
				},
			},
			expr: `..users[.name == "two"].class = "B"`,
			want: Map{
				"users": Array{
					Map{"name": StringValue("one"), "class": StringValue("A")},
					Map{"name": StringValue("two"), "class": StringValue("B")},
				},
			},
		}, {
			caseName: "filtered set with condition without spaces",
			n: Map{
				"users": Array{
					Map{"name": StringValue("one"), "class": StringValue("A")},
					Map{"name": StringValue("two"), "class": StringValue("A")},
				},
			},
			expr: `..users[.name=="two"].class="B"`, // NOTE: trim spaces
			want: Map{
				"users": Array{
					Map{"name": StringValue("one"), "class": StringValue("A")},
					Map{"name": StringValue("two"), "class": StringValue("B")},
				},
			},
		},

		// --- Pipe operations ---
		{
			caseName: "pipe with index set",
			n: Map{
				"users": Array{
					Map{"name": StringValue("one"), "class": StringValue("A")},
				},
			},
			expr: `.users[] | [0].name = "ONE"`,
			want: Map{
				"users": Array{
					Map{"name": StringValue("ONE"), "class": StringValue("A")},
				},
			},
		},

		// --- Array holder scenarios (nested array mutations) ---
		{
			caseName: "append to array nested in map nested in array",
			n: Array{
				Map{"items": Array{StringValue("a")}},
			},
			expr: `[0].items += "b"`,
			want: Array{
				Map{"items": Array{StringValue("a"), StringValue("b")}},
			},
		}, {
			caseName: "append multiple levels deep array",
			n: Map{
				"l1": Array{
					Map{
						"l2": Array{StringValue("x")},
					},
				},
			},
			expr: `.l1[0].l2 += "y"`,
			want: Map{
				"l1": Array{
					Map{
						"l2": Array{StringValue("x"), StringValue("y")},
					},
				},
			},
		}, {
			caseName: "delete from array nested in array",
			n: Array{
				Array{StringValue("a"), StringValue("b"), StringValue("c")},
			},
			expr: `[0][1] ^?`,
			want: Array{
				Array{StringValue("a"), StringValue("c")},
			},
		}, {
			caseName: "set on deeply nested array path",
			n: Map{
				"a": Array{
					Map{
						"b": Array{
							Map{"c": StringValue("old")},
						},
					},
				},
			},
			expr: `.a[0].b[0].c = "new"`,
			want: Map{
				"a": Array{
					Map{
						"b": Array{
							Map{"c": StringValue("new")},
						},
					},
				},
			},
		}, {
			caseName: "append to root array with existing elements",
			n:    Array{StringValue("a"), StringValue("b")},
			expr: `. += "c"`,
			want: Array{StringValue("a"), StringValue("b"), StringValue("c")},
		}, {
			caseName: "delete from root array",
			n:    Array{StringValue("a"), StringValue("b"), StringValue("c")},
			expr: `[1] ^?`,
			want: Array{StringValue("a"), StringValue("c")},
		}, {
			caseName: "set grows array in nested structure",
			n: Map{
				"data": Array{StringValue("x")},
			},
			expr: `.data[3] = "y"`,
			want: Map{
				"data": Array{StringValue("x"), nil, nil, StringValue("y")},
			},
		}, {
			caseName: "walk append on nested arrays",
			n: Map{
				"groups": Array{
					Map{"tags": Array{StringValue("a")}},
					Map{"tags": Array{StringValue("b")}},
				},
			},
			expr: `..tags += "z"`,
			want: Map{
				"groups": Array{
					Map{"tags": Array{StringValue("a"), StringValue("z")}},
					Map{"tags": Array{StringValue("b"), StringValue("z")}},
				},
			},
		}, {
			caseName: "walk delete from nested arrays",
			n: Map{
				"rows": Array{
					Map{"cols": Array{NumberValue(1), NumberValue(2)}},
					Map{"cols": Array{NumberValue(3), NumberValue(4)}},
				},
			},
			expr: `..cols[0] ^?`,
			want: Map{
				"rows": Array{
					Map{"cols": Array{NumberValue(2)}},
					Map{"cols": Array{NumberValue(4)}},
				},
			},
		}, {
			caseName: "filtered append on array elements",
			n: Map{
				"items": Array{
					Map{"type": StringValue("list"), "values": Array{NumberValue(1)}},
					Map{"type": StringValue("single"), "values": Array{NumberValue(2)}},
				},
			},
			expr: `.items[.type == "list"].values += 9`,
			want: Map{
				"items": Array{
					Map{"type": StringValue("list"), "values": Array{NumberValue(1), NumberValue(9)}},
					Map{"type": StringValue("single"), "values": Array{NumberValue(2)}},
				},
			},
		}, {
			caseName: "set on array in array in array",
			n:    Array{Array{Array{StringValue("deep")}}},
			expr: `[0][0][0] = "new"`,
			want: Array{Array{Array{StringValue("new")}}},
		}, {
			caseName: "append to array in array in array",
			n:    Array{Array{Array{StringValue("a")}}},
			expr: `[0][0] += "b"`,
			want: Array{Array{Array{StringValue("a"), StringValue("b")}}},
		}, {
			caseName: "delete from array in array in array",
			n:    Array{Array{Array{StringValue("a"), StringValue("b")}}},
			expr: `[0][0][0] ^?`,
			want: Array{Array{Array{StringValue("b")}}},
		}, {
			caseName: "set replaces entire nested array",
			n: Map{
				"data": Array{NumberValue(1), NumberValue(2)},
			},
			expr: `.data = [3, 4, 5]`,
			want: Map{
				"data": Array{NumberValue(3), NumberValue(4), NumberValue(5)},
			},
		}, {
			caseName: "edit with nil element in array",
			n:    Array{nil, StringValue("b")},
			expr: `[0] = "a"`,
			want: Array{StringValue("a"), StringValue("b")},
		}, {
			caseName: "delete middle element from three-element array",
			n:    Array{StringValue("a"), StringValue("b"), StringValue("c")},
			expr: `[1] ^?`,
			want: Array{StringValue("a"), StringValue("c")},
		}, {
			caseName: "append to empty nested array",
			n: Map{
				"list": Array{},
			},
			expr: `.list += "first"`,
			want: Map{
				"list": Array{StringValue("first")},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.caseName, func(t *testing.T) {
			err := Edit(&(tc.n), tc.expr)
			if tc.errstr != "" {
				if err == nil {
					t.Fatalf("no error; want %s", tc.errstr)
				}
				if err.Error() != tc.errstr {
					t.Errorf("got error %s; want %s", err.Error(), tc.errstr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %+v", err)
			}
			got := tc.n
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("got %#v; want %#v", got, tc.want)
			}
		})
	}
}

// FuzzParseQuery exercises the query parser with arbitrary input,
// ensuring it never panics. It does not assert successful parsing;
// it only requires that errors are returned cleanly for invalid input.
func FuzzParseQuery(f *testing.F) {
	seeds := []string{
		".",
		".foo",
		".foo.bar",
		".a[0]",
		".a[0:3]",
		".a[.name == \"x\"]",
		"..walk",
		".a | .b",
		".a.count()",
		".a.sort(\".name\")",
	}
	for _, s := range seeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, expr string) {
		_, _ = ParseQuery(expr)
	})
}

// FuzzEdit exercises the Edit expression parser and evaluator with
// arbitrary input against a fresh empty Map. It only requires that
// invalid expressions return an error rather than panicking.
func FuzzEdit(f *testing.F) {
	seeds := []string{
		`.a = 1`,
		`.a = "x"`,
		`.a += 1`,
		`.a.b = true`,
		`.a = [1, 2, 3]`,
		`.a = {"k": "v"}`,
		`delete .a`,
	}
	for _, s := range seeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, expr string) {
		var n Node = Map{}
		_ = Edit(&n, expr)
	})
}

const benchStoreJSON = `{
  "store": {
    "bicycle": { "color": "red", "price": 19.95 },
    "book": [
      { "author": "Nigel Rees", "category": "reference", "price": 8.95, "title": "Sayings of the Century",
        "tags": [{"name":"genre","value":"reference"},{"name":"era","value":"20th century"},{"name":"theme","value":"quotations"}] },
      { "author": "Evelyn Waugh", "category": "fiction", "price": 12.99, "title": "Sword of Honour",
        "tags": [{"name":"genre","value":"fiction"},{"name":"era","value":"20th century"},{"name":"theme","value":"WWII"}] },
      { "author": "Herman Melville", "category": "fiction", "isbn": "0-553-21311-3", "price": 8.99, "title": "Moby Dick",
        "tags": [{"name":"genre","value":"fiction"},{"name":"era","value":"19th century"},{"name":"theme","value":"whale hunting"}] },
      { "author": "J. R. R. Tolkien", "category": "fiction", "isbn": "0-395-19395-8", "price": 22.99, "title": "The Lord of the Rings",
        "tags": [{"name":"genre","value":"fantasy"},{"name":"era","value":"20th century"},{"name":"theme","value":"good vs evil"}] }
    ]
  }
}`

func mustBenchNode(b *testing.B) Node {
	b.Helper()
	n, err := UnmarshalJSON([]byte(benchStoreJSON))
	if err != nil {
		b.Fatal(err)
	}
	return n
}

// BenchmarkQueryParse measures the cost of parsing representative
// query expressions of varying complexity.
func BenchmarkQueryParse(b *testing.B) {
	exprs := []struct {
		caseName string
		expr     string
	}{
		{"simple", ".store.book[0].title"},
		{"range", ".store.book[1:3]|"},
		{"selector", `.store.book[.tags[.name == "genre" and .value == "fiction"]]|`},
		{"method", ".store.book.count()"},
	}
	for _, e := range exprs {
		b.Run(e.caseName, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				if _, err := ParseQuery(e.expr); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkFind measures end-to-end query evaluation (parse + walk)
// for a small handful of representative query shapes against the
// embedded sample document.
func BenchmarkFind(b *testing.B) {
	n := mustBenchNode(b)
	exprs := []struct {
		caseName string
		expr     string
	}{
		{"simple", ".store.book[0].title"},
		{"range", ".store.book[1:3]|"},
		{"selector", `.store.book[.tags[.name == "genre" and .value == "fiction"]]|`},
		{"walk", "..walk"},
	}
	for _, e := range exprs {
		b.Run(e.caseName, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				if _, err := Find(n, e.expr); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkEdit measures Edit performance across different operations
// and tree depths. These benchmarks serve as the baseline for
// comparing the arrayHolder implementation against future alternatives.
func BenchmarkEdit(b *testing.B) {
	testCases := []struct {
		caseName string
		setup    func() Node
		expr     string
	}{
		{
			caseName: "set_shallow",
			setup:    func() Node { return mustBenchNode(b) },
			expr:     `.store.bicycle.color = "blue"`,
		},
		{
			caseName: "set_deep_array",
			setup:    func() Node { return mustBenchNode(b) },
			expr:     `.store.book[2].title = "New Title"`,
		},
		{
			caseName: "append_to_array",
			setup:    func() Node { return mustBenchNode(b) },
			expr:     `.store.book += {"author":"New","title":"Book"}`,
		},
		{
			caseName: "delete_from_array",
			setup:    func() Node { return mustBenchNode(b) },
			expr:     `.store.book[0] ^?`,
		},
		{
			caseName: "walk_set",
			setup:    func() Node { return mustBenchNode(b) },
			expr:     `..price = 0`,
		},
		{
			caseName: "filtered_set",
			setup:    func() Node { return mustBenchNode(b) },
			expr:     `.store.book[.category == "fiction"].price = 9.99`,
		},
		{
			caseName: "nested_array_append",
			setup:    func() Node { return mustBenchNode(b) },
			expr:     `.store.book[0].tags += {"name":"new","value":"tag"}`,
		},
	}
	for _, tc := range testCases {
		b.Run(tc.caseName, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				n := tc.setup()
				if err := Edit(&n, tc.expr); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
