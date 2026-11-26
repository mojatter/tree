package tree

import (
	"bytes"
	"math"
	"regexp"
	"strings"
	"testing"
)

func TestOutputColorJSON(t *testing.T) {
	n := Map{
		"num":  ToValue(1),
		"str":  ToValue("2"),
		"bool": ToValue(true),
		"null": Nil,
	}
	want := "{\n  \x1b[1;34m\"bool\"\x1b[0m: true,\n  \x1b[1;34m\"null\"\x1b[0m: \x1b[0;90mnull\x1b[0m,\n  \x1b[1;34m\"num\"\x1b[0m: 1,\n  \x1b[1;34m\"str\"\x1b[0m: \x1b[0;32m\"2\"\x1b[0m\n}\n"

	out := new(bytes.Buffer)
	if err := OutputColorJSON(out, n); err != nil {
		t.Fatal(err)
	}
	if got := out.String(); got != want {
		t.Errorf("got %q; want %q\n%s", got, want, want)
	}
}

func TestEncodeJSON(t *testing.T) {
	testCases := []struct {
		caseName string
		e        *ColorEncoder
		n        Node
		want     string
	}{
		{
			caseName: "Map with Array, Nil and nil",
			e:        &ColorEncoder{IndentSize: 4, NoColor: true},
			n: Map{
				"a": ToValue(1),
				"b": Array{
					ToValue("2"),
					ToValue(true),
				},
				"c": Nil,
				"d": nil,
			},
			want: `{
    "a": 1,
    "b": [
        "2",
        true
    ],
    "c": null,
    "d": null
}
`,
		}, {
			caseName: "string with control char escapes",
			e:        &ColorEncoder{IndentSize: 2, NoColor: true},
			n:        ToValue("\"\n\r\t"),
			want:     "\"\\\"\\n\\r\\t\"\n",
		}, {
			caseName: "nil pointer nodes as null",
			e:        &ColorEncoder{IndentSize: 2, NoColor: true},
			n:        Map{"a": (*Map)(nil), "b": Array{(*Array)(nil)}},
			want: `{
  "a": null,
  "b": [
    null
  ]
}
`,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.caseName, func(t *testing.T) {
			out := new(bytes.Buffer)
			tc.e.Out = out
			if err := tc.e.EncodeJSON(tc.n); err != nil {
				t.Fatal(err)
			}
			if got := out.String(); got != tc.want {
				t.Errorf("got %q; want %q\n%s", got, tc.want, tc.want)
			}
		})
	}
}

var ansiPattern = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func TestOutputColorYAML(t *testing.T) {
	testCases := []struct {
		caseName string
		n        Node
		want     string
	}{
		{
			caseName: "keys, strings and nulls colored",
			n: Map{
				"num":  ToValue(1),
				"str":  ToValue("2"),
				"bool": ToValue(true),
				"null": Nil,
			},
			want: "\x1b[1;34mbool\x1b[0m: true\n\x1b[1;34m\"null\"\x1b[0m: \x1b[0;90mnull\x1b[0m\n\x1b[1;34mnum\x1b[0m: 1\n\x1b[1;34mstr\x1b[0m: \x1b[0;32m\"2\"\x1b[0m\n",
		}, {
			caseName: "explicit key and literal blocks",
			n:        Map{"a\nb": ToValue("x\ny")},
			want:     "? \x1b[1;34m|-\n  a\n  b\x1b[0m\n: \x1b[0;32m|-\n  x\n  y\x1b[0m\n",
		}, {
			caseName: "map value whose first key is explicit",
			n:        Map{"a": Map{"b\nc": ToValue(1), "d": ToValue(2)}},
			want:     "\x1b[1;34ma\x1b[0m:\n  ? \x1b[1;34m|-\n    b\n    c\x1b[0m\n  : 1\n  \x1b[1;34md\x1b[0m: 2\n",
		}, {
			caseName: "nested arrays with a kept trailing newline",
			n:        Array{Array{Array{ToValue("\n"), ToValue("a")}}},
			want:     "- - - \x1b[0;32m|2+\n\x1b[0m\n    - \x1b[0;32ma\x1b[0m\n",
		}, {
			caseName: "!!binary key and value",
			n:        Map{"\xff": StringValue("\xfe")},
			want:     "\x1b[1;34m!!binary /w==\x1b[0m: \x1b[0;32m!!binary /g==\x1b[0m\n",
		}, {
			caseName: "line separator double-quoted",
			n:        Map{"a": ToValue("b\nc\u2028")},
			want:     "\x1b[1;34ma\x1b[0m: \x1b[0;32m\"b\\nc\\L\"\x1b[0m\n",
		}, {
			caseName: "empty collections, bools and numbers uncolored",
			n:        Map{"a": Map{}, "b": Array{}, "c": ToValue(false), "d": ToValue(1.5)},
			want:     "\x1b[1;34ma\x1b[0m: {}\n\x1b[1;34mb\x1b[0m: []\n\x1b[1;34mc\x1b[0m: false\n\x1b[1;34md\x1b[0m: 1.5\n",
		}, {
			caseName: "multi-line key starting with a tab double-quoted",
			n:        Map{"\tb\nc": ToValue(1)},
			want:     "? \x1b[1;34m\"\\tb\\nc\"\x1b[0m\n: 1\n",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.caseName, func(t *testing.T) {
			out := new(bytes.Buffer)
			if err := OutputColorYAML(out, tc.n); err != nil {
				t.Fatal(err)
			}
			if got := out.String(); got != tc.want {
				t.Errorf("got %q; want %q\n%s", got, tc.want, tc.want)
			}
			plain, err := MarshalYAML(tc.n)
			if err != nil {
				t.Fatal(err)
			}
			if got := ansiPattern.ReplaceAllString(out.String(), ""); got != string(plain) {
				t.Errorf("without color %q; want MarshalYAML %q", got, plain)
			}
		})
	}
}

