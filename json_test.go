package tree

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"reflect"
	"runtime/debug"
	"strconv"
	"strings"
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
		wantErr  error
	}{
		{
			caseName: "empty input",
			data:     []byte(""),
			wantErr:  io.EOF,
		}, {
			caseName: "unterminated string",
			data:     []byte(`"`),
			wantErr:  io.ErrUnexpectedEOF,
		}, {
			caseName: "open brace only",
			data:     []byte(`{`),
			wantErr:  io.ErrUnexpectedEOF,
		}, {
			caseName: "open bracket only",
			data:     []byte(`[`),
			wantErr:  io.ErrUnexpectedEOF,
		}, {
			caseName: "object cut off after a key",
			data:     []byte(`{"a":`),
			wantErr:  io.ErrUnexpectedEOF,
		}, {
			caseName: "array cut off after a comma",
			data:     []byte(`[1,`),
			wantErr:  io.ErrUnexpectedEOF,
		}, {
			caseName: "nested container cut off",
			data:     []byte(`{"a":[1`),
			wantErr:  io.ErrUnexpectedEOF,
		}, {
			caseName: "object cut off before the colon",
			data:     []byte(`{"a"`),
			wantErr:  io.ErrUnexpectedEOF,
		}, {
			caseName: "object cut off before the closing brace",
			data:     []byte(`{"a":1`),
			wantErr:  io.ErrUnexpectedEOF,
		}, {
			caseName: "object cut off inside an array",
			data:     []byte(`[{"a":1`),
			wantErr:  io.ErrUnexpectedEOF,
		}, {
			caseName: "object cut off inside an object",
			data:     []byte(`{"a":{`),
			wantErr:  io.ErrUnexpectedEOF,
		}, {
			caseName: "stray closing brace",
			data:     []byte(`}`),
		}, {
			caseName: "invalid character after open brace",
			data:     []byte("{\n1"),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.caseName, func(t *testing.T) {
			dec := json.NewDecoder(bytes.NewReader(tc.data))
			_, err := DecodeJSON(dec)
			if err == nil {
				t.Fatal("no error")
			}
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Errorf("got %v; want %v", err, tc.wantErr)
				}
				unmarshalers := []json.Unmarshaler{new(Any)}
				switch {
				case bytes.HasPrefix(tc.data, []byte("{")):
					unmarshalers = append(unmarshalers, new(Map))
				case bytes.HasPrefix(tc.data, []byte("[")):
					unmarshalers = append(unmarshalers, new(Array))
				}
				for _, u := range unmarshalers {
					if err := u.UnmarshalJSON(tc.data); !errors.Is(err, tc.wantErr) {
						t.Errorf("%T.UnmarshalJSON: got %v; want %v", u, err, tc.wantErr)
					}
				}
				return
			}
			// Match on type, not message: Go 1.27's json/v2-backed decoder rewords syntax errors.
			var syntaxErr *json.SyntaxError
			if !errors.As(err, &syntaxErr) {
				t.Errorf("got %T %v; want *json.SyntaxError", err, err)
			}
		})
	}
}

func TestDecodeJSONStream(t *testing.T) {
	testCases := []struct {
		caseName  string
		data      string
		openOuter bool
		wantNodes int
		wantErr   error
	}{
		{
			caseName:  "second document cut off",
			data:      `{"a":1} {"b":`,
			wantNodes: 1,
			wantErr:   io.ErrUnexpectedEOF,
		}, {
			caseName:  "two complete documents",
			data:      `{"a":1} {"b":2}`,
			wantNodes: 2,
			wantErr:   io.EOF,
		}, {
			caseName:  "array opened at the end",
			data:      `[1] [`,
			wantNodes: 1,
			wantErr:   io.ErrUnexpectedEOF,
		}, {
			caseName: "empty input",
			wantErr:  io.EOF,
		}, {
			// DecodeJSON cannot see a container the caller opened, so the cut is a plain io.EOF; only dec.Token() for the closer errors.
			caseName:  "elements of a caller-opened array cut off",
			data:      `[{"a":1},`,
			openOuter: true,
			wantNodes: 1,
			wantErr:   io.EOF,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.caseName, func(t *testing.T) {
			dec := json.NewDecoder(bytes.NewReader([]byte(tc.data)))
			if tc.openOuter {
				if _, err := dec.Token(); err != nil {
					t.Fatal(err)
				}
			}
			nodes := 0
			var err error
			for {
				if _, err = DecodeJSON(dec); err != nil {
					break
				}
				nodes++
			}
			if !errors.Is(err, tc.wantErr) {
				t.Errorf("got %v; want %v", err, tc.wantErr)
			}
			if nodes != tc.wantNodes {
				t.Errorf("got %d nodes; want %d", nodes, tc.wantNodes)
			}
			if tc.openOuter {
				// The closer's error is io.EOF before Go 1.27 and a *json.SyntaxError with json/v2.
				if _, err := dec.Token(); err == nil {
					t.Error("closing delimiter: no error")
				}
			}
		})
	}
}

