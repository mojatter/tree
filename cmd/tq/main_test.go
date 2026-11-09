package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/mojatter/io2"
	"github.com/mojatter/tree"
)

// updateGolden rewrites golden files under testdata/ from the current
// output of each test case. Run `go test -update ./cmd/tq` after
// intentionally changing tq's output, then review the diff.
var updateGolden = flag.Bool("update", false, "update golden files in testdata")

func TestRun(t *testing.T) {
	stdinOrg := os.Stdin
	defer func() { os.Stdin = stdinOrg }()

	mustReadFileString := func(file string) string {
		bin, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		return string(bin)
	}

	testCases := []struct {
		caseName string
		stdin    string
		args     []string
		// golden is a path to a testdata file whose contents are the
		// expected output. Prefer this over want so that golden files
		// can be refreshed via `go test -update`.
		golden string
		want   string
		errstr string
		// pipe is fed through a pipe whose /dev/fd path is appended to args.
		pipe string
	}{
		{
			caseName: "show usage",
			args:     []string{},
			golden:   "testdata/usage",
		}, {
			caseName: "show help",
			args:     []string{"-h"},
			golden:   "testdata/usage",
		}, {
			caseName: "show version",
			args:     []string{"-v"},
			want:     tree.VERSION + "\n",
		}, {
			caseName: "first book from stdin",
			stdin:    "testdata/store.json",
			args:     []string{".store.book[0]"},
			golden:   "testdata/book-0.json",
		}, {
			caseName: "first book from stdin yaml",
			stdin:    "testdata/store.yaml",
			args:     []string{".store.book[0]"},
			golden:   "testdata/book-0.yaml",
		}, {
			caseName: "first book from stdin with -i yaml",
			stdin:    "testdata/store.yaml",
			args:     []string{"-i", "yaml", ".store.book[0]"},
			golden:   "testdata/book-0.yaml",
		}, {
			caseName: "first book from json",
			args:     []string{".store.book[0]", "testdata/store.json"},
			golden:   "testdata/book-0.json",
		}, {
			caseName: "first book from json to yaml",
			args:     []string{"-o", "yaml", ".store.book[0]", "testdata/store.json"},
			golden:   "testdata/book-0.yaml",
		}, {
			caseName: "first book from yaml to json with -o json",
			args:     []string{"-o", "json", ".store.book[0]", "testdata/store.yaml"},
			golden:   "testdata/book-0.json",
		}, {
			caseName: "first book from yaml to json with -J",
			args:     []string{"-J", ".store.book[0]", "testdata/store.yaml"},
			golden:   "testdata/book-0.json",
		}, {
			caseName: "first book from json to yaml with -Y",
			args:     []string{"-Y", ".store.book[0]", "testdata/store.json"},
			golden:   "testdata/book-0.yaml",
		}, {
			caseName: "-Y after -o json wins",
			args:     []string{"-o", "json", "-Y", ".store.book[0]", "testdata/store.yaml"},
			golden:   "testdata/book-0.yaml",
		}, {
			caseName: "-o json after -Y wins",
			args:     []string{"-Y", "-o", "json", ".store.book[0]", "testdata/store.yaml"},
			golden:   "testdata/book-0.json",
		}, {
			caseName: "-Y after -J wins",
			args:     []string{"-J", "-Y", ".store.book[0]", "testdata/store.yaml"},
			golden:   "testdata/book-0.yaml",
		}, {
			caseName: "-J after -o yaml wins",
			args:     []string{"-o", "yaml", "-J", ".store.book[0]", "testdata/store.yaml"},
			golden:   "testdata/book-0.json",
		}, {
			caseName: "-j after -i yaml wins",
			args:     []string{"-i", "yaml", "-j", ".store.book[0]", "testdata/store.json"},
			golden:   "testdata/book-0.json",
		}, {
			caseName: "-i yaml after -j wins",
			args:     []string{"-j", "-i", "yaml", ".store.book[0]", "testdata/store.json"},
			golden:   "testdata/book-0.yaml",
		}, {
			caseName: "-JY writes yaml",
			args:     []string{"-JY", ".store.book[0]", "testdata/store.json"},
			golden:   "testdata/book-0.yaml",
		}, {
			caseName: "-yj reads json",
			args:     []string{"-yj", ".store.book[0]", "testdata/store.json"},
			golden:   "testdata/book-0.json",
		}, {
			caseName: "unknown output format",
			args:     []string{"-o", "toml", ".", "testdata/store.json"},
			errstr:   `invalid argument "toml" for "-o, --output-format" flag: must be json or yaml`,
		}, {
			caseName: "output format is case sensitive",
			args:     []string{"--output-format", "JSON", ".", "testdata/store.json"},
			errstr:   `invalid argument "JSON" for "-o, --output-format" flag: must be json or yaml`,
		}, {
			caseName: "empty input format",
			args:     []string{"-i", "", ".", "testdata/store.json"},
			errstr:   `invalid argument "" for "-i, --input-format" flag: must be json or yaml`,
		}, {
			caseName: "unknown flag",
			args:     []string{"--nope", ".", "testdata/store.json"},
			errstr:   `unknown flag: --nope`,
		}, {
			caseName: "--output-yaml=false keeps json",
			args:     []string{"--output-yaml=false", ".store.book[0]", "testdata/store.json"},
			golden:   "testdata/book-0.json",
		}, {
			caseName: "-J=false after -Y keeps yaml",
			args:     []string{"-Y", "-J=false", ".store.book[0]", "testdata/store.json"},
			golden:   "testdata/book-0.yaml",
		}, {
			caseName: "-Y=false after -Y keeps yaml",
			args:     []string{"-Y", "-Y=false", ".store.book[0]", "testdata/store.json"},
			golden:   "testdata/book-0.yaml",
		}, {
			caseName: "--input-json=false keeps auto-detect",
			args:     []string{"--input-json=false", ".store.book[0]", "testdata/store.yaml"},
			golden:   "testdata/book-0.yaml",
		}, {
			caseName: "invalid alias value",
			args:     []string{"-Y=nope", ".", "testdata/store.json"},
			errstr:   `invalid argument "nope" for "-Y, --output-yaml" flag: strconv.ParseBool: parsing "nope": invalid syntax`,
		}, {
			caseName: "range[1:3] books",
			args:     []string{".store.book[1:3]|", "testdata/store.json"},
			golden:   "testdata/book-1-3.json",
		}, {
			caseName: "select books using tags",
			args:     []string{".store.book[.tags[.name == \"genre\" and .value == \"fiction\"].count() > 0]|", "testdata/store.json"},
			golden:   "testdata/book-1-3.json",
		}, {
			caseName: "select books using tags omit operators",
			args:     []string{".store.book[.tags[.name == \"genre\" and .value == \"fiction\"]]|", "testdata/store.json"},
			golden:   "testdata/book-1-3.json",
		}, {
			caseName: "single quote",
			args:     []string{".store.book[.tags[.name == 'genre' and .value == 'fiction']]|", "testdata/store.json"},
			golden:   "testdata/book-1-3.json",
		}, {
			caseName: "expand books",
			stdin:    "testdata/store.json",
			args:     []string{"-x", ".store.book"},
			golden:   "testdata/book-x",
		}, {
			caseName: "slurp books",
			stdin:    "testdata/store.json",
			args:     []string{"-s", ".store.book[]"},
			golden:   "testdata/book-s",
		}, {
			caseName: "slurp books",
			stdin:    "testdata/book-x",
			args:     []string{"-s", "."},
			golden:   "testdata/book-s",
		}, {
			caseName: "expand books",
			stdin:    "testdata/book-s",
			args:     []string{"-x", "."},
			golden:   "testdata/book-x",
		}, {
			caseName: "template output",
			stdin:    "testdata/store.json",
			args: []string{
				"-t", "{{.title}},{{.author}},{{.category}},{{.price}}",
				".store.book[]",
			},
			golden: "testdata/book.csv",
		}, {
			caseName: "output json with color",
			stdin:    "testdata/store.json",
			args:     []string{"-c", "."},
			golden:   "testdata/store-color.json",
		}, {
			caseName: "output yaml with color",
			stdin:    "testdata/store.yaml",
			args:     []string{"-c", "."},
			golden:   "testdata/store-color.yaml",
		}, {
			caseName: "edit",
			stdin:    "testdata/empty-object.json",
			args: []string{
				"-e", `.author = "Nigel Rees"`,
				"-e", `.category = "reference"`,
				"-e", `.price = 8.95`,
				"-e", `.title = "Sayings of the Century"`,
				"-e", `.tags = []`,
				"-e", `.tags += {"name": "genre", "value": "reference"}`,
				"-e", `.tags += {"name": "era", "value": "20th century"}`,
				"-e", `.tags += {"name": "theme", "value": "quotations"}`,
			},
			golden: "testdata/book-0.json",
		}, {
			caseName: "walk null",
			stdin:    "testdata/null",
			args:     []string{"..walk"},
		}, {
			caseName: "null document prints null",
			stdin:    "testdata/null",
			args:     []string{"."},
			golden:   "testdata/null",
		}, {
			caseName: "missing key prints null",
			args:     []string{".store.zz", "testdata/store.json"},
			golden:   "testdata/null",
		}, {
			caseName: "raw missing key prints null",
			args:     []string{"-r", ".store.zz", "testdata/store.json"},
			golden:   "testdata/null",
		}, {
			caseName: "raw null elements print null",
			stdin:    "testdata/null-element.json",
			args:     []string{"-r", ".a[]"},
			want:     "1\nnull\n",
		}, {
			caseName: "expand an array grown by an edit",
			stdin:    "testdata/empty-object.json",
			args:     []string{"-e", ".a[2] = 1", "-x", ".a"},
			want:     "null\nnull\n1\n",
		}, {
			caseName: "raw expand an array grown by an edit",
			stdin:    "testdata/empty-object.json",
			args:     []string{"-r", "-e", ".a[2] = 1", "-x", ".a"},
			want:     "null\nnull\n1\n",
		}, {
			caseName: "raw null in yaml prints null without separators",
			stdin:    "testdata/null-name.yaml",
			args:     []string{"-r", ".items[].name"},
			want:     "a\nnull\nb\nnull\n",
		}, {
			caseName: "raw null with yaml output prints null",
			stdin:    "testdata/null-element.json",
			args:     []string{"-r", "-Y", ".a[]"},
			want:     "1\nnull\n",
		}, {
			caseName: "template on a missing key renders no value",
			stdin:    "testdata/empty-object.json",
			args:     []string{"-x", "-t", "{{.id}}: {{.name}}", ".users"},
			want:     "<no value>: <no value>\n",
		}, {
			caseName: "template on null elements renders no value",
			stdin:    "testdata/null-object-element.json",
			args:     []string{"-t", "{{.id}}", ".a[]"},
			want:     "<no value>\n1\n",
		}, {
			caseName: "template with skips null",
			stdin:    "testdata/null-element.json",
			args:     []string{"-t", "{{with .}}{{.}}{{else}}-{{end}}", ".a[]"},
			want:     "1\n-\n",
		}, {
			caseName: "null literal selects missing keys",
			args:     []string{".store.book[.isbn == null].title", "testdata/store.json"},
			want:     "\"Sayings of the Century\"\n\"Sword of Honour\"\n",
		}, {
			caseName: "not equal keeps elements without the key",
			args:     []string{".store.book[.isbn != \"0-553-21311-3\"].title", "testdata/store.json"},
			want:     "\"Sayings of the Century\"\n\"Sword of Honour\"\n\"The Lord of the Rings\"\n",
		}, {
			caseName: "invalid json",
			args:     []string{"-i", "json", ".", "testdata/invalid-json"},
			errstr:   `failed to evaluate testdata/invalid-json: invalid character 'i' looking for beginning of value`,
		}, {
			caseName: "invalid json",
			stdin:    "testdata/invalid-json",
			args:     []string{"-i", "json", "."},
			errstr:   `failed to evaluate STDIN: invalid character 'i' looking for beginning of value`,
		}, {
			caseName: "invalid yaml",
			args:     []string{"-i", "yaml", ".", "testdata/invalid-yaml"},
			errstr:   `failed to evaluate testdata/invalid-yaml: yaml: found unexpected end of stream`,
		}, {
			caseName: "yaml starting with a json scalar is not duplicated",
			stdin:    "testdata/scalar-then-map.yaml",
			args:     []string{"."},
			want:     "1\n---\nfoo: bar\n",
		}, {
			caseName: "yaml with numeric keys is not duplicated",
			args:     []string{".", "testdata/numeric-keys.yaml"},
			golden:   "testdata/numeric-keys-out.yaml",
		}, {
			caseName: "json stream broken after two documents is not read as yaml",
			args:     []string{".a", "testdata/broken-json-stream"},
			want:     "1\n2\n",
			errstr:   `failed to evaluate testdata/broken-json-stream: invalid character 'o' in literal false (expecting 'a')`,
		}, {
			caseName: "malformed json object is not read as yaml",
			args:     []string{".", "testdata/invalid-json-object"},
			errstr:   `failed to evaluate testdata/invalid-json-object: invalid character '}' in literal null (expecting 'l')`,
		}, {
			caseName: "indented malformed json object is not read as yaml",
			args:     []string{".", "testdata/invalid-json-object-indented"},
			errstr:   `failed to evaluate testdata/invalid-json-object-indented: invalid character '}' in literal null (expecting 'l')`,
		}, {
			caseName: "json stream starting with a scalar is not read as yaml",
			args:     []string{".", "testdata/scalar-first-broken-json"},
			want:     "\"x\"\n",
			errstr:   `failed to evaluate testdata/scalar-first-broken-json: invalid character '}' in literal null (expecting 'l')`,
		}, {
			caseName: "yaml scalar folded into a flow sequence without -y fails as json",
			args:     []string{".", "testdata/folded-flow.yaml"},
			want:     "1\n",
			errstr:   `failed to evaluate testdata/folded-flow.yaml: invalid character 'a' looking for beginning of value`,
		}, {
			caseName: "yaml scalar folded into a flow sequence with -y",
			args:     []string{"-y", ".", "testdata/folded-flow.yaml"},
			want:     "1 [a]\n",
		}, {
			caseName: "json documents separated by --- without -y fail as json",
			args:     []string{".", "testdata/json-then-yaml-separator"},
			want:     "{\n  \"a\": 1\n}\n",
			errstr:   `failed to evaluate testdata/json-then-yaml-separator: invalid character '-' in numeric literal`,
		}, {
			caseName: "json documents separated by --- with -y",
			args:     []string{"-y", ".", "testdata/json-then-yaml-separator"},
			want:     "a: 1\n---\na: 2\n",
		}, {
			caseName: "yaml flow sequence without -y fails as json",
			args:     []string{".", "testdata/flow-sequence.yaml"},
			errstr:   `failed to evaluate testdata/flow-sequence.yaml: invalid character 'a' looking for beginning of value`,
		}, {
			caseName: "yaml flow sequence with -y",
			args:     []string{"-y", ".", "testdata/flow-sequence.yaml"},
			want:     "- a\n- b\n",
		}, {
			caseName: "yaml flow mapping without -y fails as json",
			args:     []string{".", "testdata/flow-mapping.yaml"},
			errstr:   `failed to evaluate testdata/flow-mapping.yaml: invalid character 'b' looking for beginning of value`,
		}, {
			caseName: "yaml flow mapping with -y",
			args:     []string{"-y", ".", "testdata/flow-mapping.yaml"},
			want:     "a: b\n",
		}, {
			caseName: "stray closing brace is an error under -j",
			args:     []string{"-j", ".", "testdata/stray-brace-json"},
			want:     "{\n  \"a\": 1\n}\n",
			errstr:   "failed to evaluate testdata/stray-brace-json: invalid character '}' looking for beginning of value",
		}, {
			caseName: "json rejected only by the number parser is not read as yaml",
			args:     []string{".", "testdata/json-number-overflow"},
			errstr:   "failed to evaluate testdata/json-number-overflow: json: cannot unmarshal number 1e400 into Go value of type float64",
		}, {
			caseName: "out-of-range number before a json array is not read as yaml",
			args:     []string{".", "testdata/number-overflow-stream"},
			errstr:   "failed to evaluate testdata/number-overflow-stream: json: cannot unmarshal number 1e400 into Go value of type float64",
		}, {
			caseName: "out-of-range number before a json array with -y",
			args:     []string{"-y", ".", "testdata/number-overflow-stream"},
			want:     "1e400 [1]\n",
		}, {
			caseName: "malformed json object from stdin is not read as yaml",
			stdin:    "testdata/invalid-json-object",
			args:     []string{"."},
			errstr:   `failed to evaluate STDIN: invalid character '}' in literal null (expecting 'l')`,
		}, {
			caseName: "truncated json container is not read as yaml",
			args:     []string{".", "testdata/truncated-json"},
			errstr:   "failed to evaluate testdata/truncated-json: unexpected EOF",
		}, {
			caseName: "truncated json container with -j",
			args:     []string{"-j", ".", "testdata/truncated-json"},
			errstr:   "failed to evaluate testdata/truncated-json: unexpected EOF",
		}, {
			caseName: "json container with utf-8 bom is auto-detected as json",
			args:     []string{".", "testdata/bom.json"},
			want:     "{\n  \"a\": 1\n}\n",
		}, {
			caseName: "json container with utf-8 bom with -j",
			args:     []string{"-j", ".", "testdata/bom.json"},
			want:     "{\n  \"a\": 1\n}\n",
		}, {
			caseName: "json container with utf-8 bom with -y",
			args:     []string{"-y", ".", "testdata/bom.json"},
			want:     "a: 1\n",
		}, {
			caseName: "yaml document with utf-8 bom is auto-detected as yaml",
			args:     []string{".", "testdata/bom.yaml"},
			want:     "a: 1\n",
		}, {
			caseName: "truncated json container with utf-8 bom is not read as yaml",
			args:     []string{".", "testdata/truncated-bom-json"},
			errstr:   "failed to evaluate testdata/truncated-bom-json: unexpected EOF",
		}, {
			caseName: "non-seekable json file with utf-8 bom is auto-detected",
			pipe:     "testdata/bom.json",
			args:     []string{"."},
			want:     "{\n  \"a\": 1\n}\n",
		}, {
			caseName: "non-seekable json file with utf-8 bom with -j",
			pipe:     "testdata/bom.json",
			args:     []string{"-j", "."},
			want:     "{\n  \"a\": 1\n}\n",
		}, {
			caseName: "non-seekable json file with utf-8 bom with -y",
			pipe:     "testdata/bom.json",
			args:     []string{"-y", "."},
			want:     "a: 1\n",
		}, {
			caseName: "yaml folding json scalars across lines is not read as json",
			stdin:    "testdata/folded-scalars.yaml",
			args:     []string{"."},
			want:     "1 2\n---\nx: 1\n",
		}, {
			caseName: "query error on yaml input is not joined with the json attempt",
			args:     []string{".a[", "testdata/store.yaml"},
			errstr:   `failed to evaluate testdata/store.yaml: syntax error: no right brackets: ".a["`,
		}, {
			caseName: "multiple yaml",
			args:     []string{".", "testdata/book-0.yaml", "testdata/book-0.yaml"},
			want:     mustReadFileString("testdata/book-0.yaml") + "---\n" + mustReadFileString("testdata/book-0.yaml"),
		}, {
			caseName: "non-seekable json file is auto-detected",
			pipe:     "testdata/store.json",
			args:     []string{".store.book[0]"},
			golden:   "testdata/book-0.json",
		}, {
			caseName: "non-seekable yaml file is auto-detected",
			pipe:     "testdata/store.yaml",
			args:     []string{".store.book[0]"},
			golden:   "testdata/book-0.yaml",
		}, {
			caseName: "non-seekable json file with -j",
			pipe:     "testdata/store.json",
			args:     []string{"-j", ".store.book[0]"},
			golden:   "testdata/book-0.json",
		}, {
			caseName: "non-seekable yaml file with -y",
			pipe:     "testdata/store.yaml",
			args:     []string{"-y", ".store.book[0]"},
			golden:   "testdata/book-0.yaml",
		}, {
			caseName: "non-seekable file with -U is rejected",
			pipe:     "testdata/store.json",
			args:     []string{"-U", ".store.book[0]"},
			errstr:   "cannot update {pipe} in place: not a regular file",
		}, {
			caseName: "non-seekable file with -U -j is rejected",
			pipe:     "testdata/store.json",
			args:     []string{"-U", "-j", ".store.book[0]"},
			errstr:   "cannot update {pipe} in place: not a regular file",
		}, {
			caseName: "non-seekable file with -s is slurped",
			pipe:     "testdata/store.json",
			args:     []string{"-s", ".store.book[]"},
			golden:   "testdata/book-s",
		}, {
			caseName: "-U with -s on stdin is rejected",
			stdin:    "testdata/store.json",
			args:     []string{"-U", "-s", ".store.book[]"},
			errstr:   "cannot combine -U with -s",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.caseName, func(t *testing.T) {
			args, errstr := tc.args, tc.errstr
			if tc.pipe != "" {
				pipe := pipePath(t, tc.pipe)
				args = append(slices.Clone(args), pipe)
				errstr = strings.ReplaceAll(errstr, "{pipe}", pipe)
			}
			if tc.stdin != "" {
				in, err := os.Open(tc.stdin)
				if err != nil {
					t.Fatal(err)
				}
				defer func() { _ = in.Close() }()

				os.Stdin = in
			}

			buf := new(bytes.Buffer)
			r := &runner{
				stderr: io2.NopWriteCloser(buf),
				out:    io2.NopWriteCloser(buf),
			}
			defer r.close()

			err := r.run(append([]string{"tq"}, args...))
			if errstr != "" {
				if err == nil {
					t.Fatal("no error")
				}
				if err.Error() != errstr {
					t.Errorf(`error %s; want %s`, err.Error(), errstr)
				}
			} else if err != nil {
				t.Fatal(err)
			}

			got := buf.String()
			if tc.golden != "" {
				if *updateGolden {
					if err := os.WriteFile(tc.golden, []byte(got), 0o644); err != nil {
						t.Fatalf("update golden %s: %v", tc.golden, err)
					}
					return
				}
				want := mustReadFileString(tc.golden)
				if got != want {
					t.Errorf("golden %s mismatch\ngot:  %q\nwant: %q", tc.golden, got, want)
				}
				return
			}
			if got != tc.want {
				t.Errorf("got %s; want %s", got, tc.want)
			}
		})
	}
}