func TestEncodeYAML(t *testing.T) {
	testCases := []struct {
		caseName string
		e        *ColorEncoder
		n        Node
		want     string
	}{
		{
			caseName: "Map with Array, Nil and nil",
			e:        &ColorEncoder{IndentSize: 2, NoColor: true},
			n: Map{
				"a": ToValue(1),
				"b": Array{
					ToValue("2"),
					ToValue(true),
				},
				"c": Nil,
				"d": nil,
			},
			want: `a: 1
b:
  - "2"
  - true
c: null
d: null
`,
		}, {
			caseName: "invalid UTF-8 as !!binary",
			e:        &ColorEncoder{IndentSize: 2, NoColor: true},
			n: Map{
				"\xff": StringValue("a\xfe\nb"),
				"b":    Array{StringValue("\xff")},
			},
			want: `!!binary /w==: !!binary Yf4KYg==
b:
  - !!binary /w==
`,
		}, {
			caseName: "trailing newline triggers literal block",
			e:        &ColorEncoder{IndentSize: 2, NoColor: true},
			n: Map{
				"a": ToValue("line1\nline2\n"),
			},
			want: `a: |
  line1
  line2
`,
		}, {
			caseName: "no trailing newline uses literal strip",
			e:        &ColorEncoder{IndentSize: 2, NoColor: true},
			n: Map{
				"a": ToValue("line1\nline2"),
			},
			want: `a: |-
  line1
  line2
`,
		}, {
			caseName: "Array of mixed types",
			e:        &ColorEncoder{IndentSize: 2, NoColor: true},
			n: Array{
				ToValue(1),
				Map{
					"a": ToValue(2),
					"b": ToValue(true),
				},
				Array{
					ToValue("c"),
					Nil,
				},
			},
			want: `- 1
- a: 2
  b: true
- - c
  - null
`,
		}, {
			caseName: "nil pointer nodes as null",
			e:        &ColorEncoder{IndentSize: 2, NoColor: true},
			n:        Map{"a": (*Map)(nil), "b": Array{(*Array)(nil)}},
			want: `a: null
b:
  - null
`,
		}, {
			caseName: "multi-line keys starting with a tab double-quoted in key order",
			e:        &ColorEncoder{IndentSize: 2, NoColor: true},
			n:        Map{"a10": ToValue(1), "a9": ToValue(2), "\tb\nc": ToValue("\td\ne"), "x": Array{Map{"\t\n": Nil}}},
			want:     "? \"\\tb\\nc\"\n: \"\\td\\ne\"\na9: 2\na10: 1\nx:\n  - ? \"\\t\\n\"\n    : null\n",
		}, {
			caseName: "Go nil pointer children",
			e:        &ColorEncoder{IndentSize: 2, NoColor: true},
			n:        Map{"a": (*NumberValue)(nil), "b": Array{(*StringValue)(nil), (*Any)(nil)}},
			want:     "a: null\nb:\n  - null\n  - null\n",
		}, {
			caseName: "reserved scalars quoted",
			e:        &ColorEncoder{IndentSize: 2, NoColor: true},
			n:        Map{"a": ToValue("True"), "b": ToValue("null"), "c": ToValue("yes")},
			want:     "a: \"True\"\nb: \"null\"\nc: \"yes\"\n",
		}, {
			caseName: "keys differing only in case",
			e:        &ColorEncoder{IndentSize: 2, NoColor: true},
			n:        Map{"true": ToValue(true), "True": ToValue(false)},
			want:     "\"True\": false\n\"true\": true\n",
		}, {
			caseName: "first line starting with spaces",
			e:        &ColorEncoder{IndentSize: 2, NoColor: true},
			n:        Map{"a": ToValue("  a\nb")},
			want:     "a: |2-\n    a\n  b\n",
		}, {
			caseName: "trailing newlines kept",
			e:        &ColorEncoder{IndentSize: 2, NoColor: true},
			n:        Map{"a": ToValue("a\n\n"), "b": ToValue("\n")},
			want:     "a: |+\n  a\n\nb: |2+\n\n",
		}, {
			caseName: "multi-line strings YAML cannot hold in a block",
			e:        &ColorEncoder{IndentSize: 2, NoColor: true},
			n:        Map{"a": ToValue("x\x01\ny"), "b": ToValue("a\r\nb"), "c": ToValue("a\u0085b\nc"), "d": ToValue(" \nb")},
			want:     "a: \"x\\x01\\ny\"\nb: \"a\\r\\nb\"\nc: \"a\\Nb\\nc\"\nd: \" \\nb\"\n",
		}, {
			caseName: "key longer than 1024 characters",
			e:        &ColorEncoder{IndentSize: 2, NoColor: true},
			n:        Map{strings.Repeat("a", 1100): ToValue(1)},
			want:     "? " + strings.Repeat("a", 1100) + "\n: 1\n",
		}, {
			caseName: "NaN and infinities",
			e:        &ColorEncoder{IndentSize: 2, NoColor: true},
			n:        Map{"a": ToValue(math.NaN()), "b": ToValue(math.Inf(1)), "c": ToValue(math.Inf(-1))},
			want:     "a: .nan\nb: .inf\nc: -.inf\n",
		}, {
			caseName: "numbers as JSON output writes them",
			e:        &ColorEncoder{IndentSize: 2, NoColor: true},
			n:        Map{"a": ToValue(1000000), "b": ToValue(0.00001), "c": ToValue(1e-7), "d": ToValue(1e21)},
			want:     "a: 1000000\nb: 0.00001\nc: 1e-7\nd: 1e+21\n",
		}, {
			caseName: "multi-line strings starting with a tab double-quoted",
			e:        &ColorEncoder{IndentSize: 2, NoColor: true},
			n:        Map{"a": ToValue("\tb\nc"), "b": Array{ToValue("\t\n")}, "c": StringValue("\t\xff\nb")},
			want:     "a: \"\\tb\\nc\"\nb:\n  - \"\\t\\n\"\nc: !!binary Cf8KYg==\n",
		}, {
			caseName: "empty maps and arrays",
			e:        &ColorEncoder{IndentSize: 2, NoColor: true},
			n:        Map{"a": Map{}, "b": Array{}, "c": Array{Array{ToValue(1), Array{}}, Map{}}, "d": ToValue(1)},
			want:     "a: {}\nb: []\nc:\n  - - 1\n    - []\n  - {}\nd: 1\n",
		}, {
			caseName: "negative zero",
			e:        &ColorEncoder{IndentSize: 2, NoColor: true},
			n:        Array{NumberValue(math.Copysign(0, -1))},
			want:     "- -0.0\n",
		}, {
			caseName: "line and paragraph separators double-quoted",
			e:        &ColorEncoder{IndentSize: 2, NoColor: true},
			n:        Map{"a": ToValue("b\nc\u2028"), "b": ToValue("x\u2029y"), "c\u2028": Array{ToValue("\u2028"), StringValue("\xff\u2028")}},
			want:     "a: \"b\\nc\\L\"\nb: \"x\\Py\"\n? \"c\\L\"\n: - \"\\L\"\n  - !!binary /+KAqA==\n",
		}, {
			caseName: "empty root map",
			e:        &ColorEncoder{IndentSize: 2, NoColor: true},
			n:        Map{},
			want:     "{}\n",
		}, {
			caseName: "indent size 4 written as 2",
			e:        &ColorEncoder{IndentSize: 4, NoColor: true},
			n:        Map{"a": Map{"b": Array{ToValue(1)}}},
			want:     "a:\n  b:\n    - 1\n",
		}, {
			caseName: "indent size 4, sequence item starting with a newline",
			e:        &ColorEncoder{IndentSize: 4, NoColor: true},
			n:        Array{StringValue("x"), StringValue("\n#'")},
			want:     "- x\n- |2-\n\n  #'\n",
		}, {
			caseName: "indent size 4, sequence item starting with a space",
			e:        &ColorEncoder{IndentSize: 4, NoColor: true},
			n:        Map{"a": Array{StringValue(" x\ny")}},
			want:     "a:\n  - |2-\n     x\n    y\n",
		}, {
			caseName: "indent size 0 written as 2",
			e:        &ColorEncoder{IndentSize: 0, NoColor: true},
			n:        Map{"a": Map{"b": Array{ToValue(1)}}},
			want:     "a:\n  b:\n    - 1\n",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.caseName, func(t *testing.T) {
			out := new(bytes.Buffer)
			tc.e.Out = out
			if err := tc.e.EncodeYAML(tc.n); err != nil {
				t.Fatal(err)
			}
			if got := out.String(); got != tc.want {
				t.Errorf("got %q; want %q\n%s", got, tc.want, tc.want)
			}
			// The output reads back to a node that writes the same output.
			n, err := UnmarshalYAML(out.Bytes())
			if err != nil {
				t.Fatal(err)
			}
			again, err := MarshalYAML(n)
			if err != nil {
				t.Fatal(err)
			}
			if string(again) != out.String() {
				t.Errorf("read back writes %q; want %q", again, out.String())
			}
		})
	}
}