func TestDecodeJSONLong(t *testing.T) {
	const n = 500_000 // the old per-element recursion needs about 100 MB here, well past the cap below
	members := make([]string, n)
	for i := range members {
		members[i] = `"` + strconv.Itoa(i) + `":0`
	}
	members[n-1] = `"` + strconv.Itoa(n-1) + `":1`
	testCases := []struct {
		caseName string
		data     string
		wantType Type
		lastKey  any
	}{
		{
			caseName: "flat array",
			data:     "[" + strings.Repeat("0,", n-1) + "1]",
			wantType: TypeArray,
			lastKey:  n - 1,
		}, {
			caseName: "flat object",
			data:     "{" + strings.Join(members, ",") + "}",
			wantType: TypeMap,
			lastKey:  strconv.Itoa(n - 1),
		},
	}
	// A small stack makes a decoder that recurses once per element crash; it is process-wide, so keep this test serial.
	defer debug.SetMaxStack(debug.SetMaxStack(32 << 20))

	for _, tc := range testCases {
		t.Run(tc.caseName, func(t *testing.T) {
			got, err := DecodeJSON(json.NewDecoder(strings.NewReader(tc.data)))
			if err != nil {
				t.Fatal(err)
			}
			if got.Type() != tc.wantType {
				t.Fatalf("got type %v; want %v", got.Type(), tc.wantType)
			}
			if l := len(got.Array()) + len(got.Map()); l != n {
				t.Errorf("got %d elements; want %d", l, n)
			}
			if v := got.Get(tc.lastKey); v != NumberValue(1) {
				t.Errorf("got last value %v; want 1", v)
			}
		})
	}
}

func TestDecodeJSONDepth(t *testing.T) {
	nest := func(n int) string { return strings.Repeat("[", n) + strings.Repeat("]", n) }
	testCases := []struct {
		caseName string
		data     string
		wantErr  bool
	}{
		{caseName: "nesting at the limit", data: nest(10000)},
		{caseName: "nesting past the limit", data: nest(10001), wantErr: true},
		{caseName: "deep nesting", data: nest(1_000_000), wantErr: true},
	}
	for _, tc := range testCases {
		t.Run(tc.caseName, func(t *testing.T) {
			_, err := DecodeJSON(json.NewDecoder(strings.NewReader(tc.data)))
			if !tc.wantErr {
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			// json/v2 reports the depth from Token as a SyntaxError, older decoders from our own limit.
			if err == nil || !strings.Contains(err.Error(), "exceeded max depth") {
				t.Errorf("got %v; want exceeded max depth", err)
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

func TestJSONElemDelim(t *testing.T) {
	testCases := []struct {
		caseName string
		delim    json.Delim
	}{
		{caseName: "closing brace", delim: '}'},
		{caseName: "closing bracket", delim: ']'},
	}
	for _, tc := range testCases {
		t.Run(tc.caseName, func(t *testing.T) {
			_, err := jsonElem(json.NewDecoder(strings.NewReader("")), tc.delim, 1)
			if err == nil || !strings.Contains(err.Error(), "unknown token") {
				t.Errorf("got %v; want unknown token", err)
			}
		})
	}
}

// BenchmarkUnmarshalJSON measures decoding the embedded sample document into a Node tree.
func BenchmarkUnmarshalJSON(b *testing.B) {
	data := []byte(testdata.StoreJSON)
	for i := 0; i < b.N; i++ {
		if _, err := UnmarshalJSON(data); err != nil {
			b.Fatal(err)
		}
	}
}
