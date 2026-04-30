package tree

import (
	"reflect"
	"testing"
)

func TestValue(t *testing.T) {
	tests := []struct {
		value Node
		b     bool
		i     int
		i64   int64
		f64   float64
		s     string
	}{
		{
			value: Nil,
		}, {
			value: StringValue("test"),
			s:     "test",
		}, {
			value: BoolValue(true),
			b:     true,
			s:     "true",
		}, {
			value: NumberValue(1),
			i:     1,
			i64:   int64(1),
			f64:   float64(1),
			s:     "1",
		}, {
			value: NumberValue(2.3),
			i:     2,
			i64:   int64(2),
			f64:   float64(2.3),
			s:     "2.3",
		},
	}
	for i, test := range tests {
		v := test.value
		vv := v.Value()
		if tt := v.Type(); tt&TypeValue == 0 {
			t.Errorf("tests[%d] Type got %v; want TypeValue", i, tt)
		}
		if a := v.Array(); a != nil {
			t.Errorf("tests[%d] Array got %v; want nil", i, a)
		}
		if m := v.Map(); m != nil {
			t.Errorf("tests[%d] Map got %v; want nil", i, m)
		}
		if vv.(Node) != v {
			t.Errorf("tests[%d] Value got %v; want %v", i, vv, v)
		}
		if b := vv.Bool(); b != test.b {
			t.Errorf("tests[%d] Bool got %v; want %v", i, b, test.b)
		}
		if ii := vv.Int(); ii != test.i {
			t.Errorf("tests[%d] Int got %v; want %v", i, ii, test.i)
		}
		if i64 := vv.Int64(); i64 != test.i64 {
			t.Errorf("tests[%d] Int64 got %v; want %v", i, i64, test.i64)
		}
		if f64 := vv.Float64(); f64 != test.f64 {
			t.Errorf("tests[%d] Float64 got %v; want %v", i, f64, test.f64)
		}
		if s := vv.String(); s != test.s {
			t.Errorf("tests[%d] String got %v; want %v", i, s, test.s)
		}
	}
}

func TestValueCompare(t *testing.T) {
	tests := []struct {
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
	for i, test := range tests {
		got := test.n.Compare(test.op, test.v)
		if got != test.want {
			t.Errorf("tests[%d] got %v; want %v", i, got, test.want)
		}
	}
}

func TestValueFind(t *testing.T) {
	tests := []struct {
		n    Node
		expr string
		want []Node
	}{
		{
			n:    StringValue("str"),
			expr: ".",
			want: []Node{StringValue("str")},
		}, {
			n:    BoolValue(true),
			expr: ".",
			want: []Node{BoolValue(true)},
		}, {
			n:    NumberValue(1),
			expr: ".",
			want: []Node{NumberValue(1)},
		},
	}
	for i, test := range tests {
		got, err := test.n.Find(test.expr)
		if err != nil {
			t.Fatalf("tests[%d] %v", i, err)
		}
		if !reflect.DeepEqual(got, test.want) {
			t.Errorf("tests[%d] got %#v; want %#v", i, got, test.want)
		}
	}
}
