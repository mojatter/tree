package tree

import "testing"

func TestMethodQueryString(t *testing.T) {
	sortByName, err := NewSortQuery(".name")
	if err != nil {
		t.Fatalf("NewSortQuery: %v", err)
	}
	rsortByAge, err := NewRSortQuery(".age")
	if err != nil {
		t.Fatalf("NewRSortQuery: %v", err)
	}

	tests := []struct {
		name string
		q    Query
		want string
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			type stringer interface{ String() string }
			s, ok := tt.q.(stringer)
			if !ok {
				t.Fatalf("%T does not implement String()", tt.q)
			}
			if got := s.String(); got != tt.want {
				t.Errorf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}
