package schema_test

import (
	"fmt"

	"github.com/mojatter/tree"
	"github.com/mojatter/tree/internal/testdata"
	"github.com/mojatter/tree/schema"
)

// Validate walks the document once per rule and aggregates every
// violation via errors.Join. The example shows a document that breaks
// four rules; all of them surface in a single error value.
func ExampleValidate() {
	doc := tree.Map{
		"name": tree.V("alice"),
		"age":  tree.V(-1),
		"tags": tree.A("ok", 1),
		"role": tree.V("guest"),
	}
	rules := schema.QueryRules{
		".name":   schema.Required(schema.String{}),
		".age":    schema.Int{Min: schema.Int64Ptr(0)},
		".tags[]": schema.String{},
		".role":   schema.String{Enum: []string{"admin", "user"}},
	}
	if err := schema.Validate(doc, rules); err != nil {
		fmt.Println(err)
	}
	// Output:
	// .age: value -1 less than min 0
	// .role: value "guest" not in enum [admin user]
	// .tags[1]: expected string, got number
}

// ExampleValidate_composite shows less-common patterns: Or for a
// polymorphic field, Not wrapping a String.Enum as a blocklist,
// per-field rules inside a nested map (`.labels.env` /
// `.labels.tier`), a Map rule at the root asserting the allowed
// key set, and Every applying per-element rules so Required fires
// on elements that are missing a sub-field.
func ExampleValidate_composite() {
	doc := tree.Map{
		"password": tree.V("12345"),
		"allow":    tree.V(42),
		"labels": tree.Map{
			"env":  tree.V("prod"),
			"tier": tree.V("one"), // should be number
		},
		"tags": tree.A(
			tree.Map{"name": tree.V("prod"), "count": tree.V(1)},
			tree.Map{"count": tree.V(2)}, // missing "name"
		),
	}
	rules := schema.QueryRules{
		".": schema.Map{Keys: []string{"password", "allow", "labels", "tags"}},
		".password": schema.And{
			schema.Required(schema.String{MinLen: schema.IntPtr(8)}),
			schema.Not{Rule: schema.String{Enum: []string{"password", "12345678"}}},
		},
		".allow":       schema.Or{schema.String{}, schema.Bool{}},
		".labels.env":  schema.String{},
		".labels.tier": schema.Int{},
		".tags": schema.Every{Rules: schema.QueryRules{
			".name": schema.Required(schema.String{}),
		}},
	}
	if err := schema.Validate(doc, rules); err != nil {
		fmt.Println(err)
	}
	// Output:
	// .allow: expected string, got number
	// .allow: expected bool, got number
	// .labels.tier: expected number, got string
	// .password: length 5 less than min 8
	// .tags[1].name: required
}

// ExampleValidate_nested validates tree's "Illustrative Object"
// fixture (a bookstore document with nested arrays and maps),
// showing Every wrapped inside Every so Required fires per element
// at arbitrary depth. The rules require every book to carry an
// isbn and every tag to carry a value; books that are missing
// isbn surface with the concrete index in the path.
func ExampleValidate_nested() {
	n, err := tree.UnmarshalJSON([]byte(testdata.StoreJSON))
	if err != nil {
		fmt.Println("unmarshal:", err)
		return
	}
	rules := schema.QueryRules{
		".store.bicycle.color": schema.Required(schema.String{Enum: []string{"red", "blue", "green"}}),
		".store.book": schema.Every{Rules: schema.QueryRules{
			".isbn":  schema.Required(schema.String{}),
			".price": schema.Required(schema.Float{Min: schema.Float64Ptr(0)}),
			".tags": schema.Every{Rules: schema.QueryRules{
				".value": schema.Required(schema.String{}),
			}},
		}},
	}
	if err := schema.Validate(n, rules); err != nil {
		fmt.Println(err)
	}
	// Output:
	// .store.book[0].isbn: required
	// .store.book[1].isbn: required
}

// ParseQueryRules decodes a tree.Node (here built directly, but
// normally produced by a yaml or json decoder) into QueryRules.
// Each entry's spec has a mandatory "type" field; "required: true"
// wraps the rule with Required.
func ExampleParseQueryRules() {
	spec := tree.Map{
		".name": tree.Map{"type": tree.V("string"), "required": tree.V(true)},
		".age":  tree.Map{"type": tree.V("int"), "min": tree.V(0), "max": tree.V(150)},
		".role": tree.Map{
			"type": tree.V("or"),
			"of": tree.A(
				tree.Map{"type": tree.V("string"), "enum": tree.A("admin", "user")},
				tree.Map{"type": tree.V("bool")},
			),
		},
	}
	rules, err := schema.ParseQueryRules(spec)
	if err != nil {
		fmt.Println("parse:", err)
		return
	}

	doc := tree.Map{
		"age":  tree.V(200),
		"role": tree.V("guest"),
	}
	if err := schema.Validate(doc, rules); err != nil {
		fmt.Println(err)
	}
	// Output:
	// .age: value 200 greater than max 150
	// .name: required
	// .role: value "guest" not in enum [admin user]
	// .role: expected bool, got string
}
