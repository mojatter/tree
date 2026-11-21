package tree

import (
	"encoding/json"
	"errors"
	"math"
	"reflect"
	"testing"
	"time"
)

func TestV(t *testing.T) {
	// V is a thin alias for ToValue; sample a few representative
	// inputs to confirm they yield identical results.
	testCases := []struct {
		caseName string
		in       any
	}{
		{caseName: "nil", in: nil},
		{caseName: "string", in: "s"},
		{caseName: "bool", in: true},
		{caseName: "int", in: 1},
		{caseName: "int64", in: int64(2)},
		{caseName: "float64", in: 3.5},
	}
	for _, tc := range testCases {
		t.Run(tc.caseName, func(t *testing.T) {
			if got, want := V(tc.in), ToValue(tc.in); got != want {
				t.Errorf("V(%v) = %#v; want %#v", tc.in, got, want)
			}
		})
	}
}

func TestA(t *testing.T) {
	// A is a thin alias for ToArrayValues.
	got := A("a", 1, true)
	want := ToArrayValues("a", 1, true)
	if !Equal(got, want) {
		t.Errorf("A(...) = %#v; want %#v", got, want)
	}
}

func TestToValue(t *testing.T) {
	testCases := []struct {
		caseName string
		v        any
		want     Node
	}{
		{caseName: "nil", v: nil, want: Nil},
		{caseName: "string", v: "string", want: StringValue("string")},
		{caseName: "bool", v: true, want: BoolValue(true)},
		{caseName: "int", v: 1, want: NumberValue(1)},
		{caseName: "int64", v: int64(2), want: NumberValue(2)},
		{caseName: "int32", v: int32(3), want: NumberValue(3)},
		{caseName: "float64", v: float64(4.4), want: NumberValue(4.4)},
		{caseName: "float32", v: float32(5.5), want: NumberValue(5.5)},
		{caseName: "uint64", v: uint64(6), want: NumberValue(uint64(6))},
		{caseName: "uint32", v: uint32(7), want: NumberValue(uint32(7))},
		{caseName: "BoolValue passthrough", v: BoolValue(true), want: BoolValue(true)},
		{caseName: "int8", v: int8(-8), want: NumberValue(-8)},
		{caseName: "int16", v: int16(16), want: NumberValue(16)},
		{caseName: "uint", v: uint(3), want: NumberValue(3)},
		{caseName: "uint8", v: uint8(255), want: NumberValue(255)},
		{caseName: "uint16", v: uint16(16), want: NumberValue(16)},
		{caseName: "named string", v: namedString("s"), want: StringValue("s")},
		{caseName: "named bool", v: namedBool(true), want: BoolValue(true)},
		{caseName: "named int", v: namedInt(9), want: NumberValue(9)},
		{caseName: "named float", v: namedFloat(1.5), want: NumberValue(1.5)},
		{caseName: "nil pointer node passthrough", v: (*Map)(nil), want: (*Map)(nil)},
		{caseName: "json.Number", v: json.Number("1.5"), want: NumberValue(1.5)},
		{caseName: "json.Number with sign and exponent", v: json.Number("-1E2"), want: NumberValue(-100)},
	}
	for _, tc := range testCases {
		t.Run(tc.caseName, func(t *testing.T) {
			if got := ToValue(tc.v); !Equal(got, tc.want) {
				t.Errorf("for %v; got %#v; want %#v", tc.v, got, tc.want)
			}
		})
	}
}

type stringerKey string

func (k stringerKey) String() string { return "stringer" }

type (
	namedString string
	namedBool   bool
	namedInt    int8
	namedFloat  float32
)