// pipePath feeds the contents of src through a pipe and returns its /dev/fd path.
func pipePath(t *testing.T, src string) string {
	t.Helper()
	if _, err := os.Stat("/dev/fd"); err != nil {
		t.Skip("/dev/fd is not available")
	}
	bin, err := os.ReadFile(src)
	if err != nil {
		t.Fatal(err)
	}
	pr, pw, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = pr.Close() })
	go func() {
		_, _ = pw.Write(bin)
		_ = pw.Close()
	}()
	return fmt.Sprintf("/dev/fd/%d", pr.Fd())
}

// TestInputFilesNextReader checks that only non-seekable files that must be rewound are spooled.
func TestInputFilesNextReader(t *testing.T) {
	stdinOrg := os.Stdin
	defer func() { os.Stdin = stdinOrg }()

	testCases := []struct {
		caseName  string
		pipe      bool
		stdin     bool
		rewind    bool
		wantSpool bool
	}{
		{caseName: "pipe without rewind is read directly", pipe: true},
		{caseName: "pipe with rewind is spooled", pipe: true, rewind: true, wantSpool: true},
		{caseName: "regular file with rewind is read directly", rewind: true},
		{caseName: "regular file as stdin with rewind is read directly", stdin: true, rewind: true},
	}
	for _, tc := range testCases {
		t.Run(tc.caseName, func(t *testing.T) {
			filename := "testdata/store.json"
			if tc.pipe {
				filename = pipePath(t, filename)
			}
			if tc.stdin {
				file, err := os.Open(filename)
				if err != nil {
					t.Fatal(err)
				}
				defer func() { _ = file.Close() }()

				os.Stdin = file
				filename = filenameStdin
			}
			in, err := newInputFiles([]string{filename}, tc.rewind).nextReader()
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = in.Close() }()

			if tc.stdin {
				// Spooling would have read stdin to EOF already.
				if off, err := os.Stdin.Seek(0, io.SeekCurrent); err != nil || off != 0 {
					t.Errorf("stdin offset %d, %v; want 0", off, err)
				}
				_ = in.Close()
				if _, err := os.Stdin.Seek(0, io.SeekCurrent); err != nil {
					t.Errorf("stdin after Close: %v", err)
				}
			} else if _, isFile := in.(*os.File); isFile == tc.wantSpool {
				t.Errorf("spooled: got %v, want %v", !isFile, tc.wantSpool)
			}
			if !tc.rewind {
				return
			}
			if _, err := io.ReadAll(in); err != nil {
				t.Fatal(err)
			}
			if _, err := in.Seek(0, io.SeekStart); err != nil {
				t.Errorf("Seek: %v", err)
			}
		})
	}
}

