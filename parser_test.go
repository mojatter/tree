package tree

import (
	"reflect"
	"testing"
)

func TestParseQuery(t *testing.T) {
	testCases := []struct {
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
			expr:   `[-3:5]`,
			errstr: `syntax error: invalid array range: "[-3:5]"`,
		}, {
			expr:   `[-3:]`,
			errstr: `syntax error: invalid array range: "[-3:]"`,
		}, {
			expr:   `[-2:-1]`,
			errstr: `syntax error: invalid array range: "[-2:-1]"`,
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