func TestColorYAML(t *testing.T) {
	testCases := []struct {
		caseName string
		data     string
		want     string
	}{
		{caseName: "unreadable uncolored", data: "a: [\n", want: "a: [\n"},
		{caseName: "flow sequence before another key uncolored", data: "a: [x, y]\nb: z\n", want: "a: [x, y]\nb: z\n"},
		{caseName: "flow sequence at the end uncolored", data: "a: [x, y]\n", want: "a: [x, y]\n"},
		{caseName: "flow mapping uncolored", data: "a: {b: c}\n", want: "a: {b: c}\n"},
		{caseName: "flow root uncolored", data: "{a: b}\n", want: "{a: b}\n"},
		{caseName: "literal block ending in a line separator", data: "a: |\n  b\n  c\u2028", want: "\x1b[1;34ma\x1b[0m: \x1b[0;32m|\n  b\n  c\x1b[0m\u2028"},
		{caseName: "no final line break", data: "a: b", want: "\x1b[1;34ma\x1b[0m: \x1b[0;32mb\x1b[0m"},
	}
	for _, tc := range testCases {
		t.Run(tc.caseName, func(t *testing.T) {
			if got := colorYAML([]byte(tc.data)); string(got) != tc.want {
				t.Errorf("got %q; want %q", got, tc.want)
			}
		})
	}
}

