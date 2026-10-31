package tree

import (
	"fmt"
	"sort"
	"strconv"
)

// Type represents the Node type.
type Type int

// These constants are the Node types.
const (
	TypeArray       Type = 0b0001
	TypeMap         Type = 0b0010
	TypeValue       Type = 0b1000
	TypeNilValue    Type = 0b1001
	TypeStringValue Type = 0b1010
	TypeBoolValue   Type = 0b1011
	TypeNumberValue Type = 0b1100
)

// IsArray returns t == TypeArray.
func (t Type) IsArray() bool {
	return t == TypeArray
}

// IsMap returns t == TypeMap.
func (t Type) IsMap() bool {
	return t == TypeMap
}

// IsValue returns true if t has the TypeValue bit set.
func (t Type) IsValue() bool {
	return t&TypeValue != 0
}

// IsNilValue returns t == TypeNilValue.
func (t Type) IsNilValue() bool {
	return t == TypeNilValue
}

// IsStringValue returns t == TypeStringValue.
func (t Type) IsStringValue() bool {
	return t == TypeStringValue
}

// IsBoolValue returns t == TypeBoolValue.
func (t Type) IsBoolValue() bool {
	return t == TypeBoolValue
}

// IsNumberValue returns t == TypeNumberValue.
func (t Type) IsNumberValue() bool {
	return t == TypeNumberValue
}

// String returns the JSON name of t, as type() does: "array", "object",
// "null", "string", "boolean" or "number"; the bare TypeValue bitmask is
// "value" and an unknown bit pattern "unknown".
func (t Type) String() string {
	switch t {
	case TypeArray:
		return "array"
	case TypeMap:
		return "object"
	case TypeNilValue:
		return "null"
	case TypeStringValue:
		return "string"
	case TypeBoolValue:
		return "boolean"
	case TypeNumberValue:
		return "number"
	case TypeValue:
		return "value"
	}
	return "unknown"
}

// A Node is an element on the tree.
type Node interface {
	// IsNil returns true if this node is nil.
	IsNil() bool
	// Type returns this node type.
	Type() Type
	// Array returns this node as an Array.
	// If this type is not Array, returns an Array(nil).
	Array() Array
	// Map returns this node as a Map.
	// If this type is not Map, returns a Map(nil).
	Map() Map
	// Value returns this node as a Value.
	// If this type is not Value, returns a NilValue.
	Value() Value
	// Has checks this node has key.
	Has(keys ...any) bool
	// Get returns the child node that matches the specified keys.
	// The key type allows int or string.
	// Get returns NilValue both when the key is missing and when the
	// stored value is nil. Use [Node.Has] to distinguish between the two.
	Get(keys ...any) Node
	// Each calls the callback function for each Array|Map values.
	// If the node type is not Array|Map then the callback called once with nil key and self as value.
	Each(cb func(key any, v Node) error) error
	// Find finds a node using the query expression.
	Find(expr string) ([]Node, error)
}

// EditorNode is an interface that defines the methods to edit this node.
type EditorNode interface {
	Node
	Append(v Node) error
	Set(key any, v Node) error
	Delete(key any) error
}

// Any wraps a Node and delegates every Node method to it, reading the Node it wraps as Nil when it is a Go nil or nil pointer.
type Any struct {
	Node
}

var _ Node = (*Any)(nil)

// IsNil delegates to the wrapped Node.
func (n Any) IsNil() bool {
	return orNil(n.Node).IsNil()
}

// Type returns the type of the wrapped Node.
func (n Any) Type() Type {
	return orNil(n.Node).Type()
}

// Array returns the wrapped Node as an Array.
func (n Any) Array() Array {
	return orNil(n.Node).Array()
}

// Map returns the wrapped Node as a Map.
func (n Any) Map() Map {
	return orNil(n.Node).Map()
}

// Value returns the wrapped Node as a Value.
func (n Any) Value() Value {
	return orNil(n.Node).Value()
}

// Has delegates to the wrapped Node.
func (n Any) Has(keys ...any) bool {
	return orNil(n.Node).Has(keys...)
}

// Get returns the child node of the wrapped Node that matches the specified keys.
func (n Any) Get(keys ...any) Node {
	return orNil(n.Node).Get(keys...)
}

// Each delegates to the wrapped Node.
func (n Any) Each(cb func(key any, n Node) error) error {
	return orNil(n.Node).Each(cb)
}

