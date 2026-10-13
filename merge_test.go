package tree

import (
	"testing"
)

func TestMergeOption(t *testing.T) {
	overrideMapAppend := MergeOptionOverrideMap | MergeOptionAppend

	testCases := []struct {
		caseName string
		is       func() bool
		want     bool
	}{
		{"Default.isOverrideMap", MergeOptionDefault.isOverrideMap, false},
		{"Default.isOverrideArray", MergeOptionDefault.isOverrideArray, false},
		{"Default.isOverrideValue", MergeOptionDefault.isOverrideValue, false},
		{"Default.isReplaceMap", MergeOptionDefault.isReplaceMap, false},
		{"Default.isReplaceArray", MergeOptionDefault.isReplaceArray, false},
		{"Default.isReplaceValue", MergeOptionDefault.isReplaceValue, false},
		{"Default.isAppend", MergeOptionDefault.isAppend, false},
		{"Default.isSlurp", MergeOptionDefault.isSlurp, false},
		{"OverrideMap.isOverrideMap", MergeOptionOverrideMap.isOverrideMap, true},
		{"OverrideMap.isOverrideArray", MergeOptionOverrideMap.isOverrideArray, false},
		{"OverrideMap.isOverrideValue", MergeOptionOverrideMap.isOverrideValue, true},
		{"OverrideMap.isReplaceMap", MergeOptionOverrideMap.isReplaceMap, false},
		{"OverrideMap.isReplaceArray", MergeOptionOverrideMap.isReplaceArray, false},
		{"OverrideMap.isReplaceValue", MergeOptionOverrideMap.isReplaceValue, false},
		{"OverrideMap.isAppend", MergeOptionOverrideMap.isAppend, false},
		{"OverrideMap.isSlurp", MergeOptionOverrideMap.isSlurp, false},
		{"OverrideArray.isOverrideMap", MergeOptionOverrideArray.isOverrideMap, false},
		{"OverrideArray.isOverrideArray", MergeOptionOverrideArray.isOverrideArray, true},
		{"OverrideArray.isOverrideValue", MergeOptionOverrideArray.isOverrideValue, true},
		{"OverrideArray.isReplaceMap", MergeOptionOverrideArray.isReplaceMap, false},
		{"OverrideArray.isReplaceArray", MergeOptionOverrideArray.isReplaceArray, false},
		{"OverrideArray.isReplaceValue", MergeOptionOverrideArray.isReplaceValue, false},
		{"OverrideArray.isAppend", MergeOptionOverrideArray.isAppend, false},
		{"OverrideArray.isSlurp", MergeOptionOverrideArray.isSlurp, false},
		{"Override.isOverrideMap", MergeOptionOverride.isOverrideMap, true},
		{"Override.isOverrideArray", MergeOptionOverride.isOverrideArray, true},
		{"Override.isOverrideValue", MergeOptionOverride.isOverrideValue, true},
		{"Override.isReplaceMap", MergeOptionOverride.isReplaceMap, false},
		{"Override.isReplaceArray", MergeOptionOverride.isReplaceArray, false},
		{"Override.isReplaceValue", MergeOptionOverride.isReplaceValue, false},
		{"Override.isAppend", MergeOptionOverride.isAppend, false},
		{"Override.isSlurp", MergeOptionOverride.isSlurp, false},
		{"ReplaceMap.isOverrideMap", MergeOptionReplaceMap.isOverrideMap, false},
		{"ReplaceMap.isOverrideArray", MergeOptionReplaceMap.isOverrideArray, false},
		{"ReplaceMap.isOverrideValue", MergeOptionReplaceMap.isOverrideValue, false},
		{"ReplaceMap.isReplaceMap", MergeOptionReplaceMap.isReplaceMap, true},
		{"ReplaceMap.isReplaceArray", MergeOptionReplaceMap.isReplaceArray, false},
		{"ReplaceMap.isReplaceValue", MergeOptionReplaceMap.isReplaceValue, true},
		{"ReplaceMap.isAppend", MergeOptionReplaceMap.isAppend, false},
		{"ReplaceMap.isSlurp", MergeOptionReplaceMap.isSlurp, false},
		{"ReplaceArray.isOverrideMap", MergeOptionReplaceArray.isOverrideMap, false},
		{"ReplaceArray.isOverrideArray", MergeOptionReplaceArray.isOverrideArray, false},
		{"ReplaceArray.isOverrideValue", MergeOptionReplaceArray.isOverrideValue, false},
		{"ReplaceArray.isReplaceMap", MergeOptionReplaceArray.isReplaceMap, false},
		{"ReplaceArray.isReplaceArray", MergeOptionReplaceArray.isReplaceArray, true},
		{"ReplaceArray.isReplaceValue", MergeOptionReplaceArray.isReplaceValue, true},
		{"ReplaceArray.isAppend", MergeOptionReplaceArray.isAppend, false},
		{"ReplaceArray.isSlurp", MergeOptionReplaceArray.isSlurp, false},
		{"Replace.isOverrideMap", MergeOptionReplace.isOverrideMap, false},
		{"Replace.isOverrideArray", MergeOptionReplace.isOverrideArray, false},
		{"Replace.isOverrideValue", MergeOptionReplace.isOverrideValue, false},
		{"Replace.isReplaceMap", MergeOptionReplace.isReplaceMap, true},
		{"Replace.isReplaceArray", MergeOptionReplace.isReplaceArray, true},
		{"Replace.isReplaceValue", MergeOptionReplace.isReplaceValue, true},
		{"Replace.isAppend", MergeOptionReplace.isAppend, false},
		{"Replace.isSlurp", MergeOptionReplace.isSlurp, false},
		{"Append.isOverrideMap", MergeOptionAppend.isOverrideMap, false},
		{"Append.isOverrideArray", MergeOptionAppend.isOverrideArray, false},
		{"Append.isOverrideValue", MergeOptionAppend.isOverrideValue, false},
		{"Append.isReplaceMap", MergeOptionAppend.isReplaceMap, false},
		{"Append.isReplaceArray", MergeOptionAppend.isReplaceArray, false},
		{"Append.isReplaceValue", MergeOptionAppend.isReplaceValue, false},
		{"Append.isAppend", MergeOptionAppend.isAppend, true},
		{"Append.isSlurp", MergeOptionAppend.isSlurp, false},
		{"Slurp.isOverrideMap", MergeOptionSlurp.isOverrideMap, false},
		{"Slurp.isOverrideArray", MergeOptionSlurp.isOverrideArray, false},
		{"Slurp.isOverrideValue", MergeOptionSlurp.isOverrideValue, false},
		{"Slurp.isReplaceMap", MergeOptionSlurp.isReplaceMap, false},
		{"Slurp.isReplaceArray", MergeOptionSlurp.isReplaceArray, false},
		{"Slurp.isReplaceValue", MergeOptionSlurp.isReplaceValue, false},
		{"Slurp.isAppend", MergeOptionSlurp.isAppend, false},
		{"Slurp.isSlurp", MergeOptionSlurp.isSlurp, true},
		{"OverrideMap|Append.isOverrideMap", overrideMapAppend.isOverrideMap, true},
		{"OverrideMap|Append.isOverrideArray", overrideMapAppend.isOverrideArray, false},
		{"OverrideMap|Append.isOverrideValue", overrideMapAppend.isOverrideValue, true},
		{"OverrideMap|Append.isReplaceMap", overrideMapAppend.isReplaceMap, false},
		{"OverrideMap|Append.isReplaceArray", overrideMapAppend.isReplaceArray, false},
		{"OverrideMap|Append.isReplaceValue", overrideMapAppend.isReplaceValue, false},
		{"OverrideMap|Append.isAppend", overrideMapAppend.isAppend, true},
		{"OverrideMap|Append.isSlurp", overrideMapAppend.isSlurp, false},
	}
	for _, tc := range testCases {
		t.Run(tc.caseName, func(t *testing.T) {
			if got := tc.is(); got != tc.want {
				t.Errorf("got %v; want %v", got, tc.want)
			}
		})
	}
}

