package tree

import (
	"bytes"
	"fmt"
	"io"
	"runtime"
	"strings"
	"unicode/utf8"

	"go.yaml.in/yaml/v3"
)

type color string

var (
	colorKey       color = "\033[1;34m"
	colorValueStr  color = "\033[0;32m"
	colorValueNull color = "\033[0;90m"
	colorReset           = "\033[0m"
)

func init() {
	if runtime.GOOS == "windows" {
		colorKey = ""
		colorValueStr = ""
		colorValueNull = ""
		colorReset = ""
	}
}

// ColorEncoder writes JSON or YAML values with color to an output stream.
type ColorEncoder struct {
	Out        io.Writer
	IndentSize int // spaces per level in JSON; EncodeYAML always indents by 2
	NoColor    bool
	indent     []byte
	buf        []byte
	err        error
}

func (e *ColorEncoder) tab() {
	e.indent = append(e.indent, bytes.Repeat([]byte{' '}, e.IndentSize)...)
}

func (e *ColorEncoder) untab() {
	e.indent = e.indent[0 : len(e.indent)-e.IndentSize]
}

func (e *ColorEncoder) write(bs ...byte) {
	if e.err != nil {
		return
	}
	_, e.err = e.Out.Write(bs)
}

func (e *ColorEncoder) writeStr(s string) {
	e.write([]byte(s)...)
}

func (e *ColorEncoder) startColor(c color) {
	if !e.NoColor {
		e.writeStr(string(c))
	}
}

func (e *ColorEncoder) endColor() {
	if !e.NoColor {
		e.writeStr(colorReset)
	}
}

func (e *ColorEncoder) writeIndent(indent bool) {
	if indent {
		e.write(e.indent...)
	}
}

func (e *ColorEncoder) writeln(bs ...byte) {
	e.write(bs...)
	e.write(0x0a)
}

func (e *ColorEncoder) writeCn(comma bool) {
	if comma {
		e.writeln(',')
	} else {
		e.writeln()
	}
}

func (e *ColorEncoder) writeNull() {
	e.startColor(colorValueNull)
	e.write('n', 'u', 'l', 'l')
	e.endColor()
}

func (e *ColorEncoder) writeQuotedJSON(s string) {
	e.buf = appendQuotedJSON(e.buf[:0], s)
	e.write(e.buf...)
}

// EncodeJSON writes JSON values with color to an output stream.
// It fails as MarshalJSON does, on a map key that is not valid UTF-8, a NaN or infinite number or nesting past 10,000 levels, before anything is written.
func (e *ColorEncoder) EncodeJSON(n Node) error {
	if err := checkJSONNode(n, 1); err != nil {
		return err
	}
	e.encodeJSON(n, false)
	e.writeln()
	return e.err
}

func (e *ColorEncoder) encodeJSON(n Node, indent bool) {
	if isGoNil(n) {
		e.writeIndent(indent)
		e.writeNull()
		return
	}
	t := n.Type()
	switch t {
	case TypeArray:
		e.writeIndent(indent)
		e.writeln('[')
		e.tab()
		a := n.Array()
		last := len(a) - 1
		for i, v := range a {
			e.encodeJSON(v, true)
			e.writeCn(i != last)
		}
		e.untab()
		e.writeIndent(true)
		e.write(']')
	case TypeMap:
		e.writeIndent(indent)
		e.writeln('{')
		e.tab()
		m := n.Map()
		i, last := 0, len(m)-1
		for _, k := range m.Keys() {
			e.writeIndent(true)
			e.startColor(colorKey)
			e.writeQuotedJSON(k)
			e.endColor()
			e.write(':', ' ')
			e.encodeJSON(m[k], false)
			e.writeCn(i != last)
			i++
		}
		e.untab()
		e.writeIndent(true)
		e.write('}')
	case TypeNilValue:
		e.writeIndent(indent)
		e.writeNull()
	case TypeStringValue:
		e.writeIndent(indent)
		e.startColor(colorValueStr)
		e.writeQuotedJSON(n.Value().String())
		e.endColor()
	case TypeBoolValue:
		e.writeIndent(indent)
		e.writeStr(n.Value().String())
	case TypeNumberValue:
		e.writeIndent(indent)
		e.buf, _ = appendJSONNumber(e.buf[:0], NumberValue(n.Value().Float64())) // EncodeJSON has rejected NaN and infinities
		e.write(e.buf...)
	default:
		panic(fmt.Errorf("unknown type %b", t))
	}
}

