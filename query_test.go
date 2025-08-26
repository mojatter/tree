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
	tests := []struct {
		n      Node
		expr   string
		want   Node
		errstr string
	}{
		{
			n:    Map{},
			expr: `.store = {}`,
			want: Map{"store": Map{}},
		}, {
			n:    Map{},
			expr: `.store={}`, // NOTE: trim spaces
			want: Map{"store": Map{}},
		}, {
			n:    Map{},
			expr: `.store.book = {}`,
			want: Map{"store": Map{"book": Map{}}},
		}, {
			n:    Map{},
			expr: `.store.pen = [{"color":"red"},{"color":"blue"}]`,
			want: Map{
				"store": Map{
					"pen": Array{
						Map{"color": StringValue("red")},
						Map{"color": StringValue("blue")},
					},
				},
			},
		}, {
			n:      StringValue("str"),
			expr:   `.key = {}`,
			errstr: `cannot index array with "key"`,
		}, {
			n:      Map{"key": StringValue("str")},
			expr:   `. += {}`,
			errstr: "cannot append to .",
		}, {
			n:      StringValue("str"),
			expr:   `. += {}`,
			errstr: "cannot append to .",
		}, {
			n:      Map{"key": StringValue("str")},
			expr:   `.key += {}`,
			errstr: `cannot append to "key"`,
		}, {
			n:      StringValue("str"),
			expr:   `.key += {}`,
			errstr: `cannot append to "key"`,
		}, {
			n:    Array{},
			expr: `[0] = "red"`,
			want: Array{StringValue("red")},
		}, {
			n:    Array{},
			expr: `[0][1] = "red"`,
			want: Array{Array{nil, StringValue("red")}},
		}, {
			n:      StringValue("str"),
			expr:   `[0] = "red"`,
			errstr: `cannot index array with 0`,
		}, {
			n:    Array{},
			expr: `.0 = "red"`,
			want: Array{StringValue("red")},
		}, {
			n:    Array{},
			expr: `.0.1 = "red"`,
			want: Array{Map{"1": StringValue("red")}},
		}, {
			n:    Array{},
			expr: `. = "red"`,
			want: StringValue("red"),
		}, {
			n:    Map{},
			expr: `.colors += "red"`,
			want: Map{"colors": Array{StringValue("red")}},
		}, {
			n:    Map{},
			expr: `.colors+="red"`, // NOTE: trime spaces
			want: Map{"colors": Array{StringValue("red")}},
		}, {
			n:    Map{"colors": Array{StringValue("red"), StringValue("green")}},
			expr: `.colors += "blue"`,
			want: Map{"colors": Array{StringValue("red"), StringValue("green"), StringValue("blue")}},
		}, {
			n:    Array{Array{StringValue("red")}},
			expr: `[0] += "blue"`,
			want: Array{Array{StringValue("red"), StringValue("blue")}},
		}, {
			n:    Array{Array{StringValue("red")}},
			expr: `[2] += "blue"`,
			want: Array{Array{StringValue("red")}, nil, Array{StringValue("blue")}},
		}, {
			n:      Array{StringValue("red")},
			expr:   `[0] += "blue"`,
			errstr: `cannot append to array with 0`,
		}, {
			n:      StringValue("red"),
			expr:   `[0] += "blue"`,
			errstr: `cannot append to array with 0`,
		}, {
			n:    Array{},
			expr: `. += "red"`,
			want: Array{StringValue("red")},
		}, {
			n:    Map{"key1": StringValue("value1"), "key2": StringValue("value2")},
			expr: `.key1 ^?`,
			want: Map{"key2": StringValue("value2")},
		}, {
			n:    Map{"key1": StringValue("value1"), "key2": StringValue("value2")},
			expr: `.key1^?`, // NOTE: trim spaces
			want: Map{"key2": StringValue("value2")},
		}, {
			n:    Array{StringValue("red")},
			expr: `[0] ^?`,
			want: Array{},
		}, {
			n:    Array{StringValue("red")},
			expr: `.0 ^?`,
			want: Array{},
		}, {
			n:      Map{},
			expr:   `. ^?`,
			errstr: "cannot delete .",
		}, {
			n:      StringValue("str"),
			expr:   `.key ^?`,
			errstr: `cannot delete "key"`,
		}, {
			n:      StringValue("str"),
			expr:   `[0] ^?`,
			errstr: `cannot delete array with 0`,
		}, {
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
		}, {
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
		}, {
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
	}
	for i, test := range tests {
		err := Edit(&(test.n), test.expr)
		if test.errstr != "" {
			if err == nil {
				t.Fatalf("tests[%d] for %v; no error", i, test.expr)
			}
			if err.Error() != test.errstr {
				t.Errorf("tests[%d] for %v; got %s; want %s", i, test.expr, err.Error(), test.errstr)
			}
			continue
		}
		if err != nil {
			t.Fatalf("tests[%d] for %v; %+v", i, test.expr, err)
		}
		got := test.n
		if !reflect.DeepEqual(got, test.want) {
			t.Errorf("tests[%d] for %v; got %#v; want %#v", i, test.expr, got, test.want)
		}
	}
}
