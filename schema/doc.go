// Package schema validates [github.com/mojatter/tree.Node] structures
// against a set of rules keyed by tree queries. Each (query, [Rule])
// entry applies the rule to every node matched by the query; [Validate]
// joins all violations via errors.Join so one call surfaces every
// failure.
//
// This comment is an API overview. Extended scenarios — the nested
// bookstore example, the filter-query caveat, the full YAML spec
// reference, dispatch-style custom rules using [ValidateWithPrefix],
// and [*ErrQuery] handling — live in docs/schema.md on GitHub:
// https://github.com/mojatter/tree/blob/main/docs/schema.md
//
// # Mental model
//
//   - [QueryRules] is a map from a tree query to the [Rule] applied
//     at every match.
//   - A missing node is passed to the rule as
//     [github.com/mojatter/tree.Nil]; only [Require] fires on Nil, so
//     [Required] (= And{Require{}, r}) is the idiomatic "must be
//     present and satisfy r".
//   - A rule query ending in "[]" expands matches: errors carry the
//     concrete index (.arr[i]) for arrays and the concrete key
//     (.obj["k"]) for maps.
//   - Only a literal "[]" suffix triggers element expansion. Filter
//     queries like ".arr[.age > 18]" still run but attribute all
//     failures to the filter expression itself; see docs/schema.md
//     for the workaround.
//
//	rules := schema.QueryRules{
//	    ".name": schema.Required(schema.String{}),
//	    ".age":  schema.Int{Min: schema.Int64Ptr(0), Max: schema.Int64Ptr(150)},
//	}
//	err := schema.Validate(doc, rules)
//
// # Rule types
//
//   - [Require]: fires when the node is Nil.
//   - [And]: passes when every child passes; collects all errors.
//   - [Or]: passes when at least one child passes.
//   - [Not]: passes when the inner rule fails; skips Nil.
//   - [Every]: apply a QueryRules set to each element of an array or
//     each value of a map. Required inside Every fires per element,
//     which a bare "[]"-suffixed query cannot do.
//   - [String]: string leaf with optional Enum, Regex, MinLen, MaxLen.
//   - [Int]: integer-valued number with optional Min/Max; rejects NaN,
//     Inf, fractional, and out-of-int64-range values.
//   - [Float]: number leaf with optional Min/Max; rejects NaN.
//   - [Bool]: boolean leaf.
//   - [Array]: array leaf with optional MinLen/MaxLen. Element rules
//     go on separate "[]"-suffixed queries.
//   - [Map]: map leaf with optional Keys allow-list.
//
// [IntPtr], [Int64Ptr], and [Float64Ptr] let you set Min/Max inline
// without a temporary variable.
//
// # YAML / JSON rule specs
//
// [ParseQueryRules] decodes a tree.Node (typically produced by a YAML
// or JSON decoder) into [QueryRules]. Every spec carries a "type"
// field and an optional meta flag "required: true". The built-in
// types, their fields, and nested spec examples are in
// docs/schema.md.
//
// # Custom rule types
//
// Any Go value implementing [Rule] can be inserted into a QueryRules
// map. To expose a custom rule to the YAML / JSON loader, register a
// [Factory] with a [Registry]:
//
//	reg := schema.BuiltinRegistry()
//	reg.Register("uuid", parseUUIDSpec, "version")
//	rules, err := schema.ParseQueryRulesWith(specNode, reg)
//
// [ValidateWithPrefix] helps composite custom rules re-apply a
// QueryRules set to a subtree and attribute sub-errors to the
// enclosing location. docs/schema.md walks through a
// dispatch-on-"type" cookbook.
//
// # Errors
//
// [Validate], [ParseQueryRules], and [Registry.Parse] join violations
// via errors.Join. [Registry.MaxReportedErrors] caps parsed-spec
// errors (default 20); overflow is summarised as "and N more errors".
//
// A malformed rule query — a bug in the rules, not the data — causes
// [Validate] and [ValidateWithPrefix] to fail fast and return a single
// [*ErrQuery] without joining any data-level violations. Callers that
// need to distinguish can errors.As into [*ErrQuery].
package schema