// Find finds nodes in the wrapped Node using the query expression.
func (n Any) Find(expr string) ([]Node, error) {
	return orNil(n.Node).Find(expr)
}

// Array represents an array of Node.
type Array []Node

var (
	_ Node       = (Array)(nil)
	_ EditorNode = (*Array)(nil)
)

// IsNil returns true if this node is nil.
func (n Array) IsNil() bool {
	return n == nil
}

// Type returns TypeArray.
func (n Array) Type() Type {
	return TypeArray
}

// Array returns this node as an Array.
func (n Array) Array() Array {
	return n
}

// Map returns a Map(nil).
func (n Array) Map() Map {
	return nil
}

// Value returns a NilValue.
func (n Array) Value() Value {
	return Nil
}

// toIndex converts a key to an array index and checks if it's valid.
// Supports both int and string representations of indices.
func (n Array) toIndex(key any) (int, bool) {
	switch tk := key.(type) {
	case int:
		if tk >= 0 {
			return tk, tk < len(n)
		}
	case string:
		k, err := strconv.Atoi(tk)
		if err == nil && k >= 0 {
			return k, k < len(n)
		}
	}
	return -1, false
}

// Has checks this node has key.
func (n Array) Has(keys ...any) bool {
	if len(keys) > 0 {
		if i, ok := n.toIndex(keys[0]); ok {
			if len(keys) > 1 {
				return orNil(n[i]).Has(keys[1:]...)
			}
			return true
		}
	}
	return false
}

// Get returns the child node at the given index.
// It returns NilValue if the index is out of range or the element is nil.
func (n Array) Get(keys ...any) Node {
	if len(keys) > 0 {
		if i, ok := n.toIndex(keys[0]); ok {
			if len(keys) > 1 {
				return orNil(n[i]).Get(keys[1:]...)
			}
			return orNil(n[i])
		}
	}
	return Nil
}

// Each calls the callback function for each element of the Array.
func (n Array) Each(cb func(key any, n Node) error) error {
	for i, v := range n {
		if err := cb(i, v); err != nil {
			return err
		}
	}
	return nil
}

// Find finds a node using the query expression.
func (n Array) Find(expr string) ([]Node, error) {
	return Find(n, expr)
}

// Append appends v to *n.
func (n *Array) Append(v Node) error {
	*n = append(*n, v)
	return nil
}

// Set sets v to n[key], growing n as needed; indexes of 1<<24 or more are rejected.
func (n *Array) Set(key any, v Node) error {
	i, ok := n.toIndex(key)
	if i == -1 {
		return errIndex(n, key)
	}
	if !ok {
		a, err := growArray(*n, i)
		if err != nil {
			return err
		}
		*n = a
	}
	(*n)[i] = v
	return nil
}

// maxArrayLen caps how far an out-of-range index may grow an array.
const maxArrayLen = 1 << 24

// growArray returns a copy of a extended so that index i exists.
func growArray(a Array, i int) (Array, error) {
	if i >= maxArrayLen {
		return nil, fmt.Errorf("array index %d exceeds maximum %d", i, maxArrayLen-1)
	}
	na := make(Array, i+1)
	copy(na, a)
	return na, nil
}

// Delete deletes n[key].
func (n *Array) Delete(key any) error {
	i, ok := n.toIndex(key)
	if i == -1 {
		return errIndex(n, key)
	}
	if ok {
		a := *n
		*n = append(a[0:i], a[i+1:]...)
	}
	return nil
}

// Map represents a map of Node.
type Map map[string]Node

var _ EditorNode = (Map)(nil)

// IsNil returns true if this node is nil.
func (n Map) IsNil() bool {
	return n == nil
}

// Type returns TypeMap.
func (n Map) Type() Type {
	return TypeMap
}

// Array returns an Array(nil).
func (n Map) Array() Array {
	return nil
}

// Map returns this node as a Map.
func (n Map) Map() Map {
	return n
}

// Value returns a NilValue.
func (n Map) Value() Value {
	return Nil
}

// toKey converts a key to a string and checks if it exists in the map.
// Supports both int and string keys.
func (n Map) toKey(key any) (string, bool) {
	switch tk := key.(type) {
	case int:
		k := strconv.Itoa(tk)
		_, ok := n[k]
		return k, ok
	case string:
		_, ok := n[tk]
		return tk, ok
	}
	return "", false
}

