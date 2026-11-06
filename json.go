package tree

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
)

// MarshalJSON returns the JSON encoding of the specified node.
func MarshalJSON(n Node) ([]byte, error) {
	return json.Marshal(n)
}

// DecodeJSON decodes the next JSON value from dec; io.EOF means the input ended before a top-level value, io.ErrUnexpectedEOF inside one.
// Inside a container the caller opened with dec.Token, a cut between its elements is also io.EOF, so read the closing delimiter to detect it; tree caps nesting at 10,000 levels from the value, json/v2 from the stream root.
// With dec.UseNumber, numbers still become float64 NumberValues, so integers beyond 2^53 lose precision.
func DecodeJSON(dec *json.Decoder) (Node, error) {
	t, err := dec.Token()
	if err != nil {
		return nil, err
	}
	return jsonElem(dec, t, 1)
}

// unexpectedEOF turns io.EOF inside a container into io.ErrUnexpectedEOF, as encoding/json does.
func unexpectedEOF(err error) error {
	if err == io.EOF {
		return io.ErrUnexpectedEOF
	}
	return err
}

// UnmarshalJSON parses the JSON-encoded data to a Node, rejecting trailing data as json.Unmarshal does.
func UnmarshalJSON(data []byte) (Node, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	n, err := DecodeJSON(dec)
	if err != nil {
		return nil, err
	}
	if err := jsonEnd(dec, data); err != nil {
		return nil, err
	}
	return n, nil
}

// jsonEnd rejects anything but whitespace after the top-level value, with json.Unmarshal's message.
func jsonEnd(dec *json.Decoder, data []byte) error {
	rest := bytes.TrimLeft(data[dec.InputOffset():], " \t\r\n")
	if len(rest) == 0 {
		return nil
	}
	return fmt.Errorf("invalid character %q after top-level value", rune(rest[0]))
}

// UnmarshalJSON implements json.Unmarshaler, leaving n as it is on error.
func (n *Any) UnmarshalJSON(data []byte) error {
	v, err := UnmarshalJSON(data)
	if err != nil {
		return err
	}
	n.Node = v
	return nil
}

// UnmarshalJSON implements json.Unmarshaler: it sets each key of the object in n, keeping n's other keys, and leaves n as it is on null.
// Like encoding/json, it decodes in place, so on error n may hold some of the new keys.
func (n *Map) UnmarshalJSON(data []byte) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	t, err := dec.Token()
	if err != nil {
		return err
	}
	if t == nil {
		return jsonEnd(dec, data)
	}
	d, ok := t.(json.Delim)
	if !ok || d != '{' {
		return fmt.Errorf("unknown token %#v", t)
	}
	m := *n
	if m == nil {
		m = make(Map) // a nil n stays nil on error
	}
	if err := jsonMap(dec, &m, 1); err != nil {
		return err
	}
	if err := jsonEnd(dec, data); err != nil {
		return err
	}
	*n = m
	return nil
}

// UnmarshalJSON implements json.Unmarshaler: it resets n to length zero and refills it, leaving n as it is on null.
// Like encoding/json, it reuses n's backing array, which other nodes may share, so on error that array may hold some of the new elements while n keeps its length.
func (n *Array) UnmarshalJSON(data []byte) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	t, err := dec.Token()
	if err != nil {
		return err
	}
	if t == nil {
		return jsonEnd(dec, data)
	}
	d, ok := t.(json.Delim)
	if !ok || d != '[' {
		return fmt.Errorf("unknown token %#v", t)
	}
	base := (*n)[:0]
	if base == nil {
		base = Array{} // [] gives an empty Array, not a nil one
	}
	a, err := jsonArray(dec, base, 1)
	if err != nil {
		return err
	}
	if err := jsonEnd(dec, data); err != nil {
		return err
	}
	*n = a
	return nil
}

// MarshalJSON implements json.Marshaler by encoding the wrapped node, or null when it is a Go nil node.
// It leaves HTML escaping to the caller's encoder and fails past 10,000 levels of nesting, so a cycle errors
// when its nodes are Maps, Arrays or Any wrapping them, and overflows the stack otherwise.
func (n Any) MarshalJSON() ([]byte, error) {
	return marshalJSONNode(n)
}

