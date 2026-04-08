package main

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"testing"

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
			caseName: "multiple yaml",
			args:     []string{".", "testdata/book-0.yaml", "testdata/book-0.yaml"},
			want:     mustReadFileString("testdata/book-0.yaml") + "---\n" + mustReadFileString("testdata/book-0.yaml"),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.caseName, func(t *testing.T) {
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

			err := r.run(append([]string{"tq"}, tc.args...))
			if tc.errstr != "" {
				if err == nil {
					t.Fatal("no error")
				}
				if err.Error() != tc.errstr {
					t.Errorf(`error %s; want %s`, err.Error(), tc.errstr)
				}
				return
			}
			if err != nil {
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
	dir := t.TempDir()
	outPath := filepath.Join(dir, "out.json")

	buf := new(bytes.Buffer)
	r := newTestRunner(buf)
	defer r.close()

	if err := r.run([]string{"tq", "-O", outPath, ".store.book[0]", "testdata/store.json"}); err != nil {
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
}

// TestRunInplace verifies the `-U` flag writes results back to the
// input file, exercising the evaluateInputFiles inplace branch.
func TestRunInplace(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "store.json")
	src, err := os.ReadFile("testdata/store.json")
	if err != nil {
		t.Fatalf("ReadFile src: %v", err)
	}
	if err := os.WriteFile(target, src, 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	buf := new(bytes.Buffer)
	r := newTestRunner(buf)
	defer r.close()

	if err := r.run([]string{"tq", "-U", ".store.book[0]", target}); err != nil {
		t.Fatalf("run: %v", err)
	}
	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("ReadFile target: %v", err)
	}
	want, err := os.ReadFile("testdata/book-0.json")
	if err != nil {
		t.Fatalf("ReadFile want: %v", err)
	}
	if string(got) != string(want) {
		t.Errorf("got %q\nwant %q", got, want)
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
