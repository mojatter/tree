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
			caseName: "value: null literal",
			q:        ValueQuery{Nil},
			n:        Map{},
			want:     []Node{Nil},
		}, {
			caseName: "value: Go nil literal reads as null",
			q:        ValueQuery{},
			n:        Map{},
			want:     []Node{Nil},
		}, {
			caseName: "map: hit",
			q:        MapQuery("key"),
			n:        Map{"key": ToValue("value")},
			want:     []Node{ToValue("value")},
		}, {
			caseName: "map: scalar reads as null",
			q:        MapQuery("key"),
			n:        ToValue("not map"),
			want:     []Node{Nil},
		}, {
			caseName: "map: null reads as null",
			q:        MapQuery("key"),
			n:        Nil,
			want:     []Node{Nil},
		}, {
			caseName: "map: missing key reads as null",
			q:        MapQuery("key"),
			n:        Map{},
			want:     []Node{Nil},
		}, {
			caseName: "array: hit",
			q:        ArrayQuery(0),
			n:        Array{ToValue(1)},
			want:     []Node{ToValue(1)},
		}, {
			caseName: "array: scalar reads as null",
			q:        ArrayQuery(0),
			n:        ToValue("not array"),
			want:     []Node{Nil},
		}, {
			caseName: "array: null reads as null",
			q:        ArrayQuery(0),
			n:        Nil,
			want:     []Node{Nil},
		}, {
			caseName: "array: index on a map reads the key",
			q:        ArrayQuery(0),
			n:        Map{"0": ToValue(1)},
			want:     []Node{ToValue(1)},
		}, {
			caseName: "array: index out of range reads as null",
			q:        ArrayQuery(3),
			n:        Array{ToValue(0)},
			want:     []Node{Nil},
		}, {
			caseName: "array: negative index hit",
			q:        ArrayQuery(-1),
			n:        Array{ToValue(0), ToValue(1), ToValue(2)},
			want:     []Node{ToValue(2)},
		}, {
			caseName: "array: negative index out of range reads as null",
			q:        ArrayQuery(-4),
			n:        Array{ToValue(0), ToValue(1), ToValue(2)},
			want:     []Node{Nil},
		}, {
			caseName: "range: 0:2",
			q:        ArrayRangeQuery{From: IntPtr(0), To: IntPtr(2)},
			n:        Array{ToValue(0), ToValue(1), ToValue(2)},
			want:     []Node{ToValue(0), ToValue(1)},
		}, {
			caseName: "range: 1:end",
			q:        ArrayRangeQuery{From: IntPtr(1), To: nil},
			n:        Array{ToValue(0), ToValue(1), ToValue(2)},
			want:     []Node{ToValue(1), ToValue(2)},
		}, {
			caseName: "range: not array yields nothing",
			q:        ArrayRangeQuery{From: IntPtr(0), To: IntPtr(1)},
			n:        Map{},
		}, {
			caseName: "range: null yields nothing",
			q:        ArrayRangeQuery{From: IntPtr(0), To: IntPtr(1)},
			n:        Nil,
		}, {
			caseName: "range: negative from to end",
			q:        ArrayRangeQuery{From: IntPtr(-3), To: nil},
			n:        Array{ToValue(0), ToValue(1), ToValue(2), ToValue(3), ToValue(4)},
			want:     []Node{ToValue(2), ToValue(3), ToValue(4)},
		}, {
			caseName: "range: 1 to negative",
			q:        ArrayRangeQuery{From: IntPtr(1), To: IntPtr(-1)},
			n:        Array{ToValue(0), ToValue(1), ToValue(2), ToValue(3), ToValue(4)},
			want:     []Node{ToValue(1), ToValue(2), ToValue(3)},
		}, {
			caseName: "range: both negative",
			q:        ArrayRangeQuery{From: IntPtr(-3), To: IntPtr(-1)},
			n:        Array{ToValue(0), ToValue(1), ToValue(2), ToValue(3), ToValue(4)},
			want:     []Node{ToValue(2), ToValue(3)},
		}, {
			caseName: "range: clamp from below 0",
			q:        ArrayRangeQuery{From: IntPtr(-100), To: IntPtr(2)},
			n:        Array{ToValue(0), ToValue(1), ToValue(2)},
			want:     []Node{ToValue(0), ToValue(1)},
		}, {
			caseName: "range: low greater than high yields empty",
			q:        ArrayRangeQuery{From: IntPtr(2), To: IntPtr(1)},
			n:        Array{ToValue(0), ToValue(1), ToValue(2)},
		}, {
			caseName: "filter: hit",
			q:        FilterQuery{MapQuery("key"), ArrayQuery(0)},
			n:        Map{"key": Array{ToValue(1)}},
			want:     []Node{ToValue(1)},
		}, {
			caseName: "filter: index on a scalar reads as null",
			q:        FilterQuery{MapQuery("key"), ArrayQuery(0)},
			n:        Map{"key": ToValue(1)},
			want:     []Node{Nil},
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
			caseName: "select: multiple match error formats quoted query without double escaping",
			q: SelectQuery{And{
				Comparator{WalkQuery("b c"), EQ, ValueQuery{ToValue(1)}},
			}},
			n:      Array{Map{"x": Map{"b c": ToValue(1)}, "y": Map{"b c": ToValue(2)}}},
			errstr: `.."b c" returns no single value [1 2]`,
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
		}, {
			caseName: "array: nil element is null",
			q:        ArrayQuery(0),
			n:        Array{nil},
			want:     []Node{Nil},
		}, {
			caseName: "range: nil element is null",
			q:        ArrayRangeQuery{},
			n:        Array{nil, ToValue(1)},
			want:     []Node{Nil, ToValue(1)},
		}, {
			caseName: "select: nil array element is null",
			q:        SelectQuery{},
			n:        Array{nil},
			want:     []Node{Nil},
		}, {
			caseName: "select: nil map value is null",
			q:        SelectQuery{},
			n:        Map{"k": nil},
			want:     []Node{Nil},
		}, {
			caseName: "select: key on a nil element compares as null",
			q:        SelectQuery{Comparator{Left: MapQuery("x"), Op: EQ, Right: ValueQuery{Nil}}},
			n:        Array{nil},
			want:     []Node{Nil},
		}, {
			caseName: "select: null literal matches null and a missing key",
			q:        SelectQuery{Comparator{Left: MapQuery("a"), Op: EQ, Right: ValueQuery{Nil}}},
			n:        Array{Map{}, Map{"a": ToValue(1)}, Map{"a": Nil}},
			want:     []Node{Map{}, Map{"a": Nil}},
		}, {
			caseName: "select: not equal matches a missing key",
			q:        SelectQuery{Comparator{Left: MapQuery("a"), Op: NE, Right: ValueQuery{ToValue(1)}}},
			n:        Array{Map{"a": ToValue(1)}, Map{"a": ToValue(2)}, Map{}},
			want:     []Node{Map{"a": ToValue(2)}, Map{}},
		}, {
			caseName: "select: two missing keys are equal",
			q:        SelectQuery{Comparator{Left: MapQuery("x"), Op: EQ, Right: MapQuery("y")}},
			n:        Array{Map{}},
			want:     []Node{Map{}},
		}, {
			caseName: "select: two missing keys are not unequal",
			q:        SelectQuery{Comparator{Left: MapQuery("x"), Op: NE, Right: MapQuery("y")}},
			n:        Array{Map{}},
		}, {
			caseName: "select: empty sides never match",
			q:        SelectQuery{Comparator{Left: FilterQuery{MapQuery("x"), SelectQuery{}}, Op: GT, Right: FilterQuery{MapQuery("y"), SelectQuery{}}}},
			n:        Array{Map{"x": Array{}, "y": Array{}}},
		}, {
			caseName: "select: empty sides are not equal",
			q:        SelectQuery{Comparator{Left: FilterQuery{MapQuery("x"), SelectQuery{}}, Op: EQ, Right: FilterQuery{MapQuery("y"), SelectQuery{}}}},
			n:        Array{Map{"x": Array{}, "y": Array{}}},
		}, {
			caseName: "select: empty side does not equal null",
			q:        SelectQuery{Comparator{Left: FilterQuery{MapQuery("x"), SelectQuery{}}, Op: EQ, Right: ValueQuery{Nil}}},
			n:        Array{Map{"x": Array{}}},
		}, {
			caseName: "select: a side with a null among several values errors",
			q:        SelectQuery{Comparator{Left: FilterQuery{MapQuery("tags"), SelectQuery{}, MapQuery("x")}, Op: EQ, Right: ValueQuery{ToValue(1)}}},
			n:        Array{Map{"tags": Array{Map{"x": ToValue(1)}, Map{"y": ToValue(2)}}}},
			errstr:   `.tags[].x returns no single value [1 null]`,
		}, {
			caseName: "select: null is not ordered",
			q:        SelectQuery{Comparator{Left: MapQuery("a"), Op: LT, Right: ValueQuery{ToValue(1)}}},
			n:        Array{Map{}},
		}, {
			caseName: "filter: nil result continues as null",
			q:        FilterQuery{ArrayQuery(0), &CountQuery{}},
			n:        Array{nil},
			want:     []Node{ToValue(0)},
		}, {
			caseName: "select: typed nil child from a custom Get compares as null",
			q:        &SelectQuery{Selector: Comparator{Left: MapQuery("a"), Op: EQ, Right: ValueQuery{ToValue(1)}}},
			n:        Array{rawGetNode{Map{"a": (*Array)(nil)}}, Map{"a": NumberValue(1)}},
			want:     []Node{Map{"a": NumberValue(1)}},
		}, {
			caseName: "select: typed nil children from a custom Get are not unequal",
			q:        &SelectQuery{Selector: Comparator{Left: MapQuery("a"), Op: NE, Right: MapQuery("b")}},
			n:        Array{rawGetNode{Map{"a": (*Array)(nil), "b": (*Map)(nil)}}},
			want:     nil,
		}, {
			caseName: "map: typed nil child from a custom Get reads as null",
			q:        MapQuery("a"),
			n:        rawGetNode{Map{"a": (*Array)(nil)}},
			want:     []Node{Nil},
		}, {
			caseName: "map: nil child from a custom Get reads as null",
			q:        MapQuery("a"),
			n:        rawGetNode{Map{"a": nil}},
			want:     []Node{Nil},
		}, {
			caseName: "walk: typed nil child from a custom Get reads as null",
			q:        WalkQuery("a"),
			n:        rawGetNode{Map{"a": (*Array)(nil)}},
			want:     []Node{Nil},
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
			q:        ArrayRangeQuery{From: IntPtr(0), To: IntPtr(2)},
			want:     "[0:2]",
		}, {
			caseName: "range to only",
			q:        ArrayRangeQuery{From: nil, To: IntPtr(2)},
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
		}, {
			caseName: "bare dot before a key",
			q:        FilterQuery{MapQuery("a"), NopQuery{}, MapQuery("b")},
			want:     ".a.b",
		}, {
			caseName: "bare dot before an index",
			q:        FilterQuery{NopQuery{}, ArrayQuery(0)},
			want:     ".[0]",
		}, {
			caseName: "bare pointer dot before a key",
			q:        FilterQuery{MapQuery("a"), &NopQuery{}, MapQuery("b")},
			want:     ".a.b",
		}, {
			caseName: "bare dots around a pipe",
			q:        FilterQuery{NopQuery{}, SlurpQuery{}, NopQuery{}},
			want:     ". | .",
		}, {
			caseName: "map quoted with spaces",
			q:        MapQuery("b c"),
			want:     `."b c"`,
		}, {
			caseName: "map quoted with escape",
			q:        MapQuery(`foo"bar`),
			want:     `."foo\"bar"`,
		}, {
			caseName: "walk quoted with spaces",
			q:        WalkQuery("b c"),
			want:     `.."b c"`,
		}, {
			caseName: "walk quoted with escape",
			q:        WalkQuery(`foo"bar`),
			want:     `.."foo\"bar"`,
		}, {
			caseName: "map empty key",
			q:        MapQuery(""),
			want:     `.""`,
		}, {
			caseName: "value string",
			q:        ValueQuery{StringValue("a")},
			want:     `"a"`,
		}, {
			caseName: "value string with escapes",
			q:        ValueQuery{StringValue(`a"b\c`)},
			want:     `"a\"b\\c"`,
		}, {
			caseName: "value string with newline",
			q:        ValueQuery{StringValue("a\nb")},
			want:     "\"a\nb\"",
		}, {
			caseName: "value string with html",
			q:        ValueQuery{StringValue("<&>")},
			want:     `"<&>"`,
		}, {
			caseName: "value number",
			q:        ValueQuery{NumberValue(1.5)},
			want:     `1.5`,
		}, {
			caseName: "value null",
			q:        ValueQuery{Nil},
			want:     `null`,
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

func TestFindNull(t *testing.T) {
	testCases := []struct {
		caseName string
		n        Node
		expr     string
		want     []Node
	}{
		{caseName: "Nil root", n: Nil, expr: ".", want: []Node{Nil}},
		{caseName: "Go nil root", n: nil, expr: ".", want: []Node{Nil}},
		{caseName: "nil pointer root", n: (*Array)(nil), expr: ".", want: []Node{Nil}},
		{caseName: "zero Any root", n: Any{}, expr: ".", want: []Node{Nil}},
		{caseName: "key on a Nil root", n: Nil, expr: ".a", want: []Node{Nil}},
		{caseName: "keys() on a Nil root", n: Nil, expr: ".keys()", want: nil},
		{caseName: "count() on a Nil root", n: Nil, expr: ".count()", want: []Node{NumberValue(0)}},
		{caseName: "[] on a Nil root", n: Nil, expr: "[]", want: nil},
		{caseName: "has() on a Nil root", n: Nil, expr: `.has("a")`, want: []Node{BoolValue(false)}},
		{caseName: "nil Map root", n: Map(nil), expr: ".", want: []Node{Map(nil)}},
		{caseName: "key on a nil Map root", n: Map(nil), expr: ".a", want: []Node{Nil}},
		{caseName: "keys() on a nil Map root", n: Map(nil), expr: ".keys()", want: []Node{Array{}}},
		{caseName: "key on a nil Array root", n: Array(nil), expr: ".a", want: []Node{Nil}},
		{caseName: "[] on a nil Array root", n: Array(nil), expr: "[]", want: nil},
		{caseName: "key on a zero Any root", n: Any{}, expr: ".a", want: []Node{Nil}},
		{caseName: "key on an Any wrapping a nil Map root", n: Any{Map(nil)}, expr: ".a", want: []Node{Nil}},
		{caseName: "key on a string root", n: StringValue("x"), expr: ".a", want: []Node{Nil}},
		{caseName: "key on a bool root", n: BoolValue(true), expr: ".a", want: []Node{Nil}},
		{caseName: "key on a number root", n: NumberValue(1), expr: ".a", want: []Node{Nil}},
		{caseName: "key on an empty Map root", n: Map{}, expr: ".a", want: []Node{Nil}},
		{caseName: "index on an empty Array root", n: Array{}, expr: "[0]", want: []Node{Nil}},
		{caseName: "key on an Any root", n: Any{Map{"a": NumberValue(1)}}, expr: ".a", want: []Node{NumberValue(1)}},
		{
			caseName: "null literal does not match a container",
			n:        Array{Map{"x": Map{"y": NumberValue(1)}}, Map{"x": Array{NumberValue(1)}}, Map{"x": Nil}, Map{}},
			expr:     `[.x == null]`,
			want:     []Node{Map{"x": Nil}, Map{}},
		}, {
			caseName: "not null keeps a container",
			n:        Array{Map{"x": Map{"y": NumberValue(1)}}, Map{"x": Array{NumberValue(1)}}, Map{"x": Nil}, Map{}},
			expr:     `[.x != null]`,
			want:     []Node{Map{"x": Map{"y": NumberValue(1)}}, Map{"x": Array{NumberValue(1)}}},
		}, {
			caseName: "map element is not null",
			n:        Array{Map{}, Nil},
			expr:     `[. == null]`,
			want:     []Node{Nil},
		}, {
			caseName: "containers compare by value",
			n: Array{
				Map{"x": Map{"a": NumberValue(1)}, "y": Map{"a": NumberValue(1)}},
				Map{"x": Map{"a": NumberValue(1)}, "y": Map{"a": NumberValue(2)}},
				Map{"x": Array{}, "y": Map{}},
				Map{"x": Array{}, "y": StringValue("")},
			},
			expr: `[.x == .y]`,
			want: []Node{Map{"x": Map{"a": NumberValue(1)}, "y": Map{"a": NumberValue(1)}}},
		}, {
			caseName: "containers differ by value",
			n: Array{
				Map{"x": Map{"a": NumberValue(1)}, "y": Map{"a": NumberValue(1)}},
				Map{"x": Array{}, "y": StringValue("")},
			},
			expr: `[.x != .y]`,
			want: []Node{Map{"x": Array{}, "y": StringValue("")}},
		}, {
			caseName: "nested select matches when any element matches",
			n:        Array{Map{"tags": Array{Map{"x": NumberValue(1)}, Map{"y": NumberValue(2)}}}, Map{"tags": Array{Map{"y": NumberValue(1)}}}},
			expr:     `[.tags[.x == 1]]`,
			want:     []Node{Map{"tags": Array{Map{"x": NumberValue(1)}, Map{"y": NumberValue(2)}}}},
		}, {
			caseName: "several null values are not truthy",
			n:        Array{Map{"tags": Array{Map{"y": NumberValue(1)}, Map{"y": NumberValue(2)}}}},
			expr:     `[.tags[].x]`,
			want:     nil,
		}, {
			caseName: "several values are truthy when any is",
			n:        Array{Map{"tags": Array{Map{"y": NumberValue(1)}, Map{"x": NumberValue(2)}}}},
			expr:     `[.tags[].x]`,
			want:     []Node{Map{"tags": Array{Map{"y": NumberValue(1)}, Map{"x": NumberValue(2)}}}},
		}, {
			caseName: "several falsy values are not truthy",
			n:        Array{Map{"tags": Array{NumberValue(0), NumberValue(0)}}, Map{"tags": Array{NumberValue(0), NumberValue(1)}}},
			expr:     `[.tags[]]`,
			want:     []Node{Map{"tags": Array{NumberValue(0), NumberValue(1)}}},
		}, {
			caseName: "ordered operator never matches a container",
			n:        Array{Map{"x": Array{NumberValue(1)}, "y": Array{NumberValue(2)}}, Map{"x": Map{}, "y": NumberValue(1)}},
			expr:     `[.x < .y or .x >= .y]`,
			want:     nil,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.caseName, func(t *testing.T) {
			got, err := Find(tc.n, tc.expr)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("got %#v; want %#v", got, tc.want)
			}
			if v := reflect.ValueOf(tc.n); !v.IsValid() || v.Kind() == reflect.Pointer && v.IsNil() {
				return // no method to call on a nil interface or pointer
			}
			got, err = tc.n.Find(tc.expr)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("method: got %#v; want %#v", got, tc.want)
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
			expr: `. | .`,
			want: []Node{Array{n}},
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
		{
			expr: `.store.book[.isbn == null].title`,
			want: []Node{StringValue("Sayings of the Century"), StringValue("Sword of Honour")},
		}, {
			expr: `.store.book[.isbn != null].title`,
			want: []Node{StringValue("Moby Dick"), StringValue("The Lord of the Rings")},
		}, {
			expr: `.store.book[].isbn`,
			want: []Node{Nil, Nil, StringValue("0-553-21311-3"), StringValue("0-395-19395-8")},
		}, {
			expr: `.store.zz`,
			want: []Node{Nil},
		}, {
			expr: `.store.zz.count()`,
			want: []Node{NumberValue(0)},
		}, {
			expr: `.store.book[0].author[0]`,
			want: []Node{Nil},
		}, {
			expr: `.store.bicycle[1:]`,
			want: nil,
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

// rawGetNode wraps a Map whose Get returns a child as stored, without turning a Go nil node into Nil.
type rawGetNode struct{ Node }

func (n rawGetNode) Get(keys ...any) Node {
	return n.Node.(Map)[keys[0].(string)]
}
