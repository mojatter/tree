# Schema

The `schema` subpackage validates `tree.Node` structures against a
set of rules keyed by tree queries.

> **Roles**
> [`schema/doc.go`](../schema/doc.go) (rendered on
> [pkg.go.dev](https://pkg.go.dev/github.com/mojatter/tree/schema))
> is the API overview and is always authoritative for type and
> function signatures. This file is the *scenario guide*: nested
> validation walkthroughs, YAML reference, custom-rule cookbooks,
> and the fine print around filter queries and error handling.
> If a code snippet here drifts from the godoc, fix the godoc first
> and treat this file as the follow-up.

## Table of contents

- [What it does](#what-it-does)
- [Getting started: flat rules](#getting-started-flat-rules)
- [Nested validation with `Every`](#nested-validation-with-every)
- [`"[]"` suffix vs `Every`](#-suffix-vs-every)
- [Filter queries and where they fall short](#filter-queries-and-where-they-fall-short)
- [Nil semantics](#nil-semantics)
- [Loading rules from YAML / JSON](#loading-rules-from-yaml--json)
- [Custom rule types](#custom-rule-types)
- [Dispatching custom rules with `ValidateWithPrefix`](#dispatching-custom-rules-with-validatewithprefix)
- [Error handling](#error-handling)
- [Topic index (godoc jump-list)](#topic-index-godoc-jump-list)

## What it does

- Each `(query, rule)` entry in a [`QueryRules`][QueryRules] map is
  applied to every tree node matched by the query.
- [`schema.Validate(doc, rules)`][Validate] collects all rule errors
  and joins them with `errors.Join`, so a single call surfaces every
  violation.
- A missing node is passed to the rule as `tree.Nil`; only
  [`Require`][Require] fires on nil, which is what makes
  [`schema.Required(r)`][Required] idiomatic for "must be present and
  satisfy `r`".
- The package is **not** JSON Schema compatible — the primitives are
  standalone validation rules, and only the aggregate `QueryRules`
  map behaves schema-like.

## Getting started: flat rules

A rule set says *what each value must be*, keyed by a tree query
that picks the value. Here is a minimal rule set written as a YAML
spec (the form most users reach for first):

```yaml
".name":   { type: string, required: true }
".age":    { type: int, min: 0, max: 150 }
".tags[]": { type: string }
".":       { type: map, keys: [name, age, tags] }
```

Loading that spec into a usable rule set is covered in
[Loading rules from YAML / JSON](#loading-rules-from-yaml--json) —
short version: decode the YAML into a `tree.Node`, then pass it to
`schema.ParseQueryRules`.

The same rules as Go values, if you prefer to build them directly
in code:

```go
doc := tree.Map{
    "name": tree.V("alice"),
    "age":  tree.V(30),
    "tags": tree.A("ops", "oncall"),
}
err := schema.Validate(doc, schema.QueryRules{
    ".name":   schema.Required(schema.String{}),
    ".age":    schema.Int{Min: schema.Int64Ptr(0), Max: schema.Int64Ptr(150)},
    ".tags[]": schema.String{},
    ".":       schema.Map{Keys: []string{"name", "age", "tags"}},
})
```

- `".tags[]"` expands matches; each non-string element surfaces at
  `.tags[i]`.
- `Map{Keys: ...}` is an allow-list on the root; any key outside the
  list is reported as an error (sorted for stable output).

[`IntPtr`][IntPtr], [`Int64Ptr`][Int64Ptr], and
[`Float64Ptr`][Float64Ptr] exist so you can set `Min`/`Max` inline
without a temporary variable.

## Nested validation with `Every`

When each element of an array or each value of a map needs its own
set of rules, wrap a `QueryRules` set in [`Every`][Every]. The
bookstore fixture from tree's README has a `.store.book` array where
every book should carry an `isbn`, a non-negative `price`, and tags
whose `value` is a non-empty string:

```go
rules := schema.QueryRules{
    ".store.bicycle.color": schema.Required(schema.String{Enum: []string{"red", "blue", "green"}}),
    ".store.book": schema.Every{Rules: schema.QueryRules{
        ".isbn":  schema.Required(schema.String{}),
        ".price": schema.Required(schema.Float{Min: schema.Float64Ptr(0)}),
        ".tags":  schema.Every{Rules: schema.QueryRules{
            ".value": schema.Required(schema.String{}),
        }},
    }},
}
err := schema.Validate(doc, rules)
```

Books missing an ISBN surface as `.store.book[0].isbn: required`.
Each rule query inside an `Every` is resolved relative to the
element, and error paths include the concrete index or key. `Every`
can be nested arbitrarily deep — above, `.tags` is another `Every`
inside the outer one.

See [`ExampleValidate_nested`](../schema/example_test.go) for a
runnable version of the full fixture.

## `"[]"` suffix vs `Every`

Both iterate over array elements or map values, but they differ in
how they treat each element:

| Feature | `"[]"` suffix | `Every` |
| ------- | ------------- | ------- |
| Applies leaf rule per element | ✓ | via inner rules |
| Applies sub-rule set per element | — | ✓ |
| `Required` fires per element | — (only on the parent) | ✓ |
| Good for | `.arr[] = String{}` type checks | elements with required sub-fields |

Use `"[]"` for cheap "every element must be type X" checks. Reach
for `Every` the moment you need per-element `Required`, nested
structure, or two or more rules per element.

## Filter queries and where they fall short

Filter-style queries like `.store.book[.category == "fiction"]` work
— `tree.Find` returns the matching nodes and the rule runs on each —
but with two caveats that make them a poor fit for most validation
jobs:

1. **Error paths stay as the filter expression.** If three matches
   fail, all three errors carry the same display path (the whole
   `.store.book[.category == "fiction"]` string). You cannot tell
   which element failed.
2. **Zero matches treat the filter as a single Nil node.** A
   `Required` rule on a filter fires once with the filter itself as
   the "missing" path, which is rarely the intent.

Only a literal `"[]"` suffix triggers per-element expansion. When
you want element-level error reporting, use `"[]"` or wrap the
parent array/map in `Every`. Filter queries are great for the
`tree.Find` side (selecting nodes to read) but clash with
validation's need for attribution.

## Nil semantics

When a query yields no matches, the rule is invoked with `tree.Nil`:

- Leaf rules (`String`, `Int`, `Float`, `Bool`, `Array`, `Map`) and
  composites (`And`, `Or`, `Not`, `Every`) return nil on `Nil`. A
  bare rule means "if present, satisfy this constraint".
- Only [`Require`][Require] fires on `Nil`. Use
  [`Required(r)`][Required] — which is `And{Require{}, r}` — to
  make presence mandatory.
- `Required` is idempotent: `Required(Required(r))` is the same as
  `Required(r)`.

## Loading rules from YAML / JSON

[`ParseQueryRules`][ParseQueryRules] decodes a `tree.Node` —
typically produced by a YAML or JSON decoder — into `QueryRules`.
The document is a map where each key is a tree query and each value
is a rule spec:

```yaml
".name":   { type: string, required: true }
".age":    { type: int, min: 0, max: 150 }
".tags[]": { type: string }
".":       { type: map, keys: [name, age, tags] }
```

Each spec must carry a `type` field. `required: true` wraps the
resulting rule with `Required`.

### Built-in types and their fields

| type | fields |
| ---- | ------ |
| `require` | *(none)* |
| `string` | `enum`, `regex`, `minLen`, `maxLen` |
| `int` | `min`, `max` |
| `float` | `min`, `max` |
| `bool` | *(none)* |
| `array` | `minLen`, `maxLen` |
| `map` | `keys` |
| `and` | `of: [spec, ...]` |
| `or` | `of: [spec, ...]` |
| `not` | `rule: spec` |
| `every` | `rules: {query: spec, ...}` |

Plus the universal meta fields `type` and `required`. Unknown
fields on a spec are rejected. Field types are checked strictly —
for example, `min: "0"` is an error, not coerced to `0`.

### Nested YAML example

The bookstore rules from earlier, written as a spec:

```yaml
".store.bicycle.color": { type: string, required: true, enum: [red, blue, green] }
".store.book":
  type: every
  rules:
    ".isbn":  { type: string, required: true }
    ".price": { type: float, required: true, min: 0 }
    ".tags":
      type: every
      rules:
        ".value": { type: string, required: true }
```

Leaf specs go on one line in flow style for density; specs with
nested content (here, `every.rules`) stay in block style so the
nesting is readable. `every.rules` recurses, so any of the
built-in or custom types can appear at any level.

## Custom rule types

Any Go value implementing [`Rule`][Rule] can be dropped into a
`QueryRules` map:

```go
type UUID struct{}

func (UUID) Validate(n tree.Node, q string) error {
    if n.IsNil() {
        return nil
    }
    if !n.Type().IsStringValue() {
        return fmt.Errorf("%s: expected string, got %s", q, n.Type())
    }
    // ... real UUID parsing ...
    return nil
}

rules := schema.QueryRules{".id": schema.Required(UUID{})}
```

Conventions to keep custom rules predictable alongside the built-in
ones:

- Return nil on `Nil`; wrapping with `Required` is the caller's job.
- Use the passed-in `q` (query path) as the error prefix so error
  paths line up with the rest of the package.
- Collect all sub-errors and `errors.Join` them rather than
  short-circuiting on the first failure.

### Registering a custom type for the YAML / JSON loader

To expose a custom rule to [`ParseQueryRulesWith`][ParseQueryRulesWith],
register a factory with a [`Registry`][Registry]:

```go
reg := schema.BuiltinRegistry()
reg.Register("uuid", parseUUIDSpec, "version")
rules, err := schema.ParseQueryRulesWith(specNode, reg)
```

The `fields` argument declares which spec keys the factory consumes
beyond the universal `type` and `required`. Any other key on the
spec is reported as an unknown-field error — this keeps typos from
silently passing.

## Dispatching custom rules with `ValidateWithPrefix`

Some configurations use a discriminator field to pick a sub-schema —
for example, a `handlers` map where each value has `type: http`,
`type: file`, or another handler kind, each with its own required
fields. The clean way to validate that in this package is a custom
rule that looks at the discriminator and calls
[`ValidateWithPrefix`][ValidateWithPrefix] on a pre-built
`QueryRules` set for the chosen kind:

```go
type HandlerDispatch struct {
    ByType map[string]schema.QueryRules
}

func (r HandlerDispatch) Validate(n tree.Node, q string) error {
    if n.IsNil() {
        return nil
    }
    typ := n.Get("type").Value().String()
    sub, ok := r.ByType[typ]
    if !ok {
        return nil // or flag unknown type, up to the caller
    }
    return schema.ValidateWithPrefix(n, sub, q)
}
```

Combined with a top-level `Every` over the handlers map:

```go
rules := schema.QueryRules{
    ".handlers": schema.Every{Rules: schema.QueryRules{
        ".": HandlerDispatch{ByType: map[string]schema.QueryRules{
            "http": {".port": schema.Required(schema.Int{})},
            "file": {".path": schema.Required(schema.String{})},
        }},
    }},
}
```

Errors from a handler missing its required field surface as, e.g.,
`.handlers["c"].port: required` — the enclosing map-key is preserved
in the path because `ValidateWithPrefix` prepended it.

Without `ValidateWithPrefix`, the custom rule would have to unwrap
the inner errors and re-wrap them with `fmt.Errorf("%s: %w", q, ...)`
by hand, which is easy to get wrong (especially for multi-error
joins). Prefer `ValidateWithPrefix` in any composite rule that
re-applies a `QueryRules` set to a subtree.

## Error handling

`Validate`, `ParseQueryRules`, and `Registry.Parse` collect errors
via `errors.Join`, so every violation is surfaced in a single call.
`Registry.MaxReportedErrors` caps the number of parsed-spec errors
(default 20); overflow is summarised as `"and N more errors"`.
`Validate` itself currently has no such cap — the number of
violations scales with the data.

### `*ErrQuery` — malformed rule queries

A malformed rule query (one that `tree.Find` rejects) is a bug in
the rules definition, not a data violation. `Validate` and
`ValidateWithPrefix` fail fast on that case and return a single
[`*ErrQuery`][ErrQuery] without joining any data-level violations:

```go
err := schema.Validate(doc, schema.QueryRules{
    ".name": schema.String{},
    ".bad[": schema.String{}, // syntax error in the rule query
})
var eq *schema.ErrQuery
if errors.As(err, &eq) {
    // The rules are broken. eq.Query is the failing path
    // (already prefixed if the rule ran inside Every or
    // ValidateWithPrefix); eq.Err wraps the underlying
    // tree.Find error.
}
```

Why fail fast? A broken rule query means every subsequent invocation
of the same rule set would emit the same error; joining it with
unrelated data violations would only produce noise. Fix the rule,
rerun, get a clean picture of what the data itself actually
violates.

## Topic index (godoc jump-list)

Quick links into the [godoc][pkg] for readers who want the API
reference alongside this guide.

- **Core:** [`Validate`][Validate], [`ValidateWithPrefix`][ValidateWithPrefix],
  [`QueryRules`][QueryRules], [`Rule`][Rule].
- **Presence / composition:** [`Require`][Require],
  [`Required`][Required], [`And`][And], [`Or`][Or], [`Not`][Not],
  [`Every`][Every].
- **Leaf rules:** [`String`][String], [`Int`][Int], [`Float`][Float],
  [`Bool`][Bool], [`Array`][Array], [`Map`][Map].
- **Helpers:** [`IntPtr`][IntPtr], [`Int64Ptr`][Int64Ptr],
  [`Float64Ptr`][Float64Ptr].
- **YAML / JSON loader:** [`ParseQueryRules`][ParseQueryRules],
  [`ParseQueryRulesWith`][ParseQueryRulesWith],
  [`BuiltinRegistry`][BuiltinRegistry], [`Registry`][Registry].
- **Errors:** [`*ErrQuery`][ErrQuery].

[pkg]: https://pkg.go.dev/github.com/mojatter/tree/schema
[Validate]: https://pkg.go.dev/github.com/mojatter/tree/schema#Validate
[ValidateWithPrefix]: https://pkg.go.dev/github.com/mojatter/tree/schema#ValidateWithPrefix
[QueryRules]: https://pkg.go.dev/github.com/mojatter/tree/schema#QueryRules
[Rule]: https://pkg.go.dev/github.com/mojatter/tree/schema#Rule
[Require]: https://pkg.go.dev/github.com/mojatter/tree/schema#Require
[Required]: https://pkg.go.dev/github.com/mojatter/tree/schema#Required
[And]: https://pkg.go.dev/github.com/mojatter/tree/schema#And
[Or]: https://pkg.go.dev/github.com/mojatter/tree/schema#Or
[Not]: https://pkg.go.dev/github.com/mojatter/tree/schema#Not
[Every]: https://pkg.go.dev/github.com/mojatter/tree/schema#Every
[String]: https://pkg.go.dev/github.com/mojatter/tree/schema#String
[Int]: https://pkg.go.dev/github.com/mojatter/tree/schema#Int
[Float]: https://pkg.go.dev/github.com/mojatter/tree/schema#Float
[Bool]: https://pkg.go.dev/github.com/mojatter/tree/schema#Bool
[Array]: https://pkg.go.dev/github.com/mojatter/tree/schema#Array
[Map]: https://pkg.go.dev/github.com/mojatter/tree/schema#Map
[IntPtr]: https://pkg.go.dev/github.com/mojatter/tree/schema#IntPtr
[Int64Ptr]: https://pkg.go.dev/github.com/mojatter/tree/schema#Int64Ptr
[Float64Ptr]: https://pkg.go.dev/github.com/mojatter/tree/schema#Float64Ptr
[ParseQueryRules]: https://pkg.go.dev/github.com/mojatter/tree/schema#ParseQueryRules
[ParseQueryRulesWith]: https://pkg.go.dev/github.com/mojatter/tree/schema#ParseQueryRulesWith
[BuiltinRegistry]: https://pkg.go.dev/github.com/mojatter/tree/schema#BuiltinRegistry
[Registry]: https://pkg.go.dev/github.com/mojatter/tree/schema#Registry
[ErrQuery]: https://pkg.go.dev/github.com/mojatter/tree/schema#ErrQuery
