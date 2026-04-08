package tree

import (
	"gopkg.in/yaml.v2"
)

// MarshalYAML returns the YAML encoding of the specified node.
func MarshalYAML(n Node) ([]byte, error) {
	return yaml.Marshal(n)
}

// DecodeYAML decodes YAML as a node using the provided decoder.
func DecodeYAML(dec *yaml.Decoder) (Node, error) {
	var v interface{}
	if err := dec.Decode(&v); err != nil {
		return nil, err
	}
	return ToNode(v), nil
}

// UnmarshalYAML returns the YAML encoding of the specified node.
func UnmarshalYAML(data []byte) (Node, error) {
	var v interface{}
	if err := yaml.Unmarshal(data, &v); err != nil {
		return nil, err
	}
	return ToNode(v), nil
}

// UnmarshalYAML is an implementation of yaml.Unmarshaler.
func (n *Map) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var v interface{}
	if err := unmarshal(&v); err != nil {
		return err
	}
	if *n == nil {
		*n = make(Map)
	}
	for k, v := range ToNode(v).Map() {
		(*n)[k] = v
	}
	return nil
}

// UnmarshalYAML is an implementation of yaml.Unmarshaler.
func (n *Array) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var v interface{}
	if err := unmarshal(&v); err != nil {
		return err
	}
	_ = ToNode(v).Array().Each(func(key interface{}, v Node) error {
		*n = append(*n, v)
		return nil
	})
	return nil
}

// MarshalYAML is an implementation of yaml.Marshaler.
func (n NilValue) MarshalYAML() (interface{}, error) {
	return nil, nil
}

// MarshalViaYAML returns the node encoding of v via "gopkg.in/yaml.v2".
func MarshalViaYAML(v interface{}) (Node, error) {
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

// UnmarshalViaYAML stores the node in the value pointed to by v via "gopkg.in/yaml.v2".
func UnmarshalViaYAML(n Node, v interface{}) error {
	data, err := MarshalYAML(n)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(data, v)
}