func TestToValuePanics(t *testing.T) {
	testCases := []struct {
		caseName string
		v        any
		want     string
	}{
		{caseName: "struct", v: struct{}{}, want: "unsupported type struct {}"},
		{caseName: "string slice", v: []string{"a"}, want: "unsupported type []string"},
		{caseName: "string map", v: map[string]string{"a": "b"}, want: "unsupported type map[string]string"},
		{caseName: "node slice", v: []Node{StringValue("a")}, want: "unsupported type []tree.Node"},
		{caseName: "nil int pointer", v: (*int)(nil), want: "unsupported type *int"},
		{caseName: "complex", v: complex(1, 2), want: "unsupported type complex128"},
		{caseName: "uintptr", v: uintptr(1), want: "unsupported type uintptr"},
		{caseName: "invalid json.Number", v: json.Number("x"), want: `invalid number "x"`},
		{caseName: "json.Number NaN", v: json.Number("NaN"), want: `invalid number "NaN"`},
		{caseName: "json.Number hex", v: json.Number("0x1p4"), want: `invalid number "0x1p4"`},
		{caseName: "json.Number leading zero", v: json.Number("01"), want: `invalid number "01"`},
		{caseName: "json.Number leading plus", v: json.Number("+1"), want: `invalid number "+1"`},
		{caseName: "json.Number bare dot", v: json.Number(".5"), want: `invalid number ".5"`},
		{caseName: "json.Number leading blank", v: json.Number(" 1"), want: `invalid number " 1"`},
	}
	for _, tc := range testCases {
		t.Run(tc.caseName, func(t *testing.T) {
			defer func() {
				err, _ := recover().(error)
				if err == nil || err.Error() != tc.want {
					t.Errorf("for %#v; got panic %v; want %q", tc.v, err, tc.want)
				}
			}()

			ToValue(tc.v)
		})
	}
}

func TestToNode(t *testing.T) {
	testCases := []struct {
		caseName string
		v        any
		want     Node
	}{
		{caseName: "nil", v: nil, want: Nil},
		{caseName: "StringValue", v: StringValue("a"), want: StringValue("a")},
		{
			caseName: "map",
			v:        map[string]any{"a": 1, "b": true},
			want:     Map{"a": NumberValue(1), "b": BoolValue(true)},
		},
		{
			caseName: "slice",
			v:        []any{"a", true, 1},
			want:     Array{StringValue("a"), BoolValue(true), NumberValue(1)},
		},
		{
			caseName: "uint in a slice",
			v:        []any{uint8(1), map[string]any{"a": uint16(2)}},
			want:     Array{NumberValue(1), Map{"a": NumberValue(2)}},
		},
		{
			caseName: "json.Number in a map",
			v:        map[string]any{"n": json.Number("1")},
			want:     Map{"n": NumberValue(1)},
		},
		{
			caseName: "non-string keys",
			v:        map[any]any{1: "a", true: "b", "c": 3},
			want:     Map{"1": StringValue("a"), "true": StringValue("b"), "c": NumberValue(3)},
		},
		{
			caseName: "nil key",
			v:        map[any]any{nil: 1},
			want:     Map{"null": NumberValue(1)},
		},
		{
			caseName: "key kinds",
			v:        map[any]any{int8(-1): 1, uint16(2): 2, float32(0.1): 3, 1.5: 4, namedString("s"): 5, stringerKey("k"): 6},
			want:     Map{"-1": NumberValue(1), "2": NumberValue(2), "0.10000000149011612": NumberValue(3), "1.5": NumberValue(4), "s": NumberValue(5), "k": NumberValue(6)},
		},
		{
			caseName: "Nil key",
			v:        map[any]any{Nil: 1},
			want:     Map{"null": NumberValue(1)},
		},
		{
			caseName: "Any keys",
			v:        map[any]any{Any{Node: StringValue("s")}: 1, Any{}: 2},
			want:     Map{"s": NumberValue(1), "null": NumberValue(2)},
		},
		{
			caseName: "value node keys",
			v:        map[any]any{StringValue("s"): 1, BoolValue(true): 2, NumberValue(1.5): 3, NumberValue(1e21): 4},
			want:     Map{"s": NumberValue(1), "true": NumberValue(2), "1.5": NumberValue(3), "1000000000000000000000": NumberValue(4)},
		},
		{
			caseName: "float keys in plain decimal",
			v:        map[any]any{1e6: 1, 1e-7: 2, float32(0.5): 3},
			want:     Map{"1000000": NumberValue(1), "0.0000001": NumberValue(2), "0.5": NumberValue(3)},
		},
		{
			caseName: "float32 and float64 keys of different numbers",
			v:        map[any]any{float32(0.1): 1, 0.1: 2},
			want:     Map{"0.10000000149011612": NumberValue(1), "0.1": NumberValue(2)},
		},
		{
			caseName: "NaN key keeps its value",
			v:        map[any]any{math.NaN(): 1},
			want:     Map{"NaN": NumberValue(1)},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.caseName, func(t *testing.T) {
			if got := ToNode(tc.v); !Equal(got, tc.want) {
				t.Errorf("for %v; got %v; want %v", tc.v, got, tc.want)
			}
		})
	}
}