// TestInputFilesNextReaderStdin checks that stdin is read without waiting for EOF unless it must be rewound.
func TestInputFilesNextReaderStdin(t *testing.T) {
	stdinOrg := os.Stdin
	defer func() { os.Stdin = stdinOrg }()

	testCases := []struct {
		caseName string
		rewind   bool
	}{
		{caseName: "stdin without rewind is read directly"},
		{caseName: "stdin with rewind is spooled", rewind: true},
	}
	for _, tc := range testCases {
		t.Run(tc.caseName, func(t *testing.T) {
			pr, pw, err := os.Pipe()
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = pr.Close() }()
			defer func() { _ = pw.Close() }()

			os.Stdin = pr
			const doc = `{"a":1}`
			if _, err := pw.WriteString(doc); err != nil {
				t.Fatal(err)
			}
			type result struct {
				in  io.ReadSeekCloser
				err error
			}
			ch := make(chan result, 1)
			go func() {
				in, err := newInputFiles([]string{filenameStdin}, tc.rewind).nextReader()
				ch <- result{in, err}
			}()
			var res result
			if tc.rewind {
				select {
				case res = <-ch:
					t.Error("nextReader returned before EOF")
				case <-time.After(100 * time.Millisecond):
					_ = pw.Close()
					select {
					case res = <-ch:
					case <-time.After(5 * time.Second):
						t.Fatal("nextReader did not return after EOF")
					}
				}
			} else {
				select {
				case res = <-ch:
				case <-time.After(5 * time.Second):
					t.Fatal("nextReader waited for EOF")
				}
			}
			if res.err != nil {
				t.Fatal(res.err)
			}
			// A pipe returned as is cannot seek; a spooled temp file can.
			if _, err := res.in.Seek(0, io.SeekCurrent); tc.rewind && err != nil {
				t.Errorf("stdin with rewind is not spooled: %v", err)
			}
			got := make([]byte, len(doc))
			if _, err := io.ReadFull(res.in, got); err != nil || string(got) != doc {
				t.Fatalf("read %q, %v; want %q", got, err, doc)
			}
			_ = res.in.Close()
			if tc.rewind {
				return
			}
			// Close must not close os.Stdin, which tq does not own.
			if _, err := pw.WriteString("x"); err != nil {
				t.Fatal(err)
			}
			if _, err := pr.Read(make([]byte, 1)); err != nil {
				t.Errorf("stdin after Close: %v", err)
			}
		})
	}
}

