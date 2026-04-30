package tree

import (
	"errors"
	"reflect"
	"testing"
)

func Test_Edit(t *testing.T) {
	testCases := []struct {
		caseName string
		n        Node
		expr     string
		want     Node
		errstr   string
	}{
		// --- Set operations on Map ---
		{
			caseName: "set empty map on map key",
			n:        Map{},
			expr:     `.store = {}`,
			want:     Map{"store": Map{}},
		}, {
			caseName: "set empty map on map key without spaces",
			n:        Map{},
			expr:     `.store={}`, // NOTE: trim spaces
			want:     Map{"store": Map{}},
		}, {
			caseName: "set nested map key",
			n:        Map{},
			expr:     `.store.book = {}`,
			want:     Map{"store": Map{"book": Map{}}},
		}, {
			caseName: "set array of maps with double quotes",
			n:        Map{},
			expr:     `.store.pen = [{"color":"red"},{"color":"blue"}]`,
			want: Map{
				"store": Map{
					"pen": Array{
						Map{"color": StringValue("red")},
						Map{"color": StringValue("blue")},
					},
				},
			},
		}, {
			caseName: "set array of maps with single quotes",
			n:        Map{},
			expr:     `.store.pen = [{'color':'red'},{'color':'blue'}]`,
			want: Map{
				"store": Map{
					"pen": Array{
						Map{"color": StringValue("red")},
						Map{"color": StringValue("blue")},
					},
				},
			},
		}, {
			caseName: "set on non-map returns error",
			n:        StringValue("str"),
			expr:     `.key = {}`,
			errstr:   `cannot index array with "key"`,
		},

		// --- Append error cases ---
		{
			caseName: "append to map root returns error",
			n:        Map{"key": StringValue("str")},
			expr:     `. += {}`,
			errstr:   "cannot append to .",
		}, {
			caseName: "append to string root returns error",
			n:        StringValue("str"),
			expr:     `. += {}`,
			errstr:   "cannot append to .",
		}, {
			caseName: "append to map string value returns error",
			n:        Map{"key": StringValue("str")},
			expr:     `.key += {}`,
			errstr:   `cannot append to "key"`,
		}, {
			caseName: "append to string value returns error",
			n:        StringValue("str"),
			expr:     `.key += {}`,
			errstr:   `cannot append to "key"`,
		},

		// --- Set operations on Array ---
		{
			caseName: "set on array by index",
			n:        Array{},
			expr:     `[0] = "red"`,
			want:     Array{StringValue("red")},
		}, {
			caseName: "set on nested array by index",
			n:        Array{},
			expr:     `[0][1] = "red"`,
			want:     Array{Array{nil, StringValue("red")}},
		}, {
			caseName: "set on non-array by index returns error",
			n:        StringValue("str"),
			expr:     `[0] = "red"`,
			errstr:   `cannot index array with 0`,
		}, {
			caseName: "set on array by dot-index",
			n:        Array{},
			expr:     `.0 = "red"`,
			want:     Array{StringValue("red")},
		}, {
			caseName: "set on array by dot-index creates map",
			n:        Array{},
			expr:     `.0.1 = "red"`,
			want:     Array{Map{"1": StringValue("red")}},
		}, {
			caseName: "set root to value",
			n:        Array{},
			expr:     `. = "red"`,
			want:     StringValue("red"),
		},

		// --- Append operations ---
		{
			caseName: "append to new map key creates array",
			n:        Map{},
			expr:     `.colors += "red"`,
			want:     Map{"colors": Array{StringValue("red")}},
		}, {
			caseName: "append to new map key without spaces",
			n:        Map{},
			expr:     `.colors+="red"`, // NOTE: trim spaces
			want:     Map{"colors": Array{StringValue("red")}},
		}, {
			caseName: "append to existing array in map",
			n:        Map{"colors": Array{StringValue("red"), StringValue("green")}},
			expr:     `.colors += "blue"`,
			want:     Map{"colors": Array{StringValue("red"), StringValue("green"), StringValue("blue")}},
		}, {
			caseName: "append to nested array by index",
			n:        Array{Array{StringValue("red")}},
			expr:     `[0] += "blue"`,
			want:     Array{Array{StringValue("red"), StringValue("blue")}},
		}, {
			caseName: "append to nested array by index with gap",
			n:        Array{Array{StringValue("red")}},
			expr:     `[2] += "blue"`,
			want:     Array{Array{StringValue("red")}, nil, Array{StringValue("blue")}},
		}, {
			caseName: "append to non-array element returns error",
			n:        Array{StringValue("red")},
			expr:     `[0] += "blue"`,
			errstr:   `cannot append to array with 0`,
		}, {
			caseName: "append to non-array root returns error",
			n:        StringValue("red"),
			expr:     `[0] += "blue"`,
			errstr:   `cannot append to array with 0`,
		}, {
			caseName: "append to array root",
			n:        Array{},
			expr:     `. += "red"`,
			want:     Array{StringValue("red")},
		},

		// --- Delete operations ---
		{
			caseName: "delete map key",
			n:        Map{"key1": StringValue("value1"), "key2": StringValue("value2")},
			expr:     `.key1 ^?`,
			want:     Map{"key2": StringValue("value2")},
		}, {
			caseName: "delete map key without spaces",
			n:        Map{"key1": StringValue("value1"), "key2": StringValue("value2")},
			expr:     `.key1^?`, // NOTE: trim spaces
			want:     Map{"key2": StringValue("value2")},
		}, {
			caseName: "delete array element by bracket-index",
			n:        Array{StringValue("red")},
			expr:     `[0] ^?`,
			want:     Array{},
		}, {
			caseName: "delete array element by dot-index",
			n:        Array{StringValue("red")},
			expr:     `.0 ^?`,
			want:     Array{},
		}, {
			caseName: "delete root returns error",
			n:        Map{},
			expr:     `. ^?`,
			errstr:   "cannot delete .",
		}, {
			caseName: "delete key on non-map returns error",
			n:        StringValue("str"),
			expr:     `.key ^?`,
			errstr:   `cannot delete "key"`,
		}, {
			caseName: "delete index on non-array returns error",
			n:        StringValue("str"),
			expr:     `[0] ^?`,
			errstr:   `cannot delete array with 0`,
		},

		// --- Recursive walk operations ---
		{
			caseName: "walk set on nested name fields",
			n: Map{
				"users": Array{
					Map{"name": StringValue("one"), "class": StringValue("A")},
					Map{"name": StringValue("two"), "job": Map{"name": StringValue("engineer")}},
				},
			},
			expr: `..name = "NAME"`,
			want: Map{
				"users": Array{
					Map{"name": StringValue("NAME"), "class": StringValue("A")},
					Map{"name": StringValue("NAME"), "job": Map{"name": StringValue("NAME")}},
				},
			},
		}, {
			caseName: "walk append to array",
			n: Map{
				"numbers": Array{
					NumberValue(1),
					NumberValue(2),
				},
			},
			expr: `..numbers += 3`,
			want: Map{
				"numbers": Array{
					NumberValue(1),
					NumberValue(2),
					NumberValue(3),
				},
			},
		}, {
			caseName: "walk delete key",
			n: Map{
				"users": Array{
					Map{"name": StringValue("one"), "class": StringValue("A")},
					Map{"name": StringValue("two"), "class": StringValue("B")},
				},
			},
			expr: `..class ^?`,
			want: Map{
				"users": Array{
					Map{"name": StringValue("one")},
					Map{"name": StringValue("two")},
				},
			},
		},

		// --- Filtered edit operations ---
		{
			caseName: "filtered set on all array elements",
			n: Map{
				"users": Array{
					Map{"name": StringValue("one"), "class": StringValue("A")},
					Map{"name": StringValue("two"), "class": StringValue("A")},
				},
			},
			expr: `..users[].class = "B"`,
			want: Map{
				"users": Array{
					Map{"name": StringValue("one"), "class": StringValue("B")},
					Map{"name": StringValue("two"), "class": StringValue("B")},
				},
			},
		}, {
			caseName: "filtered set with condition",
			n: Map{
				"users": Array{
					Map{"name": StringValue("one"), "class": StringValue("A")},
					Map{"name": StringValue("two"), "class": StringValue("A")},
				},
			},
			expr: `..users[.name == "two"].class = "B"`,
			want: Map{
				"users": Array{
					Map{"name": StringValue("one"), "class": StringValue("A")},
					Map{"name": StringValue("two"), "class": StringValue("B")},
				},
			},
		}, {
			caseName: "filtered set with condition without spaces",
			n: Map{
				"users": Array{
					Map{"name": StringValue("one"), "class": StringValue("A")},
					Map{"name": StringValue("two"), "class": StringValue("A")},
				},
			},
			expr: `..users[.name=="two"].class="B"`, // NOTE: trim spaces
			want: Map{
				"users": Array{
					Map{"name": StringValue("one"), "class": StringValue("A")},
					Map{"name": StringValue("two"), "class": StringValue("B")},
				},
			},
		},

		// --- Pipe operations ---
		{
			caseName: "pipe with index set",
			n: Map{
				"users": Array{
					Map{"name": StringValue("one"), "class": StringValue("A")},
				},
			},
			expr: `.users[] | [0].name = "ONE"`,
			want: Map{
				"users": Array{
					Map{"name": StringValue("ONE"), "class": StringValue("A")},
				},
			},
		},

		// --- Deep nested Array write-back (mutations through value-typed Arrays) ---
		{
			caseName: "append to array nested in map nested in array",
			n: Array{
				Map{"items": Array{StringValue("a")}},
			},
			expr: `[0].items += "b"`,
			want: Array{
				Map{"items": Array{StringValue("a"), StringValue("b")}},
			},
		}, {
			caseName: "append multiple levels deep array",
			n: Map{
				"l1": Array{
					Map{
						"l2": Array{StringValue("x")},
					},
				},
			},
			expr: `.l1[0].l2 += "y"`,
			want: Map{
				"l1": Array{
					Map{
						"l2": Array{StringValue("x"), StringValue("y")},
					},
				},
			},
		}, {
			caseName: "delete from array nested in array",
			n: Array{
				Array{StringValue("a"), StringValue("b"), StringValue("c")},
			},
			expr: `[0][1] ^?`,
			want: Array{
				Array{StringValue("a"), StringValue("c")},
			},
		}, {
			caseName: "set on deeply nested array path",
			n: Map{
				"a": Array{
					Map{
						"b": Array{
							Map{"c": StringValue("old")},
						},
					},
				},
			},
			expr: `.a[0].b[0].c = "new"`,
			want: Map{
				"a": Array{
					Map{
						"b": Array{
							Map{"c": StringValue("new")},
						},
					},
				},
			},
		}, {
			caseName: "append to root array with existing elements",
			n:        Array{StringValue("a"), StringValue("b")},
			expr:     `. += "c"`,
			want:     Array{StringValue("a"), StringValue("b"), StringValue("c")},
		}, {
			caseName: "delete from root array",
			n:        Array{StringValue("a"), StringValue("b"), StringValue("c")},
			expr:     `[1] ^?`,
			want:     Array{StringValue("a"), StringValue("c")},
		}, {
			caseName: "set grows array in nested structure",
			n: Map{
				"data": Array{StringValue("x")},
			},
			expr: `.data[3] = "y"`,
			want: Map{
				"data": Array{StringValue("x"), nil, nil, StringValue("y")},
			},
		}, {
			caseName: "walk append on nested arrays",
			n: Map{
				"groups": Array{
					Map{"tags": Array{StringValue("a")}},
					Map{"tags": Array{StringValue("b")}},
				},
			},
			expr: `..tags += "z"`,
			want: Map{
				"groups": Array{
					Map{"tags": Array{StringValue("a"), StringValue("z")}},
					Map{"tags": Array{StringValue("b"), StringValue("z")}},
				},
			},
		}, {
			caseName: "walk delete from nested arrays",
			n: Map{
				"rows": Array{
					Map{"cols": Array{NumberValue(1), NumberValue(2)}},
					Map{"cols": Array{NumberValue(3), NumberValue(4)}},
				},
			},
			expr: `..cols[0] ^?`,
			want: Map{
				"rows": Array{
					Map{"cols": Array{NumberValue(2)}},
					Map{"cols": Array{NumberValue(4)}},
				},
			},
		}, {
			caseName: "filtered append on array elements",
			n: Map{
				"items": Array{
					Map{"type": StringValue("list"), "values": Array{NumberValue(1)}},
					Map{"type": StringValue("single"), "values": Array{NumberValue(2)}},
				},
			},
			expr: `.items[.type == "list"].values += 9`,
			want: Map{
				"items": Array{
					Map{"type": StringValue("list"), "values": Array{NumberValue(1), NumberValue(9)}},
					Map{"type": StringValue("single"), "values": Array{NumberValue(2)}},
				},
			},
		}, {
			caseName: "set on array in array in array",
			n:        Array{Array{Array{StringValue("deep")}}},
			expr:     `[0][0][0] = "new"`,
			want:     Array{Array{Array{StringValue("new")}}},
		}, {
			caseName: "append to array in array in array",
			n:        Array{Array{Array{StringValue("a")}}},
			expr:     `[0][0] += "b"`,
			want:     Array{Array{Array{StringValue("a"), StringValue("b")}}},
		}, {
			caseName: "delete from array in array in array",
			n:        Array{Array{Array{StringValue("a"), StringValue("b")}}},
			expr:     `[0][0][0] ^?`,
			want:     Array{Array{Array{StringValue("b")}}},
		}, {
			caseName: "set replaces entire nested array",
			n: Map{
				"data": Array{NumberValue(1), NumberValue(2)},
			},
			expr: `.data = [3, 4, 5]`,
			want: Map{
				"data": Array{NumberValue(3), NumberValue(4), NumberValue(5)},
			},
		}, {
			caseName: "edit with nil element in array",
			n:        Array{nil, StringValue("b")},
			expr:     `[0] = "a"`,
			want:     Array{StringValue("a"), StringValue("b")},
		}, {
			caseName: "delete middle element from three-element array",
			n:        Array{StringValue("a"), StringValue("b"), StringValue("c")},
			expr:     `[1] ^?`,
			want:     Array{StringValue("a"), StringValue("c")},
		}, {
			caseName: "append to empty nested array",
			n: Map{
				"list": Array{},
			},
			expr: `.list += "first"`,
			want: Map{
				"list": Array{StringValue("first")},
			},
		},

		// --- Edit-level parse / unmarshal errors ---
		{
			caseName: "invalid query expression returns parse error",
			n:        Map{},
			expr:     `[ = 1`,
			errstr:   `syntax error: no right brackets: "[ "`,
		}, {
			caseName: "invalid YAML on RHS returns unmarshal error",
			n:        Map{},
			expr:     `.a = }not yaml`,
			errstr:   `yaml: did not find expected node content`,
		},

		// --- Intermediate-step edge cases in resolveAndEdit ---
		{
			caseName: "pipe-prefixed query slurps root then edits wrapper",
			n:        Map{"a": StringValue("v")},
			expr:     `| [0] = "x"`,
			want:     Map{"a": StringValue("v")},
		}, {
			caseName: "NopQuery as intermediate step uses default branch",
			n:        Map{},
			expr:     `..[0] = 1`,
			want:     Map{"0": NumberValue(1)},
		}, {
			caseName: "missing map key with non-collection next is a no-op",
			n:        Map{},
			expr:     `.missing[].x = 1`,
			want:     Map{},
		}, {
			caseName: "non-numeric map key on Array is a no-op",
			n:        Map{"arr": Array{NumberValue(1)}},
			expr:     `.arr.badkey.x = 1`,
			want:     Map{"arr": Array{NumberValue(1)}},
		}, {
			caseName: "nil element at Array MapQuery step with non-collection next is a no-op",
			n:        Map{"arr": Array{nil, nil}},
			expr:     `.arr.0[].x = 1`,
			want:     Map{"arr": Array{nil, nil}},
		}, {
			caseName: "nil element at Array ArrayQuery step with non-collection next is a no-op",
			n:        Array{nil, nil},
			expr:     `[0][].x = 1`,
			want:     Array{nil, nil},
		}, {
			caseName: "MapQuery intermediate on non-collection is a no-op",
			n:        Map{"s": StringValue("hi")},
			expr:     `.s.x.y = 1`,
			want:     Map{"s": StringValue("hi")},
		}, {
			caseName: "ArrayQuery intermediate on non-collection is a no-op",
			n:        Map{"s": StringValue("hi")},
			expr:     `.s[0].x = 1`,
			want:     Map{"s": StringValue("hi")},
		}, {
			caseName: "ArrayQuery intermediate on Map uses EditorNode branch",
			n:        Map{"m": Map{"0": Map{"x": StringValue("old")}}},
			expr:     `.m[0].x = "new"`,
			want:     Map{"m": Map{"0": Map{"x": StringValue("new")}}},
		}, {
			caseName: "ArrayQuery intermediate on Map creates missing key via emptyIntermediate",
			n:        Map{"arr": Map{"x": StringValue("v")}},
			expr:     `.arr[0].y = 1`,
			want:     Map{"arr": Map{"0": Map{"y": NumberValue(1)}, "x": StringValue("v")}},
		}, {
			caseName: "SelectQuery intermediate iterates Map values",
			n: Map{
				"items": Map{
					"a": Map{"active": BoolValue(true), "name": StringValue("old")},
					"b": Map{"active": BoolValue(false), "name": StringValue("old2")},
				},
			},
			expr: `.items[].name = "new"`,
			want: Map{
				"items": Map{
					"a": Map{"active": BoolValue(true), "name": StringValue("new")},
					"b": Map{"active": BoolValue(false), "name": StringValue("new")},
				},
			},
		},

		// --- Recursion error propagation from terminal step ---
		{
			caseName: "error from MapQuery terminal propagates through map step",
			n:        Map{"s": StringValue("hi")},
			expr:     `.s.x = 1`,
			errstr:   `cannot index array with "x"`,
		}, {
			caseName: "error from MapQuery terminal propagates through Array map step",
			n:        Map{"arr": Array{StringValue("hi")}},
			expr:     `.arr.0.x = 1`,
			errstr:   `cannot index array with "x"`,
		}, {
			caseName: "error from MapQuery terminal propagates through ArrayQuery step",
			n:        Array{StringValue("hi")},
			expr:     `[0].x = 1`,
			errstr:   `cannot index array with "x"`,
		}, {
			caseName: "error from SelectQuery Array branch propagates through iteration",
			n:        Map{"items": Array{StringValue("oops")}},
			expr:     `.items[].name = "x"`,
			errstr:   `cannot index array with "name"`,
		}, {
			caseName: "error from SelectQuery Map branch propagates through iteration",
			n:        Map{"items": Map{"a": StringValue("oops")}},
			expr:     `.items[].name = "x"`,
			errstr:   `cannot index array with "name"`,
		}, {
			caseName: "error from WalkQuery intermediate recursion propagates up",
			n:        Map{"outer": Map{"name": StringValue("str")}},
			expr:     `..name.x = 1`,
			errstr:   `cannot index array with "x"`,
		}, {
			caseName: "error from default-branch terminal step propagates up",
			n:        Array{StringValue("hi")},
			expr:     `..[0].x = 1`,
			errstr:   `cannot index array with "x"`,
		}, {
			caseName: "EditorNode ArrayQuery intermediate with missing key and non-collection next is a no-op",
			n:        Map{"arr": Map{"x": StringValue("v")}},
			expr:     `.arr[0][].y = 1`,
			want:     Map{"arr": Map{"x": StringValue("v")}},
		}, {
			caseName: "error from EditorNode ArrayQuery branch recursion propagates up",
			n:        Map{"arr": Map{"0": StringValue("oops")}},
			expr:     `.arr[0].x = 1`,
			errstr:   `cannot index array with "x"`,
		}, {
			caseName: "default-branch intermediate Exec error propagates up",
			n:        Map{"items": Map{"x": StringValue("v")}},
			expr:     `.items[1:2].x = 1`,
			errstr:   `cannot index array with range 1:2`,
		}, {
			caseName: "slurp left.Exec error propagates up",
			n:        Map{"items": Map{"x": StringValue("v")}},
			expr:     `.items[1:2] | .x = 1`,
			errstr:   `cannot index array with range 1:2`,
		}, {
			caseName: "terminal SelectQuery is an unsupported edit query",
			n:        Map{"items": Array{NumberValue(1)}},
			expr:     `.items[] = "new"`,
			errstr:   `syntax error: unsupported edit query: []`,
		}, {
			caseName: "SelectQuery intermediate on non-collection is a no-op",
			n:        Map{"s": StringValue("hi")},
			expr:     `.s[].x = 1`,
			want:     Map{"s": StringValue("hi")},
		}, {
			caseName: "SelectQuery with selector iterates Map values matching comparator",
			n: Map{
				"items": Map{
					"a": Map{"x": NumberValue(1), "name": StringValue("old")},
					"b": Map{"x": NumberValue(2), "name": StringValue("old2")},
				},
			},
			expr: `.items[.x == 1].name = "new"`,
			want: Map{
				"items": Map{
					"a": Map{"x": NumberValue(1), "name": StringValue("new")},
					"b": Map{"x": NumberValue(2), "name": StringValue("old2")},
				},
			},
		},

		// --- Negative array index ---
		{
			caseName: "set on negative index targets last element",
			n:        Map{"arr": Array{NumberValue(1), NumberValue(2), NumberValue(3)}},
			expr:     `.arr[-1] = 99`,
			want:     Map{"arr": Array{NumberValue(1), NumberValue(2), NumberValue(99)}},
		}, {
			caseName: "delete on negative index removes last element",
			n:        Map{"arr": Array{NumberValue(1), NumberValue(2), NumberValue(3)}},
			expr:     `.arr[-1] ^?`,
			want:     Map{"arr": Array{NumberValue(1), NumberValue(2)}},
		}, {
			caseName: "intermediate negative index walks last element",
			n:        Map{"arr": Array{Map{"k": NumberValue(1)}, Map{"k": NumberValue(2)}}},
			expr:     `.arr[-1].k = 99`,
			want:     Map{"arr": Array{Map{"k": NumberValue(1)}, Map{"k": NumberValue(99)}}},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.caseName, func(t *testing.T) {
			err := Edit(&(tc.n), tc.expr)
			if tc.errstr != "" {
				if err == nil {
					t.Fatalf("no error; want %s", tc.errstr)
				}
				if err.Error() != tc.errstr {
					t.Errorf("got error %s; want %s", err.Error(), tc.errstr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %+v", err)
			}
			got := tc.n
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("got %#v; want %#v", got, tc.want)
			}
		})
	}
}

func Test_Edit_resolveSelectStep_SelectorError(t *testing.T) {
	selector := &testSelectorDelegator{
		matchesFunc: func(Node) (bool, error) {
			return false, errors.New("selector boom")
		},
	}
	fq := FilterQuery{
		MapQuery("items"),
		SelectQuery{Selector: selector},
		MapQuery("name"),
	}
	testCases := []struct {
		caseName string
		n        Node
	}{
		{
			caseName: "Array branch propagates selector error",
			n:        Map{"items": Array{Map{"name": StringValue("a")}}},
		}, {
			caseName: "Map branch propagates selector error",
			n:        Map{"items": Map{"a": Map{"name": StringValue("a")}}},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.caseName, func(t *testing.T) {
			err := resolveAndEdit(&tc.n, fq, 0, "=", StringValue("x"))
			if err == nil {
				t.Fatalf("no error; want %q", "selector boom")
			}
			if err.Error() != "selector boom" {
				t.Errorf("got error %q; want %q", err.Error(), "selector boom")
			}
		})
	}
}

// FuzzEdit exercises the Edit expression parser and evaluator with
// arbitrary input against a fresh empty Map. It only requires that
// invalid expressions return an error rather than panicking.
func FuzzEdit(f *testing.F) {
	seeds := []string{
		`.a = 1`,
		`.a = "x"`,
		`.a += 1`,
		`.a.b = true`,
		`.a = [1, 2, 3]`,
		`.a = {"k": "v"}`,
		`delete .a`,
	}
	for _, s := range seeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, expr string) {
		var n Node = Map{}
		_ = Edit(&n, expr)
	})
}