// MarshalJSON implements json.Marshaler like Any's, encoding a nil Map as {} as YAML does.
// A struct embedding Map promotes it, so the struct marshals as the Map alone, as it already unmarshals.
func (n Map) MarshalJSON() ([]byte, error) {
	return marshalJSONNode(n)
}

// MarshalJSON implements json.Marshaler like Any's, encoding a nil Array as [] as YAML does.
// A struct embedding Array marshals as the Array alone, as with Map.
func (n Array) MarshalJSON() ([]byte, error) {
	return marshalJSONNode(n)
}

// marshalJSONNode encodes n into one buffer, so nested Maps, Arrays and their wrappers are not re-encoded at each level.
func marshalJSONNode(n Node) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := encodeJSONNode(&buf, enc, n, 1); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// encodeJSONNode writes a Go nil node as null, Maps and Arrays at nesting depth itself, through Any and pointers, and anything else with enc.
func encodeJSONNode(buf *bytes.Buffer, enc *json.Encoder, n Node, depth int) error {
	if isGoNil(n) {
		buf.WriteString("null")
		return nil
	}
	switch tn := n.(type) {
	case Any:
		return encodeJSONNode(buf, enc, tn.Node, depth)
	case *Any:
		return encodeJSONNode(buf, enc, tn.Node, depth)
	case *Map:
		return encodeJSONNode(buf, enc, *tn, depth)
	case *Array:
		return encodeJSONNode(buf, enc, *tn, depth)
	case Map:
		if depth > maxJSONDepth {
			return errMaxDepth
		}
		buf.WriteByte('{')
		for i, k := range tn.Keys() {
			if i > 0 {
				buf.WriteByte(',')
			}
			if err := encodeJSONValue(buf, enc, k); err != nil {
				return err
			}
			buf.WriteByte(':')
			if err := encodeJSONNode(buf, enc, tn[k], depth+1); err != nil {
				return err
			}
		}
		buf.WriteByte('}')
		return nil
	case Array:
		if depth > maxJSONDepth {
			return errMaxDepth
		}
		buf.WriteByte('[')
		for i, v := range tn {
			if i > 0 {
				buf.WriteByte(',')
			}
			if err := encodeJSONNode(buf, enc, v, depth+1); err != nil {
				return err
			}
		}
		buf.WriteByte(']')
		return nil
	}
	return encodeJSONValue(buf, enc, n)
}

// encodeJSONValue writes v with enc, which writes to buf, dropping the newline Encode appends.
func encodeJSONValue(buf *bytes.Buffer, enc *json.Encoder, v any) error {
	if err := enc.Encode(v); err != nil {
		return err
	}
	buf.Truncate(buf.Len() - 1)
	return nil
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

// maxJSONDepth bounds the recursion per decoded or encoded value; json.Decoder.Token enforces no cap before json/v2.
const maxJSONDepth = 10000

// errMaxDepth matches json/v2's depth error text, which Go 1.27 returns from Token first.
var errMaxDepth = errors.New("exceeded max depth")

// jsonElem decodes the value that starts with t at nesting depth, descending into an object or array.
func jsonElem(dec *json.Decoder, t json.Token, depth int) (Node, error) {
	d, ok := t.(json.Delim)
	if !ok {
		return jsonValue(dec, t)
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

// jsonValue converts a scalar JSON token, including a json.Number from UseNumber, to a Node value.
func jsonValue(dec *json.Decoder, t json.Token) (Node, error) {
	switch tt := t.(type) {
	case nil:
		return Nil, nil
	case string:
		return StringValue(tt), nil
	case bool:
		return BoolValue(tt), nil
	case float64:
		return NumberValue(tt), nil
	case json.Number:
		f, err := tt.Float64()
		if err != nil {
			return nil, &json.UnmarshalTypeError{Value: "number " + tt.String(), Type: reflect.TypeFor[float64](), Offset: dec.InputOffset()} // as without UseNumber
		}
		return NumberValue(f), nil
	}
	return nil, fmt.Errorf("unknown token %#v", t)
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