func TestToNodePanics(t *testing.T) {
	testCases := []struct {
		caseName string
		v        any
		want     string
	}{
		{caseName: "time in a map", v: map[string]any{"d": time.Date(2001, 12, 14, 0, 0, 0, 0, time.UTC)}, want: "unsupported type time.Time"},
		{caseName: "string slice in a map", v: map[string]any{"tags": []string{"a", "b"}}, want: "unsupported type []string"},
		{caseName: "slice of maps", v: []map[string]any{{"a": 1}}, want: "unsupported type []map[string]interface {}"},
		{caseName: "bytes", v: []byte("ab"), want: "unsupported type []uint8"},
		{caseName: "keys with the same text", v: map[any]any{1: "a", "1": "b"}, want: `duplicate key "1"`},
		{caseName: "nil and null keys", v: map[any]any{nil: 1, "null": 2}, want: `duplicate key "null"`},
		{caseName: "nil and Nil keys", v: map[any]any{nil: 1, Nil: 2}, want: `duplicate key "null"`},
		{caseName: "NumberValue and float keys", v: map[any]any{NumberValue(1e21): 1, 1e21: 2}, want: `duplicate key "1000000000000000000000"`},
		{caseName: "float32 and NumberValue keys", v: map[any]any{float32(0.1): 1, NumberValue(float32(0.1)): 2}, want: `duplicate key "0.10000000149011612"`},
		{caseName: "int and float keys", v: map[any]any{1000000: 1, 1e6: 2}, want: `duplicate key "1000000"`},
		{caseName: "NumberValue and int keys", v: map[any]any{NumberValue(1e6): 1, 1000000: 2}, want: `duplicate key "1000000"`},
		{caseName: "pointer map key", v: map[any]any{&Map{}: 1}, want: "unsupported key type *tree.Map"},
		{caseName: "pointer key", v: map[any]any{new(int): 1}, want: "unsupported key type *int"},
		{caseName: "nil pointer key", v: map[any]any{(*int)(nil): 1}, want: "unsupported key type *int"},
		{caseName: "time key", v: map[any]any{time.Time{}: 1}, want: "unsupported key type time.Time"},
		{caseName: "array key", v: map[any]any{[2]int{1, 2}: 1}, want: "unsupported key type [2]int"},
		{caseName: "uintptr key", v: map[any]any{uintptr(1): 1}, want: "unsupported key type uintptr"},
		{caseName: "nested keys with the same text", v: []any{map[any]any{true: 1, "true": 2}}, want: `duplicate key "true"`},
	}
	for _, tc := range testCases {
		t.Run(tc.caseName, func(t *testing.T) {
			defer func() {
				err, _ := recover().(error)
				if err == nil || err.Error() != tc.want {
					t.Errorf("for %#v; got panic %v; want %q", tc.v, err, tc.want)
				}
			}()

			ToNode(tc.v)
		})
	}
}

