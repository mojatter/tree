package tree

import (
	"reflect"
	"strings"
	"testing"
)

func TestParseQuery(t *testing.T) {
	nested := Query(MapQuery("a"))
	for range 40 {
		nested = SelectQuery{And{Comparator{nested, EQ, ValueQuery{NumberValue(1)}}}}
	}
	testCases := []struct {
		expr      string
		want      Query
		roundTrip Query // re-parse of want.String(); nil means want
	}{
		{
			expr: `.`,
			want: NopQuery{},
		}, {
			expr: `.[0]`,
			want: FilterQuery{NopQuery{}, ArrayQuery(0)},
		}, {
			expr: `. | .`,
			want: FilterQuery{NopQuery{}, SlurpQuery{}, NopQuery{}},
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
				ArrayRangeQuery{From: IntPtr(0), To: IntPtr(1)},
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
		}, {
			expr: `."foo\"bar"`,
			want: MapQuery(`foo"bar`),
		}, {
			expr: `."foo\\bar"`,
			want: MapQuery(`foo\bar`),
		}, {
			expr: `.'with space'`,
			want: MapQuery("with space"),
		}, {
			expr: `.'foo\'bar'`,
			want: MapQuery(`foo'bar`),
		}, {
			expr: `."b c"`,
			want: MapQuery("b c"),
		}, {
			expr:      `.a.["b c"]`,
			want:      FilterQuery{MapQuery("a"), NopQuery{}, MapQuery("b c")},
			roundTrip: FilterQuery{MapQuery("a"), MapQuery("b c")},
		}, {
			expr: `.."b c"`,
			want: WalkQuery("b c"),
		}, {
			expr: `.""`,
			want: MapQuery(""),
		}, {
			expr: `."".b`,
			want: FilterQuery{MapQuery(""), MapQuery("b")},
		}, {
			expr: `..""`,
			want: WalkQuery(""),
		}, {
			expr: `[""]`,
			want: MapQuery(""),
		}, {
			expr: `''`,
			want: ValueQuery{StringValue("")},
		}, {
			expr: `[.x == ""]`,
			want: SelectQuery{And{Comparator{MapQuery("x"), EQ, ValueQuery{StringValue("")}}}},
		}, {
			expr: `[(.x) == 1]`,
			want: SelectQuery{And{Comparator{MapQuery("x"), EQ, ValueQuery{NumberValue(1)}}}},
		}, {
			expr: `[(1) == .x]`,
			want: SelectQuery{And{Comparator{ValueQuery{NumberValue(1)}, EQ, MapQuery("x")}}},
		}, {
			expr: `[(.x | count()) == 1]`,
			want: SelectQuery{And{Comparator{FilterQuery{MapQuery("x"), SlurpQuery{}, &CountQuery{}}, EQ, ValueQuery{NumberValue(1)}}}},
		}, {
			expr: `[(.x).y == 1]`,
			want: SelectQuery{And{Comparator{FilterQuery{MapQuery("x"), MapQuery("y")}, EQ, ValueQuery{NumberValue(1)}}}},
		}, {
			expr: `[(.y[.z == 1]) == 1]`,
			want: SelectQuery{And{Comparator{FilterQuery{MapQuery("y"), SelectQuery{And{Comparator{MapQuery("z"), EQ, ValueQuery{NumberValue(1)}}}}}, EQ, ValueQuery{NumberValue(1)}}}},
		}, {
			expr: strings.Repeat("[(", 40) + ".a" + strings.Repeat(") == 1]", 40),
			want: nested,
		}, {
			expr: `[(.x) .y]`,
			want: SelectQuery{And{Evaluator{FilterQuery{MapQuery("x"), MapQuery("y")}}}},
		}, {
			expr: `[(.a | .b).c == 1]`,
			want: SelectQuery{And{Comparator{FilterQuery{FilterQuery{MapQuery("a"), SlurpQuery{}, MapQuery("b")}, MapQuery("c")}, EQ, ValueQuery{NumberValue(1)}}}},
		}, {
			expr: `[(.a | .b).c]`,
			want: SelectQuery{And{Evaluator{FilterQuery{FilterQuery{MapQuery("a"), SlurpQuery{}, MapQuery("b")}, MapQuery("c")}}}},
		}, {
			expr: `[(.x == 1) and (.y) == 2]`,
			want: SelectQuery{And{Comparator{MapQuery("x"), EQ, ValueQuery{NumberValue(1)}}, Comparator{MapQuery("y"), EQ, ValueQuery{NumberValue(2)}}}},
		}, {
			expr: "[.k == \"a\nb\"]",
			want: SelectQuery{And{Comparator{MapQuery("k"), EQ, ValueQuery{StringValue("a\nb")}}}},
		}, {
			expr: `[.k == "<&>"]`,
			want: SelectQuery{And{Comparator{MapQuery("k"), EQ, ValueQuery{StringValue("<&>")}}}},
		}, {
			expr: `[.k == "a\"b\\c"]`,
			want: SelectQuery{And{Comparator{MapQuery("k"), EQ, ValueQuery{StringValue(`a"b\c`)}}}},
		}, {
			expr: `has("a\"b")`,
			want: &HasQuery{Key: `a"b`},
		}, {
			expr: `has('a"b')`,
			want: &HasQuery{Key: `a"b`},
		}, {
			expr: `has("a\\")`,
			want: &HasQuery{Key: `a\`},
		}, {
			expr: `has("a\b")`,
			want: &HasQuery{Key: "ab"},
		}, {
			expr: `contains("x,y")`,
			want: &ContainsQuery{Value: "x,y"},
		}, {
			expr: `has('x')`,
			want: &HasQuery{Key: "x"},
		}, {
			expr: `has(isbn)`,
			want: &HasQuery{Key: "isbn"},
		}, {
			expr: `has( a )`,
			want: &HasQuery{Key: "a"},
		}, {
			expr: "has(\n\"a\"\r\n)",
			want: &HasQuery{Key: "a"},
		}, {
			expr: "has(\"a\r\nb\")",
			want: &HasQuery{Key: "a\r\nb"},
		}, {
			expr: `has("a)b")`,
			want: &HasQuery{Key: "a)b"},
		}, {
			expr: `has("a]b")`,
			want: &HasQuery{Key: "a]b"},
		}, {
			expr: `has(a(b[c]))`,
			want: &HasQuery{Key: "a(b[c])"},
		}, {
			expr: `sort((.a[0]))`,
			want: &SortQuery{Expr: "(.a[0])", Query: FilterQuery{MapQuery("a"), ArrayQuery(0)}},
		}, {
			expr: `count( )`,
			want: &CountQuery{},
		}, {
			expr: `sort(.a.count())`,
			want: &SortQuery{Expr: ".a.count()", Query: FilterQuery{MapQuery("a"), NopQuery{}, &CountQuery{}}},
		}, {
			expr: `sort(.["b c"])`,
			want: &SortQuery{Expr: `.["b c"]`, Query: FilterQuery{NopQuery{}, MapQuery("b c")}},
		}, {
			expr: `sort(."b c")`,
			want: &SortQuery{Expr: `."b c"`, Query: MapQuery("b c")},
		}, {
			expr: `rsort(."b\"c")`,
			want: &RSortQuery{Expr: `."b\"c"`, Query: MapQuery(`b"c`)},
		}, {
			expr: `sort("count()")`,
			want: &SortQuery{Expr: "count()", Query: &CountQuery{}},
		}, {
			expr: `rsort(has("a"))`,
			want: &RSortQuery{Expr: `has("a")`, Query: &HasQuery{Key: "a"}},
		}, {
			expr: `sort("(.foo)")`,
			want: &SortQuery{
				Expr:  "(.foo)",
				Query: MapQuery("foo"),
			},
		}, {
			expr: `["foo bar"]`,
			want: MapQuery("foo bar"),
		}, {
			expr: `[-1]`,
			want: ArrayQuery(-1),
		}, {
			expr: `[.x == 1.5]`,
			want: SelectQuery{
				And{
					Comparator{MapQuery("x"), EQ, ValueQuery{NumberValue(1.5)}},
				},
			},
		}, {
			expr: `[1.5 == .x]`,
			want: SelectQuery{
				And{
					Comparator{ValueQuery{NumberValue(1.5)}, EQ, MapQuery("x")},
				},
			},
		}, {
			expr: `.1.5`,
			want: FilterQuery{
				MapQuery("1"),
				MapQuery("5"),
			},
		}, {
			expr: `[.x == 1e-3]`,
			want: SelectQuery{
				And{
					Comparator{MapQuery("x"), EQ, ValueQuery{NumberValue(0.001)}},
				},
			},
		}, {
			expr: `[.x == 1e+3]`,
			want: SelectQuery{
				And{
					Comparator{MapQuery("x"), EQ, ValueQuery{NumberValue(1000)}},
				},
			},
		}, {
			expr: `[.x == 1.5e-3]`,
			want: SelectQuery{
				And{
					Comparator{MapQuery("x"), EQ, ValueQuery{NumberValue(0.0015)}},
				},
			},
		}, {
			expr: `[.x == -1]`,
			want: SelectQuery{
				And{
					Comparator{MapQuery("x"), EQ, ValueQuery{NumberValue(-1)}},
				},
			},
		}, {
			expr: `[.x == -1.5]`,
			want: SelectQuery{
				And{
					Comparator{MapQuery("x"), EQ, ValueQuery{NumberValue(-1.5)}},
				},
			},
		}, {
			expr: `[-1.5 == .x]`,
			want: SelectQuery{
				And{
					Comparator{ValueQuery{NumberValue(-1.5)}, EQ, MapQuery("x")},
				},
			},
		}, {
			expr: `[.x == -1e-3]`,
			want: SelectQuery{
				And{
					Comparator{MapQuery("x"), EQ, ValueQuery{NumberValue(-0.001)}},
				},
			},
		}, {
			expr: `[-3:5]`,
			want: ArrayRangeQuery{From: IntPtr(-3), To: IntPtr(5)},
		}, {
			expr: `[-3:]`,
			want: ArrayRangeQuery{From: IntPtr(-3), To: nil},
		}, {
			expr: `[1:-1]`,
			want: ArrayRangeQuery{From: IntPtr(1), To: IntPtr(-1)},
		}, {
			expr: `[-3:-1]`,
			want: ArrayRangeQuery{From: IntPtr(-3), To: IntPtr(-1)},
		}, {
			expr: `[:-1]`,
			want: ArrayRangeQuery{From: nil, To: IntPtr(-1)},
		}, {
			expr: `[.x == Inf]`,
			want: SelectQuery{
				And{
					Comparator{MapQuery("x"), EQ, ValueQuery{StringValue("Inf")}},
				},
			},
		}, {
			expr: `[.x == NaN]`,
			want: SelectQuery{
				And{
					Comparator{MapQuery("x"), EQ, ValueQuery{StringValue("NaN")}},
				},
			},
		}, {
			expr: `.orders`,
			want: MapQuery("orders"),
		}, {
			expr: `.android`,
			want: MapQuery("android"),
		}, {
			expr: `.origin.order_id`,
			want: FilterQuery{MapQuery("origin"), MapQuery("order_id")},
		}, {
			expr: `..orders`,
			want: WalkQuery("orders"),
		}, {
			expr: `..or`,
			want: WalkQuery("or"),
		}, {
			expr: `.or`,
			want: MapQuery("or"),
		}, {
			expr: `.a.and`,
			want: FilterQuery{MapQuery("a"), MapQuery("and")},
		}, {
			expr: `[.orders and .origin]`,
			want: SelectQuery{
				And{
					Evaluator{MapQuery("orders")},
					Evaluator{MapQuery("origin")},
				},
			},
		}, {
			expr: `[.or == 1 or .and == 2]`,
			want: SelectQuery{
				Or{
					Comparator{MapQuery("or"), EQ, ValueQuery{NumberValue(1)}},
					Comparator{MapQuery("and"), EQ, ValueQuery{NumberValue(2)}},
				},
			},
		}, {
			expr: `[. and .b]`,
			want: SelectQuery{
				And{
					Evaluator{NopQuery{}},
					Evaluator{MapQuery("b")},
				},
			},
		}, {
			expr: `[. or .x == 1]`,
			want: SelectQuery{
				Or{
					Evaluator{NopQuery{}},
					Comparator{MapQuery("x"), EQ, ValueQuery{NumberValue(1)}},
				},
			},
		}, {
			expr: `[.x == order]`,
			want: SelectQuery{
				And{
					Comparator{MapQuery("x"), EQ, ValueQuery{StringValue("order")}},
				},
			},
		}, {
			expr: `[.a == 1 or.b == 2]`,
			want: SelectQuery{
				Or{
					Comparator{MapQuery("a"), EQ, ValueQuery{NumberValue(1)}},
					Comparator{MapQuery("b"), EQ, ValueQuery{NumberValue(2)}},
				},
			},
		}, {
			expr: `[.x == null]`,
			want: SelectQuery{And{Comparator{MapQuery("x"), EQ, ValueQuery{Nil}}}},
		}, {
			expr: `[null == .x]`,
			want: SelectQuery{And{Comparator{ValueQuery{Nil}, EQ, MapQuery("x")}}},
		}, {
			expr: `[.x == "null"]`,
			want: SelectQuery{And{Comparator{MapQuery("x"), EQ, ValueQuery{StringValue("null")}}}},
		}, {
			expr: `[.x == nulls]`,
			want: SelectQuery{And{Comparator{MapQuery("x"), EQ, ValueQuery{StringValue("nulls")}}}},
		}, {
			expr: `.null`,
			want: MapQuery("null"),
		}, {
			expr: `..null`,
			want: WalkQuery("null"),
		}, {
			expr: `(.a | .[0].b).c`,
			want: FilterQuery{FilterQuery{MapQuery("a"), SlurpQuery{}, NopQuery{}, ArrayQuery(0), MapQuery("b")}, MapQuery("c")},
		}, {
			expr: `.x[](.a | .b)`,
			want: FilterQuery{MapQuery("x"), SelectQuery{}, FilterQuery{MapQuery("a"), SlurpQuery{}, MapQuery("b")}},
		}, {
			expr: `.((.a | .b) | .c).d`,
			want: FilterQuery{NopQuery{}, FilterQuery{FilterQuery{MapQuery("a"), SlurpQuery{}, MapQuery("b")}, SlurpQuery{}, MapQuery("c")}, MapQuery("d")},
		}, {
			expr:      `(.a .b).c`,
			want:      FilterQuery{FilterQuery{MapQuery("a"), MapQuery("b")}, MapQuery("c")},
			roundTrip: FilterQuery{MapQuery("a"), MapQuery("b"), MapQuery("c")},
		}, {
			expr: `."x"(.a | .b)`,
			want: FilterQuery{MapQuery("x"), FilterQuery{MapQuery("a"), SlurpQuery{}, MapQuery("b")}},
		}, {
			expr: `.x["y"](.a | .b)`,
			want: FilterQuery{MapQuery("x"), MapQuery("y"), FilterQuery{MapQuery("a"), SlurpQuery{}, MapQuery("b")}},
		}, {
			expr:      `(.a .b)(.c | .d)`,
			want:      FilterQuery{FilterQuery{MapQuery("a"), MapQuery("b")}, FilterQuery{MapQuery("c"), SlurpQuery{}, MapQuery("d")}},
			roundTrip: FilterQuery{MapQuery("a"), MapQuery("b"), FilterQuery{MapQuery("c"), SlurpQuery{}, MapQuery("d")}},
		}, {
			expr: `.x1(.a | .b)`,
			want: FilterQuery{MapQuery("x1"), FilterQuery{MapQuery("a"), SlurpQuery{}, MapQuery("b")}},
		}, {
			expr:      `(.a .).b`,
			want:      FilterQuery{FilterQuery{MapQuery("a"), NopQuery{}}, MapQuery("b")},
			roundTrip: FilterQuery{MapQuery("a"), MapQuery("b")},
		}, {
			expr: `.x count()`,
			want: FilterQuery{MapQuery("x"), &CountQuery{}},
		}, {
			expr: `.x .count()`,
			want: FilterQuery{MapQuery("x"), NopQuery{}, &CountQuery{}},
		}, {
			expr: `x`,
			want: ValueQuery{StringValue("x")},
		}, {
			expr:      `(x)`,
			want:      ValueQuery{StringValue("x")},
			roundTrip: ValueQuery{StringValue("x")},
		}, {
			expr: `.a .b`,
			want: FilterQuery{MapQuery("a"), MapQuery("b")},
		}, {
			expr: `.a [0]`,
			want: FilterQuery{MapQuery("a"), ArrayQuery(0)},
		}, {
			expr: `.a |`,
			want: FilterQuery{MapQuery("a"), SlurpQuery{}},
		}, {
			expr: `.a . count()`,
			want: FilterQuery{MapQuery("a"), NopQuery{}, &CountQuery{}},
		}, {
			expr: `[. and .b]`,
			want: SelectQuery{And{Evaluator{NopQuery{}}, Evaluator{MapQuery("b")}}},
		}, {
			expr: `.名前`,
			want: MapQuery("名前"),
		}, {
			expr: `..名前`,
			want: WalkQuery("名前"),
		}, {
			expr: `.日本foo.é_1`,
			want: FilterQuery{MapQuery("日本foo"), MapQuery("é_1")},
		}, {
			expr: `[.x == orégano]`,
			want: SelectQuery{Selector: And{Comparator{Left: MapQuery("x"), Op: EQ, Right: ValueQuery{StringValue("orégano")}}}},
		}, {
			expr: ".a\t|\r\n.b",
			want: FilterQuery{MapQuery("a"), SlurpQuery{}, MapQuery("b")},
		}, {
			expr: `.a."1e+3"`,
			want: FilterQuery{MapQuery("a"), MapQuery("1e+3")},
		}, {
			expr: `.a."1e-3"`,
			want: FilterQuery{MapQuery("a"), MapQuery("1e-3")},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.expr, func(t *testing.T) {
			got, err := ParseQuery(tc.expr)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("got %#v; want %#v", got, tc.want)
			}
			rt, err := ParseQuery(tc.want.String())
			if err != nil {
				t.Fatalf("re-parse %q: %v", tc.want.String(), err)
			}
			wantRT := tc.roundTrip
			if wantRT == nil {
				wantRT = tc.want
			}
			if !reflect.DeepEqual(rt, wantRT) {
				t.Errorf("round-trip %q -> %q: got %#v; want %#v", tc.expr, tc.want.String(), rt, wantRT)
			}
		})
	}
}

