// Package schema validates [github.com/mojatter/tree.Node] structures
// against a set of rules keyed by tree queries. Each (query, [Rule])
// entry applies the rule to every node matched by the query.
//
// The package is not JSON Schema compatible; individual primitives
// ([Require], [String], and so on) are validation rules rather than
// full schema documents. The collective form [QueryRules] behaves
// schema-like as a whole.
//
// # Programmatic usage
//
// Rules are ordinary Go values and can be composed directly. A flat
// document validates with a flat set of rules:
//
//	rules := schema.QueryRules{
//	    ".name":    schema.Required(schema.String{}),
//	    ".age":     schema.Int{Min: schema.Int64Ptr(0), Max: schema.Int64Ptr(150)},
//	    ".tags[]":  schema.String{},
//	    ".":        schema.Map{Keys: []string{"name", "age", "tags"}},
//	}
//	err := schema.Validate(doc, rules)
//
// Deeper trees mix dot-paths with [Every] to apply rules to each
// element of an array or each value of a map. Take tree's standard
// "Illustrative Object" fixture (a bookstore document shaped like
// {store: {book: [{isbn, price, tags: [{name, value}, ...]}, ...],
// bicycle: {color, price}}}):
//
//	rules := schema.QueryRules{
//	    ".store.bicycle.color": schema.Required(schema.String{Enum: []string{"red", "blue", "green"}}),
//	    ".store.book": schema.Every{Rules: schema.QueryRules{
//	        ".isbn":  schema.Required(schema.String{}),
//	        ".price": schema.Required(schema.Float{Min: schema.Float64Ptr(0)}),
//	        ".tags":  schema.Every{Rules: schema.QueryRules{
//	            ".value": schema.Required(schema.String{}),
//	        }},
//	    }},
//	}
//
// Inside [Every], each query is resolved relative to the element,
// so a book missing ".isbn" surfaces as ".store.book[0].isbn:
// required" rather than being silently absent. Every can be nested
// arbitrarily deep — here ".tags" is another Every inside the outer
// one, so per-tag ".value" is checked with the full element path.
//
// Array and map iteration use the tree query "[]" suffix. The error
// path for each element carries the concrete index ("[0]", "[1]", ...)
// for arrays and the concrete key (`["admin"]`, `["guest"]`, ...) for
// maps. When the base of "[]" resolves to more than one parent,
// numeric indices are used as a fallback.
//
// Only a literal "[]" suffix triggers this per-element expansion.
// Filter queries like ".arr[.age > 18]" still work — tree.Find returns
// the matching nodes and the rule runs on each — but the error path
// stays verbatim as the whole filter expression, so multiple failing
// matches share one display path and cannot be told apart. A filter
// that matches zero nodes also makes [Required] fire once on that
// filter, which is rarely the intent. Use "[]" (or wrap the parent
// array/map in [Every]) when element-level error paths or
// required-per-element semantics are needed.
//
// # Nil semantics
//
// When a query matches no node, the rule is invoked with
// [github.com/mojatter/tree.Nil]. Leaf rules ([String], [Int], [Float],
// [Bool], [Array], [Map]) and composites ([And], [Or], [Not], [Every])
// return nil on Nil, so a bare rule means "if present, satisfy this
// constraint". Only [Require] fires on Nil. Wrap a rule with
// [Required] (or prepend [Require] via [And]) to make the presence
// mandatory.
//
// # Rule types
//
//   - [Require]: fires when the node is Nil.
//   - [And]: passes when every child rule passes; collects all errors.
//   - [Or]: passes when at least one child rule passes.
//   - [Not]: passes when the inner rule fails; skips Nil.
//   - [Every]: applies a QueryRules set to each element of an array
//     or each value of a map. Required rules inside Every fire
//     per element, unlike a bare "[]"-suffixed query which flattens
//     matches and cannot detect a missing sub-field on a specific
//     element.
//   - [String]: a string leaf with optional Enum, Regex, and length bounds.
//   - [Int]: an integer-valued number with optional bounds; rejects NaN,
//     Inf, fractional values, and values outside int64 range.
//   - [Float]: a number leaf with optional float bounds; rejects NaN.
//   - [Bool]: a boolean leaf.
//   - [Array]: an array with optional length bounds. Element-level
//     rules are expressed as separate entries on a "[]"-suffixed query.
//   - [Map]: a map with optional Keys allow-list.
//
// # YAML / JSON format
//
// [ParseQueryRules] decodes a tree.Node (typically produced by yaml
// or json decoders) into [QueryRules]. The document is a map where
// each key is a tree query and each value is a rule spec.
//
// A rule spec is a map carrying at least a "type" field; the factory
// registered for that type consumes the rest. The meta-flag
// "required: true" wraps the resulting rule with [Required].
//
// The built-in registry ([BuiltinRegistry]) recognises these types:
//
//	type      | fields
//	---------------------------------------------------------------
//	require   | (none)
//	string    | enum, regex, minLen, maxLen
//	int       | min, max
//	float     | min, max
//	bool      | (none)
//	array     | minLen, maxLen
//	map       | keys
//	and       | of: [spec...]
//	or        | of: [spec...]
//	not       | rule: spec
//	every     | rules: {query: spec, ...}
//
// Plus the universal meta fields "type" and "required".
//
// Example YAML (decoded into a tree.Map and passed to
// [ParseQueryRules]). The shape mirrors the nested programmatic
// example above:
//
//	".store.bicycle.color":
//	  type: string
//	  required: true
//	  enum: [red, blue, green]
//	".store.book":
//	  type: every
//	  rules:
//	    ".isbn":  { type: string, required: true }
//	    ".price": { type: float, required: true, min: 0 }
//	    ".tags":
//	      type: every
//	      rules:
//	        ".value": { type: string, required: true }
//
// Unknown fields on a rule spec are rejected. "required" must be a
// bool. Field types are enforced strictly (for example, "min: \"0\""
// is an error).
//
// # Custom rule types
//
// User code can implement [Rule] directly and insert it into a
// [QueryRules] map. To expose a custom rule to the YAML / JSON
// loader, register a [Factory] with a [Registry]:
//
//	reg := schema.BuiltinRegistry()
//	reg.Register("uuid", parseUUIDSpec, "version")
//	rules, err := schema.ParseQueryRulesWith(specNode, reg)
//
// The factory's fields argument declares which type-specific keys the
// spec may carry (beyond "type" and "required"). Any other key on the
// spec is reported as an unknown-field error.
//
// # Error aggregation
//
// [Validate], [ParseQueryRules], and [Registry.Parse] collect errors
// via [errors.Join] so every violation is surfaced in one call.
// [Registry.MaxReportedErrors] caps the number of parsed-spec errors
// (default 20); overflow is summarised as "and N more errors".
// [Validate] currently has no cap.
//
// One exception: when a rule's query string itself is malformed
// (tree.Find rejects it), [Validate] and [ValidateWithPrefix] fail
// fast and return a single [*ErrQuery] without joining any
// data-level violations. A malformed query is a bug in the rules
// definition rather than in the data, so emitting it alongside
// unrelated violations would be noise. Callers that need to
// distinguish the two can errors.As into [*ErrQuery].
package schema
