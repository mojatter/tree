package tree

import (
	"errors"
	"reflect"
	"testing"
)

// valueEditorNode is a value-typed EditorNode without reference identity.
type valueEditorNode struct{ EditorNode }

// pointerEditorNode is a custom EditorNode used through a pointer.
type pointerEditorNode struct{ EditorNode }

// setFailingEditorNode is an EditorNode whose Set always fails.
type setFailingEditorNode struct{ EditorNode }

func (setFailingEditorNode) Set(any, Node) error { return errors.New("set failed") }

// deleteFailingEditorNode is an EditorNode whose Delete always fails.
type deleteFailingEditorNode struct{ EditorNode }

func (deleteFailingEditorNode) Delete(any) error { return errors.New("delete failed") }

// mapHidingEditorNode is an EditorNode whose Map returns nil, like a struct-backed node.
type mapHidingEditorNode struct{ EditorNode }

func (mapHidingEditorNode) Map() Map { return nil }

func TestEdit(t *testing.T) {
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
			caseName: "append on key starting with or",
			n:        Map{"orders": Array{}},
			expr:     `.orders += 1`,
			want:     Map{"orders": Array{NumberValue(1)}},
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
			errstr:   `syntax error: no right brackets: "["`,
		}, {
			caseName: "invalid YAML on RHS returns unmarshal error",
			n:        Map{},
			expr:     `.a = }not yaml`,
			errstr:   `yaml: did not find expected node content`,
		}, {
			caseName: "set value containing equals sign",
			n:        Map{},
			expr:     `.url = "http://x?a=1"`,
			want:     Map{"url": StringValue("http://x?a=1")},
		}, {
			caseName: "set plain value containing equals sign",
			n:        Map{},
			expr:     `.s = a=b`,
			want:     Map{"s": StringValue("a=b")},
		}, {
			caseName: "set value containing append operator",
			n:        Map{},
			expr:     `.s = "a+=b"`,
			want:     Map{"s": StringValue("a+=b")},
		}, {
			caseName: "append value containing equals sign",
			n:        Map{"a": Array{}},
			expr:     `.a += "k=v"`,
			want:     Map{"a": Array{StringValue("k=v")}},
		}, {
			caseName: "set value containing delete operator",
			n:        Map{},
			expr:     `.s = "x^?"`,
			want:     Map{"s": StringValue("x^?")},
		}, {
			caseName: "set quoted key containing operators",
			n:        Map{},
			expr:     `."a+=b" = 1`,
			want:     Map{"a+=b": NumberValue(1)},
		}, {
			caseName: "set key selected by comparison",
			n:        Array{Map{"k": StringValue("a")}, Map{"k": StringValue("b")}},
			expr:     `[.k == "b"].v = 1`,
			want:     Array{Map{"k": StringValue("a")}, Map{"k": StringValue("b"), "v": NumberValue(1)}},
		}, {
			caseName: "block scalar RHS keeps trailing newline",
			n:        Map{},
			expr:     ".a = |\n  foo\n",
			want:     Map{"a": StringValue("foo\n")},
		}, {
			caseName: "multi-line mapping RHS on following lines",
			n:        Map{},
			expr:     ".a =\n  x: 1\n  y: 2",
			want:     Map{"a": Map{"x": NumberValue(1), "y": NumberValue(2)}},
		}, {
			caseName: "RHS with tab after operator",
			n:        Map{},
			expr:     ".a=\t[1]",
			want:     Map{"a": Array{NumberValue(1)}},
		}, {
			caseName: "equals inside method args is not an operator",
			n:        Array{Map{"a=b": NumberValue(1)}},
			expr:     `[has(a=b)].v = 1`,
			want:     Array{Map{"a=b": NumberValue(1), "v": NumberValue(1)}},
		}, {
			caseName: "unbalanced paren inside method args",
			n:        Array{Map{"a(b": NumberValue(1)}},
			expr:     `[has(a(b)].v = 1`,
			want:     Array{Map{"a(b": NumberValue(1), "v": NumberValue(1)}},
		}, {
			caseName: "CSV-quoted method arg with backslash before doubled quote",
			n:        Array{Map{`a\"b`: NumberValue(1)}},
			expr:     `[has("a\""b")].v = 1`,
			want:     Array{Map{`a\"b`: NumberValue(1), "v": NumberValue(1)}},
		}, {
			caseName: "escaped quote inside grouping parens keeps lexer rules",
			n:        Array{Map{"k": StringValue(`a"=)`)}},
			expr:     `[(.k == "a\"=)")].v = 1`,
			want:     Array{Map{"k": StringValue(`a"=)`), "v": NumberValue(1)}},
		}, {
			caseName: "escaped quote inside parens after and keeps lexer rules",
			n:        Array{Map{"a": NumberValue(1), "k": StringValue(`a"=)`)}},
			expr:     `[.a == 1 and(.k == "a\"=)")].v = 1`,
			want:     Array{Map{"a": NumberValue(1), "k": StringValue(`a"=)`), "v": NumberValue(1)}},
		}, {
			caseName: "escaped quote inside parens after or keeps lexer rules",
			n:        Array{Map{"a": NumberValue(1), "k": StringValue(`a"=)`)}},
			expr:     `[.a == 2 or(.k == "a\"=)")].v = 1`,
			want:     Array{Map{"a": NumberValue(1), "k": StringValue(`a"=)`), "v": NumberValue(1)}},
		}, {
			caseName: "CSV-quoted method arg inside grouping parens",
			n:        Array{Map{`a\`: NumberValue(1)}},
			expr:     `[(has("a\") == true)].v = "x)"`,
			want:     Array{Map{`a\`: NumberValue(1), "v": StringValue("x)")}},
		}, {
			caseName: "CSV-quoted method arg inside parens after or",
			n:        Array{Map{`a\"=)`: NumberValue(1)}},
			expr:     `[.k == "x" or(has("a\""=)"))].v = 1`,
			want:     Array{Map{`a\"=)`: NumberValue(1), "v": NumberValue(1)}},
		}, {
			caseName: "unterminated CSV-quoted method arg is an error, not a match-all filter",
			n:        Array{Map{"v": NumberValue(0)}},
			expr:     `[has("a\"b")].v = 1`,
			errstr:   `syntax error: invalid edit expression "[has(\"a\\\"b\")].v = 1"`,
		}, {
			caseName: "escaped quote in parens after an uppercase-led key keeps lexer rules",
			n:        Array{Map{"Xa": NumberValue(1)}},
			expr:     `[.Xa("a\"=)")].v = 1`,
			want:     Array{Map{"Xa": NumberValue(1), "v": NumberValue(1)}},
		}, {
			caseName: "exponent sign on the LHS is not an append",
			n:        Array{Map{"x": NumberValue(1000)}},
			expr:     `[.x == 1e+3].v = 1`,
			want:     Array{Map{"x": NumberValue(1000), "v": NumberValue(1)}},
		}, {
			caseName: "second YAML document on RHS returns syntax error",
			n:        Map{},
			expr:     ".a = 1\n---\n2",
			errstr:   `syntax error: multiple YAML documents in edit value "1\n---\n2"`,
		}, {
			caseName: "trailing content after quoted RHS returns yaml error",
			n:        Map{},
			expr:     ".a = 'x'\n.b = 2",
			errstr:   `yaml: line 1: did not find expected <document start>`,
		}, {
			caseName: "trailing content after flow sequence RHS returns yaml error",
			n:        Map{},
			expr:     ".a = [1]\n2",
			errstr:   `yaml: line 1: did not find expected <document start>`,
		}, {
			caseName: "RHS trailing comment is ignored",
			n:        Map{},
			expr:     `.a = 1 # set`,
			want:     Map{"a": NumberValue(1)},
		}, {
			caseName: "RHS comment line after value is ignored",
			n:        Map{},
			expr:     ".a = 1\n#c\n",
			want:     Map{"a": NumberValue(1)},
		}, {
			caseName: "comment-only RHS assigns null",
			n:        Map{"a": NumberValue(1)},
			expr:     `.a = # c`,
			want:     Map{"a": Nil},
		}, {
			caseName: "multi-line flow sequence RHS",
			n:        Map{},
			expr:     ".a = [1,\n 2]",
			want:     Map{"a": Array{NumberValue(1), NumberValue(2)}},
		}, {
			caseName: "missing operator returns syntax error",
			n:        Map{},
			expr:     `.a`,
			errstr:   `syntax error: invalid edit expression ".a"`,
		}, {
			caseName: "comparison operator is not an assignment",
			n:        Map{},
			expr:     `.a ==1`,
			errstr:   `syntax error: invalid edit expression ".a ==1"`,
		}, {
			caseName: "not-equal operator is not an assignment",
			n:        Map{},
			expr:     `.a !=1`,
			errstr:   `syntax error: invalid edit expression ".a !=1"`,
		}, {
			caseName: "regexp operator is not an assignment",
			n:        Map{},
			expr:     `.a ~=1`,
			errstr:   `syntax error: invalid edit expression ".a ~=1"`,
		}, {
			caseName: "grouping parens in filter are skipped",
			n:        Array{Map{"k": StringValue("a")}, Map{"k": StringValue("b")}},
			expr:     `[(.k == "b")].v = 1`,
			want:     Array{Map{"k": StringValue("a")}, Map{"k": StringValue("b"), "v": NumberValue(1)}},
		}, {
			caseName: "delete with trailing space",
			n:        Map{"a": NumberValue(0)},
			expr:     `.a ^? `,
			want:     Map{},
		}, {
			caseName: "delete with trailing newline",
			n:        Map{"a": NumberValue(0)},
			expr:     ".a ^?\n",
			want:     Map{},
		}, {
			caseName: "unterminated paren on LHS returns syntax error",
			n:        Map{},
			expr:     `.a(b = 1`,
			errstr:   `syntax error: no right brackets: ".a(b"`,
		}, {
			caseName: "unterminated quote on LHS returns syntax error",
			n:        Map{},
			expr:     `.a" = 1`,
			errstr:   `syntax error: invalid edit expression ".a\" = 1"`,
		}, {
			caseName: "less-or-equal filter on LHS",
			n:        Array{Map{"n": NumberValue(1)}, Map{"n": NumberValue(2)}},
			expr:     `[.n <= 1].v = 2`,
			want:     Array{Map{"n": NumberValue(1), "v": NumberValue(2)}, Map{"n": NumberValue(2)}},
		}, {
			caseName: "greater-or-equal filter on LHS",
			n:        Array{Map{"n": NumberValue(1)}, Map{"n": NumberValue(2)}},
			expr:     `[.n >= 1].v = 2`,
			want:     Array{Map{"n": NumberValue(1), "v": NumberValue(2)}, Map{"n": NumberValue(2), "v": NumberValue(2)}},
		}, {
			caseName: "not-equal filter on LHS",
			n:        Array{Map{"k": StringValue("a")}, Map{"k": StringValue("b")}},
			expr:     `[.k != "a"].v = 1`,
			want:     Array{Map{"k": StringValue("a")}, Map{"k": StringValue("b"), "v": NumberValue(1)}},
		}, {
			caseName: "missing RHS returns syntax error",
			n:        Map{},
			expr:     `.a = `,
			errstr:   `syntax error: invalid edit expression ".a = "`,
		}, {
			caseName: "delete with RHS returns syntax error",
			n:        Map{},
			expr:     `.a ^? 1`,
			errstr:   `syntax error: invalid edit expression ".a ^? 1"`,
		}, {
			caseName: "missing LHS returns syntax error",
			n:        Map{},
			expr:     `= 1`,
			errstr:   `syntax error: invalid edit expression "= 1"`,
		}, {
			caseName: "stray plus on LHS returns syntax error",
			n:        Map{"list": Array{StringValue("a")}},
			expr:     `.list + = "x"`,
			errstr:   `syntax error: invalid edit expression ".list + = \"x\""`,
		}, {
			caseName: "double plus on LHS returns syntax error",
			n:        Map{},
			expr:     `.c++ = 1`,
			errstr:   `syntax error: invalid edit expression ".c++ = 1"`,
		}, {
			caseName: "unquoted plus in key returns syntax error",
			n:        Map{},
			expr:     `.a+b = 1`,
			errstr:   `syntax error: invalid edit expression ".a+b = 1"`,
		},

		// --- Intermediate-step edge cases in resolveAndEdit ---
		{
			caseName: "pipe-prefixed query slurps root then edits wrapper",
			n:        Map{"a": StringValue("v")},
			expr:     `| [0] = "x"`,
			want:     Map{"a": StringValue("v")},
		}, {
			caseName: "bare .. before an index is dropped like .",
			n:        Map{},
			expr:     `..[0] = 1`,
			want:     Map{"0": NumberValue(1)},
		}, {
			caseName: "dot before an index appends to an empty array",
			n:        Array{},
			expr:     `.[0] = 1`,
			want:     Array{NumberValue(1)},
		}, {
			caseName: "dot before an index creates an array under a null key",
			n:        Map{"a": Nil},
			expr:     `.a.[0] = 1`,
			want:     Map{"a": Array{NumberValue(1)}},
		}, {
			caseName: "dot before an index replaces a null root",
			n:        Nil,
			expr:     `.[0] = 1`,
			want:     Array{NumberValue(1)},
		}, {
			caseName: "dot before an index deletes the element",
			n:        Array{NumberValue(1)},
			expr:     `.[0] ^?`,
			want:     Array{},
		}, {
			caseName: "dot before an index appends under a null root",
			n:        Nil,
			expr:     `.[0] += 1`,
			want:     Array{Array{NumberValue(1)}},
		}, {
			caseName: "edit ending in a piped dot is a no-op",
			n:        Map{"a": NumberValue(1)},
			expr:     `.a | . = 1`,
			want:     Map{"a": NumberValue(1)},
		}, {
			caseName: "trailing bare dot on a null root sets the key",
			n:        Nil,
			expr:     `.a. = 1`,
			want:     Map{"a": NumberValue(1)},
		}, {
			caseName: "trailing bare dot on a missing key sets it",
			n:        Map{},
			expr:     `.a. = 1`,
			want:     Map{"a": NumberValue(1)},
		}, {
			caseName: "two trailing bare dots on a missing key set it",
			n:        Map{},
			expr:     `.a.. = 1`,
			want:     Map{"a": NumberValue(1)},
		}, {
			caseName: "trailing bare dot after a walk",
			n:        Map{"a": Map{"b": NumberValue(1)}},
			expr:     `..b. = 2`,
			want:     Map{"a": Map{"b": NumberValue(2)}},
		}, {
			caseName: "trailing bare dot after [] edits every element",
			n:        Map{"a": Array{NumberValue(1), NumberValue(2)}},
			expr:     `.a[]. = 9`,
			want:     Map{"a": Array{NumberValue(9), NumberValue(9)}},
		}, {
			caseName: "trailing bare dot after a select edits the matches",
			n:        Array{Map{"b": NumberValue(1)}, Map{"b": NumberValue(2)}},
			expr:     `.[.b == 1]. = 9`,
			want:     Array{NumberValue(9), Map{"b": NumberValue(2)}},
		}, {
			caseName: "trailing bare dot after a slice edits the elements in range",
			n:        Array{NumberValue(1), NumberValue(2)},
			expr:     `.[0:1]. = 9`,
			want:     Array{NumberValue(9), NumberValue(2)},
		}, {
			caseName: "bare dot between [] and an index is dropped",
			n:        Array{Array{NumberValue(1)}, Array{NumberValue(2)}},
			expr:     `.[].[0] = 9`,
			want:     Array{Array{NumberValue(9)}, Array{NumberValue(9)}},
		}, {
			caseName: "delete through a trailing bare dot after a key",
			n:        Map{"a": NumberValue(1)},
			expr:     `.a. ^?`,
			want:     Map{},
		}, {
			caseName: "delete through a trailing bare dot after a walk",
			n:        Map{"a": Map{"b": NumberValue(1)}},
			expr:     `..b. ^?`,
			want:     Map{"a": Map{}},
		}, {
			caseName: "delete through a trailing bare dot after [] fails",
			n:        Array{NumberValue(1), NumberValue(2)},
			expr:     `.[]. ^?`,
			errstr:   `cannot delete .`,
		}, {
			caseName: "missing map key with non-collection next is a no-op",
			n:        Map{},
			expr:     `.missing[].x = 1`,
			want:     Map{},
		}, {
			caseName: "non-numeric map key on Array errors",
			n:        Map{"arr": Array{NumberValue(1)}},
			expr:     `.arr.badkey.x = 1`,
			errstr:   `cannot index array with badkey`,
		}, {
			caseName: "non-numeric map key on empty Array errors",
			n:        Map{"x": Array{}},
			expr:     `.x.foo.c = 1`,
			errstr:   `cannot index array with foo`,
		}, {
			caseName: "non-numeric map key on empty Array at final step errors the same way",
			n:        Map{"x": Array{}},
			expr:     `.x.foo = 1`,
			errstr:   `cannot index array with foo`,
		}, {
			caseName: "delete through non-numeric map key on Array errors",
			n:        Map{"x": Array{}},
			expr:     `.x.foo.c ^?`,
			errstr:   `cannot index array with foo`,
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
			caseName: "MapQuery intermediate on non-collection errors",
			n:        Map{"s": StringValue("hi")},
			expr:     `.s.x.y = 1`,
			errstr:   `cannot index array with "x"`,
		}, {
			caseName: "delete through MapQuery intermediate on non-collection errors",
			n:        Map{"x": NumberValue(1)},
			expr:     `.x.y.c ^?`,
			errstr:   `cannot index array with "y"`,
		}, {
			caseName: "null intermediate reached through select is replaced",
			n:        Map{"a": Array{NilValue{}}},
			expr:     `.a[].x.y = 1`,
			want:     Map{"a": Array{Map{"x": Map{"y": NumberValue(1)}}}},
		}, {
			caseName: "ArrayQuery intermediate on non-collection errors",
			n:        Map{"s": StringValue("hi")},
			expr:     `.s[0].x = 1`,
			errstr:   `cannot index array with 0`,
		}, {
			caseName: "null ArrayQuery intermediate reached through select is replaced",
			n:        Map{"a": Array{NilValue{}}},
			expr:     `.a[][0].y = 1`,
			want:     Map{"a": Array{Array{Map{"y": NumberValue(1)}}}},
		}, {
			caseName: "nil element reached through select is replaced",
			n:        Map{"a": Array{nil}},
			expr:     `.a[].x = 1`,
			want:     Map{"a": Array{Map{"x": NumberValue(1)}}},
		}, {
			caseName: "unsupported last step errors before reaching a null through select",
			n:        Map{"a": Array{Nil}},
			expr:     `.a[].b[] = 1`,
			errstr:   `syntax error: unsupported edit query: []`,
		}, {
			caseName: "unsupported last step errors before reaching a null by key",
			n:        Map{"a": Nil},
			expr:     `.a.b[] = 1`,
			errstr:   `syntax error: unsupported edit query: []`,
		}, {
			caseName: "first() as the last step creates what it needs like [0]",
			n:        Map{"a": Array{nil}},
			expr:     `.a[0].b.first() = 1`,
			want:     Map{"a": Array{Map{"b": Array{NumberValue(1)}}}},
		}, {
			caseName: "nil map value reached through select is replaced",
			n:        Map{"a": Map{"k": nil}},
			expr:     `.a[].x = 1`,
			want:     Map{"a": Map{"k": Map{"x": NumberValue(1)}}},
		}, {
			caseName: "delete through nil element reached through select keeps it nil",
			n:        Map{"a": Array{nil}},
			expr:     `.a[].x ^?`,
			want:     Map{"a": Array{nil}},
		}, {
			caseName: "null element reached by index is replaced",
			n:        Map{"a": Array{Nil}},
			expr:     `.a[0].x = 1`,
			want:     Map{"a": Array{Map{"x": NumberValue(1)}}},
		}, {
			caseName: "nil element reached by index is replaced",
			n:        Map{"a": Array{nil, nil, NumberValue(1)}},
			expr:     `.a[1].x = 1`,
			want:     Map{"a": Array{nil, Map{"x": NumberValue(1)}, NumberValue(1)}},
		}, {
			caseName: "append to nil element creates an array",
			n:        Map{"a": Array{nil, nil, NumberValue(1)}},
			expr:     `.a[0] += 1`,
			want:     Map{"a": Array{Array{NumberValue(1)}, nil, NumberValue(1)}},
		}, {
			caseName: "append to null element creates an array",
			n:        Map{"a": Array{Nil}},
			expr:     `.a[-1] += 1`,
			want:     Map{"a": Array{Array{NumberValue(1)}}},
		}, {
			caseName: "append to null map value creates an array",
			n:        Map{"a": Nil},
			expr:     `.a += 1`,
			want:     Map{"a": Array{NumberValue(1)}},
		}, {
			caseName: "set through nil element in a slice creates it",
			n:        Map{"a": Array{nil, nil, NumberValue(1)}},
			expr:     `.a[1:2][0].x = 1`,
			want:     Map{"a": Array{nil, Array{Map{"x": NumberValue(1)}}, NumberValue(1)}},
		}, {
			caseName: "nil element reached through first() is created",
			n:        Map{"a": Array{nil}},
			expr:     `.a.first().x = 1`,
			want:     Map{"a": Array{Map{"x": NumberValue(1)}}},
		}, {
			caseName: "delete past null reached through walk is a no-op",
			n:        Map{"b": Nil},
			expr:     `..b.c ^?`,
			want:     Map{"b": Nil},
		}, {
			caseName: "set on nil element in a slice creates the array",
			n:        Map{"a": Array{nil, NumberValue(1)}},
			expr:     `.a[0:1][0] = 1`,
			want:     Map{"a": Array{Array{NumberValue(1)}, NumberValue(1)}},
		}, {
			caseName: "ArrayQuery intermediate on Map uses EditorNode branch",
			n:        Map{"m": Map{"0": Map{"x": StringValue("old")}}},
			expr:     `.m[0].x = "new"`,
			want:     Map{"m": Map{"0": Map{"x": StringValue("new")}}},
		}, {
			caseName: "typed-nil map reached by index through editor node is replaced",
			n:        Map{"m": Map{"0": Map(nil)}},
			expr:     `.m[0].x = 1`,
			want:     Map{"m": Map{"0": Map{"x": NumberValue(1)}}},
		}, {
			caseName: "typed-nil map in pointer array reached by index is replaced",
			n:        Map{"a": &Array{Map(nil)}},
			expr:     `.a[0].x = 1`,
			want:     Map{"a": &Array{Map{"x": NumberValue(1)}}},
		}, {
			caseName: "typed-nil array in pointer array reached by index is replaced",
			n:        Map{"a": &Array{Array(nil)}},
			expr:     `.a[0][0] = 1`,
			want:     Map{"a": &Array{Array{NumberValue(1)}}},
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
			caseName: "slice step on a map errors",
			n:        Map{"items": Map{"x": StringValue("v")}},
			expr:     `.items[1:2].x = 1`,
			errstr:   `cannot index array with range [1:2]`,
		}, {
			caseName: "slurp left.Exec error propagates up",
			n:        Map{"items": Map{"x": StringValue("v")}},
			expr:     `.items[1:2] | .x = 1`,
			errstr:   `cannot index array with range [1:2]`,
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
		}, {
			caseName: "negative index set on map errors",
			n:        Map{"x": StringValue("v")},
			expr:     `[-1] = "z"`,
			errstr:   `cannot index array with -1`,
		}, {
			caseName: "negative index append on map errors",
			n:        Map{"x": StringValue("v")},
			expr:     `[-1] += "z"`,
			errstr:   `cannot append to array with -1`,
		}, {
			caseName: "negative index delete on map errors",
			n:        Map{"x": StringValue("v")},
			expr:     `[-1] ^?`,
			errstr:   `cannot delete array with -1`,
		}, {
			caseName: "walk set with value containing the same key",
			n:        Map{"a": Map{"b": NumberValue(1)}},
			expr:     `..b = {"b": 1}`,
			want:     Map{"a": Map{"b": Map{"b": NumberValue(1)}}},
		}, {
			caseName: "walk append with value containing the same key",
			n:        Map{"b": Array{}},
			expr:     `..b += {"b": []}`,
			want:     Map{"b": Array{Map{"b": Array{}}}},
		}, {
			caseName: "set index at array limit errors",
			n:        Map{"a": Array{}},
			expr:     `.a[16777216] = 1`,
			errstr:   `array index 16777216 exceeds maximum 16777215`,
		}, {
			caseName: "set int32 max index errors",
			n:        Map{},
			expr:     `.a[2147483647] = 1`,
			errstr:   `array index 2147483647 exceeds maximum 16777215`,
		}, {
			caseName: "set through index beyond array limit errors",
			n:        Map{"a": Array{}},
			expr:     `.a[16777216].b = 1`,
			errstr:   `array index 16777216 exceeds maximum 16777215`,
		}, {
			caseName: "set through numeric key beyond array limit errors",
			n:        Map{"a": Array{}},
			expr:     `.a.16777216.b = 1`,
			errstr:   `array index 16777216 exceeds maximum 16777215`,
		}, {
			caseName: "append at index beyond array limit errors",
			n:        Map{"a": Array{}},
			expr:     `.a[16777216] += 1`,
			errstr:   `array index 16777216 exceeds maximum 16777215`,
		}, {
			caseName: "walk intermediate set with value containing the same path",
			n:        Map{"b": Map{"c": NumberValue(1)}},
			expr:     `..b.c = {"b": {"c": 1}}`,
			want:     Map{"b": Map{"c": Map{"b": Map{"c": NumberValue(1)}}}},
		}, {
			caseName: "walk intermediate append with value containing the same path",
			n:        Map{"b": Map{"c": Array{}}},
			expr:     `..b.c += {"b": {"c": []}}`,
			want:     Map{"b": Map{"c": Array{Map{"b": Map{"c": Array{}}}}}},
		}, {
			caseName: "negative index out of range on intermediate array errors",
			n:        Array{},
			expr:     `[-5].a = 1`,
			errstr:   `cannot index array with -5`,
		}, {
			caseName: "negative index out of range on nested intermediate array errors",
			n:        Map{"a": Array{}},
			expr:     `.a[-5].b = 1`,
			errstr:   `cannot index array with -5`,
		}, {
			caseName: "set via negative index on map key",
			n:        Map{"m": Map{}},
			expr:     `.m[-1].x = 1`,
			want:     Map{"m": Map{"-1": Map{"x": NumberValue(1)}}},
		}, {
			caseName: "negative index on scalar errors",
			n:        Map{"a": NumberValue(1)},
			expr:     `.a[-1].b = 1`,
			errstr:   `cannot index array with -1`,
		}, {
			caseName: "walk intermediate set edits inner match first and fails on its shape",
			n:        Map{"b": Map{"c": Map{"b": Array{}}}},
			expr:     `..b.c = 1`,
			errstr:   `cannot index array with c`,
		}, {
			caseName: "walk intermediate set fails on scalar inner match",
			n:        Map{"b": Map{"c": Map{"b": NumberValue(1)}}},
			expr:     `..b.c = 1`,
			errstr:   `cannot index array with "c"`,
		}, {
			caseName: "walk set does not edit the assigned value",
			n:        Map{"b": Map{"b": NumberValue(0)}},
			expr:     `..b = {"b": 9}`,
			want:     Map{"b": Map{"b": NumberValue(9)}},
		}, {
			caseName: "set past end grows with nil holes",
			n:        Array{},
			expr:     `[2] = 1`,
			want:     Array{nil, nil, NumberValue(1)},
		}, {
			caseName: "walk intermediate set skips the assigned value",
			n:        Map{"b": Map{"b": Map{"b": Map{}}}},
			expr:     `..b.b = {"x": 1}`,
			want:     Map{"b": Map{"b": Map{"x": NumberValue(1)}}},
		}, {
			caseName: "walk intermediate set fails on scalar below inner match",
			n:        Map{"b": Map{"b": Map{"b": NumberValue(0)}}},
			expr:     `..b.b.b = {"x": 1}`,
			errstr:   `cannot index array with "b"`,
		}, {
			caseName: "walk intermediate set fails on scalar inner match with index",
			n:        Map{"b": Array{Map{"b": NumberValue(0)}}},
			expr:     `..b[0].b = [{"c": 0}]`,
			errstr:   `cannot index array with 0`,
		}, {
			caseName: "walk intermediate append edits inner matches first",
			n:        Map{"b": Map{"b": Map{"b": Map{"c": Array{}}}}},
			expr:     `..b.b.c += {"b": {"c": []}}`,
			want: Map{"b": Map{"b": Map{
				"b": Map{"b": Map{"c": Array{Map{"b": Map{"c": Array{}}}}}, "c": Array{Map{"b": Map{"c": Array{}}}}},
				"c": Array{Map{"b": Map{"c": Array{}}}},
			}}},
		}, {
			caseName: "walk intermediate delete edits inner matches first",
			n:        Map{"b": Map{"b": Map{"b": Map{}}}},
			expr:     `..b.b ^?`,
			want:     Map{"b": Map{}},
		}, {
			caseName: "nested walk set lets the outer match overwrite the inner one",
			n:        Map{"b": Map{"b": Map{"c": NumberValue(0)}}},
			expr:     `..b..c = {"c": 1}`,
			want:     Map{"b": Map{"b": Map{"c": Map{"c": NumberValue(1)}}}},
		}, {
			caseName: "nested walk append skips the value appended by the inner match",
			n:        Map{"b": Map{"b": Map{"c": Array{}}}},
			expr:     `..b..c += {"c": []}`,
			want:     Map{"b": Map{"b": Map{"c": Array{Map{"c": Array{}}, Map{"c": Array{}}}}}},
		}, {
			caseName: "nested walk set with a path after it skips the value set by the inner match",
			n:        Map{"b": Map{"b": Map{"c": Map{}}}},
			expr:     `..b..c.d = {"c": 1}`,
			want:     Map{"b": Map{"b": Map{"c": Map{"d": Map{"c": NumberValue(1)}}}}},
		}, {
			caseName: "walk intermediate set fails on scalar inner match before assigning",
			n:        Map{"b": Map{"b": Map{"b": NumberValue(0)}}},
			expr:     `..b.b = []`,
			errstr:   `cannot index array with "b"`,
		}, {
			caseName: "walk set on value-typed editor node does not edit the assigned value",
			n:        Map{"b": valueEditorNode{Map{"b": NumberValue(0)}}},
			expr:     `..b = {"b": 9}`,
			want:     Map{"b": Map{"b": NumberValue(9)}},
		}, {
			caseName: "walk intermediate set grows an existing empty array",
			n:        Map{"b": Array{}, "x": Map{"b": Array{}}},
			expr:     `..b[0] = []`,
			want:     Map{"b": Array{Array{}}, "x": Map{"b": Array{Array{}}}},
		}, {
			caseName: "walk set on value-typed editor node",
			n:        Map{"a": Map{"b": NumberValue(0)}, "x": valueEditorNode{Map{"b": NumberValue(1)}}},
			expr:     `..b = 9`,
			want:     Map{"a": Map{"b": NumberValue(9)}, "x": valueEditorNode{Map{"b": NumberValue(9)}}},
		}, {
			caseName: "walk intermediate set on value-typed editor node",
			n:        Map{"x": valueEditorNode{Map{"b": Map{"c": NumberValue(1)}}}},
			expr:     `..b.c = 9`,
			want:     Map{"x": valueEditorNode{Map{"b": Map{"c": NumberValue(9)}}}},
		}, {
			caseName: "negative index out of range set reports the index",
			n:        Array{NumberValue(1), NumberValue(2)},
			expr:     `[-5] = 1`,
			errstr:   `cannot index array with -5`,
		}, {
			caseName: "negative index out of range set on pointer array reports the index",
			n:        &Array{NumberValue(1), NumberValue(2)},
			expr:     `[-5] = 1`,
			errstr:   `cannot index array with -5`,
		}, {
			caseName: "walk set reports set error of editor node",
			n:        Map{"x": setFailingEditorNode{Map{"b": NumberValue(0)}}},
			expr:     `..b = 1`,
			errstr:   `set failed`,
		}, {
			caseName: "walk append reports set error of editor node",
			n:        Map{"x": setFailingEditorNode{Map{"b": Array{}}}},
			expr:     `..b += 1`,
			errstr:   `set failed`,
		}, {
			caseName: "walk set fails on editor node an outer match would replace",
			n:        Map{"b": setFailingEditorNode{Map{"b": NumberValue(0)}}},
			expr:     `..b = 1`,
			errstr:   `set failed`,
		}, {
			caseName: "walk delete fails on editor node under a key an outer match would delete",
			n:        Map{"b": deleteFailingEditorNode{Map{"b": NumberValue(0)}}},
			expr:     `..b ^?`,
			errstr:   `delete failed`,
		}, {
			caseName: "walk delete reports delete error of editor node",
			n:        Map{"x": deleteFailingEditorNode{Map{"b": NumberValue(0)}}},
			expr:     `..b ^?`,
			errstr:   `delete failed`,
		}, {
			caseName: "intermediate map step reports set error of editor node",
			n:        Map{"x": setFailingEditorNode{Map{"y": Map{"c": NumberValue(0)}}}},
			expr:     `.x.y.c = 1`,
			errstr:   `set failed`,
		}, {
			caseName: "intermediate map step edits editor node without map",
			n:        Map{"x": mapHidingEditorNode{Map{"y": Map{"c": NumberValue(0)}}}},
			expr:     `.x.y.c = 1`,
			want:     Map{"x": mapHidingEditorNode{Map{"y": Map{"c": NumberValue(1)}}}},
		}, {
			caseName: "intermediate map step creates missing key in editor node without map",
			n:        Map{"x": mapHidingEditorNode{Map{}}},
			expr:     `.x.new.c = 1`,
			want:     Map{"x": mapHidingEditorNode{Map{"new": Map{"c": NumberValue(1)}}}},
		}, {
			caseName: "intermediate map step replaces nil value in editor node",
			n:        Map{"x": valueEditorNode{Map{"y": nil}}},
			expr:     `.x.y.c = 1`,
			want:     Map{"x": valueEditorNode{Map{"y": Map{"c": NumberValue(1)}}}},
		}, {
			caseName: "intermediate map step replaces nil value in editor node without map",
			n:        Map{"x": mapHidingEditorNode{Map{"y": nil}}},
			expr:     `.x.y.c = 1`,
			want:     Map{"x": mapHidingEditorNode{Map{"y": Map{"c": NumberValue(1)}}}},
		}, {
			caseName: "first() on an editor node without map errors as on a map",
			n:        Map{"x": mapHidingEditorNode{Map{"k": NumberValue(1)}}},
			expr:     `.x.first() = 1`,
			errstr:   `cannot edit first() of a map`,
		}, {
			caseName: "last() step on an editor node without map errors as on a map",
			n:        Map{"x": mapHidingEditorNode{Map{"k": NumberValue(1)}}},
			expr:     `.x.last().y = 1`,
			errstr:   `cannot edit last() of a map`,
		}, {
			caseName: "intermediate map step leaves editor node wrapping nil map",
			n:        Map{"x": valueEditorNode{Map(nil)}},
			expr:     `.x.y.c = 1`,
			want:     Map{"x": valueEditorNode{Map(nil)}},
		}, {
			caseName: "intermediate map step replaces null value",
			n:        Map{"y": Nil},
			expr:     `.y.c = 1`,
			want:     Map{"y": Map{"c": NumberValue(1)}},
		}, {
			caseName: "intermediate map step keeps pointer array",
			n:        Map{"x": &Array{Map{"c": NumberValue(0)}}},
			expr:     `.x.0.c = 1`,
			want:     Map{"x": &Array{Map{"c": NumberValue(1)}}},
		}, {
			caseName: "intermediate map step grows pointer array",
			n:        Map{"x": &Array{}},
			expr:     `.x.2.c = 1`,
			want:     Map{"x": &Array{nil, nil, Map{"c": NumberValue(1)}}},
		}, {
			caseName: "intermediate map step rejects non-numeric key on pointer array",
			n:        Map{"x": &Array{}},
			expr:     `.x.foo.c = 1`,
			errstr:   `cannot index array with foo`,
		}, {
			caseName: "intermediate map step keeps nil pointer array",
			n:        Map{"x": new(Array)},
			expr:     `.x[0].c = 1`,
			want:     Map{"x": &Array{Map{"c": NumberValue(1)}}},
		}, {
			caseName: "delete through null intermediate is a no-op",
			n:        Map{"x": Map{"y": Nil}},
			expr:     `.x.y.z ^?`,
			want:     Map{"x": Map{"y": Nil}},
		}, {
			caseName: "delete through missing intermediate is a no-op",
			n:        Map{"x": Map{}},
			expr:     `.x.y[0].z ^?`,
			want:     Map{"x": Map{}},
		},
		// --- Null-like nodes are replaced however they are reached (#147) ---
		{
			caseName: "set past null reached through walk creates a map",
			n:        Map{"a": Map{"b": Nil}},
			expr:     `..b.x = 1`,
			want:     Map{"a": Map{"b": Map{"x": NumberValue(1)}}},
		}, {
			caseName: "set two keys past null reached through walk creates nested maps",
			n:        Map{"a": Map{"b": Nil}},
			expr:     `..b.x.y = 1`,
			want:     Map{"a": Map{"b": Map{"x": Map{"y": NumberValue(1)}}}},
		}, {
			caseName: "set index past null reached through walk creates an array",
			n:        Map{"a": Map{"b": Nil}},
			expr:     `..b[0] = 1`,
			want:     Map{"a": Map{"b": Array{NumberValue(1)}}},
		}, {
			caseName: "append past null reached through walk creates an array",
			n:        Map{"a": Map{"b": Nil}},
			expr:     `..b.x += 1`,
			want:     Map{"a": Map{"b": Map{"x": Array{NumberValue(1)}}}},
		}, {
			caseName: "set past go nil reached through walk creates a map",
			n:        Map{"a": Map{"b": nil}},
			expr:     `..b.x = 1`,
			want:     Map{"a": Map{"b": Map{"x": NumberValue(1)}}},
		}, {
			caseName: "set on nil map value replaces it",
			n:        Map{"m": Map(nil)},
			expr:     `.m.k = 1`,
			want:     Map{"m": Map{"k": NumberValue(1)}},
		}, {
			caseName: "append on nil map value replaces it",
			n:        Map{"m": Map(nil)},
			expr:     `.m.k += 1`,
			want:     Map{"m": Map{"k": Array{NumberValue(1)}}},
		}, {
			caseName: "set two keys past nil map value replaces it",
			n:        Map{"m": Map(nil)},
			expr:     `.m.k.x = 1`,
			want:     Map{"m": Map{"k": Map{"x": NumberValue(1)}}},
		}, {
			caseName: "set index on nil array value replaces it",
			n:        Map{"m": Array(nil)},
			expr:     `.m[0] = 1`,
			want:     Map{"m": Array{NumberValue(1)}},
		}, {
			caseName: "delete on nil map value is a no-op",
			n:        Map{"m": Map(nil)},
			expr:     `.m.k ^?`,
			want:     Map{"m": Map(nil)},
		}, {
			caseName: "set on nil map element reached through select replaces it",
			n:        Map{"a": Array{Map(nil)}},
			expr:     `.a[].k = 1`,
			want:     Map{"a": Array{Map{"k": NumberValue(1)}}},
		}, {
			caseName: "set on nil map reached through walk replaces it",
			n:        Map{"a": Map{"b": Map(nil)}},
			expr:     `..b.k = 1`,
			want:     Map{"a": Map{"b": Map{"k": NumberValue(1)}}},
		}, {
			caseName: "set on null root creates a map",
			n:        Nil,
			expr:     `.k = 1`,
			want:     Map{"k": NumberValue(1)},
		}, {
			caseName: "set two keys on null root creates nested maps",
			n:        Nil,
			expr:     `.a.b = 1`,
			want:     Map{"a": Map{"b": NumberValue(1)}},
		}, {
			caseName: "set on nil map root replaces it",
			n:        Map(nil),
			expr:     `.k = 1`,
			want:     Map{"k": NumberValue(1)},
		}, {
			caseName: "delete on null root is a no-op",
			n:        Nil,
			expr:     `.k ^?`,
			want:     Nil,
		}, {
			caseName: "delete through a method step on a null root is a no-op",
			n:        Nil,
			expr:     `.a.first().b ^?`,
			want:     Nil,
		}, {
			caseName: "delete with an unsupported query on a null root is a syntax error",
			n:        Nil,
			expr:     `"x" ^?`,
			errstr:   `syntax error: unsupported edit query: "x"`,
		}, {
			caseName: "delete with [] on a null root is a syntax error",
			n:        Nil,
			expr:     `.[] ^?`,
			errstr:   `syntax error: unsupported edit query: []`,
		}, {
			caseName: "append to nil array value creates the array",
			n:        Map{"m": Array(nil)},
			expr:     `.m += 1`,
			want:     Map{"m": Array{NumberValue(1)}},
		}, {
			caseName: "append to nil map value creates the array",
			n:        Map{"m": Map(nil)},
			expr:     `.m += 1`,
			want:     Map{"m": Array{NumberValue(1)}},
		}, {
			caseName: "append to nil array element by index creates the array",
			n:        Map{"a": Array{Array(nil)}},
			expr:     `.a[0] += 1`,
			want:     Map{"a": Array{Array{NumberValue(1)}}},
		}, {
			caseName: "walk append on null creates the array",
			n:        Map{"b": Nil},
			expr:     `..b += 1`,
			want:     Map{"b": Array{NumberValue(1)}},
		}, {
			caseName: "walk append on nil array creates the array",
			n:        Map{"b": Array(nil)},
			expr:     `..b += 1`,
			want:     Map{"b": Array{NumberValue(1)}},
		}, {
			caseName: "walk append on nested null creates the array",
			n:        Map{"x": Map{"b": Nil}},
			expr:     `..b += 1`,
			want:     Map{"x": Map{"b": Array{NumberValue(1)}}},
		}, {
			caseName: "nil map reached through first() is replaced for an index step",
			n:        Map{"a": Array{Map(nil)}},
			expr:     `.a.first()[0].x = 1`,
			want:     Map{"a": Array{Array{Map{"x": NumberValue(1)}}}},
		}, {
			caseName: "nil map reached through first() is replaced",
			n:        Map{"a": Array{Map(nil)}},
			expr:     `.a.first().k = 1`,
			want:     Map{"a": Array{Map{"k": NumberValue(1)}}},
		},
		// --- Edit paths: groups, slices and methods ---
		{
			caseName: "group without a pipe is flattened",
			n:        Map{"a": Array{}},
			expr:     `(.a[0]).b = 1`,
			want:     Map{"a": Array{Map{"b": NumberValue(1)}}},
		}, {
			caseName: "group without a pipe is flattened on a null root",
			n:        Nil,
			expr:     `(.a[0]).b = 1`,
			want:     Map{"a": Array{Map{"b": NumberValue(1)}}},
		}, {
			caseName: "group with a pipe is an unsupported edit query",
			n:        Map{"a": Array{Map{"b": NumberValue(0)}}},
			expr:     `(.a | .[0]).b = 1`,
			errstr:   `syntax error: unsupported edit query: (.a | .[0])`,
		}, {
			caseName: "slice step edits each element in range",
			n:        Map{"a": Array{Map{"b": NumberValue(0)}, Map{"b": NumberValue(5)}}},
			expr:     `.a[1:].b = 1`,
			want:     Map{"a": Array{Map{"b": NumberValue(0)}, Map{"b": NumberValue(1)}}},
		}, {
			caseName: "slice step writes back a grown element",
			n:        Map{"a": Array{Array{}, Array{}}},
			expr:     `.a[0:1][0] = 1`,
			want:     Map{"a": Array{Array{NumberValue(1)}, Array{}}},
		}, {
			caseName: "delete through a slice step",
			n:        Map{"a": Array{Map{"b": NumberValue(0), "c": NumberValue(1)}, Map{"b": NumberValue(5)}}},
			expr:     `.a[:1].b ^?`,
			want:     Map{"a": Array{Map{"c": NumberValue(1)}, Map{"b": NumberValue(5)}}},
		}, {
			caseName: "empty slice step is a no-op",
			n:        Map{"a": Array{NumberValue(1), NumberValue(2)}},
			expr:     `.a[2:1][0] = 9`,
			want:     Map{"a": Array{NumberValue(1), NumberValue(2)}},
		}, {
			caseName: "slice step on a string errors",
			n:        Map{"s": StringValue("hi")},
			expr:     `.s[0:1].x = 1`,
			errstr:   `cannot index array with range [0:1]`,
		}, {
			caseName: "last() step edits the last element",
			n:        Map{"a": Array{Map{"b": NumberValue(0)}, Map{"b": NumberValue(5)}}},
			expr:     `.a.last().b = 1`,
			want:     Map{"a": Array{Map{"b": NumberValue(0)}, Map{"b": NumberValue(1)}}},
		}, {
			caseName: "first() as the last step sets the first element",
			n:        Map{"a": Array{NumberValue(1), NumberValue(2)}},
			expr:     `.a.first() = 9`,
			want:     Map{"a": Array{NumberValue(9), NumberValue(2)}},
		}, {
			caseName: "last() as the last step deletes the last element",
			n:        Map{"a": Array{NumberValue(1), NumberValue(2)}},
			expr:     `.a.last() ^?`,
			want:     Map{"a": Array{NumberValue(1)}},
		}, {
			caseName: "first() step on null creates the array",
			n:        Map{"a": Nil},
			expr:     `.a.first().x = 1`,
			want:     Map{"a": Array{Map{"x": NumberValue(1)}}},
		}, {
			caseName: "first() step on a map errors",
			n:        Map{"m": Map{"k": NumberValue(1)}},
			expr:     `.m.first().x = 1`,
			errstr:   `cannot edit first() of a map`,
		}, {
			caseName: "last() as the last step on a map errors",
			n:        Map{"m": Map{"k": NumberValue(1)}},
			expr:     `.m.last() = 1`,
			errstr:   `cannot edit last() of a map`,
		}, {
			caseName: "other method step is an unsupported edit query",
			n:        Map{"a": Array{Map{"b": NumberValue(0)}}},
			expr:     `.a.sort()[0] = 9`,
			errstr:   `syntax error: unsupported edit query: sort()`,
		}, {
			caseName: "other method as the last step is an unsupported edit query",
			n:        Map{"a": Map{"b": NumberValue(0)}},
			expr:     `.a.keys() = 1`,
			errstr:   `syntax error: unsupported edit query: keys()`,
		}, {
			caseName: "value step is an unsupported edit query",
			n:        Map{},
			expr:     `"x".b = 1`,
			errstr:   `syntax error: unsupported edit query: "x"`,
		}, {
			caseName: "empty group step is an unsupported edit query",
			n:        Map{"a": Array{NumberValue(1)}},
			expr:     `.a.first ().b = 1`,
			errstr:   `syntax error: unsupported edit query: ()`,
		}, {
			caseName: "empty group as the last step is an unsupported edit query",
			n:        Map{},
			expr:     `.a.() = 1`,
			errstr:   `syntax error: unsupported edit query: ()`,
		}, {
			caseName: "unsupported last step errors before reaching missing data",
			n:        Map{},
			expr:     `.a[] = 1`,
			errstr:   `syntax error: unsupported edit query: []`,
		}, {
			caseName: "unsupported step errors before reaching missing data",
			n:        Map{},
			expr:     `.a.sort().b = 1`,
			errstr:   `syntax error: unsupported edit query: sort()`,
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

func TestEditSequence(t *testing.T) {
	testCases := []struct {
		caseName string
		n        Node
		exprs    []string
		want     Node
		errstr   string
	}{
		{
			caseName: "walk set empty array then append through separate paths",
			n:        Map{"a": Map{"b": Nil}, "c": Map{"b": Nil}},
			exprs:    []string{`..b = []`, `.a.b += 1`, `.c.b += 2`},
			want:     Map{"a": Map{"b": Array{NumberValue(1)}}, "c": Map{"b": Array{NumberValue(2)}}},
		}, {
			caseName: "walk append after select set skips the value appended to the shared map",
			n:        Map{"a": Array{Map{}, Map{}}},
			exprs:    []string{`.a[].x = {"b": []}`, `..b += {"b": []}`},
			want: Map{"a": Array{
				Map{"x": Map{"b": Array{Map{"b": Array{}}, Map{"b": Array{}}}}},
				Map{"x": Map{"b": Array{Map{"b": Array{}}, Map{"b": Array{}}}}},
			}},
		}, {
			caseName: "walk set map then edit one match leaves the others",
			n:        Map{"a": Map{"b": Nil}, "c": Map{"b": Nil}},
			exprs:    []string{`..b = {}`, `.a.b.x = 1`},
			want:     Map{"a": Map{"b": Map{"x": NumberValue(1)}}, "c": Map{"b": Map{}}},
		}, {
			caseName: "walk intermediate set map then edit one match leaves the others",
			n:        Map{"a": Map{"b": Map{}}, "c": Map{"b": Map{}}},
			exprs:    []string{`..b.d = {}`, `.a.b.d.x = 1`},
			want:     Map{"a": Map{"b": Map{"d": Map{"x": NumberValue(1)}}}, "c": Map{"b": Map{"d": Map{}}}},
		}, {
			caseName: "walk append map then edit one match leaves the others",
			n:        Map{"a": Map{"b": Array{}}, "c": Map{"b": Array{}}},
			exprs:    []string{`..b += {}`, `.a.b[0].x = 1`},
			want:     Map{"a": Map{"b": Array{Map{"x": NumberValue(1)}}}, "c": Map{"b": Array{Map{}}}},
		}, {
			caseName: "select set empty array then append through separate elements",
			n:        Array{Map{"b": NumberValue(0)}, Map{"b": NumberValue(0)}},
			exprs:    []string{`.[].b = []`, `[0].b += 1`, `[1].b += 2`},
			want:     Array{Map{"b": Array{NumberValue(1)}}, Map{"b": Array{NumberValue(2)}}},
		}, {
			caseName: "select set on holes left by array growth",
			n:        Map{"a": Array{}},
			exprs:    []string{`.a[2] = 1`, `.a[].x = 1`},
			errstr:   `cannot index array with "x"`,
		}, {
			caseName: "select intermediate set on holes left by array growth",
			n:        Map{"a": Array{}},
			exprs:    []string{`.a[2] = 1`, `.a[].x.y = 1`},
			errstr:   `cannot index array with "x"`,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.caseName, func(t *testing.T) {
			var err error
			for _, expr := range tc.exprs {
				if err = Edit(&(tc.n), expr); err != nil {
					break
				}
			}
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
			if !reflect.DeepEqual(tc.n, tc.want) {
				t.Errorf("got %#v; want %#v", tc.n, tc.want)
			}
		})
	}
}

func TestEditResolveSelectStepSelectorError(t *testing.T) {
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
			err := resolveAndEdit(&tc.n, fq, 0, "=", StringValue("x"), nil)
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
// arbitrary JSON input and two edit expressions applied in turn. It
// only requires that invalid input or expressions return an error
// rather than panicking.
func FuzzEdit(f *testing.F) {
	seeds := [][3]string{
		{`{}`, `.a = 1`, ``},
		{`{}`, `.a = "x"`, ``},
		{`{}`, `.a += 1`, ``},
		{`{}`, `.a.b = true`, ``},
		{`{}`, `.a = [1, 2, 3]`, ``},
		{`{}`, `.a = {"k": "v"}`, ``},
		{`{}`, `delete .a`, ``},
		{`{"a":[]}`, `.a[2] = 1`, `.a[].x = 1`},
		{`{"a":[]}`, `.a[2] = 1`, `.a[].x.y = 1`},
		{`{"a":[]}`, `.a[2] = 1`, `.a[0] += 1`},
		{`{"a":[]}`, `.a[2] = 1`, `.a[1:2][0].x = 1`},
		{`{"a":[]}`, `.a[2] = 1`, `.a[.x == 1]`},
		{`{"a":[]}`, `.a[2] = 1`, `.a[0].x.sort(.y)`},
	}
	for _, s := range seeds {
		f.Add(s[0], s[1], s[2])
	}
	f.Fuzz(func(t *testing.T, input, expr1, expr2 string) {
		n, err := UnmarshalJSON([]byte(input))
		if err != nil {
			return
		}
		if err := Edit(&n, expr1); err != nil {
			return
		}
		_ = Edit(&n, expr2)
		_, _ = Find(n, expr2)
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

func TestEditWalkValue(t *testing.T) {
	custom := mapHidingEditorNode{Map{"k": NumberValue(1)}}
	shared := func() Node {
		m := Map{"b": Array{}}
		return Map{"a": Array{m, m}}
	}
	sharedInner := func() Node {
		m := Map{"b": Map{}}
		return Map{"a": Array{m, m}}
	}
	appended := Map{"b": Array{Map{"b": Array{}}, Map{"b": Array{}}}}
	ptr := &pointerEditorNode{Map{"b": NumberValue(0)}}
	docSet := Map{"x": Map{"a": NumberValue(1)}, "a": NumberValue(0)}
	docAppend := Map{"x": Map{"a": Array{NumberValue(1)}}, "a": Array{}}
	docNested := Map{"y": Map{"x": Map{"a": NumberValue(1)}, "a": NumberValue(2)}}
	resliced := Array{NumberValue(1), Map{"k": NumberValue(0)}}
	reslicedAppend := Array{NumberValue(1), Map{"k": Array{}}}
	sharedScalar := func() Node {
		m := Map{"b": NumberValue(0)}
		return Map{"a": m, "c": m}
	}
	sharedArray := func() Node {
		m := Map{"b": Array{}}
		return Map{"a": m, "c": m}
	}
	testCases := []struct {
		caseName string
		n        Node
		q        Query
		op       string
		v        Node
		want     Node
	}{
		{
			caseName: "walk set stores a custom value as given",
			n:        Map{"x": Map{}},
			q:        WalkQuery("x"),
			op:       "=",
			v:        custom,
			want:     Map{"x": custom},
		}, {
			caseName: "walk append stores a custom value as given",
			n:        Map{"x": Array{}},
			q:        WalkQuery("x"),
			op:       "+=",
			v:        custom,
			want:     Map{"x": Array{custom}},
		}, {
			caseName: "walk set stores a nil value as given",
			n:        Map{"x": Map{}},
			q:        WalkQuery("x"),
			op:       "=",
			v:        nil,
			want:     Map{"x": nil},
		}, {
			caseName: "walk set stores a custom value nested in a map as given",
			n:        Map{"x": Map{}},
			q:        WalkQuery("x"),
			op:       "=",
			v:        Map{"c": custom},
			want:     Map{"x": Map{"c": custom}},
		}, {
			caseName: "walk set stores a nil map as given",
			n:        Map{"x": Map{}},
			q:        WalkQuery("x"),
			op:       "=",
			v:        Map(nil),
			want:     Map{"x": Map(nil)},
		}, {
			caseName: "walk set keeps a pointer array",
			n:        Map{"x": Map{}},
			q:        WalkQuery("x"),
			op:       "=",
			v:        &Array{NumberValue(1)},
			want:     Map{"x": &Array{NumberValue(1)}},
		}, {
			caseName: "walk set stores a nil element as given",
			n:        Map{"x": Map{}},
			q:        WalkQuery("x"),
			op:       "=",
			v:        Array{nil},
			want:     Map{"x": Array{nil}},
		}, {
			caseName: "walk append on a shared map skips the value it appended",
			n:        shared(),
			q:        WalkQuery("b"),
			op:       "+=",
			v:        Map{"b": Array{}},
			want:     Map{"a": Array{appended, appended}},
		}, {
			caseName: "walk intermediate set on a shared map skips the value it set",
			n:        sharedInner(),
			q:        FilterQuery{WalkQuery("b"), MapQuery("b")},
			op:       "=",
			v:        Array{NumberValue(1)},
			want:     Map{"a": Array{Map{"b": Map{"b": Array{NumberValue(1)}}}, Map{"b": Map{"b": Array{NumberValue(1)}}}}},
		}, {
			caseName: "walk set on a shared map does not edit inside the pointer custom value it stored",
			n:        sharedScalar(),
			q:        WalkQuery("b"),
			op:       "=",
			v:        ptr,
			want:     Map{"a": Map{"b": ptr}, "c": Map{"b": ptr}},
		}, {
			caseName: "walk append on a shared map does not edit inside the pointer custom value it appended",
			n:        sharedArray(),
			q:        WalkQuery("b"),
			op:       "+=",
			v:        ptr,
			want:     Map{"a": Map{"b": Array{ptr, ptr}}, "c": Map{"b": Array{ptr, ptr}}},
		}, {
			caseName: "walk set with a value from the document does not edit inside it",
			n:        docSet,
			q:        WalkQuery("a"),
			op:       "=",
			v:        docSet.Get("x"),
			want:     Map{"a": Map{"a": NumberValue(1)}, "x": Map{"a": NumberValue(1)}},
		}, {
			caseName: "walk append with a value from the document does not edit inside it",
			n:        docAppend,
			q:        WalkQuery("a"),
			op:       "+=",
			v:        docAppend.Get("x"),
			want:     Map{"a": Array{Map{"a": Array{NumberValue(1)}}}, "x": Map{"a": Array{NumberValue(1)}}},
		}, {
			caseName: "walk set with a reslice of a document array still edits the other elements",
			n:        Map{"a": resliced},
			q:        WalkQuery("k"),
			op:       "=",
			v:        resliced[:1],
			want:     Map{"a": Array{NumberValue(1), Map{"k": Array{NumberValue(1)}}}},
		}, {
			caseName: "walk append with a reslice of a document array still edits the other elements",
			n:        Map{"a": reslicedAppend},
			q:        WalkQuery("k"),
			op:       "+=",
			v:        reslicedAppend[:1],
			want:     Map{"a": Array{NumberValue(1), Map{"k": Array{Array{NumberValue(1)}}}}},
		}, {
			caseName: "walk intermediate set skips a value taken from the document",
			n:        docNested,
			q:        FilterQuery{MapQuery("y"), WalkQuery("a")},
			op:       "=",
			v:        docNested.Get("y", "x"),
			want:     Map{"y": Map{"a": Map{"a": NumberValue(1)}, "x": Map{"a": NumberValue(1)}}},
		}, {
			caseName: "walk intermediate set stores a custom value as given",
			n:        Map{"x": Map{}},
			q:        FilterQuery{WalkQuery("x"), MapQuery("y")},
			op:       "=",
			v:        custom,
			want:     Map{"x": Map{"y": custom}},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.caseName, func(t *testing.T) {
			if err := editQuery(&tc.n, tc.q, tc.op, tc.v); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(tc.n, tc.want) {
				t.Errorf("got %#v; want %#v", tc.n, tc.want)
			}
			if !reflect.DeepEqual(ptr.EditorNode, Map{"b": NumberValue(0)}) {
				t.Errorf("custom value changed: %#v", ptr.EditorNode)
			}
		})
	}
}

func TestNewAssigned(t *testing.T) {
	testCases := []struct {
		caseName string
		expr     string
		want     bool
	}{
		{caseName: "no walk", expr: ".a.b", want: false},
		{caseName: "leading walk", expr: "..a.b", want: true},
		{caseName: "single walk after a step", expr: ".a..b", want: true},
		{caseName: "two walks", expr: "..a..b", want: true},
		{caseName: "two walks with a step between", expr: "..a.x..b", want: true},
	}
	for _, tc := range testCases {
		t.Run(tc.caseName, func(t *testing.T) {
			q, err := ParseQuery(tc.expr)
			if err != nil {
				t.Fatal(err)
			}
			fq, ok := q.(FilterQuery)
			if !ok {
				t.Fatalf("got %T; want FilterQuery", q)
			}
			if got := newAssigned(fq) != nil; got != tc.want {
				t.Errorf("got %v; want %v", got, tc.want)
			}
		})
	}
}

func TestEditNopQuerySteps(t *testing.T) {
	testCases := []struct {
		caseName string
		q        Query
	}{
		{caseName: "value", q: FilterQuery{NopQuery{}, ArrayQuery(0)}},
		{caseName: "pointer", q: FilterQuery{&NopQuery{}, ArrayQuery(0)}},
	}
	for _, tc := range testCases {
		t.Run(tc.caseName, func(t *testing.T) {
			var n Node = Array{}
			if err := editQuery(&n, tc.q, "=", NumberValue(1)); err != nil {
				t.Fatal(err)
			}
			if want := (Array{NumberValue(1)}); !reflect.DeepEqual(n, want) {
				t.Errorf("got %#v; want %#v", n, want)
			}
		})
	}
}
