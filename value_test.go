package tree

import (
	"fmt"
	"testing"
)

func TestValue(t *testing.T) {
	testCases := []struct {
		caseName string
		value    Node
		want     Node
	}{
		{caseName: "Nil", value: Nil, want: Nil},
		{caseName: "StringValue", value: StringValue("test"), want: StringValue("test")},
		{caseName: "BoolValue", value: BoolValue(true), want: BoolValue(true)},
		{caseName: "NumberValue int", value: NumberValue(1), want: NumberValue(1)},
		{caseName: "NumberValue float", value: NumberValue(2.3), want: NumberValue(2.3)},
	}
	for _, tc := range testCases {
		t.Run(tc.caseName, func(t *testing.T) {
			got := tc.value.Value()
			if !Equal(got, tc.want) {
				t.Errorf("Value got %v; want %v", got, tc.want)
			}
		})
	}
}

func TestValueAs(t *testing.T) {
	testCases := []struct {
		caseName string
		value    Value
		b        bool
		i        int
		i64      int64
		f64      float64
		s        string
	}{
		{caseName: "Nil", value: Nil},
		{caseName: "StringValue", value: StringValue("test"), s: "test"},
		{caseName: "BoolValue true", value: BoolValue(true), b: true, s: "true"},
		{caseName: "NumberValue int", value: NumberValue(1), i: 1, i64: 1, f64: 1, s: "1"},
		{caseName: "NumberValue float", value: NumberValue(2.3), i: 2, i64: 2, f64: 2.3, s: "2.3"},
	}
	for _, tc := range testCases {
		t.Run(tc.caseName, func(t *testing.T) {
			if got := tc.value.Bool(); got != tc.b {
				t.Errorf("Bool got %v; want %v", got, tc.b)
			}
			if got := tc.value.Int(); got != tc.i {
				t.Errorf("Int got %v; want %v", got, tc.i)
			}
			if got := tc.value.Int64(); got != tc.i64 {
				t.Errorf("Int64 got %v; want %v", got, tc.i64)
			}
			if got := tc.value.Float64(); got != tc.f64 {
				t.Errorf("Float64 got %v; want %v", got, tc.f64)
			}
			if got := tc.value.String(); got != tc.s {
				t.Errorf("String got %v; want %v", got, tc.s)
			}
		})
	}
}

func TestValueCompare(t *testing.T) {
	testCases := []struct {
		n    Value
		op   Operator
		v    Value
		want bool
	}{
		{StringValue("x"), EQ, nil, false},
		{StringValue("x"), EQ, StringValue("x"), true},
		{StringValue("x"), EQ, StringValue("y"), false},
		{StringValue("1"), EQ, NumberValue(1), false},
		{StringValue("x"), GT, StringValue("a"), true},
		{StringValue("x"), GT, StringValue("x"), false},
		{StringValue("x"), GT, StringValue("y"), false},
		{StringValue("x"), GE, StringValue("a"), true},
		{StringValue("x"), GE, StringValue("x"), true},
		{StringValue("x"), GE, StringValue("y"), false},
		{StringValue("x"), LT, StringValue("a"), false},
		{StringValue("x"), LT, StringValue("x"), false},
		{StringValue("x"), LT, StringValue("y"), true},
		{StringValue("x"), LE, StringValue("a"), false},
		{StringValue("x"), LE, StringValue("x"), true},
		{StringValue("x"), LE, StringValue("y"), true},
		{StringValue("x"), NE, nil, true},
		{StringValue("x"), NE, StringValue("x"), false},
		{StringValue("x"), NE, StringValue("y"), true},
		{StringValue("1"), NE, NumberValue(1), true},
		{StringValue("xyz"), RE, StringValue(`x`), true},
		{StringValue("xyz"), RE, StringValue(`z$`), true},
		{StringValue("xyz"), RE, StringValue(`^[a-z]+$`), true},
		{StringValue("xyz"), RE, StringValue(`a`), false},
		{StringValue("xyz"), RE, StringValue(`^z`), false},
		{StringValue("xyz"), RE, StringValue(`^[0-9]+$`), false},
		{StringValue("x"), Operator("unknown"), StringValue("x"), false},
		{NumberValue(1), EQ, nil, false},
		{NumberValue(1), EQ, NumberValue(1), true},
		{NumberValue(1), EQ, NumberValue(0), false},
		{NumberValue(1), EQ, NumberValue(1.0), true},
		{NumberValue(1), EQ, StringValue("1"), false},
		{NumberValue(1), GT, NumberValue(0), true},
		{NumberValue(1), GT, NumberValue(1), false},
		{NumberValue(1), GT, NumberValue(2), false},
		{NumberValue(1), GE, NumberValue(0), true},
		{NumberValue(1), GE, NumberValue(1), true},
		{NumberValue(1), GE, NumberValue(2), false},
		{NumberValue(1), LT, NumberValue(0), false},
		{NumberValue(1), LT, NumberValue(1), false},
		{NumberValue(1), LT, NumberValue(2), true},
		{NumberValue(1), LE, NumberValue(0), false},
		{NumberValue(1), LE, NumberValue(1), true},
		{NumberValue(1), LE, NumberValue(2), true},
		{NumberValue(1), NE, nil, true},
		{NumberValue(1), NE, NumberValue(1), false},
		{NumberValue(1), NE, NumberValue(0), true},
		{NumberValue(1), NE, NumberValue(1.0), false},
		{NumberValue(1), Operator("unknown"), NumberValue(1), false},
		{BoolValue(true), EQ, BoolValue(true), true},
		{BoolValue(true), EQ, BoolValue(false), false},
		{BoolValue(true), EQ, StringValue("true"), false},
		{BoolValue(true), LT, BoolValue(true), false},
		{BoolValue(true), GT, BoolValue(true), false},
		{BoolValue(true), NE, BoolValue(true), false},
		{BoolValue(true), NE, BoolValue(false), true},
		{BoolValue(true), NE, StringValue("true"), true},
	}
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%v %s %v", tc.n, tc.op, tc.v), func(t *testing.T) {
			got := tc.n.Compare(tc.op, tc.v)
			if got != tc.want {
				t.Errorf("got %v; want %v", got, tc.want)
			}
		})
	}
}

func TestValueFind(t *testing.T) {
	testCases := []struct {
		caseName string
		n        Node
		expr     string
		want     []Node
	}{
		{
			caseName: "StringValue",
			n:        StringValue("str"),
			expr:     ".",
			want:     []Node{StringValue("str")},
		}, {
			caseName: "BoolValue",
			n:        BoolValue(true),
			expr:     ".",
			want:     []Node{BoolValue(true)},
		}, {
			caseName: "NumberValue",
			n:        NumberValue(1),
			expr:     ".",
			want:     []Node{NumberValue(1)},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.caseName, func(t *testing.T) {
			got, err := tc.n.Find(tc.expr)
			if err != nil {
				t.Fatal(err)
			}
			if !Equal(Array(got), Array(tc.want)) {
				t.Errorf("got %#v; want %#v", got, tc.want)
			}
		})
	}
}
