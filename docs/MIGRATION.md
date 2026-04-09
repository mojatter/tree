# Migration guide

This document collects upgrade notes that do not fit into CHANGELOG
bullets. If you hit a rough edge moving between versions (or from the
old `jarxorg/tree` repository), start here.

## Table of contents

- [Migrating from jarxorg/tree to mojatter/tree](#migrating-from-jarxorgtree-to-mojattertree)
- [v0.9.x -> v0.10.0 breaking changes](#v09x---v0100-breaking-changes)

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