// TestColorEncoderWriteQuotedJSONEscapes exercises the escape branches
// of writeQuotedJSON that were previously uncovered: control characters
// written via \u00xx, U+2028 / U+2029 line separators, and invalid
// UTF-8 replacement.
func TestColorEncoderWriteQuotedJSONEscapes(t *testing.T) {
	testCases := []struct {
		caseName string
		in       string
		want     string
	}{
		{
			caseName: "control char uses u00xx",
			in:       "a\x01b",
			want:     `"a\u0001b"`,
		},
		{
			caseName: "backspace and form feed",
			in:       "\b\f",
			want:     `"\b\f"`,
		},
		{
			caseName: "line separator U+2028",
			in:       "a\u2028b",
			want:     `"a\u2028b"`,
		},
		{
			caseName: "paragraph separator U+2029",
			in:       "a\u2029b",
			want:     `"a\u2029b"`,
		},
		{
			caseName: "invalid utf8 becomes fffd",
			in:       "a\xffb",
			want:     `"a\ufffdb"`,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.caseName, func(t *testing.T) {
			out := new(bytes.Buffer)
			e := &ColorEncoder{Out: out, IndentSize: 2, NoColor: true}
			e.writeQuotedJSON(tc.in)
			if err := e.err; err != nil {
				t.Fatalf("writeQuotedJSON err: %v", err)
			}
			if got := out.String(); got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

// failingWriter returns an error on Write so that we can exercise the
// early-return path in (*ColorEncoder).write when e.err is already set.
type failingWriter struct{}

func (failingWriter) Write(p []byte) (int, error) {
	return 0, errWrite
}

var errWrite = &writeErr{}

type writeErr struct{}

func (*writeErr) Error() string { return "write failed" }

// TestColorEncoderWriteErrorShortCircuits verifies that once the
// encoder records a write error it stops issuing further writes, and
// that EncodeJSON surfaces the recorded error.
func TestColorEncoderWriteErrorShortCircuits(t *testing.T) {
	e := &ColorEncoder{Out: failingWriter{}, IndentSize: 2, NoColor: true}
	err := e.EncodeJSON(ToValue("hello"))
	if err == nil {
		t.Fatal("expected error from failing writer")
	}
	if !strings.Contains(err.Error(), "write failed") {
		t.Errorf("unexpected error: %v", err)
	}
}
