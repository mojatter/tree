package tree

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/mojatter/tree/internal/testdata"
)

func TestMarshalJSON(t *testing.T) {
	want := `{"a":["1",2,true,null,null]}`
	n := Map{
		"a": Array{
			StringValue("1"),
			NumberValue(2),
			BoolValue(true),
			Nil,
			nil,
		},
	}
	got, err := MarshalJSON(n)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != want {
		t.Errorf("got %s; want %s", string(got), want)
	}
}

func TestNodeMarshalJSON(t *testing.T) {
	testCases := []struct {
		caseName string
		n        Node
		want     string
	}{
		{
			caseName: "Map",
			n: Map{
				"a": Array{
					StringValue("1"),
					NumberValue(2),
					BoolValue(true),
				},
			},
			want: `{"a":["1",2,true]}`,
		},
		{
			caseName: "Array",
			n: Array{
				StringValue("1"),
				NumberValue(2),
				BoolValue(true),
			},
			want: `["1",2,true]`,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.caseName, func(t *testing.T) {
			got, err := json.Marshal(tc.n)
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != tc.want {
				t.Errorf("got %s; want %s", string(got), tc.want)
			}
		})
	}
}

func TestDecodeJSONErrors(t *testing.T) {
	testCases := []struct {
		caseName string
		data     []byte
		errstr   string
	}{
		{
			caseName: "unterminated string",
			data:     []byte(`"`),
			errstr:   "unexpected EOF",
		}, {
			caseName: "stray closing brace",
			data:     []byte(`}`),
			errstr:   "invalid character '}' looking for beginning of value",
		}, {
			caseName: "invalid character after open brace",
			data:     []byte("{\n1"),
			errstr:   `invalid character '1'`,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.caseName, func(t *testing.T) {
			dec := json.NewDecoder(bytes.NewReader(tc.data))
			_, err := DecodeJSON(dec)
			if err == nil {
				t.Fatal("no error")
			}
			if err.Error() != tc.errstr {
				t.Errorf("got %s; want %s", err.Error(), tc.errstr)
			}
		})
	}
}

func TestUnmarshalJSON(t *testing.T) {
	testCases := []struct {
		caseName string
		data     string
		want     Node
	}{
		{
			caseName: "Map with nested Array and Map",
			data:     `{"a":1,"b":true,"c":null,"d":["1",2,true],"e":{"x":"x"}}`,
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
			data:     `["1",2,true,null,{"a":1,"b":true,"c":null},["x"]]`,
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
		{caseName: "NumberValue", data: `1`, want: NumberValue(1)},
		{caseName: "StringValue", data: `"str"`, want: StringValue("str")},
		{caseName: "BoolValue", data: `true`, want: BoolValue(true)},
		{caseName: "Nil", data: `null`, want: Nil},
	}
	for _, tc := range testCases {
		t.Run(tc.caseName, func(t *testing.T) {
			got, err := UnmarshalJSON([]byte(tc.data))
			if err != nil {
				t.Fatal(err)
			}
			if !Equal(got, tc.want) {
				t.Errorf("got %#v; want %#v", got, tc.want)
			}
		})
	}
}

func TestMapUnmarshalJSON(t *testing.T) {
	want := Map{
		"a": NumberValue(1),
		"b": BoolValue(true),
		"c": Nil,
	}
	data := []byte(`{"a":1,"b":true,"c":null}`)
	var got Map
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if !Equal(got, want) {
		t.Errorf("got %#v; want %#v", got, want)
	}
}

func TestArrayUnmarshalJSON(t *testing.T) {
	want := Array{
		StringValue("1"),
		NumberValue(2),
		BoolValue(true),
	}
	data := []byte(`["1",2,true]`)
	got := Array{}
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if !Equal(got, want) {
		t.Errorf("got %#v; want %#v", got, want)
	}
}

func TestMarshalViaJSON(t *testing.T) {
	testCases := []struct {
		caseName string
		v        any
		want     Node
	}{
		{
			caseName: "struct",
			v: struct {
				ID     int      `json:"id"`
				Name   string   `json:"name"`
				Colors []string `json:"colors"`
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
			got, err := MarshalViaJSON(tc.v)
			if err != nil {
				t.Fatal(err)
			}
			if !Equal(got, tc.want) {
				t.Errorf("got %#v; want %#v", got, tc.want)
			}
		})
	}
}

func TestUnmarshalViaJSON(t *testing.T) {
	m := Map{
		"id":     ToValue(1),
		"name":   ToValue("Reds"),
		"colors": ToArrayValues("Crimson", "Red", "Ruby", "Maroon"),
	}
	v := []struct {
		ID     int      `json:"id"`
		Name   string   `json:"name"`
		Colors []string `json:"colors"`
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

	if err := UnmarshalViaJSON(m, &got); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %#v; want %#v", got, want)
	}
}

// FuzzUnmarshalJSON exercises the JSON unmarshaler with arbitrary
// bytes, ensuring it never panics on malformed input.
func FuzzUnmarshalJSON(f *testing.F) {
	seeds := [][]byte{
		[]byte(`null`),
		[]byte(`true`),
		[]byte(`1`),
		[]byte(`"s"`),
		[]byte(`[]`),
		[]byte(`{}`),
		[]byte(`{"a": [1, 2, {"b": null}]}`),
	}
	for _, s := range seeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		_, _ = UnmarshalJSON(data)
	})
}

// BenchmarkUnmarshalJSON measures decoding the embedded sample
// document into a Node tree.
func BenchmarkUnmarshalJSON(b *testing.B) {
	data := []byte(testdata.StoreJSON)
	for i := 0; i < b.N; i++ {
		if _, err := UnmarshalJSON(data); err != nil {
			b.Fatal(err)
		}
	}
}
