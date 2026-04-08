package tree

import (
	"bytes"
	"reflect"
	"testing"

	"gopkg.in/yaml.v2"
)

func Test_MarshalYAML(t *testing.T) {
	want := `a:
- "1"
- 2
- true
- null
- null
`
	n := Map{
		"a": Array{
			StringValue("1"),
			NumberValue(2),
			BoolValue(true),
			Nil,
			nil,
		},
	}
	got, err := MarshalYAML(n)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != want {
		t.Errorf("for %#v; got %s; want %s", n, string(got), want)
	}
}

func Test_Map_MarshalYAML(t *testing.T) {
	want := `a:
- "1"
- 2
- true
`
	n := Map{
		"a": Array{
			StringValue("1"),
			NumberValue(2),
			BoolValue(true),
		},
	}
	got, err := yaml.Marshal(n)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != want {
		t.Errorf("for %#v; got %s; want %s", n, string(got), want)
	}
}

func Test_Array_MarshalYAML(t *testing.T) {
	want := `- "1"
- 2
- true
`
	n := Array{
		StringValue("1"),
		NumberValue(2),
		BoolValue(true),
	}
	got, err := yaml.Marshal(n)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != want {
		t.Errorf("for %#v; marshaled %s; want %s", n, string(got), want)
	}
}

func Test_DecodeYAML_Errors(t *testing.T) {
	tests := []struct {
		data   []byte
		errstr string
	}{
		{
			data:   []byte(`"`),
			errstr: "yaml: found unexpected end of stream",
		}, {
			data:   []byte(`}`),
			errstr: "yaml: did not find expected node content",
		}, {
			data:   []byte("{\n1"),
			errstr: `yaml: line 2: did not find expected ',' or '}'`,
		},
	}
	for i, test := range tests {
		dec := yaml.NewDecoder(bytes.NewReader(test.data))
		_, err := DecodeYAML(dec)
		if err == nil {
			t.Fatalf("tests[%d] no error", i)
		}
		if err.Error() != test.errstr {
			t.Errorf("tests[%d] got %s; want %s", i, err.Error(), test.errstr)
		}
	}
}

func Test_UnmarshalYAML(t *testing.T) {
	tests := []struct {
		want Node
		data []byte
	}{
		{
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
			data: []byte(`a: 1
b: true
c: null
d: ["1",2,true]
e: {"x":"x"}
`),
		}, {
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
			data: []byte(`- "1"
- 2
- true
- null
- {"a":1,"b":true,"c":null}
- ["x"]
`),
		},
	}
	for i, test := range tests {
		got, err := UnmarshalYAML(test.data)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, test.want) {
			t.Errorf("tests[%d] got %#v; want %#v", i, got, test.want)
		}
	}
}

func Test_Map_UnmarshalYAML(t *testing.T) {
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
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %#v; want %#v", got, want)
	}
}

func Test_Array_UnmarshalYAML(t *testing.T) {
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
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %#v; want %#v", got, want)
	}
}

func Test_MarshalViaYAML(t *testing.T) {
	tests := []struct {
		v    interface{}
		want Node
	}{
		{
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
		}, {
			v:    "str",
			want: StringValue("str"),
		}, {
			v:    true,
			want: BoolValue(true),
		}, {
			v:    1,
			want: NumberValue(1),
		}, {
			v:    nil,
			want: Nil,
		}, {
			v:    BoolValue(true),
			want: BoolValue(true),
		},
	}

	for i, test := range tests {
		got, err := MarshalViaYAML(test.v)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, test.want) {
			t.Errorf("tests[%d] got %#v; want %#v", i, got, test.want)
		}
	}
}

func Test_UnmarshalViaYAML(t *testing.T) {
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
