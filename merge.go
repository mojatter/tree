package tree

import (
	"fmt"
	"strings"
)

// MergeOption represents different merge strategies for combining nodes.
type MergeOption int

const (
	// MergeOptionDefault merges with the following default rules.
	// For examples:
	// - {"a": 1, "b": 2} and {"a": 3, "c": 4} merges to {"a": 1, "b": 2, "c": 4}
	// - [1, 2] and [3, 4, 5] merges to [1, 2, 5]
	// - "a" and "b" merges to "a"
	MergeOptionDefault MergeOption = 0
	// MergeOptionOverrideMap overrides duplicate map keys.
	// For examples:
	// - {"a": 1, "b": 2} and {"a": 3} merges to {"a": 3, "b": 2}
	// - "a" and "b" merges to "b
	MergeOptionOverrideMap MergeOption = 0b000001
	// MergeOptionOverrideArray overrides duplicate array indexes.
	// For examples:
	// - [1, 2, 3] and [4, 5] merges to [4, 5, 3]
	// - "a" and "b" merges to "b
	MergeOptionOverrideArray MergeOption = 0b000010
	// MergeOptionOverride overrides duplicate map keys and array indexes.
	// For examples:
	// - {"a": 1, "b": 2} and {"a": 3} merges to {"a": 3, "b": 2}
	// - [1, 2, 3] and [4, 5] merges to [4, 5, 3]
	// - "a" and "b" merges to "b"
	MergeOptionOverride MergeOption = MergeOptionOverrideMap | MergeOptionOverrideArray
	// MergeOptionReplaceMap merges with replace map.
	// For examples:
	// - {"a": 1, "b": 2} and {"a": 3} merges to {"a": 3}
	// - "a" and "b" merges to "b"
	MergeOptionReplaceMap MergeOption = 0b000100
	// MergeOptionReplaceArray merges with replace array.
	// For examples:
	// - [1, 2, 3] and [4, 5] merges to [4, 5]
	// - "a" and "b" merges to "b"
	MergeOptionReplaceArray MergeOption = 0b001000
	// MergeOptionReplace merges with replace.
	// For examples:
	// - {"a": 1, "b": 2} and {"a": 3} merges to {"a": 3}
	// - [1, 2, 3] and [4, 5] merges to [4, 5]
	// - "a" and "b" merges to "b"
	MergeOptionReplace MergeOption = MergeOptionReplaceMap | MergeOptionReplaceArray
	// MergeOptionAppend acts when both are arrays and append them.
	// Inside array merging this is checked before Override and Replace,
	// so when both operands are arrays Append wins over those flags.
	// For examples:
	// - [1, 2, 3] and [4, 5] merges to [1, 2, 3, 4, 5]
	MergeOptionAppend MergeOption = 0b010000
	// MergeOptionSlurp acts on an array or value and converts it to an array and
	// merges it, even if the value is not an array. When the two operands have
	// non-matching types (e.g. array vs scalar) it wins over Override and Replace
	// because the slurp wrapping happens before the type-mismatch fallback.
	// Inside Map merging Slurp follows the same per-key recursion as
	// MergeOptionOverrideMap, so leaf type-mismatched values are wrapped into
	// an array (see "Slurp Map Map" example below).
	// For examples:
	// - [1, 2, 3] and [4, 5] merges to [1, 2, 3, 4, 5]
	// - [1, 2, 3] and 4 merges to [1, 2, 3, 4]
	// - 1 and 2 merges to [1, 2]
	// - {"a": 1} and {"a": 2} merges to {"a": [1, 2]}
	MergeOptionSlurp MergeOption = 0b100000
)

// isOverrideMap checks if the merge option includes map override behavior.
func (o MergeOption) isOverrideMap() bool {
	return o&MergeOptionOverrideMap == MergeOptionOverrideMap
}

// isOverrideArray checks if the merge option includes array override behavior.
func (o MergeOption) isOverrideArray() bool {
	return o&MergeOptionOverrideArray == MergeOptionOverrideArray
}

// isOverrideValue checks if the merge option includes any override behavior.
func (o MergeOption) isOverrideValue() bool {
	return o.isOverrideMap() || o.isOverrideArray()
}

// isReplaceMap checks if the merge option includes map replacement behavior.
func (o MergeOption) isReplaceMap() bool {
	return o&MergeOptionReplaceMap == MergeOptionReplaceMap
}

// isReplaceArray checks if the merge option includes array replacement behavior.
func (o MergeOption) isReplaceArray() bool {
	return o&MergeOptionReplaceArray == MergeOptionReplaceArray
}

// isReplaceValue checks if the merge option includes any replacement behavior.
func (o MergeOption) isReplaceValue() bool {
	return o.isReplaceMap() || o.isReplaceArray()
}

// isAppend checks if the merge option includes array append behavior.
func (o MergeOption) isAppend() bool {
	return o&MergeOptionAppend == MergeOptionAppend
}

// isSlurp checks if the merge option includes slurp behavior (convert to array).
func (o MergeOption) isSlurp() bool {
	return o&MergeOptionSlurp == MergeOptionSlurp
}

