package tree

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"math"
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

func TestDecodeJSONUseNumber(t *testing.T) {
	decode := func(data string, useNumber bool) (Node, error) {
		dec := json.NewDecoder(strings.NewReader(data))
		if useNumber {
			dec.UseNumber()
		}
		return DecodeJSON(dec)
	}
	testCases := []struct {
		caseName  string
		data      string
		want      Node
		wantValue string // the UnmarshalTypeError Value, when decoding fails
		// wantOffset is the end of the number, checked with UseNumber only: the stdlib's own Offset differs between v1 and v2.
		wantOffset int64
		negZero    bool // DeepEqual treats -0 as 0, so the sign is checked separately
	}{
		{caseName: "top-level number", data: `1`, want: NumberValue(1)},
		{caseName: "nested numbers", data: `[1, {"a":2.5}]`, want: Array{NumberValue(1), Map{"a": NumberValue(2.5)}}},
		{caseName: "negative zero", data: `[-0]`, want: Array{NumberValue(math.Copysign(0, -1))}, negZero: true},
		{caseName: "underflow to zero", data: `[1e-400]`, want: Array{NumberValue(0)}},
		{caseName: "decimal fraction", data: `[0.1]`, want: Array{NumberValue(0.1)}},
		{caseName: "integer beyond 2^53 loses precision", data: `[12345678901234567891]`, want: Array{NumberValue(12345678901234567891)}},
		{caseName: "out of float64 range", data: `{"a":[1e400]}`, wantValue: "number 1e400", wantOffset: 11},
	}
	for _, tc := range testCases {
		for _, useNumber := range []bool{true, false} {
			t.Run(fmt.Sprintf("%s/UseNumber=%v", tc.caseName, useNumber), func(t *testing.T) {
				got, err := decode(tc.data, useNumber)
				if tc.wantValue != "" {
					var ute *json.UnmarshalTypeError
					if !errors.As(err, &ute) || ute.Value != tc.wantValue || ute.Type != reflect.TypeFor[float64]() {
						t.Fatalf("got %v; want an UnmarshalTypeError for %s into float64", err, tc.wantValue)
					}
					if useNumber && ute.Offset != tc.wantOffset {
						t.Errorf("got Offset %d; want %d", ute.Offset, tc.wantOffset)
					}
					return
				}
				if err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(got, tc.want) {
					t.Errorf("got %#v; want %#v", got, tc.want)
				}
				if tc.negZero && !math.Signbit(float64(got.(Array)[0].(NumberValue))) {
					t.Errorf("got %#v; want negative zero", got)
				}
			})
		}
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
	testCases := []struct {
		caseName string
		init     Map
		data     string
		want     Map
		wantErr  bool
	}{
		{
			caseName: "nil map",
			data:     `{"a":1,"b":true,"c":null}`,
			want:     Map{"a": NumberValue(1), "b": BoolValue(true), "c": Nil},
		}, {
			caseName: "reused map keeps other keys",
			init:     Map{"a": NumberValue(0), "x": NumberValue(0)},
			data:     `{"a":1,"b":true,"c":null}`,
			want:     Map{"a": NumberValue(1), "b": BoolValue(true), "c": Nil, "x": NumberValue(0)},
		}, {
			caseName: "null leaves the map as it is",
			init:     Map{"x": NumberValue(0)},
			data:     `null`,
			want:     Map{"x": NumberValue(0)},
		}, {
			caseName: "null with trailing data is an error",
			init:     Map{"x": NumberValue(0)},
			data:     `null x`,
			want:     Map{"x": NumberValue(0)},
			wantErr:  true,
		}, {
			caseName: "trailing data after the object keeps the keys already set",
			init:     Map{"k": NumberValue(1)},
			data:     `{"a":1} x`,
			want:     Map{"a": NumberValue(1), "k": NumberValue(1)},
			wantErr:  true,
		}, {
			caseName: "nil map stays nil on a syntax error",
			data:     `{"a":`,
			wantErr:  true,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.caseName, func(t *testing.T) {
			got := maps.Clone(tc.init)
			if err := got.UnmarshalJSON([]byte(tc.data)); (err != nil) != tc.wantErr {
				t.Fatalf("got %v; wantErr %v", err, tc.wantErr)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("got %#v; want %#v", got, tc.want)
			}
			if tc.wantErr {
				return
			}
			s := struct{ M Map }{M: maps.Clone(tc.init)}
			if err := json.Unmarshal([]byte(`{"M":`+tc.data+`}`), &s); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(s.M, tc.want) {
				t.Errorf("json.Unmarshal: got %#v; want %#v", s.M, tc.want)
			}
		})
	}
}

func TestArrayUnmarshalJSON(t *testing.T) {
	// exactCopy keeps nil and has cap == len, so growth past len reallocates as the rows assume.
	exactCopy := func(a Array) Array {
		if a == nil {
			return nil
		}
		c := make(Array, len(a))
		copy(c, a)
		return c
	}
	testCases := []struct {
		caseName string
		init     Array
		data     string
		want     Array
		wantErr  bool
	}{
		{
			caseName: "empty array",
			init:     Array{},
			data:     `["1",2,true]`,
			want:     Array{StringValue("1"), NumberValue(2), BoolValue(true)},
		}, {
			caseName: "nil array",
			data:     `["1",2,true]`,
			want:     Array{StringValue("1"), NumberValue(2), BoolValue(true)},
		}, {
			caseName: "reused array is reset",
			init:     Array{NumberValue(0), NumberValue(0), NumberValue(0), NumberValue(0)},
			data:     `["1",2,true]`,
			want:     Array{StringValue("1"), NumberValue(2), BoolValue(true)},
		}, {
			caseName: "empty input into a nil array is empty, not null",
			data:     `[]`,
			want:     Array{},
		}, {
			caseName: "null leaves the array as it is",
			init:     Array{NumberValue(1)},
			data:     `null`,
			want:     Array{NumberValue(1)},
		}, {
			caseName: "null with trailing data is an error",
			init:     Array{NumberValue(1)},
			data:     `null x`,
			want:     Array{NumberValue(1)},
			wantErr:  true,
		}, {
			caseName: "syntax error keeps the length with the elements already decoded",
			init:     Array{NumberValue(9), NumberValue(9)},
			data:     `[1, {`,
			want:     Array{NumberValue(1), NumberValue(9)},
			wantErr:  true,
		}, {
			caseName: "error past capacity keeps the old length",
			init:     Array{NumberValue(1)},
			data:     `[5,6,{`,
			want:     Array{NumberValue(5)},
			wantErr:  true,
		}, {
			caseName: "trailing data after the value keeps the old length",
			init:     Array{NumberValue(9)},
			data:     `[1,2] x`,
			want:     Array{NumberValue(1)},
			wantErr:  true,
		}, {
			caseName: "an object is rejected",
			init:     Array{NumberValue(1)},
			data:     `{"a":1}`,
			want:     Array{NumberValue(1)},
			wantErr:  true,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.caseName, func(t *testing.T) {
			got := exactCopy(tc.init)
			if err := got.UnmarshalJSON([]byte(tc.data)); (err != nil) != tc.wantErr {
				t.Fatalf("got %v; wantErr %v", err, tc.wantErr)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("got %#v; want %#v", got, tc.want)
			}
			if tc.wantErr {
				return
			}
			s := struct{ A Array }{A: exactCopy(tc.init)}
			if err := json.Unmarshal([]byte(`{"A":`+tc.data+`}`), &s); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(s.A, tc.want) {
				t.Errorf("json.Unmarshal: got %#v; want %#v", s.A, tc.want)
			}
		})
	}
}

func TestAnyUnmarshalJSON(t *testing.T) {
	testCases := []struct {
		caseName string
		data     string
		want     Node
		wantErr  error
	}{
		{caseName: "null gives Nil", data: `null`, want: Nil},
		{caseName: "error leaves the node as it is", data: `{"a":`, want: NumberValue(1), wantErr: io.ErrUnexpectedEOF},
	}
	for _, tc := range testCases {
		t.Run(tc.caseName, func(t *testing.T) {
			a := Any{Node: NumberValue(1)}
			if err := a.UnmarshalJSON([]byte(tc.data)); !errors.Is(err, tc.wantErr) {
				t.Errorf("got %v; want %v", err, tc.wantErr)
			}
			if !reflect.DeepEqual(a.Node, tc.want) {
				t.Errorf("got %#v; want %#v", a.Node, tc.want)
			}
		})
	}
}

func TestUnmarshalJSONTrailingData(t *testing.T) {
	unmarshalers := map[string]func([]byte) error{
		"tree":  func(b []byte) error { _, err := UnmarshalJSON(b); return err },
		"Any":   func(b []byte) error { var n Any; return n.UnmarshalJSON(b) },
		"Map":   func(b []byte) error { var n Map; return n.UnmarshalJSON(b) },
		"Array": func(b []byte) error { var n Array; return n.UnmarshalJSON(b) },
		"json":  func(b []byte) error { var n Any; return json.Unmarshal(b, &n) },
	}
	testCases := []struct {
		caseName string
		via      []string
		data     string
		wantErr  bool
		errText  string
	}{
		{caseName: "trailing whitespace", via: []string{"tree", "Any", "Map", "json"}, data: "{\"a\":1} \n"},
		{caseName: "trailing whitespace after an array", via: []string{"tree", "Any", "Array", "json"}, data: "[1] \n"},
		{caseName: "trailing word", via: []string{"tree", "Any", "Map", "json"}, data: `{"a":1} x`, wantErr: true, errText: "invalid character 'x' after top-level value"},
		{caseName: "extra closing brace", via: []string{"tree", "Any", "Map", "json"}, data: `{"a":1}}`, wantErr: true, errText: "invalid character '}' after top-level value"},
		{caseName: "second array", via: []string{"tree", "Any", "Array", "json"}, data: `[1] [2]`, wantErr: true, errText: "invalid character '[' after top-level value"},
		{caseName: "second scalar", via: []string{"tree", "Any", "json"}, data: `1 2`, wantErr: true, errText: "invalid character '2' after top-level value"},
		{caseName: "trailing string", via: []string{"tree", "Any", "Array", "json"}, data: `[1] "foo"`, wantErr: true, errText: `invalid character '"' after top-level value`},
		{caseName: "trailing null", via: []string{"tree", "Any", "Map", "json"}, data: `{} null`, wantErr: true, errText: "invalid character 'n' after top-level value"},
	}
	for _, tc := range testCases {
		for _, via := range tc.via {
			t.Run(tc.caseName+"/"+via, func(t *testing.T) {
				err := unmarshalers[via]([]byte(tc.data))
				if (err != nil) != tc.wantErr {
					t.Fatalf("got %v; wantErr %v", err, tc.wantErr)
				}
				if tc.errText != "" && err.Error() != tc.errText {
					t.Errorf("got %q; want %q", err.Error(), tc.errText)
				}
			})
		}
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
		[]byte(`[1] x`),
		[]byte(`[1] [2]`),
		[]byte(`{"a":1}}`),
	}
	for _, s := range seeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		if _, err := UnmarshalJSON(data); err == nil && !json.Valid(data) {
			t.Errorf("accepted invalid JSON %q", data) // the reverse does not hold: 1e400 is valid but overflows float64
		}
	})
}

func TestJSONElemUnknownToken(t *testing.T) {
	testCases := []struct {
		caseName string
		tok      json.Token
	}{
		{caseName: "closing brace", tok: json.Delim('}')},
		{caseName: "closing bracket", tok: json.Delim(']')},
		{caseName: "unknown token type", tok: struct{}{}},
	}
	for _, tc := range testCases {
		t.Run(tc.caseName, func(t *testing.T) {
			_, err := jsonElem(json.NewDecoder(strings.NewReader("")), tc.tok, 1)
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
