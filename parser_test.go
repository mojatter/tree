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
	for _, tc := range testCases {
		t.Run(tc.expr, func(t *testing.T) {
			got, err := ParseQuery(tc.expr)
			if err != nil {
				t.Fatalf("got err %s", err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("got %#v; want %#v", got, tc.want)
			}
		})
	}
}

func TestParseQuery_Errors(t *testing.T) {
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
