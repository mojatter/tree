package tree

import (
	"reflect"
	"testing"

	"github.com/mojatter/tree/internal/testdata"
)

func TestQuery(t *testing.T) {
	testCases := []struct {
		caseName string
		q        Query
		n        Node
		want     []Node
		errstr   string
	}{
		{
			caseName: "nop",
			q:        NopQuery{},
			n:        Array{},
			want:     []Node{Array{}},
		}, {
			caseName: "map: hit",
			q:        MapQuery("key"),
			n:        Map{"key": ToValue("value")},
			want:     []Node{ToValue("value")},
		}, {
			caseName: "map: not map",
			q:        MapQuery("key"),
			n:        ToValue("not map"),
			errstr:   `cannot index array with "key"`,
		}, {
			caseName: "array: hit",
			q:        ArrayQuery(0),
			n:        Array{ToValue(1)},
			want:     []Node{ToValue(1)},
		}, {
			caseName: "array: not array",
			q:        ArrayQuery(0),
			n:        ToValue("not array"),
			errstr:   `cannot index array with 0`,
		}, {
			caseName: "range: 0:2",
			q:        ArrayRangeQuery{0, 2},
			n:        Array{ToValue(0), ToValue(1), ToValue(2)},
			want:     []Node{ToValue(0), ToValue(1)},
		}, {
			caseName: "range: 1:end",
			q:        ArrayRangeQuery{1, -1},
			n:        Array{ToValue(0), ToValue(1), ToValue(2)},
			want:     []Node{ToValue(1), ToValue(2)},
		}, {
			caseName: "range: invalid arity",
			q:        ArrayRangeQuery{0, 1, 2},
			n:        Array{},
			errstr:   `invalid array range [0:1:2]`,
		}, {
			caseName: "range: not array",
			q:        ArrayRangeQuery{0, 1},
			n:        Map{},
			errstr:   `cannot index array with range 0:1`,
		}, {
			caseName: "filter: hit",
			q:        FilterQuery{MapQuery("key"), ArrayQuery(0)},
			n:        Map{"key": Array{ToValue(1)}},
			want:     []Node{ToValue(1)},
		}, {
			caseName: "filter: error",
			q:        FilterQuery{MapQuery("key"), ArrayQuery(0)},
			n:        Map{"key": ToValue(1)},
			errstr:   `cannot index array with 0`,
		}, {
			caseName: "select: match",
			q: SelectQuery{And{
				Comparator{MapQuery("key"), EQ, ValueQuery{ToValue(1)}},
			}},
			n:    Array{Map{"key": ToValue(1)}, Map{"key": ToValue(2)}},
			want: []Node{Map{"key": ToValue(1)}},
		}, {
			caseName: "select: all",
			q:        SelectQuery{},
			n:        Array{Map{"key": ToValue(1)}, Map{"key": ToValue(2)}},
			want:     []Node{Map{"key": ToValue(1)}, Map{"key": ToValue(2)}},
		}, {
			caseName: "select: no match on map",
			q: SelectQuery{And{
				Comparator{MapQuery("key"), EQ, ValueQuery{ToValue(1)}},
			}},
			n: Map{},
		}, {
			caseName: "select: complex",
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
			caseName: "walk",
			q:        WalkQuery("key1"),
			n: Array{
				Map{"key1": ToValue(1), "key2": ToValue("a")},
				Map{"key1": ToValue(2), "key2": ToValue("b")},
				Map{"key1": ToValue(3), "key2": ToValue("c")},
			},
			want: []Node{ToValue(1), ToValue(2), ToValue(3)},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.caseName, func(t *testing.T) {
			got, err := tc.q.Exec(tc.n)
			if tc.errstr != "" {
				if err == nil {
					t.Fatalf("for %v; no error", tc.q)
				}
				if err.Error() != tc.errstr {
					t.Errorf("for %v; got %s; want %s", tc.q, err.Error(), tc.errstr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("for %v; got %v; want %v", tc.q, got, tc.want)
			}
		})
	}
}

func TestQueryString(t *testing.T) {
	testCases := []struct {
		caseName string
		q        Query
		want     string
	}{
		{
			caseName: "nop",
			q:        NopQuery{},
			want:     ".",
		}, {
			caseName: "map",
			q:        MapQuery("key"),
			want:     ".key",
		}, {
			caseName: "array index",
			q:        ArrayQuery(1),
			want:     "[1]",
		}, {
			caseName: "range full",
			q:        ArrayRangeQuery{0, 2},
			want:     "[0:2]",
		}, {
			caseName: "range to only",
			q:        ArrayRangeQuery{-1, 2},
			want:     "[:2]",
		}, {
			caseName: "slurp",
			q:        SlurpQuery{},
			want:     " | ",
		}, {
			caseName: "filter",
			q:        FilterQuery{MapQuery("key1"), ArrayQuery(0), MapQuery("key2")},
			want:     ".key1[0].key2",
		}, {
			caseName: "select complex",
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
			caseName: "walk",
			q:        WalkQuery("key"),
			want:     "..key",
		}, {
			caseName: "filter walk",
			q:        FilterQuery{MapQuery("key1"), WalkQuery("key2")},
			want:     ".key1..key2",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.caseName, func(t *testing.T) {
			got := tc.q.String()
			if got != tc.want {
				t.Errorf("got %v; want %v", got, tc.want)
			}
		})
	}
}

func TestFind(t *testing.T) {
	n, err := UnmarshalJSON([]byte(testdata.StoreJSON))
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
		t.Run(tc.expr, func(t *testing.T) {
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

func mustBenchNode(b *testing.B) Node {
	b.Helper()
	n, err := UnmarshalJSON([]byte(testdata.StoreJSON))
	if err != nil {
		b.Fatal(err)
	}
	return n
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

// testSelectorDelegator is a test-only Selector whose Matches behavior
// can be configured per-test by assigning matchesFunc. It exists so
// that tests can exercise Selector error paths and custom match logic
// that parser-produced selectors cannot easily produce.
type testSelectorDelegator struct {
	matchesFunc func(Node) (bool, error)
}

func (d *testSelectorDelegator) Matches(n Node) (bool, error) {
	if d.matchesFunc != nil {
		return d.matchesFunc(n)
	}
	return false, nil
}

func (d *testSelectorDelegator) String() string {
	return "testSelectorDelegator"
}
