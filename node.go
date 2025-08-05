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
	Has(keys ...interface{}) bool
	// Get returns array/map value that matched by the specified key.
	// The key type allows int or string.
	// If the specified keys does not match, returns NilValue.
	Get(keys ...interface{}) Node
	// Each calls the callback function for each Array|Map values.
	// If the node type is not Array|Map then the callback called once with nil key and self as value.
	Each(cb func(key interface{}, v Node) error) error
	// Find finds a node using the query expression.
	Find(expr string) ([]Node, error)
}

// EditorNode is an interface that defines the methods to edit this node.
type EditorNode interface {
	Node
	Append(v Node) error
	Set(key interface{}, v Node) error
	Delete(key interface{}) error
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
func (n Any) Has(keys ...interface{}) bool {
	return n.Node.Has(keys...)
}

// Get returns an array value as Node.
func (n Any) Get(keys ...interface{}) Node {
	return n.Node.Get(keys...)
}

// Each calls the callback function for each Array values.
func (n Any) Each(cb func(key interface{}, n Node) error) error {
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
func (n Array) toIndex(key interface{}) (int, bool) {
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
func (n Array) Has(keys ...interface{}) bool {
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

// Get returns an array value as Node.
func (n Array) Get(keys ...interface{}) Node {
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
func (n Array) Each(cb func(key interface{}, n Node) error) error {
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
func (n *Array) Set(key interface{}, v Node) error {
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
func (n *Array) Delete(key interface{}) error {
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
func (n Map) toKey(key interface{}) (string, bool) {
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
func (n Map) Has(keys ...interface{}) bool {
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

// Get returns an array value as Node.
func (n Map) Get(keys ...interface{}) Node {
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
func (n Map) Set(key interface{}, v Node) error {
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
func (n Map) Delete(key interface{}) error {
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
func (n Map) Each(cb func(key interface{}, n Node) error) error {
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
