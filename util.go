package tree

import (
	"errors"
	"fmt"
	"regexp"
	"sync"
)

// V is a short alias for [ToValue]. It is intended to keep
// composite-literal construction of [Map] / [Array] concise:
//
//	tree.Map{"Name": tree.V("Alice"), "Age": tree.V(30)}
func V(v any) Node {
	return ToValue(v)
}

// IntPtr, Int64Ptr, and Float64Ptr return a pointer to v. They are
// convenience helpers for constructing pointer-valued struct fields
// (e.g. ArrayRangeQuery.From, schema rule bounds) inline.

// IntPtr returns a pointer to v.
func IntPtr(v int) *int { return &v }

// Int64Ptr returns a pointer to v.
func Int64Ptr(v int64) *int64 { return &v }

// Float64Ptr returns a pointer to v.
func Float64Ptr(v float64) *float64 { return &v }

// A is a short alias for [ToArrayValues]. It pairs with [V] for concise
// construction of arrays:
//
//	tree.A("Crimson", "Red", "Ruby")
func A(vs ...any) Array {
	return ToArrayValues(vs...)
}

// ToValue converts the specified v to a Value as Node.
// Node.Value() returns converted value.
func ToValue(v any) Node {
	if v == nil {
		return Nil
	}
	switch tv := v.(type) {
	case string:
		return StringValue(tv)
	case bool:
		return BoolValue(tv)
	case int:
		return NumberValue(int64(tv))
	case int64:
		return NumberValue(tv)
	case int32:
		return NumberValue(int64(tv))
	case float64:
		return NumberValue(tv)
	case float32:
		return NumberValue(float64(tv))
	case uint64:
		return NumberValue(float64(tv))
	case uint32:
		return NumberValue(float64(tv))
	case Node:
		return v.(Node)
	}
	// NOTE: Unsupported type.
	return StringValue(fmt.Sprintf("%#v", v))
}

// ToArrayValues calls ToValue for each provided vs and returns them as an Array.
func ToArrayValues(vs ...any) Array {
	a := make(Array, len(vs))
	for i, v := range vs {
		a[i] = ToValue(v)
	}
	return a
}

// ToNodeValues calls ToValue for each provided vs and returns them as []Node.
func ToNodeValues(vs ...any) []Node {
	ns := make([]Node, len(vs))
	for i, v := range vs {
		ns[i] = ToValue(v)
	}
	return ns
}

// ToNode converts the specified v to an Node.
func ToNode(v any) Node {
	if v == nil {
		return Nil
	}
	switch tv := v.(type) {
	case Node:
		return tv
	case []any:
		a := make(Array, len(tv))
		for i, vv := range tv {
			a[i] = ToNode(vv)
		}
		return a
	case map[string]any:
		m := Map{}
		for k := range tv {
			m[k] = ToNode(tv[k])
		}
		return m
	case map[any]any:
		m := Map{}
		for k := range tv {
			m[fmt.Sprintf("%v", k)] = ToNode(tv[k])
		}
		return m
	}
	return ToValue(v)
}

// ToAny converts a Node back to a native Go any value.
// This is the reverse operation of ToNode.
func ToAny(n Node) any {
	if n == nil {
		return nil
	}
	t := n.Type()
	switch t {
	case TypeArray:
		a := n.Array()
		x := make([]any, len(a))
		for i, v := range a {
			x[i] = ToAny(v)
		}
		return x
	case TypeMap:
		m := n.Map()
		x := make(map[string]any, len(m))
		for k, v := range m {
			x[k] = ToAny(v)
		}
		return x
	case TypeNilValue:
		return nil
	case TypeStringValue:
		return n.Value().String()
	case TypeBoolValue:
		return n.Value().Bool()
	case TypeNumberValue:
		return n.Value().Float64()
	}
	panic(fmt.Errorf("unknown type %v", t))
}

// SkipWalk is used as a return value from WalkFunc to indicate that
// the node and its children in the call are to be skipped.
// It is not returned as an error by any function.
//
// Like [io/fs.SkipDir] and [io/fs.SkipAll], this is a sentinel control
// value rather than a failure, so it intentionally does not use the
// "Err" prefix recommended by ST1012.
var SkipWalk = errors.New("skip") //nolint:staticcheck // ST1012: sentinel value, see godoc

// WalkFunc is the type of the function called by Walk to visit each nodes.
//
// The keys argument contains that parent keys and the node key that
// type is int (array index) or string (map key).
type WalkFunc func(n Node, keys []any) error

// Walk walks the node tree rooted at root, calling fn for each node or
// that children in the tree, including root.
func Walk(n Node, fn WalkFunc) error {
	return walk(n, []any{}, fn)
}

// walk is a recursive helper function that traverses the node tree.
// It maintains the path of keys from root to current node.
func walk(n Node, lastKeys []any, fn WalkFunc) error {
	if n == nil {
		return nil
	}
	if err := fn(n, lastKeys); err != nil {
		if err == SkipWalk {
			return nil
		}
		return err
	}

	last := len(lastKeys)
	keys := make([]any, last+1)
	copy(keys, lastKeys)

	return n.Each(func(key any, v Node) error {
		if key == nil {
			return nil
		}
		keys[last] = key
		return walk(v, keys, fn)
	})
}

var regexpPool = sync.Pool{
	New: func() any {
		return map[string]*regexp.Regexp{}
	},
}

// pooledRegexp retrieves a compiled regexp from the pool or creates a new one.
// Uses sync.Pool for efficient regexp reuse to avoid recompilation.
func pooledRegexp(expr string) (*regexp.Regexp, error) {
	cache := regexpPool.Get().(map[string]*regexp.Regexp)
	defer regexpPool.Put(cache)

	if re, ok := cache[expr]; ok {
		return re, nil
	}
	re, err := regexp.Compile(expr)
	if err != nil {
		return nil, err
	}
	cache[expr] = re
	return re, nil
}

// regexpMatchString matches a string against a regular expression pattern.
// Uses pooled regexp compilation for better performance.
func regexpMatchString(expr, value string) (bool, error) {
	re, err := pooledRegexp(expr)
	if err != nil {
		return false, err
	}
	return re.MatchString(value), nil
}

// Clone clones the node.
func Clone(n Node) Node {
	return clone(n, false)
}

// CloneDeep clones the node.
func CloneDeep(n Node) Node {
	return clone(n, true)
}

// clone creates a copy of the node with optional deep cloning.
// If deep is true, recursively clones all child nodes.
func clone(n Node, deep bool) Node {
	switch n.Type() {
	case TypeArray:
		a := n.Array()
		aa := make(Array, len(a))
		for i := 0; i < len(a); i++ {
			if deep {
				aa[i] = Clone(a[i])
			} else {
				aa[i] = a[i]
			}
		}
		return aa
	case TypeMap:
		m := n.Map()
		mm := make(Map, len(m))
		for k, v := range m {
			if deep {
				mm[k] = Clone(v)
			} else {
				mm[k] = v
			}
		}
		return mm
	}
	return n
}
