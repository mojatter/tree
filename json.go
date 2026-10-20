package tree

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

// MarshalJSON returns the JSON encoding of the specified node.
func MarshalJSON(n Node) ([]byte, error) {
	return json.Marshal(n)
}

// DecodeJSON decodes the next JSON value from dec; io.EOF means the input ended before a top-level value, io.ErrUnexpectedEOF inside one.
// Inside a container the caller opened with dec.Token, a cut between its elements is also io.EOF, so read the closing delimiter to detect it; tree caps nesting at 10,000 levels from the value, json/v2 from the stream root.
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
			if err := jsonMap(dec, &m, 1); err != nil {
				return nil, err
			}
			return m, nil
		case "[":
			return jsonArray(dec, Array{}, 1)
		}
	}
	return nil, fmt.Errorf("unknown token %#v", t)
}

// unexpectedEOF turns io.EOF inside a container into io.ErrUnexpectedEOF, as encoding/json does.
func unexpectedEOF(err error) error {
	if err == io.EOF {
		return io.ErrUnexpectedEOF
	}
	return err
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
			return jsonMap(dec, &m, 1)
		case "[":
			a, err := jsonArray(dec, Array{}, 1)
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
	return jsonMap(dec, n, 1)
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
	*n, err = jsonArray(dec, *n, 1)
	return err
}

// MarshalJSON is an implementation of json.Marshaler.
func (n NilValue) MarshalJSON() ([]byte, error) {
	return []byte("null"), nil
}

// jsonMap decodes the members of a JSON object into m, after its opening brace has been read.
func jsonMap(dec *json.Decoder, m *Map, depth int) error {
	for {
		t, err := dec.Token()
		if err != nil {
			return unexpectedEOF(err)
		}
		if d, ok := t.(json.Delim); ok {
			if d == '}' {
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
			return unexpectedEOF(err)
		}
		v, err := jsonElem(dec, t, depth+1)
		if err != nil {
			return err
		}
		(*m)[key] = v
	}
}

// jsonArray appends the elements of a JSON array to a, after its opening bracket has been read.
func jsonArray(dec *json.Decoder, a Array, depth int) (Array, error) {
	for {
		t, err := dec.Token()
		if err != nil {
			return nil, unexpectedEOF(err)
		}
		if d, ok := t.(json.Delim); ok && d == ']' {
			return a, nil
		}
		v, err := jsonElem(dec, t, depth+1)
		if err != nil {
			return nil, err
		}
		a = append(a, v)
	}
}

// maxJSONDepth bounds the recursion per decoded value; json.Decoder.Token enforces no cap before json/v2.
const maxJSONDepth = 10000

// errMaxDepth matches json/v2's depth error text, which Go 1.27 returns from Token first.
var errMaxDepth = errors.New("exceeded max depth")

// jsonElem decodes the value that starts with t at nesting depth, descending into an object or array.
func jsonElem(dec *json.Decoder, t json.Token, depth int) (Node, error) {
	d, ok := t.(json.Delim)
	if !ok {
		return jsonValue(t), nil
	}
	if d != '{' && d != '[' {
		return nil, fmt.Errorf("unknown token %#v", t)
	}
	if depth > maxJSONDepth {
		return nil, errMaxDepth
	}
	if d == '{' {
		m := Map{}
		if err := jsonMap(dec, &m, depth); err != nil {
			return nil, err
		}
		return m, nil
	}
	a, err := jsonArray(dec, Array{}, depth)
	if err != nil {
		return nil, err
	}
	return a, nil
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
func MarshalViaJSON(v any) (Node, error) {
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
func UnmarshalViaJSON(n Node, v any) error {
	data, err := MarshalJSON(n)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}
