package tree

import (
	"bytes"
	"maps"

	"go.yaml.in/yaml/v3"
)

// yamlIndent is the block sequence/mapping indent used by tree's
// YAML encoder. yaml.v3 defaults to 4 spaces; we pick 2 to stay closer
// to yaml.v2's output and to common convention.
const yamlIndent = 2

// MarshalYAML returns the YAML encoding of the specified node using
// tree's default 2-space indentation.
func MarshalYAML(n Node) ([]byte, error) {
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(yamlIndent)
	if err := enc.Encode(n); err != nil {
		_ = enc.Close()
		return nil, err
	}
	if err := enc.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// DecodeYAML decodes YAML as a node using the provided decoder, reading a timestamp as its source text and a null key as "null".
// Any other non-string scalar key is read as its string form (e.g. 1.0 as "1"), and two keys of one mapping with the same string form are a duplicate-key error.
func DecodeYAML(dec *yaml.Decoder) (Node, error) {
	var yn yaml.Node
	if err := dec.Decode(&yn); err != nil {
		return nil, err
	}
	return decodeYAMLNode(&yn)
}

// UnmarshalYAML decodes YAML data as a node, reading a timestamp as its source text and a null key as "null".
// Any other non-string scalar key is read as its string form (e.g. 1.0 as "1"), and two keys of one mapping with the same string form are a duplicate-key error.
func UnmarshalYAML(data []byte) (Node, error) {
	var yn yaml.Node
	if err := yaml.Unmarshal(data, &yn); err != nil {
		return nil, err
	}
	return decodeYAMLNode(&yn)
}

// decodeYAMLNode decodes yn as a node without changing it, reading a timestamp as its source text and a null key as "null".
// Any other non-string scalar key is read as its string form (e.g. 1.0 as "1"), and two keys of one mapping with the same string form are a duplicate-key error.
func decodeYAMLNode(yn *yaml.Node) (Node, error) {
	if yn.Kind == 0 {
		return Nil, nil // empty document
	}
	var v any
	if err := sourceTextYAML(yn, map[*yaml.Node]*yaml.Node{}).Decode(&v); err != nil {
		return nil, err
	}
	return ToNode(v), nil
}

// sourceTextYAML returns a copy of yn with timestamps tagged as strings and non-string or alias scalar keys replaced by their string text, sharing copies via seen.
// A scalar that does not decode as written keeps its tag, so the decode reports it in yaml.v3's order.
func sourceTextYAML(yn *yaml.Node, seen map[*yaml.Node]*yaml.Node) *yaml.Node {
	if yn == nil {
		return nil
	}
	if c, ok := seen[yn]; ok {
		return c
	}
	c := *yn
	seen[yn] = &c
	var v any
	if c.Kind == yaml.ScalarNode && c.ShortTag() == "!!timestamp" && yn.Decode(&v) == nil {
		c.Tag = "!!str"
	}
	c.Alias = sourceTextYAML(yn.Alias, seen)
	isMap := c.Kind == yaml.MappingNode
	c.Content = make([]*yaml.Node, len(yn.Content))
	for i, cn := range yn.Content {
		c.Content[i] = sourceTextYAML(cn, seen)
		if !isMap || i%2 != 0 {
			continue
		}
		k := cn
		if k.Kind == yaml.AliasNode && k.Alias != nil {
			k = k.Alias
		}
		if k.Kind != yaml.ScalarNode {
			continue
		}
		tag := k.ShortTag()
		if k == cn && (tag == "!!merge" || tag == "!!str" || tag == "!!timestamp") {
			continue // merge key untouched; a non-alias string or timestamp key already carries its text
		}
		var kv any
		if k.Decode(&kv) != nil {
			continue
		}
		// Retag as the text ToNode reads, so yaml.v3 also reports an alias key that collides; a timestamp alias keeps its source text.
		s := k.Value
		if tag != "!!timestamp" {
			s = mapKey(kv)
		}
		c.Content[i] = &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: s, Line: cn.Line, Column: cn.Column}
	}
	return &c
}

// UnmarshalYAML is an implementation of yaml.Unmarshaler (yaml.v3).
func (n *Map) UnmarshalYAML(value *yaml.Node) error {
	v, err := decodeYAMLNode(value)
	if err != nil {
		return err
	}
	if *n == nil {
		*n = make(Map)
	}
	maps.Copy(*n, v.Map())
	return nil
}

// UnmarshalYAML is an implementation of yaml.Unmarshaler (yaml.v3).
func (n *Array) UnmarshalYAML(value *yaml.Node) error {
	v, err := decodeYAMLNode(value)
	if err != nil {
		return err
	}
	_ = v.Array().Each(func(key any, v Node) error {
		*n = append(*n, v)
		return nil
	})
	return nil
}

// UnmarshalYAML implements yaml.Unmarshaler (yaml.v3), leaving n as it is on error or null (yaml.v3 skips it for null).
func (n *Any) UnmarshalYAML(value *yaml.Node) error {
	v, err := decodeYAMLNode(value)
	if err != nil {
		return err
	}
	n.Node = v
	return nil
}

// MarshalYAML implements yaml.Marshaler by encoding the wrapped node, or null when it is a Go nil node.
func (n Any) MarshalYAML() (any, error) {
	if isGoNil(n.Node) {
		return nil, nil
	}
	return n.Node, nil
}

// MarshalYAML is an implementation of yaml.Marshaler.
func (n NilValue) MarshalYAML() (any, error) {
	return nil, nil
}

// MarshalViaYAML returns the node encoding of v via "go.yaml.in/yaml/v3".
func MarshalViaYAML(v any) (Node, error) {
	if v == nil {
		return Nil, nil
	}
	if n, ok := v.(Node); ok {
		return n, nil
	}
	data, err := yaml.Marshal(v)
	if err != nil {
		return nil, err
	}
	return UnmarshalYAML(data)
}

// UnmarshalViaYAML stores the node in the value pointed to by v via "go.yaml.in/yaml/v3".
func UnmarshalViaYAML(n Node, v any) error {
	data, err := MarshalYAML(n)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(data, v)
}