func TestParseQueryErrors(t *testing.T) {
	testCases := []struct {
		expr   string
		errstr string
	}{
		{
			expr:   `<`,
			errstr: `syntax error: invalid token <: "<"`,
		}, {
			expr:   ``,
			errstr: `syntax error: empty expression: ""`,
		}, {
			expr:   ` `,
			errstr: `syntax error: empty expression: " "`,
		}, {
			expr:   `()`,
			errstr: `syntax error: empty expression: "()"`,
		}, {
			expr:   `[.x == ]`,
			errstr: `syntax error: empty expression: "[.x == ]"`,
		}, {
			expr:   `[ == 1]`,
			errstr: `syntax error: empty expression: "[ == 1]"`,
		}, {
			expr:   `[.x and ]`,
			errstr: `syntax error: empty expression: "[.x and ]"`,
		}, {
			expr:   `.a | ()`,
			errstr: `syntax error: empty expression: ".a | ()"`,
		}, {
			expr:   `[()]`,
			errstr: `syntax error: empty expression: "[()]"`,
		}, {
			expr:   `(`,
			errstr: `syntax error: no right brackets: "("`,
		}, {
			expr:   `[.x == `,
			errstr: `syntax error: no right brackets: "[.x == "`,
		}, {
			expr:   `sort("")`,
			errstr: `syntax error: empty expression: ""`,
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
		}, {
			expr:   `.foo-bar`,
			errstr: `syntax error: invalid token -: ".foo-bar"`,
		}, {
			expr:   `[foo-bar]`,
			errstr: `syntax error: invalid token -: "[foo-bar]"`,
		}, {
			expr:   `has("a""b")`,
			errstr: `syntax error: unexpected character "\"" at offset 7: "has(\"a\"\"b\")"`,
		}, {
			expr:   `[has("abc]`,
			errstr: `syntax error: unterminated string at offset 5: "[has(\"abc]"`,
		}, {
			expr:   `has(it's)`,
			errstr: `syntax error: unterminated string at offset 6: "has(it's)"`,
		}, {
			expr:   `has("a" b)`,
			errstr: `syntax error: unexpected character "b" at offset 8: "has(\"a\" b)"`,
		}, {
			expr:   `has(,)`,
			errstr: `syntax error: empty argument at offset 4: "has(,)"`,
		}, {
			expr:   `has(a, )`,
			errstr: `syntax error: empty argument at offset 7: "has(a, )"`,
		}, {
			expr:   `has("a", "b")`,
			errstr: `has() requires exactly 1 argument, got 2`,
		}, {
			expr:   `.a.first(1)`,
			errstr: `first() requires no arguments, got 1`,
		}, {
			expr:   `.a.sort(.k, .x)`,
			errstr: `sort() requires at most 1 argument, got 2`,
		}, {
			expr:   `.a.sort(".b"`,
			errstr: `syntax error: no right brackets: ".a.sort(\".b\""`,
		}, {
			expr:   `sort(.a`,
			errstr: `syntax error: no right brackets: "sort(.a"`,
		}, {
			expr:   `has(a[b)`,
			errstr: `syntax error: unexpected character ")" at offset 7: "has(a[b)"`,
		}, {
			expr:   `has(a(b])`,
			errstr: `syntax error: unexpected character "]" at offset 7: "has(a(b])"`,
		}, {
			expr:   `has(a[b))`,
			errstr: `syntax error: unexpected character ")" at offset 7: "has(a[b))"`,
		}, {
			expr:   `has(a]b)`,
			errstr: `syntax error: unexpected character "]" at offset 5: "has(a]b)"`,
		}, {
			expr:   `sort(.a[0)`,
			errstr: `syntax error: unexpected character ")" at offset 9: "sort(.a[0)"`,
		}, {
			expr:   `[has(a]`,
			errstr: `syntax error: unexpected character "]" at offset 6: "[has(a]"`,
		}, {
			expr:   `sort(.a[0`,
			errstr: `syntax error: no right brackets: "sort(.a[0"`,
		}, {
			expr:   `.foo!!!`,
			errstr: `syntax error: unexpected character "!" at offset 4: ".foo!!!"`,
		}, {
			expr:   `.$foo`,
			errstr: `syntax error: unexpected character "$" at offset 1: ".$foo"`,
		}, {
			expr:   `.$or`,
			errstr: `syntax error: unexpected character "$" at offset 1: ".$or"`,
		}, {
			expr:   `.a*or`,
			errstr: `syntax error: unexpected character "*" at offset 2: ".a*or"`,
		}, {
			expr:   `.a+foo`,
			errstr: `syntax error: unexpected character "+" at offset 2: ".a+foo"`,
		}, {
			expr:   `.foo?`,
			errstr: `syntax error: unexpected character "?" at offset 4: ".foo?"`,
		}, {
			expr:   `.foo\bar`,
			errstr: `syntax error: unexpected character "\\" at offset 4: ".foo\\bar"`,
		}, {
			expr:   `[.foo = 1]`,
			errstr: `syntax error: unexpected character "=" at offset 6: "[.foo = 1]"`,
		}, {
			expr:   ".foo\u3000",
			errstr: `syntax error: unexpected character "\u3000" at offset 4: ".foo\u3000"`,
		}, {
			expr:   ".foo\xff",
			errstr: `syntax error: unexpected character "\xff" at offset 4: ".foo\xff"`,
		}, {
			expr:   `.名前!`,
			errstr: `syntax error: unexpected character "!" at offset 7: ".名前!"`,
		}, {
			expr:   `.a.1e+3`,
			errstr: `syntax error: unexpected character "+" at offset 5: ".a.1e+3"`,
		}, {
			expr:   `.a.1e-3`,
			errstr: `syntax error: invalid token -: ".a.1e-3"`,
		}, {
			expr:   `.1e-3`,
			errstr: `syntax error: invalid token -: ".1e-3"`,
		}, {
			expr:   `[.a == .1e-3]`,
			errstr: `syntax error: invalid token -: "[.a == .1e-3]"`,
		}, {
			expr:   `.foo "bar`,
			errstr: `syntax error: unterminated string at offset 5: ".foo \"bar"`,
		}, {
			expr:   `.a['b]`,
			errstr: `syntax error: unterminated string at offset 3: ".a['b]"`,
		}, {
			expr:   `1.or`,
			errstr: `syntax error: unexpected literal 1: "1.or"`,
		}, {
			expr:   `true (.a | .b)`,
			errstr: `syntax error: unexpected literal true: "true (.a | .b)"`,
		}, {
			expr:   `.x true`,
			errstr: `syntax error: unexpected literal true: ".x true"`,
		}, {
			expr:   `.x null`,
			errstr: `syntax error: unexpected literal null: ".x null"`,
		}, {
			expr:   `.x 1.5`,
			errstr: `syntax error: unexpected literal 1.5: ".x 1.5"`,
		}, {
			expr:   `1 1`,
			errstr: `syntax error: unexpected literal 1: "1 1"`,
		}, {
			expr:   `1.5.5`,
			errstr: `syntax error: unexpected literal 1.5: "1.5.5"`,
		}, {
			expr:   `.a.("x")`,
			errstr: `syntax error: unexpected literal "x": ".a.(\"x\")"`,
		}, {
			expr:   `.(true)`,
			errstr: `syntax error: unexpected literal true: ".(true)"`,
		}, {
			expr:   `.a x`,
			errstr: `syntax error: unexpected literal "x": ".a x"`,
		}, {
			expr:   `x.count()`,
			errstr: `syntax error: unexpected literal "x": "x.count()"`,
		}, {
			expr:   `.a | length`,
			errstr: `syntax error: unexpected literal "length": ".a | length"`,
		}, {
			expr:   `[.a == x.y]`,
			errstr: `syntax error: unexpected literal "x": "[.a == x.y]"`,
		}, {
			expr:   `[.host == example.com]`,
			errstr: `syntax error: unexpected literal "example": "[.host == example.com]"`,
		}, {
			expr:   `[.x == 1 or 2]`,
			errstr: `syntax error: unexpected literal 2: "[.x == 1 or 2]"`,
		}, {
			expr:   `[not .x]`,
			errstr: `syntax error: unexpected literal "not": "[not .x]"`,
		}, {
			expr:   `[(x)]`,
			errstr: `syntax error: unexpected literal "x": "[(x)]"`,
		}, {
			expr:   `[(1)]`,
			errstr: `syntax error: unexpected literal 1: "[(1)]"`,
		}, {
			expr:   `[(.x]`,
			errstr: `syntax error: no right brackets: "[(.x]"`,
		}, {
			expr:   `[(.x) ==]`,
			errstr: `syntax error: empty expression: "[(.x) ==]"`,
		}, {
			expr:   `[(.x == 1) == true]`,
			errstr: `syntax error: no right brackets: "[(.x == 1) == true]"`,
		}, {
			expr:   `[(.a or .b) == 1]`,
			errstr: `syntax error: no right brackets: "[(.a or .b) == 1]"`,
		}, {
			expr:   `[(-1)]`,
			errstr: `syntax error: invalid token -: "[(-1)]"`,
		}, {
			expr:   `[(-1) == .a]`,
			errstr: `syntax error: invalid token -: "[(-1) == .a]"`,
		}, {
			expr:   `[(.x`,
			errstr: `syntax error: no right brackets: "[(.x"`,
		}, {
			expr:   `[(1) .x == 1]`,
			errstr: `syntax error: unexpected literal 1: "[(1) .x == 1]"`,
		}, {
			expr:   `[(.a) x]`,
			errstr: `syntax error: unexpected literal "x": "[(.a) x]"`,
		}, {
			expr:   `[(.a)(.b == 1)]`,
			errstr: `syntax error: invalid token ==: "[(.a)(.b == 1)]"`,
		}, {
			expr:   `[(.a) - 1]`,
			errstr: `syntax error: invalid token -: "[(.a) - 1]"`,
		}, {
			expr:   `[.a == -1 .b]`,
			errstr: `syntax error: unexpected literal -1: "[.a == -1 .b]"`,
		}, {
			expr:   `[.a == -1 | .b]`,
			errstr: `syntax error: unexpected literal -1: "[.a == -1 | .b]"`,
		}, {
			expr:   `. a`,
			errstr: `syntax error: unexpected blank at offset 1: ". a"`,
		}, {
			expr:   `.a . b`,
			errstr: `syntax error: unexpected blank at offset 4: ".a . b"`,
		}, {
			expr:   `.. a`,
			errstr: `syntax error: unexpected blank at offset 2: ".. a"`,
		}, {
			expr:   ".a.\t\"b\"",
			errstr: `syntax error: unexpected blank at offset 3: ".a.\t\"b\""`,
		}, {
			expr:   `. 1.5`,
			errstr: `syntax error: unexpected blank at offset 1: ". 1.5"`,
		}, {
			expr:   `| .a`,
			errstr: `syntax error: empty expression: "| .a"`,
		}, {
			expr:   `|`,
			errstr: `syntax error: empty expression: "|"`,
		}, {
			expr:   `||`,
			errstr: `syntax error: empty expression: "||"`,
		}, {
			expr:   `.a || .b`,
			errstr: `syntax error: empty expression: ".a || .b"`,
		}, {
			expr:   `(| .a)`,
			errstr: `syntax error: empty expression: "(| .a)"`,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.expr, func(t *testing.T) {
			_, err := ParseQuery(tc.expr)
			if err == nil {
				t.Fatal("no error")
			}
			if err.Error() != tc.errstr {
				t.Errorf("got %s; want %s", err.Error(), tc.errstr)
			}
		})
	}
}