func TestMerge(t *testing.T) {
	testCases := []struct {
		caseName string
		a        Node
		b        Node
		opts     MergeOption
		want     Node
	}{
		{
			caseName: "Default Map merges new key",
			a:        Map{"a": ToValue(1), "b": ToValue(2)},
			b:        Map{"a": ToValue(3), "c": ToValue(4)},
			want:     Map{"a": ToValue(1), "b": ToValue(2), "c": ToValue(4)},
		}, {
			caseName: "Default Map vs Nil keeps a",
			a:        Map{"a": ToValue(1), "b": ToValue(2)},
			b:        Nil,
			want:     Map{"a": ToValue(1), "b": ToValue(2)},
		}, {
			caseName: "Default Array overlays trailing",
			a:        ToArrayValues(1, 2),
			b:        ToArrayValues(3, 4, 5),
			want:     ToArrayValues(1, 2, 5),
		}, {
			caseName: "Default Array longer a",
			a:        ToArrayValues(1, 2, 3),
			b:        ToArrayValues(4, 5),
			want:     ToArrayValues(1, 2, 3),
		}, {
			caseName: "Default Value keeps a",
			a:        ToValue("a"),
			b:        ToValue("b"),
			want:     ToValue("a"),
		}, {
			caseName: "Default Map vs Value keeps a",
			a:        Map{"a": ToValue(1), "b": ToValue(2)},
			b:        ToValue("c"),
			want:     Map{"a": ToValue(1), "b": ToValue(2)},
		}, {
			caseName: "Default Array vs Value keeps a",
			a:        ToArrayValues("a"),
			b:        ToValue("b"),
			want:     ToArrayValues("a"),
		}, {
			caseName: "OverrideMap Map overlays existing key",
			a:        Map{"a": ToValue(1), "b": ToValue(2)},
			b:        Map{"a": ToValue(3)},
			opts:     MergeOptionOverrideMap,
			want:     Map{"a": ToValue(3), "b": ToValue(2)},
		}, {
			caseName: "OverrideMap Map vs Nil",
			a:        Map{"a": ToValue(1), "b": ToValue(2)},
			b:        Nil,
			opts:     MergeOptionOverrideMap,
			want:     Nil,
		}, {
			caseName: "OverrideMap Value override",
			a:        ToValue("a"),
			b:        ToValue("b"),
			opts:     MergeOptionOverrideMap,
			want:     ToValue("b"),
		}, {
			caseName: "OverrideMap mixed types",
			a:        Map{"a": ToValue(1), "b": ToValue(2), "c": ToValue(3)},
			b:        Map{"a": ToValue(4), "b": ToArrayValues(5, 6), "d": ToValue(7)},
			opts:     MergeOptionOverrideMap,
			want:     Map{"a": ToValue(4), "b": ToArrayValues(5, 6), "c": ToValue(3), "d": ToValue(7)},
		}, {
			caseName: "OverrideArray shorter b",
			a:        ToArrayValues(1, 2, 3),
			b:        ToArrayValues(4, 5),
			opts:     MergeOptionOverrideArray,
			want:     ToArrayValues(4, 5, 3),
		}, {
			caseName: "OverrideArray longer b",
			a:        ToArrayValues(1, 2, 3),
			b:        ToArrayValues(4, 5, 6, 7),
			opts:     MergeOptionOverrideArray,
			want:     ToArrayValues(4, 5, 6, 7),
		}, {
			caseName: "OverrideArray Value override",
			a:        ToValue("a"),
			b:        ToValue("b"),
			opts:     MergeOptionOverrideArray,
			want:     ToValue("b"),
		}, {
			caseName: "Override Map",
			a:        Map{"a": ToValue(1), "b": ToValue(2)},
			b:        Map{"a": ToValue(3)},
			opts:     MergeOptionOverride,
			want:     Map{"a": ToValue(3), "b": ToValue(2)},
		}, {
			caseName: "Override Array",
			a:        ToArrayValues(1, 2, 3),
			b:        ToArrayValues(4, 5),
			opts:     MergeOptionOverride,
			want:     ToArrayValues(4, 5, 3),
		}, {
			caseName: "Override Value",
			a:        ToValue("a"),
			b:        ToValue("b"),
			opts:     MergeOptionOverride,
			want:     ToValue("b"),
		}, {
			caseName: "ReplaceMap Map",
			a:        Map{"a": ToValue(1), "b": ToValue(2)},
			b:        Map{"a": ToValue(3)},
			opts:     MergeOptionReplaceMap,
			want:     Map{"a": ToValue(3)},
		}, {
			caseName: "ReplaceMap Value first",
			a:        ToValue("a"),
			b:        ToValue("b"),
			opts:     MergeOptionReplaceMap,
			want:     ToValue("b"),
		}, {
			caseName: "ReplaceMap Value second",
			a:        ToValue("a"),
			b:        ToValue("b"),
			opts:     MergeOptionReplaceMap,
			want:     ToValue("b"),
		}, {
			caseName: "ReplaceArray Array",
			a:        ToArrayValues(1, 2, 3),
			b:        ToArrayValues(4, 5),
			opts:     MergeOptionReplaceArray,
			want:     ToArrayValues(4, 5),
		}, {
			caseName: "ReplaceArray Value",
			a:        ToValue("a"),
			b:        ToValue("b"),
			opts:     MergeOptionReplaceArray,
			want:     ToValue("b"),
		}, {
			caseName: "Replace Value",
			a:        ToValue("a"),
			b:        ToValue("b"),
			opts:     MergeOptionReplace,
			want:     ToValue("b"),
		}, {
			caseName: "Replace Array",
			a:        ToArrayValues(1, 2, 3),
			b:        ToArrayValues(4, 5),
			opts:     MergeOptionReplace,
			want:     ToArrayValues(4, 5),
		}, {
			caseName: "Replace Value second",
			a:        ToValue("a"),
			b:        ToValue("b"),
			opts:     MergeOptionReplace,
			want:     ToValue("b"),
		}, {
			caseName: "Append Array",
			a:        ToArrayValues(1, 2, 3),
			b:        ToArrayValues(4, 5),
			opts:     MergeOptionAppend,
			want:     ToArrayValues(1, 2, 3, 4, 5),
		}, {
			caseName: "Slurp Array Array",
			a:        ToArrayValues(1, 2, 3),
			b:        ToArrayValues(4, 5),
			opts:     MergeOptionSlurp,
			want:     ToArrayValues(1, 2, 3, 4, 5),
		}, {
			caseName: "Slurp Array Value",
			a:        ToArrayValues(1, 2, 3),
			b:        ToValue(4),
			opts:     MergeOptionSlurp,
			want:     ToArrayValues(1, 2, 3, 4),
		}, {
			caseName: "Slurp Value Value",
			a:        ToValue(1),
			b:        ToValue(2),
			opts:     MergeOptionSlurp,
			want:     ToArrayValues(1, 2),
		}, {
			caseName: "Slurp Map Map",
			a:        Map{"a": ToValue(1)},
			b:        Map{"a": ToValue(2)},
			opts:     MergeOptionSlurp,
			want:     Map{"a": ToArrayValues(1, 2)},
		}, {
			caseName: "OverrideMap|Append nested",
			a: Map{
				"map":          Map{"a": ToValue(1), "b": ToValue(2)},
				"array":        ToArrayValues(3, 4, 5),
				"arrayOrValue": ToArrayValues(6, 7, 8),
			},
			b: Map{
				"map":          Map{"a": ToValue(9)},
				"array":        ToArrayValues(10, 11),
				"arrayOrValue": ToValue(12),
			},
			opts: MergeOptionOverrideMap | MergeOptionAppend,
			want: Map{
				"map":          Map{"a": ToValue(9), "b": ToValue(2)},
				"array":        ToArrayValues(3, 4, 5, 10, 11),
				"arrayOrValue": ToValue(12),
			},
		}, {
			caseName: "Append|Slurp nested",
			a: Map{
				"map":          Map{"a": ToValue(1), "b": ToValue(2)},
				"array":        ToArrayValues(3, 4, 5),
				"arrayOrValue": ToArrayValues(6, 7, 8),
			},
			b: Map{
				"map":          Map{"a": ToValue(9)},
				"array":        ToArrayValues(10, 11),
				"arrayOrValue": ToValue(12),
			},
			opts: MergeOptionAppend | MergeOptionSlurp,
			want: Map{
				"map":          Map{"a": ToArrayValues(1, 9), "b": ToValue(2)},
				"array":        ToArrayValues(3, 4, 5, 10, 11),
				"arrayOrValue": ToArrayValues(6, 7, 8, 12),
			},
		}, {
			caseName: "ReplaceMap|Append nested",
			a: Map{
				"map":   Map{"a": ToValue(1), "b": ToValue(2)},
				"array": ToArrayValues(3, 4, 5),
			},
			b: Map{
				"map":   Map{"a": ToValue(6)},
				"array": ToArrayValues(7, 8),
			},
			opts: MergeOptionReplaceMap | MergeOptionAppend,
			want: Map{
				"map":   Map{"a": ToValue(6)},
				"array": ToArrayValues(7, 8),
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.caseName, func(t *testing.T) {
			a, b := CloneDeep(tc.a), CloneDeep(tc.b)
			got := Merge(CloneDeep(tc.a), CloneDeep(tc.b), tc.opts)
			if !Equal(got, tc.want) {
				t.Errorf("unexpected %v; want %v", got, tc.want)
			}
			if !Equal(a, tc.a) {
				t.Errorf("a was mutated: %v; want %v", a, tc.a)
			}
			if !Equal(b, tc.b) {
				t.Errorf("b was mutated: %v; want %v", b, tc.b)
			}
		})
	}
}
