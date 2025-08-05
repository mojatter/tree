package tree

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// MarshalJSON returns the JSON encoding of the specified node.
func MarshalJSON(n Node) ([]byte, error) {
	return json.Marshal(n)
}

// DecodeJSON decodes JSON as a node using the provided decoder.
func DecodeJSON(dec *json.Decoder) (Node, error) {
	t, err := dec.Token()
	if err != nil {
		return nil, err
	}
	if t == nil {
		return Nil, nil
	}
	switch tt := t.(type) {
	case string:
		return StringValue(tt), nil
	case float64:
		return NumberValue(tt), nil
	case bool:
		return BoolValue(tt), nil
	case json.Delim:
		switch ds := tt.String(); ds {
		case "{":
			m := Map{}
			if err := jsonMap(dec, &m); err != nil {
				return nil, err
			}
			return m, nil
		case "[":
			return jsonArray(dec, Array{})
		}
	}
	return nil, fmt.Errorf("unknown token %#v", t)
}

// UnmarshalJSON parses the JSON-encoded data to a Node.
func UnmarshalJSON(data []byte) (Node, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	return DecodeJSON(dec)
}

// UnmarshalJSON is an implementation of json.Unmarshaler.
func (n *Any) UnmarshalJSON(data []byte) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	t, err := dec.Token()
	if err != nil {
		return err
	}
	d, ok := t.(json.Delim)
	if ok {
		switch d.String() {
		case "{":
			m := Map{}
			n.Node = m
			return jsonMap(dec, &m)
		case "[":
			a, err := jsonArray(dec, Array{})
			if err != nil {
				return err
			}
			n.Node = a
			return nil
		}
	}
	n.Node = jsonValue(t)
	return nil
}

// UnmarshalJSON is an implementation of json.Unmarshaler.
func (n *Map) UnmarshalJSON(data []byte) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	t, err := dec.Token()
	if err != nil {
		return err
	}
	d, ok := t.(json.Delim)
	if !ok || d.String() != "{" {
		return fmt.Errorf("unknown token %#v", t)
	}
	if *n == nil {
		*n = make(Map)
	}
	return jsonMap(dec, n)
}

// UnmarshalJSON is an implementation of json.Unmarshaler.
func (n *Array) UnmarshalJSON(data []byte) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	t, err := dec.Token()
	if err != nil {
		return err
	}
	d, ok := t.(json.Delim)
	if !ok || d.String() != "[" {
		return fmt.Errorf("unknown token %#v", t)
	}
	*n, err = jsonArray(dec, *n)
	return err
}

// MarshalJSON is an implementation of json.Marshaler.
func (n NilValue) MarshalJSON() ([]byte, error) {
	return []byte("null"), nil
}

// jsonMap recursively decodes a JSON object into a Map.
// Handles nested objects and arrays during JSON parsing.
func jsonMap(dec *json.Decoder, m *Map) error {
	t, err := dec.Token()
	if err != nil {
		return err
	}
	if d, ok := t.(json.Delim); ok {
		if d.String() == "}" {
			return nil
		}
		return fmt.Errorf("unknown token %#v", t)
	}

	key, ok := t.(string)
	if !ok {
		return fmt.Errorf("unknown token %#v", t)
	}

	t, err = dec.Token()
	if err != nil {
		return err
	}
	if d, ok := t.(json.Delim); ok {
		switch ds := d.String(); ds {
		case "{":
			mm := Map{}
			if err := jsonMap(dec, &mm); err != nil {
				return err
			}
			(*m)[key] = mm
			return jsonMap(dec, m)
		case "[":
			aa, err := jsonArray(dec, Array{})
			if err != nil {
				return err
			}
			(*m)[key] = aa
			return jsonMap(dec, m)
		}
	}

	(*m)[key] = jsonValue(t)
	return jsonMap(dec, m)
}

// jsonArray recursively decodes a JSON array.
// Handles nested objects and arrays during JSON parsing.
func jsonArray(dec *json.Decoder, a Array) (Array, error) {
	t, err := dec.Token()
	if err != nil {
		return nil, err
	}
	if d, ok := t.(json.Delim); ok {
		switch ds := d.String(); ds {
		case "]":
			return a, nil
		case "{":
			mm := Map{}
			if err := jsonMap(dec, &mm); err != nil {
				return nil, err
			}
			return jsonArray(dec, append(a, mm))
		case "[":
			aa, err := jsonArray(dec, Array{})
			if err != nil {
				return nil, err
			}
			return jsonArray(dec, append(a, aa))
		}
	}
	return jsonArray(dec, append(a, jsonValue(t)))
}

// jsonValue converts a JSON token to a Node value.
// Handles primitive types: string, bool, float64, and nil.
func jsonValue(t json.Token) Node {
	if t == nil {
		return Nil
	}
	switch tt := t.(type) {
	case string:
		return StringValue(tt)
	case bool:
		return BoolValue(tt)
	case float64:
		return NumberValue(tt)
	}
	return StringValue(fmt.Sprintf("%#v", t))
}

// MarshalViaJSON returns the node encoding of v via "encoding/json".
func MarshalViaJSON(v interface{}) (Node, error) {
	if v == nil {
		return Nil, nil
	}
	if n, ok := v.(Node); ok {
		return n, nil
	}
	data, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return UnmarshalJSON(data)
}

// UnmarshalViaJSON stores the node in the value pointed to by v via "encoding/json".
func UnmarshalViaJSON(n Node, v interface{}) error {
	data, err := MarshalJSON(n)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}