// TestNewRunnerDefaults ensures newRunner() wires stderr/out to the
// process streams. Previously this constructor was uncovered.
func TestNewRunnerDefaults(t *testing.T) {
	r := newRunner()
	if r == nil {
		t.Fatal("newRunner returned nil")
	}
	if r.stderr != os.Stderr {
		t.Errorf("stderr not os.Stderr")
	}
	if r.out == nil {
		t.Errorf("out is nil")
	}
}

// newTestRunner builds a runner that buffers its output/error streams
// so tests can assert on them without touching real FDs.
func newTestRunner(buf *bytes.Buffer) *runner {
	return &runner{
		stderr: io2.NopWriteCloser(buf),
		out:    io2.NopWriteCloser(buf),
	}
}

// TestRunOutputFile verifies the `-O` flag writes results to a file
// instead of stdout, covering the os.Create path in (*runner).run.
func TestRunOutputFile(t *testing.T) {
	testCases := []struct {
		caseName string
		args     []string
	}{
		{
			caseName: "regular file",
			args:     []string{"tq", "-O", "{out}", ".store.book[0]", "testdata/store.json"},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.caseName, func(t *testing.T) {
			outPath := filepath.Join(t.TempDir(), "out.json")
			args := make([]string, len(tc.args))
			for i, a := range tc.args {
				args[i] = strings.ReplaceAll(a, "{out}", outPath)
			}

			buf := new(bytes.Buffer)
			r := newTestRunner(buf)
			defer r.close()

			if err := r.run(args); err != nil {
				t.Fatalf("run: %v", err)
			}
			got, err := os.ReadFile(outPath)
			if err != nil {
				t.Fatalf("ReadFile: %v", err)
			}
			want, err := os.ReadFile("testdata/book-0.json")
			if err != nil {
				t.Fatalf("ReadFile want: %v", err)
			}
			if string(got) != string(want) {
				t.Errorf("got %q\nwant %q", got, want)
			}
		})
	}
}

