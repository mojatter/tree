package jsonschema

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	googlejsonschema "github.com/google/jsonschema-go/jsonschema"
	"github.com/mojatter/tree"
	"github.com/santhosh-tekuri/jsonschema/v5"
)

const groupSchema = `{
	"$schema": "https://json-schema.org/draft/2020-12/schema",
	"type": "object",
	"properties": {
		"ID":     { "type": "integer" },
		"Name":   { "type": "string" },
		"Colors": { "type": "array", "items": { "type": "string" } }
	},
	"required": ["ID", "Name", "Colors"]
}`

// santhosh-tekuri/jsonschema/v5 type-switches on map[string]any / []any /
// primitives internally, so tree nodes must be converted via tree.ToAny.
func Example_santhoshtekuriValid() {
	compiler := jsonschema.NewCompiler()
	const id = "https://example.com/schema.json"
	if err := compiler.AddResource(id, strings.NewReader(groupSchema)); err != nil {
		log.Fatal(err)
	}
	schema, err := compiler.Compile(id)
	if err != nil {
		log.Fatal(err)
	}

	group := tree.Map{
		"ID":     tree.V(1),
		"Name":   tree.V("Reds"),
		"Colors": tree.A("Crimson", "Red", "Ruby", "Maroon"),
	}
	// tree.ToAny copies the whole tree into plain Go values, which is
	// expensive; Example_googleValid shows a validator that accepts tree
	// nodes directly and skips this cost.
	if err := schema.Validate(tree.ToAny(group)); err != nil {
		log.Fatal(err)
	}
	fmt.Println("valid")

	// Output:
	// valid
}

func Example_santhoshtekuriInvalid() {
	compiler := jsonschema.NewCompiler()
	const id = "https://example.com/schema.json"
	if err := compiler.AddResource(id, strings.NewReader(groupSchema)); err != nil {
		log.Fatal(err)
	}
	schema, err := compiler.Compile(id)
	if err != nil {
		log.Fatal(err)
	}

	group := tree.Map{
		"ID":     tree.V("one"), // schema requires integer; this triggers the failure
		"Name":   tree.V("Reds"),
		"Colors": tree.A("Crimson"),
	}
	// tree.ToAny copies the whole tree into plain Go values, which is
	// expensive; Example_googleValid shows a validator that accepts tree
	// nodes directly and skips this cost.
	fmt.Println(schema.Validate(tree.ToAny(group)))

	// Output:
	// jsonschema: '/ID' does not validate with https://example.com/schema.json#/properties/ID/type: expected integer, but got string
}

// google/jsonschema-go walks values via reflection, so tree.Map / tree.Array /
// tree.StringValue / tree.NumberValue / tree.BoolValue are accepted directly.
// Note that the tree.Any wrapper (a struct embedding Node) is not accepted;
// unmarshal into tree.Map / tree.Array directly when using this validator.
func Example_googleValid() {
	var s googlejsonschema.Schema
	if err := json.Unmarshal([]byte(groupSchema), &s); err != nil {
		log.Fatal(err)
	}
	resolved, err := s.Resolve(nil)
	if err != nil {
		log.Fatal(err)
	}

	group := tree.Map{
		"ID":     tree.V(1),
		"Name":   tree.V("Reds"),
		"Colors": tree.A("Crimson", "Red", "Ruby", "Maroon"),
	}
	if err := resolved.Validate(group); err != nil {
		log.Fatal(err)
	}
	fmt.Println("valid")

	// Output:
	// valid
}

func Example_googleInvalid() {
	var s googlejsonschema.Schema
	if err := json.Unmarshal([]byte(groupSchema), &s); err != nil {
		log.Fatal(err)
	}
	resolved, err := s.Resolve(nil)
	if err != nil {
		log.Fatal(err)
	}

	group := tree.Map{
		"ID":     tree.V("one"), // schema requires integer; this triggers the failure
		"Name":   tree.V("Reds"),
		"Colors": tree.A("Crimson"),
	}
	fmt.Println(resolved.Validate(group))

	// Output:
	// validating root: validating /properties/ID: type: one has type "string", want "integer"
}