func TestWalk(t *testing.T) {
	root := Array{
		Map{"ID": ToValue(1)},
		Map{"ID": ToValue(2), "Sub": Array{Map{"ID": ToValue(20)}}},
		Map{"ID": ToValue(3), "Sub": Array{Map{"ID": ToValue(30)}}},
	}

	expected := []struct {
		n    Node
		keys []any
		skip bool
	}{
		{
			n:    root,
			keys: []any{},
		}, {
			n:    root.Get(0),
			keys: []any{0},
		}, {
			n:    root.Get(0).Get("ID"),
			keys: []any{0, "ID"},
		}, {
			n:    root.Get(1),
			keys: []any{1},
			skip: true,
		}, {
			n:    root.Get(2),
			keys: []any{2},
		}, {
			n:    root.Get(2).Get("ID"),
			keys: []any{2, "ID"},
		}, {
			n:    root.Get(2).Get("Sub"),
			keys: []any{2, "Sub"},
		}, {
			n:    root.Get(2).Get("Sub").Get(0),
			keys: []any{2, "Sub", 0},
		}, {
			n:    root.Get(2).Get("Sub").Get(0).Get("ID"),
			keys: []any{2, "Sub", 0, "ID"},
		},
	}

	i := 0
	err := Walk(root, func(n Node, keys []any) error {
		if i >= len(expected) {
			t.Fatalf("fn is called too many times %d", i)
			return nil
		}
		want := expected[i]
		i++

		if !Equal(n, want.n) {
			t.Errorf("walk[%d] got %#v; want %#v", i, n, want.n)
		}
		if !reflect.DeepEqual(keys, want.keys) {
			t.Errorf("walk[%d] got %#v; want %#v", i, keys, want.keys)
		}
		if want.skip {
			return SkipWalk
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(expected) != i {
		t.Errorf("fn is called %d times; want %d", i, len(expected))
	}
}

// rawEachNode wraps a Map whose Each passes a child as stored, without turning a Go nil node into Nil.
type rawEachNode struct{ Node }

func (n rawEachNode) Each(cb func(key any, v Node) error) error {
	m := n.Node.(Map)
	for _, k := range m.Keys() {
		if err := cb(k, m[k]); err != nil {
			return err
		}
	}
	return nil
}

func TestWalkVisits(t *testing.T) {
	type visit struct {
		keys []any
		n    Node
	}
	custom := bareValueNode{Map{"a": ToValue(1)}}
	errStop := errors.New("stop")
	testCases := []struct {
		caseName string
		root     Node
		stopAt   Node
		want     []visit
	}{
		{
			caseName: "Go nil children are visited as Nil",
			root:     Map{"a": nil, "b": Array{nil, Any{}}},
			want: []visit{
				{[]any{}, Map{"a": nil, "b": Array{nil, Any{}}}},
				{[]any{"a"}, Nil},
				{[]any{"b"}, Array{nil, Any{}}},
				{[]any{"b", 0}, Nil},
				{[]any{"b", 1}, Nil},
			},
		}, {
			caseName: "custom Each passing Go nil children visits them as Nil",
			root:     rawEachNode{Map{"a": nil, "b": (*Map)(nil)}},
			want: []visit{
				{[]any{}, rawEachNode{Map{"a": nil, "b": (*Map)(nil)}}},
				{[]any{"a"}, Nil},
				{[]any{"b"}, Nil},
			},
		}, {
			caseName: "Go nil root is visited as Nil",
			root:     nil,
			want:     []visit{{[]any{}, Nil}},
		}, {
			caseName: "zero Any root is visited as Nil",
			root:     Any{},
			want:     []visit{{[]any{}, Nil}},
		}, {
			caseName: "nil *Map root is visited as Nil",
			root:     (*Map)(nil),
			want:     []visit{{[]any{}, Nil}},
		}, {
			caseName: "value-typed custom node is descended through Each",
			root:     custom,
			want:     []visit{{[]any{}, custom}, {[]any{"a"}, ToValue(1)}},
		}, {
			caseName: "Any wrapping a custom node is descended through Each",
			root:     Any{custom},
			want:     []visit{{[]any{}, Any{custom}}, {[]any{"a"}, ToValue(1)}},
		}, {
			caseName: "custom scalar node is visited once",
			root:     bareValueNode{StringValue("x")},
			want:     []visit{{[]any{}, bareValueNode{StringValue("x")}}},
		}, {
			caseName: "scalar is visited once",
			root:     StringValue("x"),
			want:     []visit{{[]any{}, StringValue("x")}},
		}, {
			caseName: "Any wrapping a Map is descended",
			root:     Any{Map{"a": ToValue(1)}},
			want:     []visit{{[]any{}, Any{Map{"a": ToValue(1)}}}, {[]any{"a"}, ToValue(1)}},
		}, {
			caseName: "Any wrapping an Array is descended",
			root:     Any{Array{ToValue(1)}},
			want:     []visit{{[]any{}, Any{Array{ToValue(1)}}}, {[]any{0}, ToValue(1)}},
		}, {
			caseName: "non-nil *Map is descended through Each",
			root:     &Map{"a": ToValue(1)},
			want:     []visit{{[]any{}, &Map{"a": ToValue(1)}}, {[]any{"a"}, ToValue(1)}},
		}, {
			caseName: "non-nil *Array is descended through Each",
			root:     &Array{ToValue(1), nil},
			want:     []visit{{[]any{}, &Array{ToValue(1), nil}}, {[]any{0}, ToValue(1)}, {[]any{1}, Nil}},
		}, {
			caseName: "an error from fn in an Array stops the walk",
			root:     Array{ToValue(1), ToValue(2), ToValue(3)},
			stopAt:   ToValue(2),
			want: []visit{
				{[]any{}, Array{ToValue(1), ToValue(2), ToValue(3)}},
				{[]any{0}, ToValue(1)},
				{[]any{1}, ToValue(2)},
			},
		}, {
			caseName: "an error from fn in a Map stops the walk",
			root:     Map{"a": ToValue(1), "b": ToValue(2), "c": ToValue(3)},
			stopAt:   ToValue(2),
			want: []visit{
				{[]any{}, Map{"a": ToValue(1), "b": ToValue(2), "c": ToValue(3)}},
				{[]any{"a"}, ToValue(1)},
				{[]any{"b"}, ToValue(2)},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.caseName, func(t *testing.T) {
			var got []visit
			err := Walk(tc.root, func(n Node, keys []any) error {
				got = append(got, visit{keys, n})
				if tc.stopAt != nil && reflect.DeepEqual(n, tc.stopAt) {
					return errStop
				}
				return nil
			})
			var wantErr error
			if tc.stopAt != nil {
				wantErr = errStop
			}
			if !errors.Is(err, wantErr) {
				t.Fatalf("got error %v; want %v", err, wantErr)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("got %#v; want %#v", got, tc.want)
			}
		})
	}
}

func TestWalkKeys(t *testing.T) {
	testCases := []struct {
		caseName string
		root     Node
		record   func(keys []any) []any
		want     [][]any
	}{
		{
			caseName: "retained keys are not overwritten by the next sibling or by walk",
			root:     Map{"a": Map{"b": Map{"c": Map{"d1": nil, "d2": nil}}}},
			record:   func(keys []any) []any { return keys },
			want:     [][]any{{}, {"a"}, {"a", "b"}, {"a", "b", "c"}, {"a", "b", "c", "d1"}, {"a", "b", "c", "d2"}},
		}, {
			caseName: "keys appended to in fn are not overwritten",
			root:     Map{"a": Map{"b": Map{"c": Map{"d1": nil, "d2": nil}}}},
			record:   func(keys []any) []any { return append(keys, "x") },
			want: [][]any{
				{"x"}, {"a", "x"}, {"a", "b", "x"}, {"a", "b", "c", "x"},
				{"a", "b", "c", "d1", "x"}, {"a", "b", "c", "d2", "x"},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.caseName, func(t *testing.T) {
			var got [][]any
			err := Walk(tc.root, func(_ Node, keys []any) error {
				got = append(got, tc.record(keys))
				return nil
			})
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("got %#v; want %#v", got, tc.want)
			}
		})
	}
}

func TestRegexpMatchString(t *testing.T) {
	testCases := []struct {
		caseName string
		expr     string
		value    string
		want     bool
		errstr   string
	}{
		{
			caseName: "substring match",
			expr:     `a`,
			value:    "abc",
			want:     true,
		}, {
			caseName: "anchored full match",
			expr:     `^[a-z]+$`,
			value:    "abc",
			want:     true,
		}, {
			caseName: "no match",
			expr:     `x`,
			value:    "abc",
			want:     false,
		}, {
			caseName: "invalid regex",
			expr:     `(`,
			value:    "abc",
			errstr:   "error parsing regexp: missing closing ): `(`",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.caseName, func(t *testing.T) {
			got, err := regexpMatchString(tc.expr, tc.value)
			if tc.errstr != "" {
				if err == nil {
					t.Fatalf("for %v; no error", tc.expr)
				}
				if err.Error() != tc.errstr {
					t.Errorf("for %v; got %v want %v", tc.expr, err.Error(), tc.errstr)
				}
				return
			}
			if err != nil {
				t.Fatalf("for %v; %+v", tc.expr, err)
			}
			if got != tc.want {
				t.Errorf("for %v; got %v; want %v", tc.expr, got, tc.want)
			}
		})
	}
}

func TestClone(t *testing.T) {
	testCases := []struct {
		caseName string
		n        Node
		want     Node
	}{
		{caseName: "Value", n: ToValue(1), want: ToValue(1)},
		{
			caseName: "Map",
			n:        Map{"a": ToValue(1), "b": ToValue(2)},
			want:     Map{"a": ToValue(1), "b": ToValue(2)},
		},
		{caseName: "nil", n: nil, want: Nil},
		{caseName: "Array(nil)", n: Array(nil), want: Array{}},
		{caseName: "Map(nil)", n: Map(nil), want: Map{}},
	}
	for _, tc := range testCases {
		t.Run(tc.caseName, func(t *testing.T) {
			if got := Clone(tc.n); !reflect.DeepEqual(got, tc.want) {
				t.Errorf("unexpected %v; want %v", got, tc.want)
			}
		})
	}
}

func TestCloneDeep(t *testing.T) {
	testCases := []struct {
		caseName string
		n        Node
		want     Node
		update   func(n Node)
	}{
		{
			caseName: "Value",
			n:        ToValue(1),
			want:     ToValue(1),
			update:   func(n Node) {},
		}, {
			caseName: "Map of Arrays mutated post-clone",
			n:        Map{"a": ToArrayValues(1, 2), "b": ToArrayValues(3, 4)},
			want:     Map{"a": ToArrayValues(1, 2), "b": ToArrayValues(3, 4)},
			update: func(n Node) {
				n.Map().Get("a").Array()[0] = ToValue(5)
			},
		}, {
			caseName: "Map three levels mutated post-clone",
			n:        Map{"a": Map{"b": Map{"c": ToValue(1)}}},
			want:     Map{"a": Map{"b": Map{"c": ToValue(1)}}},
			update: func(n Node) {
				n.Map()["a"].Map()["b"].Map()["c"] = ToValue(2)
			},
		}, {
			caseName: "Array three levels mutated post-clone",
			n:        Array{Array{ToArrayValues(1, 2)}},
			want:     Array{Array{ToArrayValues(1, 2)}},
			update: func(n Node) {
				n.Array()[0].Array()[0].Array()[0] = ToValue(3)
			},
		}, {
			caseName: "nil",
			n:        nil,
			want:     Nil,
			update:   func(n Node) {},
		}, {
			caseName: "nil children normalised to Nil",
			n:        Map{"a": nil, "b": Array{nil, ToValue(1)}},
			want:     Map{"a": Nil, "b": Array{Nil, ToValue(1)}},
			update: func(n Node) {
				n.Map()["b"].Array()[1] = ToValue(2)
			},
		}, {
			caseName: "Array(nil) child",
			n:        Map{"a": Array(nil)},
			want:     Map{"a": Array{}},
			update:   func(n Node) {},
		}, {
			caseName: "pointer Array child mutated post-clone",
			n:        Map{"a": Map{"b": &Array{ToValue(1)}}},
			want:     Map{"a": Map{"b": Array{ToValue(1)}}},
			update: func(n Node) {
				(*n.Map()["a"].Map()["b"].(*Array))[0] = ToValue(9)
			},
		}, {
			caseName: "nil pointer child normalised to Nil",
			n:        Map{"a": (*Array)(nil)},
			want:     Map{"a": Nil},
			update:   func(n Node) {},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.caseName, func(t *testing.T) {
			got := CloneDeep(tc.n)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("unexpected %v; want %v", got, tc.want)
			}
			tc.update(tc.n)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("after update unexpected %v; want %v", got, tc.want)
			}
		})
	}
}

func TestIsGoNil(t *testing.T) {
	testCases := []struct {
		caseName string
		n        Node
		want     bool
	}{
		{caseName: "untyped nil", n: nil, want: true},
		{caseName: "nil *Array", n: (*Array)(nil), want: true},
		{caseName: "nil *Map", n: (*Map)(nil), want: true},
		{caseName: "nil custom pointer", n: (*leafEditorNode)(nil), want: true},
		{caseName: "non-nil *Array", n: &Array{}, want: false},
		{caseName: "Map(nil)", n: Map(nil), want: false},
		{caseName: "Nil", n: Nil, want: false},
		{caseName: "non-nil *Map", n: &Map{}, want: false},
		{caseName: "Any wrapping nil", n: Any{}, want: true},
		{caseName: "Any wrapping a nil pointer", n: Any{Node: (*Array)(nil)}, want: true},
		{caseName: "Any wrapping a Map", n: Any{Node: Map{}}, want: false},
		{caseName: "pointer Any wrapping nil", n: &Any{}, want: true},
		{caseName: "pointer Any wrapping a nil pointer", n: &Any{Node: (*Array)(nil)}, want: true},
		{caseName: "pointer Any wrapping a Map", n: &Any{Node: Map{}}, want: false},
	}
	for _, tc := range testCases {
		t.Run(tc.caseName, func(t *testing.T) {
			if got := isGoNil(tc.n); got != tc.want {
				t.Errorf("got %v; want %v", got, tc.want)
			}
		})
	}
}

func BenchmarkWalk(b *testing.B) {
	store := mustBenchNode(b)
	items := make(Array, 200)
	for i := range items {
		items[i] = Map{"id": ToValue(i), "name": ToValue("item"), "ok": ToValue(true)}
	}
	testCases := []struct {
		caseName string
		n        Node
	}{
		{caseName: "store", n: store},
		{caseName: "array200", n: items},
		{caseName: "storeAny", n: wrapAny(store)},
		{caseName: "array200Any", n: wrapAny(items)},
	}
	for _, tc := range testCases {
		b.Run(tc.caseName, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				if err := Walk(tc.n, func(Node, []any) error { return nil }); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// wrapAny returns n with every node in it wrapped in Any.
func wrapAny(n Node) Node {
	switch x := n.(type) {
	case Array:
		a := make(Array, len(x))
		for i, v := range x {
			a[i] = wrapAny(v)
		}
		return Any{a}
	case Map:
		m := make(Map, len(x))
		for k, v := range x {
			m[k] = wrapAny(v)
		}
		return Any{m}
	}
	return Any{n}
}