// TestRunInplace verifies the `-U` flag writes results back to the
// input file, exercising the evaluateInputFiles inplace branch.
func TestRunInplace(t *testing.T) {
	testCases := []struct {
		caseName   string
		src        string
		args       []string
		golden     string
		errstr     string
		twoFiles   bool
		stdin      string
		wantStdout string
		mode       os.FileMode
		symlink    bool
		// wantErr is checked with errors.Is, for errors whose text holds a temp or missing path.
		wantErr     error
		readonlyDir bool
		missing     bool
		longName    bool
		// unreadableSecond makes the second of twoFiles unreadable.
		unreadableSecond bool
	}{
		{
			caseName: "json in place",
			src:      "testdata/store.json",
			args:     []string{"-U", ".store.book[0]"},
			golden:   "testdata/book-0.json",
		}, {
			caseName: "yaml to json in place with -J",
			src:      "testdata/store.yaml",
			args:     []string{"-U", "-J", ".store.book[0]"},
			golden:   "testdata/book-0.json",
		}, {
			caseName: "json to yaml in place with -Y",
			src:      "testdata/store.json",
			args:     []string{"-U", "-Y", ".store.book[0]"},
			golden:   "testdata/book-0.yaml",
		}, {
			caseName: "missing key in place writes null",
			src:      "testdata/store.json",
			args:     []string{"-U", ".zz"},
			golden:   "testdata/null",
		}, {
			caseName: "yaml with numeric keys in place is not duplicated",
			src:      "testdata/numeric-keys.yaml",
			args:     []string{"-U", "."},
			golden:   "testdata/numeric-keys-out.yaml",
		}, {
			caseName: "broken json stream in place leaves the file untouched",
			src:      "testdata/broken-json-stream",
			args:     []string{"-U", ".a"},
			golden:   "testdata/broken-json-stream",
			errstr:   "failed to evaluate {target}: invalid character 'o' in literal false (expecting 'a')",
		}, {
			caseName: "multi-document yaml in place keeps its documents",
			src:      "testdata/multi-doc.yaml",
			args:     []string{"-U", "."},
			golden:   "testdata/multi-doc.yaml",
		}, {
			caseName: "json stream to yaml in place with -Y keeps its documents",
			src:      "testdata/multi-doc-json",
			args:     []string{"-U", "-Y", "."},
			golden:   "testdata/multi-doc.yaml",
		}, {
			caseName: "expanded results in place are separate documents",
			src:      "testdata/store.yaml",
			args:     []string{"-U", "-x", ".store.book"},
			golden:   "testdata/book-x.yaml",
		}, {
			caseName: "second file in place does not start with a separator",
			src:      "testdata/multi-doc.yaml",
			args:     []string{"-U", "."},
			golden:   "testdata/multi-doc.yaml",
			twoFiles: true,
		}, {
			caseName:   "stdin after an in-place file goes to stdout",
			src:        "testdata/multi-doc-json",
			args:       []string{"-U", "-Y", "."},
			golden:     "testdata/multi-doc.yaml",
			stdin:      "testdata/multi-doc.yaml",
			wantStdout: "a: 1\n---\na: 2\n",
		}, {
			caseName: "file mode is kept",
			src:      "testdata/store.json",
			args:     []string{"-U", ".store.book[0]"},
			golden:   "testdata/book-0.json",
			mode:     0o640,
		}, {
			caseName: "setuid bit is dropped",
			src:      "testdata/store.json",
			args:     []string{"-U", ".store.book[0]"},
			golden:   "testdata/book-0.json",
			mode:     os.ModeSetuid | 0o640,
		}, {
			caseName: "symlink is kept and its file is updated",
			src:      "testdata/store.json",
			args:     []string{"-U", ".store.book[0]"},
			golden:   "testdata/book-0.json",
			symlink:  true,
		}, {
			caseName:    "unwritable directory leaves the file untouched",
			src:         "testdata/store.json",
			args:        []string{"-U", ".store.book[0]"},
			golden:      "testdata/store.json",
			wantErr:     fs.ErrPermission,
			readonlyDir: true,
		}, {
			caseName: "missing second target updates nothing",
			src:      "testdata/store.json",
			args:     []string{"-U", ".store.book[0]"},
			golden:   "testdata/store.json",
			wantErr:  fs.ErrNotExist,
			missing:  true,
		}, {
			caseName: "target with a name near NAME_MAX",
			src:      "testdata/store.json",
			args:     []string{"-U", ".store.book[0]"},
			golden:   "testdata/book-0.json",
			longName: true,
		}, {
			caseName:         "unreadable second target updates nothing",
			src:              "testdata/store.json",
			args:             []string{"-U", ".store.book[0]"},
			golden:           "testdata/store.json",
			twoFiles:         true,
			wantErr:          fs.ErrPermission,
			unreadableSecond: true,
		}, {
			caseName: "-U with -s is rejected",
			src:      "testdata/store.json",
			args:     []string{"-U", "-s", ".store.book[]"},
			golden:   "testdata/store.json",
			errstr:   "cannot combine -U with -s",
		}, {
			caseName: "-U with -O is rejected",
			src:      "testdata/store.json",
			args:     []string{"-U", "-O", os.DevNull, ".store.book[0]"},
			golden:   "testdata/store.json",
			errstr:   "cannot combine -U with -O",
		}, {
			caseName: "-U with -c is rejected",
			src:      "testdata/store.json",
			args:     []string{"-U", "-c", ".store.book[0]"},
			golden:   "testdata/store.json",
			errstr:   "cannot combine -U with -c",
		},
	}
	stdinOrg := os.Stdin
	defer func() { os.Stdin = stdinOrg }()

	for _, tc := range testCases {
		t.Run(tc.caseName, func(t *testing.T) {
			dir := t.TempDir()
			target := filepath.Join(dir, filepath.Base(tc.src))
			if tc.longName {
				target = filepath.Join(dir, strings.Repeat("a", 245)+".json")
			}
			targets := []string{target}
			if tc.twoFiles {
				targets = append(targets, filepath.Join(dir, "second-"+filepath.Base(tc.src)))
			}
			src, err := os.ReadFile(tc.src)
			if err != nil {
				t.Fatalf("ReadFile src: %v", err)
			}
			mode := tc.mode
			if mode == 0 {
				mode = 0o644
			}
			for _, f := range targets {
				if err := os.WriteFile(f, src, mode.Perm()); err != nil {
					t.Fatalf("WriteFile: %v", err)
				}
				// WriteFile's perm is masked by the umask.
				if err := os.Chmod(f, mode); err != nil {
					t.Fatalf("Chmod: %v", err)
				}
			}
			// The OS may drop bits it does not grant, so want what Chmod produced.
			fi, err := os.Stat(target)
			if err != nil {
				t.Fatalf("Stat: %v", err)
			}
			wantMode := fi.Mode().Perm()
			wantNames := []string{filepath.Base(target)}
			if tc.twoFiles {
				wantNames = append(wantNames, "second-"+filepath.Base(tc.src))
			}
			args := slices.Concat([]string{"tq"}, tc.args, targets)
			if tc.symlink {
				link := filepath.Join(dir, "link")
				if err := os.Symlink(filepath.Base(target), link); err != nil {
					t.Fatalf("Symlink: %v", err)
				}
				wantNames = append(wantNames, "link")
				args = slices.Concat([]string{"tq"}, tc.args, []string{link})
			}
			if tc.missing {
				args = append(args, filepath.Join(dir, "missing.json"))
			}
			if tc.readonlyDir || tc.unreadableSecond {
				if os.Geteuid() == 0 {
					t.Skip("root ignores permissions")
				}
			}
			if tc.unreadableSecond {
				if err := os.Chmod(targets[1], 0); err != nil {
					t.Fatalf("Chmod: %v", err)
				}
			}
			if tc.readonlyDir {
				if err := os.Chmod(dir, 0o555); err != nil {
					t.Fatalf("Chmod dir: %v", err)
				}
				t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })
			}

			buf := new(bytes.Buffer)
			r := newTestRunner(buf)
			defer r.close()

			if tc.stdin != "" {
				in, err := os.Open(tc.stdin)
				if err != nil {
					t.Fatal(err)
				}
				defer func() { _ = in.Close() }()

				os.Stdin = in
				args = append(args, "-")
			}
			err = r.run(args)
			if tc.unreadableSecond {
				if err := os.Chmod(targets[1], wantMode); err != nil {
					t.Fatalf("Chmod: %v", err)
				}
			}
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("got error %v; want %v", err, tc.wantErr)
				}
				if tc.readonlyDir && !strings.HasPrefix(err.Error(), "failed to write "+target+": ") {
					t.Errorf("got error %v; want it to wrap the target", err)
				}
			} else if tc.errstr != "" {
				want := strings.ReplaceAll(tc.errstr, "{target}", target)
				if err == nil || err.Error() != want {
					t.Fatalf("got error %v; want %s", err, want)
				}
			} else if err != nil {
				t.Fatalf("run: %v", err)
			} else if buf.String() != tc.wantStdout {
				t.Errorf("stdout: got %q; want %q", buf.String(), tc.wantStdout)
			}
			want, err := os.ReadFile(tc.golden)
			if err != nil {
				t.Fatalf("ReadFile want: %v", err)
			}
			for _, f := range targets {
				got, err := os.ReadFile(f)
				if err != nil {
					t.Fatalf("ReadFile target: %v", err)
				}
				if string(got) != string(want) {
					t.Errorf("%s: got %q\nwant %q", filepath.Base(f), got, want)
				}
				fi, err := os.Stat(f)
				if err != nil {
					t.Fatalf("Stat target: %v", err)
				}
				if got := fi.Mode() &^ os.ModeType; got != wantMode {
					t.Errorf("%s: mode %v; want %v", filepath.Base(f), got, wantMode)
				}
			}
			if tc.symlink {
				fi, err := os.Lstat(filepath.Join(dir, "link"))
				if err != nil {
					t.Fatalf("Lstat link: %v", err)
				}
				if fi.Mode()&os.ModeSymlink == 0 {
					t.Errorf("link is replaced with %v", fi.Mode())
				}
			}
			entries, err := os.ReadDir(dir)
			if err != nil {
				t.Fatalf("ReadDir: %v", err)
			}
			var names []string
			for _, e := range entries {
				names = append(names, e.Name())
			}
			slices.Sort(wantNames)
			if !slices.Equal(names, wantNames) {
				t.Errorf("files %v; want %v", names, wantNames)
			}
		})
	}
}

