// Package tree provides a simple structure for dealing with dynamic or unknown JSON/YAML structures.
package tree

// VERSION is the version number. The default "dev" value is
// overridden at release build time via -ldflags by goreleaser:
//
//	-X github.com/mojatter/tree.VERSION={{.Version}}
var VERSION = "dev"
