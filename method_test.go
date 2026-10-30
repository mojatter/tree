package tree

import (
	"math"
	"reflect"
	"testing"
)

// bareValueNode is a custom node whose Type is the unspecialised TypeValue.
type bareValueNode struct{ Node }

func (bareValueNode) Type() Type { return TypeValue }

func TestMethodQuery(t *testing.T) {
	testCases := []struct {
		caseName string
		q        Query
		n        Node
		want     []Node
		errstr   string
	}{
		{
			caseName: "count map",
			q:        &CountQuery{},
			n:        Map{"key1": ToValue(1), "key2": ToValue("a")},
			want:     []Node{ToValue(2)},
		}, {
			caseName: "count array",
			q:        &CountQuery{},
			n:        ToArrayValues(1, 2, 3),
			want:     []Node{ToValue(3)},
		}, {
			caseName: "keys map",
			q:        &KeysQuery{},
			n:        Map{"key1": ToValue(1), "key2": ToValue("a")},
			want:     []Node{ToArrayValues("key1", "key2")},
		}, {
			caseName: "keys array",
			q:        &KeysQuery{},
			n:        ToArrayValues(1, 2, 3),
			want:     []Node{ToArrayValues(0, 1, 2)},
		}, {
			caseName: "values map",
			q:        &ValuesQuery{},
			n:        Map{"key1": ToValue(1), "key2": ToValue("a")},
			want:     []Node{ToArrayValues(1, "a")},
		}, {
			caseName: "values array",
			q:        &ValuesQuery{},
			n:        ToArrayValues(1, 2, 3),
			want:     []Node{ToArrayValues(1, 2, 3)},
		}, {
			caseName: "values map with Go nil values",
			q:        &ValuesQuery{},
			n:        Map{"a": nil, "b": Any{}, "c": (*Map)(nil), "d": Any{StringValue("1")}},
			want:     []Node{Array{Nil, Nil, Nil, Any{StringValue("1")}}},
		}, {
			caseName: "values array with Go nil elements",
			q:        &ValuesQuery{},
			n:        Array{nil, ToValue(1), Any{}},
			want:     []Node{Array{Nil, ToValue(1), Nil}},
		}, {
			caseName: "empty array true",
			q:        &EmptyQuery{},
			n:        Array{},
			want:     []Node{BoolValue(true)},
		}, {
			caseName: "empty array false",
			q:        &EmptyQuery{},
			n:        ToArrayValues(1, 2),
			want:     []Node{BoolValue(false)},
		}, {
			caseName: "empty map true",
			q:        &EmptyQuery{},
			n:        Map{},
			want:     []Node{BoolValue(true)},
		}, {
			caseName: "empty string true",
			q:        &EmptyQuery{},
			n:        StringValue(""),
			want:     []Node{BoolValue(true)},
		}, {
			caseName: "empty string false",
			q:        &EmptyQuery{},
			n:        StringValue("test"),
			want:     []Node{BoolValue(false)},
		}, {
			caseName: "empty nil",
			q:        &EmptyQuery{},
			n:        Nil,
			want:     []Node{BoolValue(true)},
		}, {
			caseName: "type array",
			q:        &TypeQuery{},
			n:        ToArrayValues(1, 2),
			want:     []Node{StringValue("array")},
		}, {
			caseName: "type object",
			q:        &TypeQuery{},
			n:        Map{"key": ToValue("value")},
			want:     []Node{StringValue("object")},
		}, {
			caseName: "type string",
			q:        &TypeQuery{},
			n:        StringValue("test"),
			want:     []Node{StringValue("string")},
		}, {
			caseName: "type number",
			q:        &TypeQuery{},
			n:        NumberValue(42),
			want:     []Node{StringValue("number")},
		}, {
			caseName: "type boolean",
			q:        &TypeQuery{},
			n:        BoolValue(true),
			want:     []Node{StringValue("boolean")},
		}, {
			caseName: "type null",
			q:        &TypeQuery{},
			n:        Nil,
			want:     []Node{StringValue("null")},
		}, {
			caseName: "type bare value",
			q:        &TypeQuery{},
			n:        bareValueNode{Nil},
			want:     []Node{StringValue("value")},
		}, {
			caseName: "has key in map true",
			q:        &HasQuery{Key: "name"},
			n:        Map{"name": StringValue("test"), "age": NumberValue(30)},
			want:     []Node{BoolValue(true)},
		}, {
			caseName: "has key in map false",
			q:        &HasQuery{Key: "missing"},
			n:        Map{"name": StringValue("test")},
			want:     []Node{BoolValue(false)},
		}, {
			caseName: "has index in array true",
			q:        &HasQuery{Key: "1"},
			n:        ToArrayValues("a", "b", "c"),
			want:     []Node{BoolValue(true)},
		}, {
			caseName: "has index in array false",
			q:        &HasQuery{Key: "5"},
			n:        ToArrayValues("a", "b"),
			want:     []Node{BoolValue(false)},
		}, {
			caseName: "contains value in array true",
			q:        &ContainsQuery{Value: "b"},
			n:        ToArrayValues("a", "b", "c"),
			want:     []Node{BoolValue(true)},
		}, {
			caseName: "contains value in array false",
			q:        &ContainsQuery{Value: "x"},
			n:        ToArrayValues("a", "b", "c"),
			want:     []Node{BoolValue(false)},
		}, {
			caseName: "contains value in map true",
			q:        &ContainsQuery{Value: "test"},
			n:        Map{"name": StringValue("test"), "age": NumberValue(30)},
			want:     []Node{BoolValue(true)},
		}, {
			caseName: "contains value in map false",
			q:        &ContainsQuery{Value: "missing"},
			n:        Map{"name": StringValue("test")},
			want:     []Node{BoolValue(false)},
		}, {
			caseName: "contains substring in string true",
			q:        &ContainsQuery{Value: "est"},
			n:        StringValue("testing"),
			want:     []Node{BoolValue(true)},
		}, {
			caseName: "contains substring in string false",
			q:        &ContainsQuery{Value: "xyz"},
			n:        StringValue("testing"),
			want:     []Node{BoolValue(false)},
		}, {
			caseName: "contains empty string",
			q:        &ContainsQuery{Value: ""},
			n:        StringValue("test"),
			want:     []Node{BoolValue(true)},
		}, {
			caseName: "contains exact string match",
			q:        &ContainsQuery{Value: "test"},
			n:        StringValue("test"),
			want:     []Node{BoolValue(true)},
		}, {
			caseName: "contains in non-container type",
			q:        &ContainsQuery{Value: "test"},
			n:        NumberValue(42),
			want:     []Node{BoolValue(false)},
		}, {
			caseName: "first element",
			q:        &FirstQuery{},
			n:        ToArrayValues("a", "b", "c"),
			want:     []Node{StringValue("a")},
		}, {
			caseName: "first empty array",
			q:        &FirstQuery{},
			n:        Array{},
			want:     []Node{Nil},
		}, {
			caseName: "first non-array",
			q:        &FirstQuery{},
			n:        StringValue("test"),
			want:     []Node{Nil},
		}, {
			caseName: "last element",
			q:        &LastQuery{},
			n:        ToArrayValues("a", "b", "c"),
			want:     []Node{StringValue("c")},
		}, {
			caseName: "last empty array",
			q:        &LastQuery{},
			n:        Array{},
			want:     []Node{Nil},
		}, {
			caseName: "sort array of numbers",
			q:        &SortQuery{},
			n:        ToArrayValues(3, 1, 2),
			want:     []Node{ToArrayValues(1, 2, 3)},
		}, {
			caseName: "sort array of strings",
			q:        &SortQuery{},
			n:        ToArrayValues("c", "a", "b"),
			want:     []Node{ToArrayValues("a", "b", "c")},
		}, {
			caseName: "sort array of maps by field",
			q:        &SortQuery{Expr: ".age", Query: MapQuery("age")},
			n: Array{
				Map{"name": StringValue("Alice"), "age": NumberValue(30)},
				Map{"name": StringValue("Bob"), "age": NumberValue(25)},
				Map{"name": StringValue("Charlie"), "age": NumberValue(35)},
			},
			want: []Node{Array{
				Map{"name": StringValue("Bob"), "age": NumberValue(25)},
				Map{"name": StringValue("Alice"), "age": NumberValue(30)},
				Map{"name": StringValue("Charlie"), "age": NumberValue(35)},
			}},
		}, {
			caseName: "sort by a method expression",
			q:        &SortQuery{Expr: ".b.count()", Query: FilterQuery{MapQuery("b"), &CountQuery{}}},
			n:        Array{Map{"b": ToArrayValues(1, 2)}, Map{"b": ToArrayValues(1)}},
			want:     []Node{Array{Map{"b": ToArrayValues(1)}, Map{"b": ToArrayValues(1, 2)}}},
		}, {
			caseName: "sort by field on nil element sorts as null",
			q:        &SortQuery{Expr: ".age", Query: MapQuery("age")},
			n:        Array{Map{"age": NumberValue(1)}, nil},
			want:     []Node{Array{Nil, Map{"age": NumberValue(1)}}},
		}, {
			caseName: "first nil element is null",
			q:        &FirstQuery{},
			n:        Array{nil},
			want:     []Node{Nil},
		}, {
			caseName: "last nil element is null",
			q:        &LastQuery{},
			n:        Array{nil},
			want:     []Node{Nil},
		}, {
			caseName: "contains reads nil array element as null",
			q:        &ContainsQuery{Value: "1"},
			n:        Array{nil, ToValue(1)},
			want:     []Node{ToValue(true)},
		}, {
			caseName: "contains reads nil map value as null",
			q:        &ContainsQuery{Value: "1"},
			n:        Map{"a": nil},
			want:     []Node{ToValue(false)},
		}, {
			caseName: "contains ignores null elements",
			q:        &ContainsQuery{Value: ""},
			n:        Array{Nil, ToValue(1)},
			want:     []Node{ToValue(false)},
		}, {
			caseName: "contains ignores nil elements",
			q:        &ContainsQuery{Value: ""},
			n:        Array{nil},
			want:     []Node{ToValue(false)},
		}, {
			caseName: "contains ignores null map values",
			q:        &ContainsQuery{Value: ""},
			n:        Map{"a": Nil},
			want:     []Node{ToValue(false)},
		}, {
			caseName: "contains ignores nested arrays",
			q:        &ContainsQuery{Value: ""},
			n:        Array{Array{ToValue(1)}},
			want:     []Node{ToValue(false)},
		}, {
			caseName: "contains ignores nested maps",
			q:        &ContainsQuery{Value: ""},
			n:        Map{"a": Map{"b": ToValue(1)}},
			want:     []Node{ToValue(false)},
		}, {
			caseName: "contains matches an empty string element",
			q:        &ContainsQuery{Value: ""},
			n:        Array{StringValue("")},
			want:     []Node{ToValue(true)},
		}, {
			caseName: "sort array of maps by nested field",
			q: func() Query {
				q, _ := NewSortQuery(".meta.id")
				return q
			}(),
			n: Array{
				Map{"name": StringValue("A"), "meta": Map{"id": NumberValue(3)}},
				Map{"name": StringValue("B"), "meta": Map{"id": NumberValue(1)}},
				Map{"name": StringValue("C"), "meta": Map{"id": NumberValue(2)}},
			},
			want: []Node{Array{
				Map{"name": StringValue("B"), "meta": Map{"id": NumberValue(1)}},
				Map{"name": StringValue("C"), "meta": Map{"id": NumberValue(2)}},
				Map{"name": StringValue("A"), "meta": Map{"id": NumberValue(3)}},
			}},
		}, {
			caseName: "sort non-array",
			q:        &SortQuery{},
			n:        StringValue("test"),
			want:     []Node{StringValue("test")},
		}, {
			caseName: "sort Go nil elements",
			q:        &SortQuery{},
			n:        Array{ToValue(2), nil, Any{}, ToValue(1)},
			want:     []Node{Array{Nil, Nil, ToValue(1), ToValue(2)}},
		}, {
			caseName: "sort zero Any receiver",
			q:        &SortQuery{},
			n:        Any{},
			want:     []Node{Nil},
		}, {
			caseName: "rsort zero Any receiver",
			q:        &RSortQuery{},
			n:        Any{},
			want:     []Node{Nil},
		}, {
			caseName: "rsort Go nil elements",
			q:        &RSortQuery{},
			n:        Array{ToValue(2), nil, Any{}, ToValue(1)},
			want:     []Node{Array{ToValue(2), ToValue(1), Nil, Nil}},
		}, {
			caseName: "rsort array of numbers",
			q:        &RSortQuery{},
			n:        ToArrayValues(1, 3, 2),
			want:     []Node{ToArrayValues(3, 2, 1)},
		}, {
			caseName: "rsort array of maps by field",
			q:        &RSortQuery{Expr: ".age", Query: MapQuery("age")},
			n: Array{
				Map{"name": StringValue("Bob"), "age": NumberValue(25)},
				Map{"name": StringValue("Alice"), "age": NumberValue(30)},
				Map{"name": StringValue("Charlie"), "age": NumberValue(35)},
			},
			want: []Node{Array{
				Map{"name": StringValue("Charlie"), "age": NumberValue(35)},
				Map{"name": StringValue("Alice"), "age": NumberValue(30)},
				Map{"name": StringValue("Bob"), "age": NumberValue(25)},
			}},
		}, {
			caseName: "sort mixed types",
			q:        &SortQuery{},
			n:        Array{NumberValue(3), StringValue("a"), BoolValue(true), Nil, NumberValue(1), BoolValue(false)},
			want:     []Node{Array{Nil, BoolValue(false), BoolValue(true), NumberValue(1), NumberValue(3), StringValue("a")}},
		}, {
			caseName: "rsort mixed types",
			q:        &RSortQuery{},
			n:        Array{NumberValue(3), StringValue("a"), BoolValue(true), Nil, NumberValue(1), BoolValue(false)},
			want:     []Node{Array{StringValue("a"), NumberValue(3), NumberValue(1), BoolValue(true), BoolValue(false), Nil}},
		}, {
			caseName: "sort arrays and objects by type rank keeping order",
			q:        &SortQuery{},
			n:        Array{Map{"k": NumberValue(1)}, Array{NumberValue(2)}, StringValue("s"), Array{NumberValue(1)}, Map{"k": NumberValue(0)}},
			want:     []Node{Array{StringValue("s"), Array{NumberValue(2)}, Array{NumberValue(1)}, Map{"k": NumberValue(1)}, Map{"k": NumberValue(0)}}},
		}, {
			caseName: "sort keeps order of equal keys",
			q:        &SortQuery{Expr: ".age", Query: MapQuery("age")},
			n: Array{
				Map{"name": StringValue("A"), "age": NumberValue(2)},
				Map{"name": StringValue("B"), "age": NumberValue(1)},
				Map{"name": StringValue("C"), "age": NumberValue(2)},
			},
			want: []Node{Array{
				Map{"name": StringValue("B"), "age": NumberValue(1)},
				Map{"name": StringValue("A"), "age": NumberValue(2)},
				Map{"name": StringValue("C"), "age": NumberValue(2)},
			}},
		}, {
			caseName: "rsort keeps order of equal keys",
			q:        &RSortQuery{Expr: ".age", Query: MapQuery("age")},
			n: Array{
				Map{"name": StringValue("A"), "age": NumberValue(1)},
				Map{"name": StringValue("B"), "age": NumberValue(1)},
				Map{"name": StringValue("C"), "age": NumberValue(2)},
			},
			want: []Node{Array{
				Map{"name": StringValue("C"), "age": NumberValue(2)},
				Map{"name": StringValue("A"), "age": NumberValue(1)},
				Map{"name": StringValue("B"), "age": NumberValue(1)},
			}},
		}, {
			caseName: "sort missing key sorts as null",
			q:        &SortQuery{Expr: ".age", Query: MapQuery("age")},
			n:        Array{Map{"age": NumberValue(2)}, Map{"name": StringValue("x")}},
			want:     []Node{Array{Map{"name": StringValue("x")}, Map{"age": NumberValue(2)}}},
		}, {
			caseName: "sort key on a scalar element sorts as null",
			q:        &SortQuery{Expr: ".age", Query: MapQuery("age")},
			n:        Array{Map{"age": NumberValue(1)}, NumberValue(5)},
			want:     []Node{Array{NumberValue(5), Map{"age": NumberValue(1)}}},
		}, {
			caseName: "rsort key on a scalar element sorts as null",
			q:        &RSortQuery{Expr: ".age", Query: MapQuery("age")},
			n:        Array{NumberValue(5), Map{"age": NumberValue(1)}},
			want:     []Node{Array{Map{"age": NumberValue(1)}, NumberValue(5)}},
		}, {
			caseName: "sort key query error",
			q:        &SortQuery{Expr: "[..k == 1]", Query: SelectQuery{And{Comparator{WalkQuery("k"), EQ, ValueQuery{NumberValue(1)}}}}},
			n:        Array{Array{Map{"k": NumberValue(1), "z": Map{"k": NumberValue(2)}}}, Array{}},
			errstr:   `..k returns more than one value [1 2]`,
		}, {
			caseName: "rsort key query error",
			q:        &RSortQuery{Expr: "[..k == 1]", Query: SelectQuery{And{Comparator{WalkQuery("k"), EQ, ValueQuery{NumberValue(1)}}}}},
			n:        Array{Array{Map{"k": NumberValue(1), "z": Map{"k": NumberValue(2)}}}, Array{}},
			errstr:   `..k returns more than one value [1 2]`,
		}, {
			caseName: "sort key on a null element sorts as null",
			q:        &SortQuery{Expr: ".age", Query: MapQuery("age")},
			n:        Array{Map{"age": NumberValue(1)}, Nil},
			want:     []Node{Array{Nil, Map{"age": NumberValue(1)}}},
		}, {
			caseName: "sort nil containers by type",
			q:        &SortQuery{},
			n:        Array{NumberValue(1), Map(nil), StringValue("a"), Array(nil)},
			want:     []Node{Array{NumberValue(1), StringValue("a"), Array(nil), Map(nil)}},
		}, {
			caseName: "sort key ties keep nil and empty containers together",
			q:        &SortQuery{Expr: ".a", Query: MapQuery("a")},
			n:        Array{Map{"a": Map{}}, Map{"a": Nil}, Map{"a": Map(nil)}},
			want:     []Node{Array{Map{"a": Nil}, Map{"a": Map{}}, Map{"a": Map(nil)}}},
		}, {
			caseName: "sort nil pointer element as null",
			q:        &SortQuery{},
			n:        Array{ToValue(1), (*Map)(nil)},
			want:     []Node{Array{Nil, ToValue(1)}},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.caseName, func(t *testing.T) {
			got, err := tc.q.Exec(tc.n)
			if tc.errstr != "" {
				if err == nil {
					t.Fatalf("expected error %q, got nil", tc.errstr)
				}
				if err.Error() != tc.errstr {
					t.Errorf("got error %q; want %q", err.Error(), tc.errstr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("got %v; want %v", got, tc.want)
			}
		})
	}
}

// TestSortQueryNaN is separate because reflect.DeepEqual never matches NaN.
func TestSortQueryNaN(t *testing.T) {
	nan := math.NaN()
	testCases := []struct {
		caseName string
		q        Query
		want     []float64
	}{
		{
			caseName: "sort puts NaN first",
			q:        &SortQuery{},
			want:     []float64{nan, 1, 2, 3},
		}, {
			caseName: "rsort puts NaN last",
			q:        &RSortQuery{},
			want:     []float64{3, 2, 1, nan},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.caseName, func(t *testing.T) {
			got, err := tc.q.Exec(Array{NumberValue(3), NumberValue(nan), NumberValue(1), NumberValue(2)})
			if err != nil {
				t.Fatal(err)
			}
			a := got[0].Array()
			if len(a) != len(tc.want) {
				t.Fatalf("got %v; want %v", a, tc.want)
			}
			for i, w := range tc.want {
				g := a[i].Value().Float64()
				if g != w && !(math.IsNaN(g) && math.IsNaN(w)) {
					t.Errorf("got %v; want %v", a, tc.want)
					break
				}
			}
		})
	}
}

func TestNewMethodQuery(t *testing.T) {
	testCases := []struct {
		caseName string
		method   string
		args     []string
		wantType string
		errstr   string
	}{
		{
			caseName: "count method",
			method:   "count",
			args:     []string{},
			wantType: "*tree.CountQuery",
		},
		{
			caseName: "keys method",
			method:   "keys",
			args:     []string{},
			wantType: "*tree.KeysQuery",
		},
		{
			caseName: "values method",
			method:   "values",
			args:     []string{},
			wantType: "*tree.ValuesQuery",
		},
		{
			caseName: "empty method",
			method:   "empty",
			args:     []string{},
			wantType: "*tree.EmptyQuery",
		},
		{
			caseName: "type method",
			method:   "type",
			args:     []string{},
			wantType: "*tree.TypeQuery",
		},
		{
			caseName: "has method with arg",
			method:   "has",
			args:     []string{"key"},
			wantType: "*tree.HasQuery",
		},
		{
			caseName: "has method no args",
			method:   "has",
			args:     []string{},
			errstr:   "has() requires exactly 1 argument, got 0",
		},
		{
			caseName: "has method too many args",
			method:   "has",
			args:     []string{"key1", "key2"},
			errstr:   "has() requires exactly 1 argument, got 2",
		},
		{
			caseName: "contains method with arg",
			method:   "contains",
			args:     []string{"value"},
			wantType: "*tree.ContainsQuery",
		},
		{
			caseName: "contains method no args",
			method:   "contains",
			args:     []string{},
			errstr:   "contains() requires exactly 1 argument, got 0",
		},
		{
			caseName: "contains method too many args",
			method:   "contains",
			args:     []string{"value1", "value2"},
			errstr:   "contains() requires exactly 1 argument, got 2",
		},
		{
			caseName: "first method",
			method:   "first",
			args:     []string{},
			wantType: "*tree.FirstQuery",
		},
		{
			caseName: "last method",
			method:   "last",
			args:     []string{},
			wantType: "*tree.LastQuery",
		},
		{
			caseName: "sort method no args",
			method:   "sort",
			args:     []string{},
			wantType: "*tree.SortQuery",
		},
		{
			caseName: "sort method with field",
			method:   "sort",
			args:     []string{"name"},
			wantType: "*tree.SortQuery",
		},
		{
			caseName: "rsort method no args",
			method:   "rsort",
			args:     []string{},
			wantType: "*tree.RSortQuery",
		},
		{
			caseName: "rsort method with field",
			method:   "rsort",
			args:     []string{"name"},
			wantType: "*tree.RSortQuery",
		},
		{
			caseName: "sort method with a method argument",
			method:   "sort",
			args:     []string{"count()"},
			wantType: "*tree.SortQuery",
		},
		{
			caseName: "rsort method with a method argument",
			method:   "rsort",
			args:     []string{`has("a")`},
			wantType: "*tree.RSortQuery",
		},
		{
			caseName: "unknown method",
			method:   "unknown",
			args:     []string{},
			errstr:   "unknown method: unknown",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.caseName, func(t *testing.T) {
			got, err := NewMethodQuery(tc.method, tc.args...)
			if tc.errstr != "" {
				if err == nil {
					t.Fatalf("expected error %q, got nil", tc.errstr)
				}
				if err.Error() != tc.errstr {
					t.Errorf("got error %q; want %q", err.Error(), tc.errstr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got == nil {
				t.Fatal("got nil query")
			}

			gotType := reflect.TypeOf(got).String()
			if gotType != tc.wantType {
				t.Errorf("got type %s; want %s", gotType, tc.wantType)
			}
		})
	}
}

func TestRegisterNewMethodQueryFunc(t *testing.T) {
	// Save original state
	methodQueryMux.Lock()
	original := make(map[string]NewMethodQueryFunc)
	for k, v := range newMethodQueryFuncs {
		original[k] = v
	}
	methodQueryMux.Unlock()

	// Restore original state after test
	defer func() {
		methodQueryMux.Lock()
		newMethodQueryFuncs = original
		methodQueryMux.Unlock()
	}()

	// Test registration
	testFunc := func(args ...string) (Query, error) {
		return &CountQuery{}, nil
	}

	RegisterNewMethodQueryFunc("test", testFunc)

	// Test that it was registered
	query, err := NewMethodQuery("test")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := query.(*CountQuery); !ok {
		t.Errorf("expected CountQuery, got %T", query)
	}

	// Test overwriting existing registration
	testFunc2 := func(args ...string) (Query, error) {
		return &KeysQuery{}, nil
	}

	RegisterNewMethodQueryFunc("test", testFunc2)

	query2, err := NewMethodQuery("test")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := query2.(*KeysQuery); !ok {
		t.Errorf("expected KeysQuery, got %T", query2)
	}
}

func TestMethodQueryString(t *testing.T) {
	sortByName, err := NewSortQuery(".name")
	if err != nil {
		t.Fatalf("NewSortQuery: %v", err)
	}
	rsortByAge, err := NewRSortQuery(".age")
	if err != nil {
		t.Fatalf("NewRSortQuery: %v", err)
	}

	testCases := []struct {
		caseName string
		q        Query
		want     string
	}{
		{"count", &CountQuery{}, "count()"},
		{"keys", &KeysQuery{}, "keys()"},
		{"values", &ValuesQuery{}, "values()"},
		{"empty", &EmptyQuery{}, "empty()"},
		{"type", &TypeQuery{}, "type()"},
		{"has", &HasQuery{Key: "foo"}, `has("foo")`},
		{"contains", &ContainsQuery{Value: "bar"}, `contains("bar")`},
		{"first", &FirstQuery{}, "first()"},
		{"last", &LastQuery{}, "last()"},
		{"sort noarg", &SortQuery{}, "sort()"},
		{"sort expr", sortByName, `sort(".name")`},
		{"rsort noarg", &RSortQuery{}, "rsort()"},
		{"rsort expr", rsortByAge, `rsort(".age")`},
		{"has quote", &HasQuery{Key: `a"b`}, `has("a""b")`},
		{"has backslash", &HasQuery{Key: `a\`}, `has("a\")`},
		{"contains comma", &ContainsQuery{Value: "x,y"}, `contains("x,y")`},
		{"sort quoted key", &SortQuery{Expr: `."b c"`, Query: MapQuery("b c")}, `sort(".""b c""")`},
	}

	for _, tc := range testCases {
		t.Run(tc.caseName, func(t *testing.T) {
			type stringer interface{ String() string }
			s, ok := tc.q.(stringer)
			if !ok {
				t.Fatalf("%T does not implement String()", tc.q)
			}
			if got := s.String(); got != tc.want {
				t.Errorf("String() = %q, want %q", got, tc.want)
			}
		})
	}
}
