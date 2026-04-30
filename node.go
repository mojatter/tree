package tree

import (
	"fmt"
	"sort"
	"strconv"
)

// Type represents the Node type.
type Type int

// These variables are the Node types.
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

// IsValue returns true if t is TypeStringValue or TypeBoolValue or TypeNumberValue.
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

// String returns a short, lowercase label suitable for use in error
// messages: "array", "map", "nil", "string", "bool", or "number".
// The unspecialised "value" bitmask returns "value"; an unknown
// bit pattern returns "unknown".
func (t Type) String() string {
	switch t {
	case TypeArray:
		return "array"
	case TypeMap:
		return "map"
	case TypeNilValue:
		return "nil"
	case TypeStringValue:
		return "string"
	case TypeBoolValue:
		return "bool"
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
	// If this type is not Array, returns a Array(nil).
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

// Any is an interface that defines any node.
type Any struct {
	Node
}

var _ Node = (*Any)(nil)

// IsNil returns true if this node is nil.
func (n Any) IsNil() bool {
	return n.Node.IsNil()
}

// Type returns TypeArray.
func (n Any) Type() Type {
	return n.Node.Type()
}

// Array returns this node as an Array.
func (n Any) Array() Array {
	return n.Node.Array()
}

// Map returns nil.
func (n Any) Map() Map {
	return n.Node.Map()
}

// Value returns nil.
func (n Any) Value() Value {
	return n.Node.Value()
}

// Has checks this node has key.
func (n Any) Has(keys ...any) bool {
	return n.Node.Has(keys...)
}

// Get returns the child node that matches the specified keys.
func (n Any) Get(keys ...any) Node {
	return n.Node.Get(keys...)
}

// Each calls the callback function for each Array values.
func (n Any) Each(cb func(key any, n Node) error) error {
	return n.Node.Each(cb)
}

// Find finds a node using the query expression.
func (n Any) Find(expr string) ([]Node, error) {
	return n.Node.Find(expr)
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

// Map returns nil.
func (n Array) Map() Map {
	return nil
}

// Value returns nil.
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
				return n[i].Has(keys[1:]...)
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
				return n[i].Get(keys[1:]...)
			}
			if n[i] == nil {
				return Nil
			}
			return n[i]
		}
	}
	return Nil
}

// Each calls the callback function for each Array values.
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

// Set sets v to n[key].
func (n *Array) Set(key any, v Node) error {
	i, ok := n.toIndex(key)
	if i == -1 {
		return fmt.Errorf("cannot index array with %v", key)
	}
	if !ok {
		a := make([]Node, i+1)
		copy(a, *n)
		*n = a
	}
	(*n)[i] = v
	return nil
}

// Delete deletes n[key].
func (n *Array) Delete(key any) error {
	i, ok := n.toIndex(key)
	if i == -1 {
		return fmt.Errorf("cannot index array with %v", key)
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

// Array returns nil.
func (n Map) Array() Array {
	return nil
}

// Map returns this node as a Map.
func (n Map) Map() Map {
	return n
}

// Value returns nil.
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
				return n[k].Has(keys[1:]...)
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
				return n[k].Get(keys[1:]...)
			}
			if n[k] == nil {
				return Nil
			}
			return n[k]
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

// Values returns values of the map.
func (n Map) Values() []Node {
	values := make([]Node, len(n))
	for i, k := range n.Keys() {
		values[i] = n[k]
	}
	return values
}

// Append returns a error.
func (n Map) Append(v Node) error {
	return fmt.Errorf("cannot append to map")
}

// Set sets v to n[key].
func (n Map) Set(key any, v Node) error {
	switch tk := key.(type) {
	case int:
		n[strconv.Itoa(tk)] = v
		return nil
	case string:
		n[tk] = v
		return nil
	}
	return fmt.Errorf("cannot index array with %v", key)
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
	return fmt.Errorf("cannot index array with %v", key)
}

// Each calls the callback function for each Map values.
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
//   - Untyped nil and the Nil sentinel (NilValue{}) compare equal to
//     each other.
//   - Map equality is order-independent. Map{"a":1, "b":2} equals
//     Map{"b":2, "a":1}.
//   - Array equality is order-dependent. Array{1, 2} does not equal
//     Array{2, 1}.
//   - NumberValue equality follows IEEE 754. NumberValue(NaN) does
//     not equal NumberValue(NaN).
//   - Map(nil) equals Map{} (both empty Map); the same applies to
//     Array(nil) and Array{}. Neither equals Nil because the type
//     tags differ.
//   - Any wrappers are unwrapped before comparison.
//
// For comparing []Node slices (e.g. results from Find), wrap each
// side as Array before calling Equal.
func Equal(n1, n2 Node) bool {
	if a, ok := n1.(Any); ok {
		n1 = a.Node
	}
	if a, ok := n2.(Any); ok {
		n2 = a.Node
	}

	if n1 == nil {
		return n2 == nil || n2 == Nil
	}
	if n2 == nil {
		return n1 == Nil
	}

	if n1.Type() != n2.Type() {
		return false
	}

	switch v1 := n1.(type) {
	case NilValue:
		return true
	case StringValue:
		return v1 == n2.(StringValue)
	case NumberValue:
		return v1 == n2.(NumberValue)
	case BoolValue:
		return v1 == n2.(BoolValue)
	case Map:
		v2 := n2.(Map)
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
	case Array:
		v2 := n2.(Array)
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
