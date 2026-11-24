package tree

import (
	"bytes"
	"fmt"
	"io"
	"regexp"
	"runtime"
	"strconv"
	"strings"
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
	IndentSize int
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

var yamlNoNeedQuotePattern = regexp.MustCompile(`^[a-zA-Z][0-9a-zA-Z._\-]*$`)

// NOTE: Simple implementation
func (e *ColorEncoder) writeQuotedYAMLIfNeed(s string, multiline bool) bool {
	if yamlNoNeedQuotePattern.MatchString(s) {
		e.writeStr(s)
		return false
	}
	if multiline && strings.Contains(s, "\n") {
		if strings.HasSuffix(s, "\n") {
			s = strings.TrimRight(s, "\n")
			e.writeln('|')
		} else {
			e.writeln('|', '-')
		}
		e.tab()
		for _, line := range strings.Split(s, "\n") {
			e.writeIndent(true)
			e.writeln([]byte(line)...)
		}
		e.untab()
		return true
	}
	e.writeStr(strconv.Quote(s))
	return false
}

// EncodeJSON writes JSON values with color to an output stream.
// A map key that is not valid UTF-8 or nesting past 10,000 levels is an error, before anything is written.
func (e *ColorEncoder) EncodeJSON(n Node) error {
	if err := checkJSONKeys(n, 1); err != nil {
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
	case TypeBoolValue, TypeNumberValue:
		e.writeIndent(indent)
		e.writeStr(n.Value().String())
	default:
		panic(fmt.Errorf("unknown type %b", t))
	}
}

// EncodeJSON writes JSON values with color to an output stream.
func (e *ColorEncoder) EncodeYAML(n Node) error {
	e.encodeYAML(n, true)
	return e.err
}

func (e *ColorEncoder) encodeYAML(n Node, noIndentFirstKey bool) {
	if isGoNil(n) {
		e.writeNull()
		e.writeln()
		return
	}
	t := n.Type()
	switch t {
	case TypeArray:
		for _, v := range n.Array() {
			e.writeIndent(true)
			if isGoNil(v) || v.Type().IsValue() {
				e.write('-', ' ')
				e.encodeYAML(v, false)
			} else if v.Type().IsMap() {
				e.write('-', ' ')
				e.tab()
				e.encodeYAML(v, true)
				e.untab()
			} else {
				e.writeln('-')
				e.tab()
				e.encodeYAML(v, false)
				e.untab()
			}
		}
	case TypeMap:
		i := 0
		m := n.Map()
		for _, k := range m.Keys() {
			v := m[k]
			e.writeIndent(i != 0 || !noIndentFirstKey)
			e.startColor(colorKey)
			e.writeQuotedYAMLIfNeed(k, false)
			e.endColor()
			if isGoNil(v) || v.Type().IsValue() {
				e.write(':', ' ')
				e.encodeYAML(v, false)
			} else {
				e.writeln(':')
				e.tab()
				e.encodeYAML(v, false)
				e.untab()
			}
			i++
		}
	case TypeNilValue:
		e.writeNull()
		e.writeln()
	case TypeStringValue:
		e.startColor(colorValueStr)
		ln := e.writeQuotedYAMLIfNeed(n.Value().String(), true)
		e.endColor()
		if !ln {
			e.writeln()
		}
	case TypeBoolValue, TypeNumberValue:
		e.writeStr(n.Value().String())
		e.writeln()
	default:
		panic(fmt.Errorf("unknown type %b", t))
	}
}

// OutputColorJSON writes JSON values with color to out.
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
