package tree

import (
	"reflect"
	"testing"
)

func Test_MethodQuery(t *testing.T) {
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

func Test_NewMethodQuery(t *testing.T) {
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

func Test_RegisterNewMethodQueryFunc(t *testing.T) {
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