// EncodeYAML writes n as MarshalYAML does, with keys, strings and nulls colored; IndentSize applies to JSON only.
func (e *ColorEncoder) EncodeYAML(n Node) error {
	data, err := MarshalYAML(n)
	if err != nil {
		return err
	}
	if !e.NoColor && colorReset != "" {
		data = colorYAML(data)
	}
	e.write(data...)
	return e.err
}

// yamlBreaks are the line breaks yaml.v3 writes and counts in a node position.
const yamlBreaks = "\n\u0085\u2028\u2029"

// colorYAML returns data, which yaml.v3 wrote, with keys, strings and nulls colored, or data as it is if yaml.v3 cannot read it back or it has a flow collection.
func colorYAML(data []byte) []byte {
	var yn yaml.Node
	if yaml.Unmarshal(data, &yn) != nil {
		return data
	}
	lines := []int{0}
	for i := 0; i < len(data); {
		r, size := utf8.DecodeRune(data[i:])
		i += size
		if strings.ContainsRune(yamlBreaks, r) {
			lines = append(lines, i)
		}
	}
	type mark struct {
		pos   int
		color color
	}
	var marks []mark // every node in document order; color is empty for one written as it is
	flow := false    // a non-empty flow collection, whose scalars do not end at a line break
	var walk func(yn *yaml.Node, key bool)
	walk = func(yn *yaml.Node, key bool) {
		flow = flow || yn.Style&yaml.FlowStyle != 0 && len(yn.Content) > 0
		if yn.Kind != yaml.DocumentNode {
			m := mark{pos: lines[yn.Line-1]}
			for c := 1; c < yn.Column; c++ {
				_, size := utf8.DecodeRune(data[m.pos:])
				m.pos += size
			}
			switch {
			case yn.Kind != yaml.ScalarNode:
			case key:
				m.color = colorKey
			case yn.Tag == "!!str" || yn.Tag == "!!binary":
				m.color = colorValueStr
			case yn.Tag == "!!null":
				m.color = colorValueNull
			}
			marks = append(marks, m)
		}
		for i, c := range yn.Content {
			walk(c, yn.Kind == yaml.MappingNode && i%2 == 0)
		}
	}
	walk(&yn, false)
	if flow {
		return data
	}
	out := make([]byte, 0, len(data)+len(marks)*(len(colorKey)+len(colorReset)))
	pos := 0
	for i, m := range marks {
		if m.color == "" {
			continue
		}
		next := len(data)
		if i+1 < len(marks) {
			next = marks[i+1].pos
		}
		s := data[m.pos:next]
		if m.color == colorKey {
			// Up to the value, less the ": " or "\n: " before it and the "? " of an explicit key that starts it.
			s = bytes.TrimRight(bytes.TrimSuffix(bytes.TrimRight(s, " ?"+yamlBreaks), []byte{':'}), " "+yamlBreaks)
		} else if i := bytes.LastIndexAny(s, yamlBreaks); i >= 0 {
			s = s[:i] // up to the line break that ends the scalar
		}
		out = append(out, data[pos:m.pos]...)
		out = append(out, m.color...)
		out = append(out, s...)
		out = append(out, colorReset...)
		pos = m.pos + len(s)
	}
	return append(out, data[pos:]...)
}

// OutputColorJSON writes JSON values with color to out, failing as EncodeJSON does.
func OutputColorJSON(out io.Writer, n Node) error {
	e := &ColorEncoder{
		Out:        out,
		IndentSize: 2,
	}
	return e.EncodeJSON(n)
}

// OutputColorYAML writes YAML values with color to out.
func OutputColorYAML(out io.Writer, n Node) error {
	e := &ColorEncoder{
		Out:        out,
		IndentSize: 2,
	}
	return e.EncodeYAML(n)
}