// TestInplaceFileWrite checks that a failed write to the temp file is recorded for the write-back error.
func TestInplaceFileWrite(t *testing.T) {
	path := filepath.Join(t.TempDir(), "readonly")
	if err := os.WriteFile(path, nil, 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	f := &inplaceFile{file: file, target: path}
	defer f.discard()

	_, err = f.Write([]byte("x"))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if f.writeErr != err {
		t.Errorf("writeErr: got %v, want %v", f.writeErr, err)
	}
	if _, err := f.Write([]byte("y")); err == nil || f.writeErr == err {
		t.Errorf("writeErr is not the first error: %v", f.writeErr)
	}
}

// TestEvaluateInputFilesInplaceErrorRestoresOutput checks that a failed in-place file restores the output and separator state.
func TestEvaluateInputFilesInplaceErrorRestoresOutput(t *testing.T) {
	src, err := os.ReadFile("testdata/invalid-yaml")
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	path := filepath.Join(t.TempDir(), "invalid-yaml")
	if err := os.WriteFile(path, src, 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	buf := new(bytes.Buffer)
	r := newTestRunner(buf)
	if err := r.initFlagSet([]string{"tq", "-U", "."}); err != nil {
		t.Fatalf("initFlagSet: %v", err)
	}
	r.outputYAMLCalled = 2
	if err := r.evaluateInputFiles(newInputFiles([]string{path}, false)); err == nil {
		t.Fatal("expected error, got nil")
	}
	if r.outputYAMLCalled != 2 {
		t.Errorf("outputYAMLCalled: got %d, want 2", r.outputYAMLCalled)
	}
	if _, err := r.out.Write([]byte("x")); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if got := buf.String(); got != "x" {
		t.Errorf("output not restored: got %q, want %q", got, "x")
	}
}

// TestRunRawString covers the `-r` flag path in (*runner).output which
// prints string values without JSON quoting.
func TestRunRawString(t *testing.T) {
	buf := new(bytes.Buffer)
	r := newTestRunner(buf)
	defer r.close()

	if err := r.run([]string{"tq", "-r", ".store.book[0].author", "testdata/store.json"}); err != nil {
		t.Fatalf("run: %v", err)
	}
	if got, want := buf.String(), "Nigel Rees\n"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// TestRunErrors covers error paths in (*runner).run where we only care
// that an error is returned: missing input file, bad -t template, and
// an -O path whose parent directory does not exist.
func TestRunErrors(t *testing.T) {
	missingOut := filepath.Join(t.TempDir(), "nope", "out.json")

	testCases := []struct {
		caseName string
		args     []string
	}{
		{
			caseName: "file not found",
			args:     []string{"tq", ".", "testdata/does-not-exist.json"},
		},
		{
			caseName: "bad template",
			args:     []string{"tq", "-t", "{{.unclosed", ".", "testdata/store.json"},
		},
		{
			caseName: "output file create error",
			args:     []string{"tq", "-O", missingOut, ".", "testdata/store.json"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.caseName, func(t *testing.T) {
			buf := new(bytes.Buffer)
			r := newTestRunner(buf)
			defer r.close()

			if err := r.run(tc.args); err == nil {
				t.Fatalf("expected error, got nil")
			}
		})
	}
}

func TestSkipBOM(t *testing.T) {
	testCases := []struct {
		caseName   string
		input      []byte
		consumed   int
		wantOffset int64
		wantOutput []byte
	}{
		{
			caseName:   "empty",
			input:      []byte{},
			wantOffset: 0,
			wantOutput: []byte{},
		},
		{
			caseName:   "1-byte",
			input:      []byte("{"),
			wantOffset: 0,
			wantOutput: []byte("{"),
		},
		{
			caseName:   "2-byte",
			input:      []byte("{}"),
			wantOffset: 0,
			wantOutput: []byte("{}"),
		},
		{
			caseName:   "3-byte non-BOM",
			input:      []byte("foo"),
			wantOffset: 0,
			wantOutput: []byte("foo"),
		},
		{
			caseName:   "UTF-8 BOM",
			input:      append([]byte{0xEF, 0xBB, 0xBF}, []byte(`{"a":1}`)...),
			wantOffset: 3,
			wantOutput: []byte(`{"a":1}`),
		},
		{
			caseName:   "double UTF-8 BOM skips only first BOM",
			input:      append([]byte{0xEF, 0xBB, 0xBF, 0xEF, 0xBB, 0xBF}, []byte(`{"a":1}`)...),
			wantOffset: 3,
			wantOutput: append([]byte{0xEF, 0xBB, 0xBF}, []byte(`{"a":1}`)...),
		},
		{
			caseName:   "non-BOM input keeps its current position",
			input:      []byte(`xx{"a":1}`),
			consumed:   2,
			wantOffset: 0,
			wantOutput: []byte(`{"a":1}`),
		},
		{
			caseName:   "UTF-8 BOM after consumed bytes",
			input:      append([]byte("xx\xEF\xBB\xBF"), []byte(`{"a":1}`)...),
			consumed:   2,
			wantOffset: 3,
			wantOutput: []byte(`{"a":1}`),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.caseName, func(t *testing.T) {
			in := bytes.NewReader(tc.input)
			if _, err := in.Seek(int64(tc.consumed), io.SeekStart); err != nil {
				t.Fatal(err)
			}

			reader, offset, err := skipBOM(in)
			if err != nil {
				t.Fatalf("skipBOM: unexpected error %v", err)
			}
			if offset != tc.wantOffset {
				t.Errorf("offset = %d, want %d", offset, tc.wantOffset)
			}
			got, err := io.ReadAll(reader)
			if err != nil {
				t.Fatalf("ReadAll: unexpected error %v", err)
			}
			if !bytes.Equal(got, tc.wantOutput) {
				t.Errorf("got %q, want %q", got, tc.wantOutput)
			}
		})
	}
}

func TestEvaluateFromOffset(t *testing.T) {
	bom := "\xEF\xBB\xBF"
	testCases := []struct {
		caseName string
		args     []string
		input    string
		consumed int
		want     string
	}{
		{
			caseName: "auto-detected json after consumed bytes",
			args:     []string{"."},
			input:    "skip\n{\"a\":1}\n",
			consumed: 5,
			want:     "{\n  \"a\": 1\n}\n",
		},
		{
			caseName: "auto-detected json with utf-8 bom after consumed bytes",
			args:     []string{"."},
			input:    "skip\n" + bom + "{\"a\":1}\n",
			consumed: 5,
			want:     "{\n  \"a\": 1\n}\n",
		},
		{
			caseName: "auto-detected second json document",
			args:     []string{"."},
			input:    "{\"a\":1}\n{\"a\":2}\n",
			consumed: 8,
			want:     "{\n  \"a\": 2\n}\n",
		},
		{
			caseName: "json with -j after consumed bytes",
			args:     []string{"-j", "."},
			input:    "skip\n{\"a\":1}\n",
			consumed: 5,
			want:     "{\n  \"a\": 1\n}\n",
		},
		{
			caseName: "json with -y after consumed bytes",
			args:     []string{"-y", "."},
			input:    "skip\n{\"a\":1}\n",
			consumed: 5,
			want:     "a: 1\n",
		},
		{
			caseName: "auto-detected yaml after consumed bytes",
			args:     []string{"."},
			input:    "skip\nfoo: bar\n",
			consumed: 5,
			want:     "foo: bar\n",
		},
		{
			caseName: "json with -j and utf-8 bom after consumed bytes",
			args:     []string{"-j", "."},
			input:    "skip\n" + bom + "{\"a\":1}\n",
			consumed: 5,
			want:     "{\n  \"a\": 1\n}\n",
		},
		{
			caseName: "json with -y and utf-8 bom after consumed bytes",
			args:     []string{"-y", "."},
			input:    "skip\n" + bom + "{\"a\":1}\n",
			consumed: 5,
			want:     "a: 1\n",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.caseName, func(t *testing.T) {
			buf := new(bytes.Buffer)
			r := newTestRunner(buf)
			if err := r.initFlagSet(append([]string{"tq"}, tc.args...)); err != nil {
				t.Fatalf("initFlagSet: %v", err)
			}
			in := io2.NopReadSeekCloser(strings.NewReader(tc.input))
			if _, err := in.Seek(int64(tc.consumed), io.SeekStart); err != nil {
				t.Fatal(err)
			}

			if err := r.evaluate(in); err != nil {
				t.Fatalf("evaluate: %v", err)
			}
			if got := buf.String(); got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}
