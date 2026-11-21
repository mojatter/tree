package tree

import (
	"bytes"
	"errors"
	"io"
	"reflect"
	"strings"
	"testing"
	"time"

	"go.yaml.in/yaml/v3"
)

func TestMarshalYAML(t *testing.T) {
	testCases := []struct {
		caseName string
		n        Node
		want     string
	}{
		{
			caseName: "Map with Array containing Nil values",
			n: Map{
				"a": Array{
					StringValue("1"),
					NumberValue(2),
					BoolValue(true),
					Nil,
					nil,
				},
			},
			want: `a:
  - "1"
  - 2
  - true
  - null
  - null
`,
		},
		{
			caseName: "Map with Array",
			n: Map{
				"a": Array{
					StringValue("1"),
					NumberValue(2),
					BoolValue(true),
				},
			},
			want: `a:
  - "1"
  - 2
  - true
`,
		},
		{
			caseName: "Array",
			n: Array{
				StringValue("1"),
				NumberValue(2),
				BoolValue(true),
			},
			want: `- "1"
- 2
- true
`,
		},
		{caseName: "Any wrapping Map", n: Any{Node: Map{"k": NumberValue(1)}}, want: "k: 1\n"},
		{caseName: "pointer Any wrapping String", n: &Any{Node: StringValue("x")}, want: "x\n"},
		{caseName: "nested Any", n: Any{Node: &Any{Node: BoolValue(true)}}, want: "true\n"},
		{caseName: "zero Any", n: Any{}, want: "null\n"},
		{caseName: "Any wrapping nil pointer", n: Any{Node: (*Array)(nil)}, want: "null\n"},
		{caseName: "nil pointer Any", n: (*Any)(nil), want: "null\n"},
		{caseName: "Map with Any values", n: Map{"a": Any{Node: StringValue("x")}, "b": Any{}}, want: "a: x\nb: null\n"},
	}
	for _, tc := range testCases {
		t.Run(tc.caseName, func(t *testing.T) {
			got, err := MarshalYAML(tc.n)
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != tc.want {
				t.Errorf("got %s; want %s", string(got), tc.want)
			}
		})
	}
}

func TestDecodeYAMLErrors(t *testing.T) {
	testCases := []struct {
		caseName string
		data     []byte
		errstr   string
	}{
		{
			caseName: "unexpected end of stream",
			data:     []byte(`"`),
			errstr:   "yaml: found unexpected end of stream",
		}, {
			caseName: "unexpected node content",
			data:     []byte(`}`),
			errstr:   "yaml: did not find expected node content",
		}, {
			caseName: "missing comma or brace",
			data:     []byte("{\n1"),
			errstr:   `yaml: line 2: did not find expected ',' or '}'`,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.caseName, func(t *testing.T) {
			dec := yaml.NewDecoder(bytes.NewReader(tc.data))
			_, err := DecodeYAML(dec)
			if err == nil {
				t.Fatal("no error")
			}
			if err.Error() != tc.errstr {
				t.Errorf("got %s; want %s", err.Error(), tc.errstr)
			}
		})
	}
}

func TestDecodeYAML(t *testing.T) {
	testCases := []struct {
		caseName string
		data     string
		want     []Node
	}{
		{
			caseName: "documents with a timestamp, a null key and an empty document",
			data:     "---\na: 2001-12-14\n---\n~: b\n---\n---\n- 1\n",
			want:     []Node{Map{"a": StringValue("2001-12-14")}, Map{"null": StringValue("b")}, Nil, Array{NumberValue(1)}},
		},
		{caseName: "trailing empty document", data: "a: 1\n---\n", want: []Node{Map{"a": NumberValue(1)}, Nil}},
		{caseName: "comment only", data: "# only comment\n"},
	}
	for _, tc := range testCases {
		t.Run(tc.caseName, func(t *testing.T) {
			dec := yaml.NewDecoder(strings.NewReader(tc.data))
			var got []Node
			for {
				n, err := DecodeYAML(dec)
				if errors.Is(err, io.EOF) {
					break
				}
				if err != nil {
					t.Fatal(err)
				}
				got = append(got, n)
			}
			if len(got) != len(tc.want) {
				t.Fatalf("got %#v; want %#v", got, tc.want)
			}
			for i := range got {
				if !Equal(got[i], tc.want[i]) {
					t.Errorf("document %d: got %#v; want %#v", i, got[i], tc.want[i])
				}
			}
		})
	}
}

