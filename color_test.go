package tree

import (
	"bytes"
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
	want := "{\n  \x1b[1;34m\"bool\"\x1b[0m: true,\n  \x1b[1;34m\"null\"\x1b[0m: \x1b[1;30mnull\x1b[0m,\n  \x1b[1;34m\"num\"\x1b[0m: 1,\n  \x1b[1;34m\"str\"\x1b[0m: \x1b[0;32m\"2\"\x1b[0m\n}\n"

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

func TestOutputColorYAML(t *testing.T) {
	n := Map{
		"num":  ToValue(1),
		"str":  ToValue("2"),
		"bool": ToValue(true),
		"null": Nil,
	}
	want := "\x1b[1;34mbool\x1b[0m: true\n\x1b[1;34mnull\x1b[0m: \x1b[1;30mnull\x1b[0m\n\x1b[1;34mnum\x1b[0m: 1\n\x1b[1;34mstr\x1b[0m: \x1b[0;32m\"2\"\x1b[0m\n"

	out := new(bytes.Buffer)
	if err := OutputColorYAML(out, n); err != nil {
		t.Fatal(err)
	}
	if got := out.String(); got != want {
		t.Errorf("got %q; want %q\n%s", got, want, want)
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
-
  - c
  - null
`,
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
			want:     `"\u0008\u000c"`,
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