// MergeOptionFromString parses a comma-separated merge strategy spec
// into a MergeOption. Recognised names map directly to the exported
// MergeOption* values:
//
//   - "default"        — MergeOptionDefault (zero value)
//   - "override"       — MergeOptionOverride (= override-map | override-array)
//   - "override-map"   — MergeOptionOverrideMap
//   - "override-array" — MergeOptionOverrideArray
//   - "replace"        — MergeOptionReplace (= replace-map | replace-array)
//   - "replace-map"    — MergeOptionReplaceMap
//   - "replace-array"  — MergeOptionReplaceArray
//   - "append"         — MergeOptionAppend
//   - "slurp"          — MergeOptionSlurp
//
// Multiple names may be combined with commas: "override,append" yields
// MergeOptionOverride | MergeOptionAppend. Whitespace around each name
// is trimmed. An empty spec returns MergeOptionDefault. An unknown name
// produces a non-nil error and a zero MergeOption.
//
// Intended for CLI / config consumers (e.g. tq's --merge flag) so the
// option-name vocabulary lives next to the option definitions.
func MergeOptionFromString(spec string) (MergeOption, error) {
	if spec == "" {
		return MergeOptionDefault, nil
	}
	var opts MergeOption
	for name := range strings.SplitSeq(spec, ",") {
		name = strings.TrimSpace(name)
		switch name {
		case "default":
			// no flag bits
		case "override":
			opts |= MergeOptionOverride
		case "override-map":
			opts |= MergeOptionOverrideMap
		case "override-array":
			opts |= MergeOptionOverrideArray
		case "replace":
			opts |= MergeOptionReplace
		case "replace-map":
			opts |= MergeOptionReplaceMap
		case "replace-array":
			opts |= MergeOptionReplaceArray
		case "append":
			opts |= MergeOptionAppend
		case "slurp":
			opts |= MergeOptionSlurp
		default:
			return 0, fmt.Errorf("unknown merge strategy %q", name)
		}
	}
	return opts, nil
}

// Merge merges two nodes with MergeOption. The opts value propagates
// recursively into every nested merge call, so a flag set at the top
// level (e.g. Append) also applies to arrays found at any depth.
//
// Merge mutates a in place and may return either a or b depending on
// the rules. If you need to preserve the inputs, clone before calling:
//
//	merged := tree.Merge(tree.CloneDeep(a), tree.CloneDeep(b), opts)
func Merge(a, b Node, opts MergeOption) Node {
	if a.Type().IsMap() {
		if b.Type().IsMap() {
			return mergeMap(a.Map(), b.Map(), opts)
		}
		return mergeNoMatchType(a, b, opts)
	}
	if a.Type().IsArray() {
		if b.Type().IsArray() {
			return mergeArray(a.Array(), b.Array(), opts)
		}
		if opts.isSlurp() {
			return mergeArray(a.Array(), Array{b}, opts)
		}
		return mergeNoMatchType(a, b, opts)
	}
	if opts.isSlurp() {
		if !b.Type().IsMap() {
			return mergeArray(Array{a}, Array{b}, opts)
		}
	}
	return mergeNoMatchType(a, b, opts)
}

// mergeNoMatchType handles merging when node types don't match.
// Returns b if any Override*/Replace* flag is set, otherwise a.
//
// Note: the predicate is "any override-or-replace flag", not the
// specific one matching the operand types. Setting only
// MergeOptionOverrideArray therefore still causes a Map vs scalar
// merge to choose b. Distinguishing per-shape override on type
// mismatch would be a behaviour change; revisit at v1.0.
func mergeNoMatchType(a Node, b Node, opts MergeOption) Node {
	if opts.isOverrideValue() || opts.isReplaceValue() {
		return b
	}
	return a
}

// mergeArray merges two arrays according to the specified merge options.
// Supports append, override, replace, and slurp behaviors.
func mergeArray(a, b Array, opts MergeOption) Array {
	if opts.isAppend() || opts.isSlurp() {
		return append(a, b...)
	}
	if opts.isOverrideArray() {
		for i, v := range b {
			if i < len(a) {
				// Set only fails on out-of-range index; the i < len(a)
				// guard above makes that impossible here.
				_ = a.Set(i, Merge(a[i], v, opts))
			} else {
				a = append(a, v)
			}
		}
		return a
	}
	if opts.isReplaceArray() {
		return b
	}
	if len(a) < len(b) {
		return append(a, b[len(a):]...)
	}
	return a
}

// mergeMap merges two maps according to the specified merge options.
// Supports override, replace, and recursive merging of nested structures.
func mergeMap(a, b Map, opts MergeOption) Map {
	if opts.isSlurp() || opts.isOverrideMap() {
		for k, v := range b {
			if vv, exists := a[k]; exists {
				a[k] = Merge(vv, v, opts)
			} else {
				a[k] = v
			}
		}
		return a
	}
	if opts.isReplaceMap() {
		return b
	}
	for k, v := range b {
		if _, exists := a[k]; !exists {
			a[k] = v
		}
	}
	return a
}