func TestUnmarshalYAML(t *testing.T) {
	testCases := []struct {
		caseName string
		data     []byte
		want     Node
		errstr   string
	}{
		{
			caseName: "Map with nested Array and Map",
			data: []byte(`a: 1
b: true
c: null
d: ["1",2,true]
e: {"x":"x"}
`),
			want: Map{
				"a": NumberValue(1),
				"b": BoolValue(true),
				"c": Nil,
				"d": Array{
					StringValue("1"),
					NumberValue(2),
					BoolValue(true),
				},
				"e": Map{
					"x": StringValue("x"),
				},
			},
		}, {
			caseName: "Array with nested Map and Array",
			data: []byte(`- "1"
- 2
- true
- null
- {"a":1,"b":true,"c":null}
- ["x"]
`),
			want: Array{
				StringValue("1"),
				NumberValue(2),
				BoolValue(true),
				Nil,
				Map{
					"a": NumberValue(1),
					"b": BoolValue(true),
					"c": Nil,
				},
				Array{
					StringValue("x"),
				},
			},
		},
		{caseName: "date as its source text", data: []byte("d: 2001-12-14\n"), want: Map{"d": StringValue("2001-12-14")}},
		{caseName: "timestamp as its source text", data: []byte("t: 2001-12-14T21:59:43.10-05:00\n"), want: Map{"t": StringValue("2001-12-14T21:59:43.10-05:00")}},
		{caseName: "tagged timestamp as its source text", data: []byte("t: !!timestamp 2001-12-14\n"), want: Map{"t": StringValue("2001-12-14")}},
		{caseName: "timestamp key as its source text", data: []byte("2001-12-14: x\n"), want: Map{"2001-12-14": StringValue("x")}},
		{caseName: "timestamp alias as its source text", data: []byte("a: &x 2001-12-14\nb: *x\n"), want: Map{"a": StringValue("2001-12-14"), "b": StringValue("2001-12-14")}},
		{caseName: "null key as null", data: []byte("null: n\n"), want: Map{"null": StringValue("n")}},
		{caseName: "tilde key as null", data: []byte("~: n\n"), want: Map{"null": StringValue("n")}},
		{caseName: "null alias key as null", data: []byte("a: &x null\n*x : n\n"), want: Map{"a": Nil, "null": StringValue("n")}},
		{caseName: "empty document", data: []byte(""), want: Nil},
		{caseName: "tilde and null keys are duplicates", data: []byte("~: a\nnull: b\n"), errstr: `line 2: mapping key "null" already defined at line 1`},
		{caseName: "null and quoted null keys are duplicates", data: []byte("~: a\n\"null\": b\n"), errstr: `line 2: mapping key "null" already defined at line 1`},
		{caseName: "tilde and quoted tilde keys are distinct", data: []byte("~: a\n\"~\": b\n"), want: Map{"null": StringValue("a"), "~": StringValue("b")}},
		{caseName: "empty and quoted empty keys are distinct", data: []byte("? ''\n: a\n? \n: b\n"), want: Map{"": StringValue("a"), "null": StringValue("b")}},
		{caseName: "float key as its source text", data: []byte("1.0: a\n"), want: Map{"1": StringValue("a")}},
		{caseName: "hex key as its source text", data: []byte("0x10: a\n"), want: Map{"16": StringValue("a")}},
		{caseName: "nan key keeps its value", data: []byte(".nan: c\n"), want: Map{"NaN": StringValue("c")}},
		{caseName: "float and string keys are duplicates", data: []byte("1.0: a\n\"1\": b\n"), errstr: `mapping key "1" already defined at line 1`},
		{caseName: "hex and decimal keys are duplicates", data: []byte("0x10: a\n16: b\n"), errstr: `mapping key "16" already defined at line 1`},
		{caseName: "int and float keys are duplicates", data: []byte("1: a\n1.0: b\n"), errstr: `mapping key "1" already defined at line 1`},
		{caseName: "explicit key wins over a merged key", data: []byte("1: a\n<<: {\"1\": b}\n"), want: Map{"1": StringValue("a")}},
		{caseName: "bool keys differing in case are duplicates", data: []byte("true: a\nTrue: b\n"), errstr: `mapping key "true" already defined at line 1`},
		{caseName: "negative zero key stays distinct", data: []byte("0.0: a\n-0.0: b\n"), want: Map{"0": StringValue("a"), "-0": StringValue("b")}},
		{caseName: "nan keys differing in case are duplicates", data: []byte(".nan: a\n.NaN: b\n"), errstr: `mapping key "NaN" already defined at line 1`},
		{caseName: "string alias key and string key are duplicates", data: []byte("&k foo: 1\n*k : 2\n"), errstr: `line 2: mapping key "foo" already defined at line 1`},
		{caseName: "timestamp alias key and string key are duplicates", data: []byte("a: &k 2001-01-01\nb: {\"2001-01-01\": x, *k : y}\n"), errstr: `mapping key "2001-01-01" already defined at line 2`},
		{caseName: "int alias key and string key are duplicates", data: []byte("a: &k 1\nb: {\"1\": x, *k : y}\n"), errstr: `mapping key "1" already defined at line 2`},
		{caseName: "int alias keys to a reused anchor are distinct", data: []byte("a: &x 1\n*x : 1\nb: &x 2\n*x : 2\n"), want: Map{"a": NumberValue(1), "1": NumberValue(1), "b": NumberValue(2), "2": NumberValue(2)}},
		{caseName: "string alias keys to a reused anchor are distinct", data: []byte("a: &x foo\n*x : 1\nb: &x bar\n*x : 2\n"), want: Map{"a": StringValue("foo"), "foo": NumberValue(1), "b": StringValue("bar"), "bar": NumberValue(2)}},
		{caseName: "timestamp alias keys to a reused anchor are distinct", data: []byte("a: &x 2001-01-01\n*x : 1\nb: &x 2002-02-02\n*x : 2\n"), want: Map{"a": StringValue("2001-01-01"), "2001-01-01": NumberValue(1), "b": StringValue("2002-02-02"), "2002-02-02": NumberValue(2)}},
		{caseName: "merge-text alias key and string key are duplicates", data: []byte("a: &k <<\nb: {*k : 1, \"<<\": 2}\n"), errstr: `mapping key "<<" already defined at line 2`},
		{caseName: "merge-text alias key is a plain key", data: []byte("a: &k <<\nb: {*k : 1}\n"), want: Map{"a": StringValue("<<"), "b": Map{"<<": NumberValue(1)}}},
		{caseName: "merge-text alias key and merge key are duplicates", data: []byte("a: &k <<\nb: {<<: {x: 1}, *k : 2}\n"), errstr: `mapping key "<<" already defined at line 2`},
		{caseName: "alias key to an anchored merge key and the merge key are duplicates", data: []byte("&k <<: {x: 1}\n*k : 2\n"), errstr: `line 2: mapping key "<<" already defined at line 1`},
		{caseName: "malformed tagged timestamp", data: []byte("t: !!timestamp foo\n"), errstr: "cannot decode !!str `foo` as a !!timestamp"},
		{caseName: "malformed tagged timestamp key", data: []byte("!!timestamp foo: x\n"), errstr: "cannot decode !!str `foo` as a !!timestamp"},
		{caseName: "malformed tagged null key", data: []byte("? !!null x\n: a\n"), errstr: "cannot decode !!str `x` as a !!null"},
		{caseName: "earlier error before malformed timestamp", data: []byte("a: !!int x\nb: !!timestamp foo\n"), errstr: "cannot decode !!str `x` as a !!int"},
		{caseName: "duplicate key before malformed timestamp", data: []byte("a: 1\na: 2\nb: !!timestamp foo\n"), errstr: `mapping key "a" already defined at line 1`},
		{caseName: "earlier error before malformed null key", data: []byte("a: !!int x\n? !!null x\n: y\n"), errstr: "cannot decode !!str `x` as a !!int"},
		{caseName: "duplicate key before malformed null key", data: []byte("a: 1\na: 2\n? !!null x\n: y\n"), errstr: `mapping key "a" already defined at line 1`},
	}
	for _, tc := range testCases {
		t.Run(tc.caseName, func(t *testing.T) {
			got, err := UnmarshalYAML(tc.data)
			if tc.errstr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.errstr) {
					t.Errorf("got %v; want error containing %q", err, tc.errstr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if !Equal(got, tc.want) {
				t.Errorf("got %#v; want %#v", got, tc.want)
			}
		})
	}
}

func TestMapUnmarshalYAML(t *testing.T) {
	want := Map{
		"a":    NumberValue(1),
		"b":    BoolValue(true),
		"c":    Nil,
		"d":    StringValue("2001-12-14"),
		"null": StringValue("n"),
	}
	data := []byte(`a: 1
b: true
c: null
d: 2001-12-14
~: n
`)
	var got Map
	if err := yaml.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if !Equal(got, want) {
		t.Errorf("got %#v; want %#v", got, want)
	}
}

func TestArrayUnmarshalYAML(t *testing.T) {
	want := Array{
		StringValue("1"),
		NumberValue(2),
		BoolValue(true),
		StringValue("2001-12-14"),
		StringValue("2001-12-15"),
	}
	data := []byte(`- "1"
- 2
- true
- 2001-12-14
- !!timestamp 2001-12-15
`)
	got := Array{}
	if err := yaml.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if !Equal(got, want) {
		t.Errorf("got %#v; want %#v", got, want)
	}
}

func TestAnyUnmarshalYAML(t *testing.T) {
	testCases := []struct {
		caseName string
		data     string
		want     Node
		errstr   string
	}{
		{caseName: "map", data: "k: 1\n", want: Map{"k": NumberValue(1)}},
		{caseName: "array with null", data: "- 1\n- null\n", want: Array{NumberValue(1), Nil}},
		{caseName: "null leaves the node as it is", data: "null\n", want: NumberValue(1)},
		{caseName: "timestamp and null key", data: "null: 2001-12-14\n", want: Map{"null": StringValue("2001-12-14")}},
		{caseName: "malformed tagged timestamp leaves the node as it is", data: "!!timestamp foo\n", want: NumberValue(1), errstr: "cannot decode !!str `foo` as a !!timestamp"},
		{caseName: "error leaves the node as it is", data: "!!int x\n", want: NumberValue(1), errstr: "cannot decode !!str `x` as a !!int"},
	}
	for _, tc := range testCases {
		t.Run(tc.caseName, func(t *testing.T) {
			a := Any{Node: NumberValue(1)}
			err := yaml.Unmarshal([]byte(tc.data), &a)
			if tc.errstr == "" && err != nil {
				t.Fatal(err)
			}
			if tc.errstr != "" && (err == nil || !strings.Contains(err.Error(), tc.errstr)) {
				t.Errorf("got %v; want error containing %q", err, tc.errstr)
			}
			if !reflect.DeepEqual(a.Node, tc.want) {
				t.Errorf("got %#v; want %#v", a.Node, tc.want)
			}
		})
	}
}

func TestUnmarshalYAMLKeepsNode(t *testing.T) {
	testCases := []struct {
		caseName string
		data     string
	}{
		{caseName: "timestamp, alias and null key", data: "a: &x 2001-12-14\nb: *x\n~: n\n"},
		{caseName: "tagged timestamp and nested null key", data: "- !!timestamp 2001-12-14\n- {~: 1}\n"},
		{caseName: "malformed tagged timestamp", data: "t: !!timestamp foo\n"},
	}
	for _, tc := range testCases {
		t.Run(tc.caseName, func(t *testing.T) {
			var yn, want yaml.Node
			if err := yaml.Unmarshal([]byte(tc.data), &yn); err != nil {
				t.Fatal(err)
			}
			if err := yaml.Unmarshal([]byte(tc.data), &want); err != nil {
				t.Fatal(err)
			}
			// Errors are ignored: only the node is checked.
			_ = (&Map{}).UnmarshalYAML(&yn)
			_ = (&Array{}).UnmarshalYAML(&yn)
			_ = (&Any{}).UnmarshalYAML(&yn)
			if !reflect.DeepEqual(&yn, &want) {
				t.Errorf("got %#v; want %#v", yn, want)
			}
		})
	}
}

func TestMarshalViaYAML(t *testing.T) {
	testCases := []struct {
		caseName string
		v        any
		want     Node
	}{
		{
			caseName: "struct",
			v: struct {
				ID     int      `yaml:"id"`
				Name   string   `yaml:"name"`
				Colors []string `yaml:"colors"`
			}{
				ID:     1,
				Name:   "Reds",
				Colors: []string{"Crimson", "Red", "Ruby", "Maroon"},
			},
			want: Map{
				"id":     ToValue(1),
				"name":   ToValue("Reds"),
				"colors": ToArrayValues("Crimson", "Red", "Ruby", "Maroon"),
			},
		},
		{caseName: "string", v: "str", want: StringValue("str")},
		{caseName: "bool", v: true, want: BoolValue(true)},
		{caseName: "int", v: 1, want: NumberValue(1)},
		{caseName: "nil", v: nil, want: Nil},
		{caseName: "BoolValue passthrough", v: BoolValue(true), want: BoolValue(true)},
		{caseName: "time", v: map[string]time.Time{"t": time.Date(2001, 12, 14, 21, 59, 43, 0, time.UTC)}, want: Map{"t": StringValue("2001-12-14T21:59:43Z")}},
		{
			caseName: "struct with Any field",
			v: struct {
				Extra Any `yaml:"extra"`
			}{Extra: Any{Node: Map{"k": NumberValue(1)}}},
			want: Map{"extra": Map{"k": NumberValue(1)}},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.caseName, func(t *testing.T) {
			got, err := MarshalViaYAML(tc.v)
			if err != nil {
				t.Fatal(err)
			}
			if !Equal(got, tc.want) {
				t.Errorf("got %#v; want %#v", got, tc.want)
			}
		})
	}
}

func TestUnmarshalViaYAML(t *testing.T) {
	m := Map{
		"id":     ToValue(1),
		"name":   ToValue("Reds"),
		"colors": ToArrayValues("Crimson", "Red", "Ruby", "Maroon"),
		"extra":  Map{"k": NumberValue(1)},
	}
	v := []struct {
		ID     int      `yaml:"id"`
		Name   string   `yaml:"name"`
		Colors []string `yaml:"colors"`
		Extra  Any      `yaml:"extra"`
	}{
		{},
		{
			ID:     1,
			Name:   "Reds",
			Colors: []string{"Crimson", "Red", "Ruby", "Maroon"},
			Extra:  Any{Node: Map{"k": NumberValue(1)}},
		},
	}
	got := v[0]
	want := v[1]

	if err := UnmarshalViaYAML(m, &got); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %#v; want %#v", got, want)
	}
}

