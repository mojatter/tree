# Migration guide

This document collects upgrade notes that do not fit into CHANGELOG
bullets. If you hit a rough edge moving between versions (or from the
old `jarxorg/tree` repository), start here.

## Table of contents

- [Migrating from jarxorg/tree to mojatter/tree](#migrating-from-jarxorgtree-to-mojattertree)
- [v0.9.x -> v0.10.0 breaking changes](#v09x---v0100-breaking-changes)
- [v0.11.x -> v0.12.0 breaking changes](#v011x---v0120-breaking-changes)

---

## Migrating from jarxorg/tree to mojatter/tree

The repository was transferred from `jarxorg` to `mojatter` in 2026.
The Go module path and the Homebrew tap both changed as a result.

- Old: `github.com/jarxorg/tree` / tap `jarxorg/tree`
- New: `github.com/mojatter/tree` / tap `mojatter/tree`

The API is the same — only the import path and the distribution
channels moved.

### Go library consumers

Update every import:

```go
// before
import "github.com/jarxorg/tree"

// after
import "github.com/mojatter/tree"
```

Then rewrite `go.mod`:

```sh
# Option 1: rely on go mod tidy after updating imports
go mod tidy

# Option 2: use a replace directive during a gradual migration
go mod edit -replace=github.com/jarxorg/tree=github.com/mojatter/tree@latest
go mod tidy
```

Once every file is updated, remove the `replace` directive if you
used one.

### `tq` installed via `go install`

Install the new module and remove the old binary from `$GOBIN`:

```sh
go install github.com/mojatter/tree/cmd/tq@latest
rm -f "$(go env GOBIN)/tq"  # only if it came from the old module
go install github.com/mojatter/tree/cmd/tq@latest
```

### `tq` installed via Homebrew

This is where most friction happens, because the old tap may leave a
stale `tq` binary in your `PATH` that shadows the new install. The
full recovery looks like this:

```sh
# 1. Remove the old tap entirely
brew uninstall --force jarxorg/tree/tq 2>/dev/null
brew untap jarxorg/tree 2>/dev/null

# 2. Remove any stale binary at the linked location
rm -f /opt/homebrew/bin/tq /usr/local/bin/tq

# 3. Install from the new tap
brew install mojatter/tree/tq

# 4. Verify
which -a tq   # should print exactly one path
tq -v         # should print the version you just installed
```

#### Troubleshooting

**`brew link` fails with "Target /opt/homebrew/bin/tq already exists"**

The old `jarxorg/tree/tq` formula left its binary behind. Homebrew
will not overwrite it automatically. Resolve with:

```sh
brew unlink tq 2>/dev/null
rm -f /opt/homebrew/bin/tq
brew link --overwrite mojatter/tree/tq
```

**`which -a tq` shows more than one path**

Something other than Homebrew (a manual `curl` install under
`/usr/local/bin`, or an old `go install` binary under
`$GOBIN`/`$GOPATH/bin`) is also providing `tq`. Remove the extras
until only the Homebrew path remains.

**`tq -v` still prints an old version after `brew install`**

Almost always the same root cause: the shell is running a stale
binary because a directory earlier in `$PATH` has its own copy. Run
`which -a tq` and remove whichever entries should not be there.

---

## v0.9.x -> v0.10.0 breaking changes

### YAML backend switched from `gopkg.in/yaml.v2` to `go.yaml.in/yaml/v3`

`yaml.v2` is no longer maintained. tree now depends on
`go.yaml.in/yaml/v3`, the supported successor. Consumers that rely on
tree's YAML surface need to adjust:

- Any custom `UnmarshalYAML` implementation on a type embedded in
  `tree.Map` or `tree.Array` must satisfy the v3 signature
  `UnmarshalYAML(*yaml.Node) error` instead of the v2
  `UnmarshalYAML(func(interface{}) error) error`.
- `tree.DecodeYAML` now takes a `*go.yaml.in/yaml/v3.Decoder`. If you
  were constructing a `*gopkg.in/yaml.v2.Decoder` and passing it in,
  update the import and the constructor call.
- `tree.MarshalYAML` now explicitly sets a 2-space block indent to
  stay close to the previous v2 output. If you were comparing
  serialized YAML byte-for-byte, regenerate your fixtures after
  upgrading. YAML produced by `tq -o yaml` changes its nested
  sequence indentation from flush-left to 2-space indented to match.

### `tree.VERSION` is now a `var`, not a `const`

To let `goreleaser` inject the tag at build time, `tree.VERSION`
became a package-level `var` defaulting to `"dev"`. Library consumers
that used it in a constant expression will no longer compile:

```go
// before: compiles
const myVersion = "app/" + tree.VERSION

// after: error - tree.VERSION is not a constant
var myVersion = "app/" + tree.VERSION
```

Switch the surrounding declaration from `const` to `var` (or move the
concatenation to a package init) to fix it.

### `tree` binaries no longer hard-code a version string

Release builds (via `goreleaser`) inject the tag into `tree.VERSION`
with `-ldflags`. `go install`-built binaries will report `dev`
instead of a number; this is intentional. Use a tagged release from
`brew` or a download if you need the version string to be
meaningful.

---

## v0.11.x -> v0.12.0 breaking changes

v0.12.0 replaces the regex-based query parser with a hand-written
recursive-descent parser. The rewrite fixes six pre-existing parser
bugs and unlocks Python-style negative array indexing, but it also
tightens parsing in ways that can surface latent issues. This section
walks through every break and silent-behavior change, with the code
shape needed to migrate.

### `ArrayRangeQuery` is now a struct, not `[]int`

The previous `[]int` shape used `-1` as an "omitted" sentinel:
`ArrayRangeQuery{-1, 5}` meant `[:5]`, and `ArrayRangeQuery{1, -1}`
meant `[1:]`. This collided with the new Python-style "last element"
meaning of `-1`, so the type was migrated to a struct with explicit
nil-able pointer fields.

```go
// before: v0.11.x
import "github.com/mojatter/tree"

q1 := tree.ArrayRangeQuery{-1, 5}    // meant [:5]
q2 := tree.ArrayRangeQuery{1, -1}    // meant [1:]
q3 := tree.ArrayRangeQuery{1, 5}     // meant [1:5]
```

```go
// after: v0.12.0
import "github.com/mojatter/tree"

q1 := tree.ArrayRangeQuery{To: tree.IntPtr(5)}                       // [:5]
q2 := tree.ArrayRangeQuery{From: tree.IntPtr(1)}                     // [1:]
q3 := tree.ArrayRangeQuery{From: tree.IntPtr(1), To: tree.IntPtr(5)} // [1:5]
```

`nil` means "omitted" (whole-end slice). A non-nil negative pointer
now means "from / to the end", Python-style:

```go
// new in v0.12.0: previously a parse error or silently wrong
tree.ArrayRangeQuery{From: tree.IntPtr(-3)}                       // [-3:]   (last 3 elements)
tree.ArrayRangeQuery{From: tree.IntPtr(1), To: tree.IntPtr(-1)}   // [1:-1]  (drop first, drop last)
```

`tree.IntPtr` is the canonical helper. `tree.Int64Ptr` and
`tree.Float64Ptr` ship alongside it for symmetry. The same-named
helpers in the `schema` package (`schema.IntPtr` etc.) are kept as
thin re-exports but marked `Deprecated`; new code should import from
`tree`.

This change only affects code that constructs `ArrayRangeQuery`
directly. Query strings parsed via `tree.ParseQuery` are unaffected
by the type shape — but see the next section for query-string
behavior changes.

### `[-1]` now resolves to the last element

In v0.11.x, `tree.ParseQuery("[-1]")` silently produced
`tree.ArrayQuery(1)` — the leading `-` was dropped during lexing. On
a 3-element array, `[-1]` would resolve to index `1`, not the last
element.

v0.12.0 lexes `-` as a real token. `[-1]` parses to
`tree.ArrayQuery(-1)` and resolves jq-style to the last element.

```go
n, _ := tree.UnmarshalJSON([]byte(`["a", "b", "c"]`))
q, _ := tree.ParseQuery("[-1]")
got, _ := q.Exec(n)
// v0.11.x: got == ["b"]
// v0.12.0: got == ["c"]
```

If you had code that relied on the buggy v0.11.x behavior, switch to
the explicit positive index.

`tree.ArrayQuery(-1).Set(...)`, `Append(...)`, and `Delete(...)` on
a value that is a `Map` (not an `Array`) used to silently write or
no-op against the literal key `"-1"`. They now return an error
because negative indices have no meaning on maps.

### `.foo-bar` is now a syntax error

Hyphens in unquoted path keys used to silently split the path:
`.foo-bar` parsed as `FilterQuery{MapQuery("foo"), MapQuery("bar")}`,
which was almost never what the caller intended.

v0.12.0 returns a syntax error. Quote the key explicitly:

```go
// before: v0.11.x — silently parsed as .foo.bar
q, _ := tree.ParseQuery(".foo-bar")

// after: v0.12.0
q, err := tree.ParseQuery(`."foo-bar"`)
```

The same applies to keys containing other punctuation; quoting is the
universal escape hatch. Single quotes also work
(`.'foo-bar'`, new in v0.12.0).

### `[:]` and `[2:1]` no longer panic

v0.11.x panicked on degenerate range expressions:

- `[:]` (no `From`, no `To`)
- `[2:1]` (`From > To`)
- `[-3:5]` and other end-relative ranges

v0.12.0 resolves these without panicking:

- `[:]` returns every element
- `[2:1]` returns an empty slice
- `[-3:5]`, `[1:-1]`, `[-3:-1]` clamp Python-style

If you wrapped any of these in `defer recover()` to swallow the
panic, the recover branch is now dead code.

### `Inf` / `NaN` in selector operands are strings, not numbers

v0.11.x parsed `Inf`, `+Inf`, `-Inf`, and `NaN` ident tokens as
floating-point literals via `strconv.ParseFloat`. Inside selectors
this caused a footgun: `[.x == NaN]` compiled against
`NumberValue(NaN)`, and `NaN != NaN` by IEEE rules, so the selector
silently never matched.

v0.12.0 treats these tokens as strings:

```go
// v0.11.x: tries (and fails) to compare to numeric NaN
q, _ := tree.ParseQuery("[.x == NaN]")

// v0.12.0: compares .x to the literal string "NaN"
q, _ := tree.ParseQuery(`[.x == "NaN"]`)
```

If you really need a numeric `+Inf` or `NaN` comparison, build the
selector programmatically instead of going through
`tree.ParseQuery`.

### Float literals in selectors now work

v0.11.x failed to parse decimals, signed exponents, and negative
numbers in selector operand position:

```go
// v0.11.x: parse error
tree.ParseQuery("[.x == 1.5]")
tree.ParseQuery("[.x == -1.5]")
tree.ParseQuery("[.x == 1e-3]")
```

v0.12.0 parses all three correctly. This is purely additive — code
that was already working continues to work.

### Summary table

| Expression | v0.11.x | v0.12.0 |
|---|---|---|
| `ArrayRangeQuery{1, -1}` literal | `[1:]` (sentinel `-1`) | type error (use struct + `IntPtr`) |
| `[-1]` on `[a, b, c]` | silently `[1]` -> `b` | `c` (last element) |
| `[:]` | panic | every element |
| `[2:1]` | panic | empty slice |
| `[-3:]`, `[1:-1]` | parse error / silently wrong | Python-style end-relative |
| `.foo-bar` | silent `FilterQuery{foo, bar}` | syntax error (quote it) |
| `[foo-bar]` | silent meaningless selector | error |
| `ArrayQuery(-1).Set/Delete` on `Map` | wrote / no-op'd `"-1"` key | error |
| `[.x == Inf]`, `[.x == NaN]` | numeric `+Inf` / `NaN` | string `"Inf"` / `"NaN"` |
| `[.x == 1.5]`, `[.x == -1.5]`, `[.x == 1e-3]` | parse error | parsed correctly |