// FuzzParseQuery checks that the parser never panics and that a parsed query's String() parses back to the same String().
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
		"[.a == 1 or .b == 2]",
		".orders",
		".名前",
		".foo!!!",
	}
	for _, s := range seeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, expr string) {
		q, err := ParseQuery(expr)
		if err != nil {
			return
		}
		s := q.String()
		rt, err := ParseQuery(s)
		if err != nil {
			t.Fatalf("re-parse %q (from %q): %v", s, expr, err)
		}
		if got := rt.String(); got != s {
			t.Errorf("round-trip %q (from %q): got %q", s, expr, got)
		}
	})
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

func TestLexEdit(t *testing.T) {
	testCases := []struct {
		caseName string
		expr     string
		tokens   int // including tkEOF
		pos      int
		op       string
	}{
		{caseName: "edit stops at the operator", expr: `.a[.b == 1] += 2`, tokens: 9, pos: 12, op: "+="},
		{caseName: "edit skips method arguments and quotes", expr: `.a.has(x=1)."=" ^? `, tokens: 7, pos: 16, op: "^?"},
		{caseName: "edit reads == as a comparison, not an operator", expr: `.a ==1`, tokens: 5, pos: 6},
		{caseName: "edit without an operator", expr: `.a `, tokens: 3, pos: 3},
		{caseName: "edit operator after an unopened bracket", expr: `.a] = 1`, tokens: 4, pos: 4, op: "="},
		{caseName: "edit operator in a group after an unopened parenthesis", expr: `.a)(.b = 1`, tokens: 7, pos: 7, op: "="},
	}
	for _, tc := range testCases {
		t.Run(tc.caseName, func(t *testing.T) {
			l := &lexer{expr: tc.expr, edit: true}
			tokens, err := l.lex()
			if err != nil {
				t.Fatal(err)
			}
			if len(tokens) != tc.tokens || l.pos != tc.pos || l.op != tc.op {
				t.Errorf("got %d tokens, pos %d, op %q; want %d, %d, %q", len(tokens), l.pos, l.op, tc.tokens, tc.pos, tc.op)
			}
			want, err := (&lexer{expr: tc.expr[:l.pos]}).lex()
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(tokens, want) {
				t.Errorf("got tokens %v; want %v", tokens, want)
			}
		})
	}
}