// FuzzUnmarshalYAML exercises the YAML unmarshaler with arbitrary
// bytes, ensuring it never panics on malformed input.
func FuzzUnmarshalYAML(f *testing.F) {
	seeds := [][]byte{
		[]byte(`null`),
		[]byte(`true`),
		[]byte(`1`),
		[]byte("a: 1\nb: [1, 2]\n"),
		[]byte("- 1\n- 2\n- 3\n"),
	}
	for _, s := range seeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		_, _ = UnmarshalYAML(data)
	})
}

const benchStoreYAML = `store:
  bicycle:
    color: red
    price: 19.95
  book:
    - author: Nigel Rees
      category: reference
      price: 8.95
      title: Sayings of the Century
      tags:
        - {name: genre, value: reference}
        - {name: era, value: 20th century}
        - {name: theme, value: quotations}
    - author: Evelyn Waugh
      category: fiction
      price: 12.99
      title: Sword of Honour
      tags:
        - {name: genre, value: fiction}
        - {name: era, value: 20th century}
        - {name: theme, value: WWII}
    - author: Herman Melville
      category: fiction
      isbn: 0-553-21311-3
      price: 8.99
      title: Moby Dick
      tags:
        - {name: genre, value: fiction}
        - {name: era, value: 19th century}
        - {name: theme, value: whale hunting}
    - author: J. R. R. Tolkien
      category: fiction
      isbn: 0-395-19395-8
      price: 22.99
      title: The Lord of the Rings
      tags:
        - {name: genre, value: fantasy}
        - {name: era, value: 20th century}
        - {name: theme, value: good vs evil}
`

// BenchmarkUnmarshalYAML measures decoding the embedded sample
// document into a Node tree.
func BenchmarkUnmarshalYAML(b *testing.B) {
	data := []byte(benchStoreYAML)
	for i := 0; i < b.N; i++ {
		if _, err := UnmarshalYAML(data); err != nil {
			b.Fatal(err)
		}
	}
}