// BenchmarkEdit measures Edit performance across different operations
// and tree depths. These benchmarks serve as the baseline for
// comparing the arrayHolder implementation against future alternatives.
func BenchmarkEdit(b *testing.B) {
	testCases := []struct {
		caseName string
		setup    func() Node
		expr     string
	}{
		{
			caseName: "set_shallow",
			setup:    func() Node { return mustBenchNode(b) },
			expr:     `.store.bicycle.color = "blue"`,
		},
		{
			caseName: "set_deep_array",
			setup:    func() Node { return mustBenchNode(b) },
			expr:     `.store.book[2].title = "New Title"`,
		},
		{
			caseName: "append_to_array",
			setup:    func() Node { return mustBenchNode(b) },
			expr:     `.store.book += {"author":"New","title":"Book"}`,
		},
		{
			caseName: "delete_from_array",
			setup:    func() Node { return mustBenchNode(b) },
			expr:     `.store.book[0] ^?`,
		},
		{
			caseName: "walk_set",
			setup:    func() Node { return mustBenchNode(b) },
			expr:     `..price = 0`,
		},
		{
			caseName: "filtered_set",
			setup:    func() Node { return mustBenchNode(b) },
			expr:     `.store.book[.category == "fiction"].price = 9.99`,
		},
		{
			caseName: "nested_array_append",
			setup:    func() Node { return mustBenchNode(b) },
			expr:     `.store.book[0].tags += {"name":"new","value":"tag"}`,
		},
	}
	for _, tc := range testCases {
		b.Run(tc.caseName, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				n := tc.setup()
				if err := Edit(&n, tc.expr); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
