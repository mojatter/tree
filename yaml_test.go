package tree

import (
	"bytes"
	"reflect"
	"testing"

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

func TestUnmarshalYAML(t *testing.T) {
	testCases := []struct {
		caseName string
		data     []byte
		want     Node
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
	}
	for _, tc := range testCases {
		t.Run(tc.caseName, func(t *testing.T) {
			got, err := UnmarshalYAML(tc.data)
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
		"a": NumberValue(1),
		"b": BoolValue(true),
		"c": Nil,
	}
	data := []byte(`a: 1
b: true
c: null
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
	}
	data := []byte(`- "1"
- 2
- true
`)
	got := Array{}
	if err := yaml.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if !Equal(got, want) {
		t.Errorf("got %#v; want %#v", got, want)
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
	}
	v := []struct {
		ID     int      `yaml:"id"`
		Name   string   `yaml:"name"`
		Colors []string `yaml:"colors"`
	}{
		{},
		{
			ID:     1,
			Name:   "Reds",
			Colors: []string{"Crimson", "Red", "Ruby", "Maroon"},
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
