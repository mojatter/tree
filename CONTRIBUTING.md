# Contributing to tree

Thanks for taking an interest in `tree` / `tq`. This document explains
the local development workflow and the conventions the project follows.

## Prerequisites

- Go matching the `toolchain` directive in [go.mod](go.mod) (currently
  `go1.24.4`). Newer Go versions automatically download the pinned
  toolchain via `go.mod`.
- [`golangci-lint`](https://golangci-lint.run/) v2.x for linting.
- `make` for running the convenience targets below.

## Getting started

```sh
git clone https://github.com/mojatter/tree.git
cd tree
make test       # run the unit tests
make lint       # run golangci-lint
```

`make help` lists every available target.

## Common tasks

| Goal | Command |
|---|---|
| Run unit tests | `make test` |
| Run tests with the race detector | `make test-race` |
| Generate a coverage profile | `make cover` |
| Run linters | `make lint` |
| Run fuzz targets (default 30s each) | `make fuzz` |
| Run benchmarks | `make bench` |
| Refresh `cmd/tq` golden files after intentional output changes | `make update-golden` |
| `go mod tidy` for the root and `examples/` modules | `make tidy` |
| Run `govulncheck` | `make vuln` |
| Everything CI cares about | `make all` |

Variables can be overridden on the command line, for example:

```sh
make fuzz FUZZTIME=2m
make bench BENCHTIME=10s BENCHCOUNT=3
```

## Code conventions

- **Test layout.** Each `foo.go` has a single corresponding
  `foo_test.go`. Add new tests by appending to the existing file
  rather than creating split files like `foo_extra_test.go`.
- **Table-driven tests** are preferred. The slice variable should be
  `testCases`, the loop variable `tc`, and the per-case name field
  `caseName`, matching the existing tests.
- **Lint suppressions.** Prefer narrow `//nolint:<linter> // reason`
  comments at the call site. Use `.golangci.yml` exclusions only for
  patterns that repeat across many files.
- **`SkipWalk`** is intentionally named without an `Err` prefix; it is
  a sentinel control value modeled after `io/fs.SkipDir`. The
  in-source godoc explains the staticcheck suppression.

## Pull requests

- Open one PR per logical change. Prefer a single squashable commit
  with a descriptive subject; iterate via `git commit --amend` or
  `git reset --soft origin/main` rather than stacking fixup commits.
- Subject style: `type(scope): Capitalized summary` (Conventional
  Commits style prefix, sentence case after the colon).
- Keep PR descriptions concrete: a one-paragraph **Summary** and a
  **Test plan** checklist that mirrors what you actually ran locally
  is enough.
- CI must be green before requesting review. `make all` is a quick way
  to reproduce most of what CI checks.

## Tests, lint, and security gates

These run automatically in CI:

- `tests` workflow — `go test -race -coverprofile=...` on Ubuntu plus
  `golangci-lint`.
- `release` workflow — runs the same `tests` job on tag pushes, then
  `goreleaser`.
- `govulncheck` workflow — weekly schedule plus PR runs that touch
  `go.mod` / `go.sum`.

If you need to add a new dependency or upgrade one, please run
`make tidy` and `make vuln` locally before opening the PR.

## Reporting issues

Open an issue on GitHub for bug reports and feature requests. For
security disclosures, please open a private security advisory on
GitHub instead of a public issue.