// Has checks this node has key.
func (n Map) Has(keys ...any) bool {
	if len(keys) > 0 {
		if k, ok := n.toKey(keys[0]); ok {
			if len(keys) > 1 {
				return orNil(n[k]).Has(keys[1:]...)
			}
			return true
		}
	}
	return false
}

// Get returns the child node for the given key.
// It returns NilValue if the key is missing or the stored value is nil.
func (n Map) Get(keys ...any) Node {
	if len(keys) > 0 {
		if k, ok := n.toKey(keys[0]); ok {
			if len(keys) > 1 {
				return orNil(n[k]).Get(keys[1:]...)
			}
			return orNil(n[k])
		}
	}
	return Nil
}

// Keys returns sorted keys of the map.
func (n Map) Keys() []string {
	keys := make([]string, len(n))
	i := 0
	for k := range n {
		keys[i] = k
		i++
	}
	sort.Strings(keys)
	return keys
}

// Values returns values of the map, ordered by sorted key, with Nil for a nil value.
func (n Map) Values() []Node {
	values := make([]Node, len(n))
	for i, k := range n.Keys() {
		values[i] = orNil(n[k])
	}
	return values
}

// Append returns an error.
func (n Map) Append(v Node) error {
	return errAppend(n)
}

// Set sets v to n[key].
func (n Map) Set(key any, v Node) error {
	if n == nil {
		return fmt.Errorf("cannot set %s in a nil map", formatStep(key))
	}
	switch tk := key.(type) {
	case int:
		n[strconv.Itoa(tk)] = v
		return nil
	case string:
		n[tk] = v
		return nil
	}
	return errIndex(n, key)
}

// Delete deletes n[key].
func (n Map) Delete(key any) error {
	switch tk := key.(type) {
	case int:
		delete(n, strconv.Itoa(tk))
		return nil
	case string:
		delete(n, tk)
		return nil
	}
	return errIndex(n, key)
}

// Each calls the callback function for each value of the Map, in sorted key order.
func (n Map) Each(cb func(key any, n Node) error) error {
	for _, k := range n.Keys() {
		if err := cb(k, n[k]); err != nil {
			return err
		}
	}
	return nil
}

// Find finds a node using the query expression.
func (n Map) Find(expr string) ([]Node, error) {
	return Find(n, expr)
}

// Equal reports whether two Node values represent the same tree.
//
// Equality semantics:
//
//   - Type tags must match. NumberValue(1) and StringValue("1") are
//     not equal even though they format to the same string.
//   - Untyped nil, a nil pointer node such as (*Array)(nil) and the
//     Nil sentinel (NilValue{}) compare equal to each other.
//   - Map equality is order-independent. Map{"a":1, "b":2} equals
//     Map{"b":2, "a":1}.
//   - Array equality is order-dependent. Array{1, 2} does not equal
//     Array{2, 1}.
//   - NumberValue equality follows IEEE 754. NumberValue(NaN) does
//     not equal NumberValue(NaN).
//   - Map(nil) equals Map{} (both empty Map); the same applies to
//     Array(nil) and Array{}. Neither equals Nil because the type
//     tags differ.
//   - Nodes are compared through the Node interface, so Any, pointer
//     nodes such as *Map and custom Node types compare by Type and content.
//   - Trees must be acyclic; a cycle such as a Map holding a pointer to
//     itself recurses without bound.
//
// For comparing []Node slices (e.g. results from Find), wrap each
// side as Array before calling Equal.
func Equal(n1, n2 Node) bool {
	if isNull(n1) || isNull(n2) {
		return isNull(n1) && isNull(n2)
	}

	t := n1.Type()
	if t != n2.Type() {
		return false
	}

	switch t {
	case TypeStringValue:
		return n1.Value().String() == n2.Value().String()
	case TypeNumberValue:
		return n1.Value().Float64() == n2.Value().Float64()
	case TypeBoolValue:
		return n1.Value().Bool() == n2.Value().Bool()
	case TypeMap:
		v1, v2 := n1.Map(), n2.Map()
		if len(v1) != len(v2) {
			return false
		}
		for k, c1 := range v1 {
			c2, exists := v2[k]
			if !exists {
				return false
			}
			if !Equal(c1, c2) {
				return false
			}
		}
		return true
	case TypeArray:
		v1, v2 := n1.Array(), n2.Array()
		if len(v1) != len(v2) {
			return false
		}
		for i := range v1 {
			if !Equal(v1[i], v2[i]) {
				return false
			}
		}
		return true
	}

	return false
}
