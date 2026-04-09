package tree

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"
)

func Test_MarshalJSON(t *testing.T) {
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

func Test_Map_MarshalJSON(t *testing.T) {
	want := `{"a":["1",2,true]}`
	n := Map{
		"a": Array{
			StringValue("1"),
			NumberValue(2),
			BoolValue(true),
		},
	}
	got, err := json.Marshal(n)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != want {
		t.Errorf("got %s; want %s", string(got), want)
	}
}

func Test_Array_MarshalJSON(t *testing.T) {
	want := `["1",2,true]`
	n := Array{
		StringValue("1"),
		NumberValue(2),
		BoolValue(true),
	}
	got, err := json.Marshal(n)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != want {
		t.Errorf("got %s; want %s", string(got), want)
	}
}

func Test_DecodeJSON_Errors(t *testing.T) {
	tests := []struct {
		data   []byte
		errstr string
	}{
		{
			data:   []byte(`"`),
			errstr: "unexpected EOF",
		}, {
			data:   []byte(`}`),
			errstr: "invalid character '}' looking for beginning of value",
		}, {
			data:   []byte("{\n1"),
			errstr: `invalid character '1'`,
		},
	}
	for i, test := range tests {
		dec := json.NewDecoder(bytes.NewReader(test.data))
		_, err := DecodeJSON(dec)
		if err == nil {
			t.Fatalf("tests[%d] no error", i)
		}
		if err.Error() != test.errstr {
			t.Errorf("tests[%d] got %s; want %s", i, err.Error(), test.errstr)
		}
	}
}

func Test_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		want Node
		data string
	}{
		{
			data: `{"a":1,"b":true,"c":null,"d":["1",2,true],"e":{"x":"x"}}`,
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
			data: `["1",2,true,null,{"a":1,"b":true,"c":null},["x"]]`,
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
		}, {
			data: `1`,
			want: NumberValue(1),
		}, {
			data: `"str"`,
			want: StringValue("str"),
		}, {
			data: `true`,
			want: BoolValue(true),
		}, {
			data: `null`,
			want: Nil,
		},
	}
	for i, test := range tests {
		got, err := UnmarshalJSON([]byte(test.data))
		if err != nil {
			t.Fatalf("tests[%d] %v", i, err)
		}
		if !reflect.DeepEqual(got, test.want) {
			t.Errorf("tests[%d] got %#v; want %#v", i, got, test.want)
		}
	}
}

func Test_Map_UnmarshalJSON(t *testing.T) {
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
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %#v; want %#v", got, want)
	}
}

func Test_Array_UnmarshalJSON(t *testing.T) {
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
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %#v; want %#v", got, want)
	}
}

func Test_MarshalViaJSON(t *testing.T) {
	tests := []struct {
		v    any
		want Node
	}{
		{
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
		got, err := MarshalViaJSON(test.v)
		if err != nil {
			t.Fatalf("tests[%d] %v", i, err)
		}
		if !reflect.DeepEqual(got, test.want) {
			t.Errorf("tests[%d] got %#v; want %#v", i, got, test.want)
		}
	}
}

func Test_UnmarshalViaJSON(t *testing.T) {
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
	data := []byte(benchStoreJSON)
	for i := 0; i < b.N; i++ {
		if _, err := UnmarshalJSON(data); err != nil {
			b.Fatal(err)
		}
	}
}
