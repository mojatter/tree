package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
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
			caseName: "-o wins over -Y",
			args:     []string{"-o", "json", "-Y", ".store.book[0]", "testdata/store.yaml"},
			golden:   "testdata/book-0.json",
		}, {
			caseName: "-J wins over -Y",
			args:     []string{"-J", "-Y", ".store.book[0]", "testdata/store.yaml"},
			golden:   "testdata/book-0.json",
		}, {
			caseName: "-o wins over -J",
			args:     []string{"-o", "yaml", "-J", ".store.book[0]", "testdata/store.yaml"},
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
			caseName: "yaml starting with a json scalar is not duplicated",
			stdin:    "testdata/scalar-then-map.yaml",
			args:     []string{"."},
			want:     "1\n---\nfoo: bar\n",
		}, {
			caseName: "yaml with numeric keys is not duplicated",
			args:     []string{".", "testdata/numeric-keys.yaml"},
			golden:   "testdata/numeric-keys-out.yaml",
		}, {
			caseName: "json stream broken after two documents reports both errors",
			args:     []string{".a", "testdata/broken-json-stream"},
			want:     "1\n",
			errstr:   `failed to evaluate testdata/broken-json-stream: invalid character 'o' in literal false (expecting 'a'); yaml: line 1: did not find expected <document start>`,
		}, {
			caseName: "stray closing brace is an error under -j",
			args:     []string{"-j", ".", "testdata/stray-brace-json"},
			want:     "{\n  \"a\": 1\n}\n",
			errstr:   "failed to evaluate testdata/stray-brace-json: invalid character '}' looking for beginning of value",
		}, {
			caseName: "json rejected only by the number parser falls back to yaml",
			args:     []string{".", "testdata/json-number-overflow"},
			want:     "b: 1e400\n",
		}, {
			caseName: "truncated json container is not accepted by the pre-scan",
			args:     []string{".", "testdata/truncated-json"},
			errstr:   "failed to evaluate testdata/truncated-json: unexpected EOF; yaml: line 1: did not find expected node content",
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
			caseName: "non-seekable file with -U fails as before",
			pipe:     "testdata/store.json",
			args:     []string{"-U", ".store.book[0]"},
			errstr:   "failed to evaluate {pipe}: seek {pipe}: illegal seek",
		}, {
			caseName: "non-seekable file with -U -s is slurped",
			pipe:     "testdata/store.json",
			args:     []string{"-U", "-s", ".store.book[]"},
			golden:   "testdata/book-s",
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
	testCases := []struct {
		caseName  string
		pipe      bool
		rewind    bool
		wantSpool bool
	}{
		{caseName: "pipe without rewind is read directly", pipe: true},
		{caseName: "pipe with rewind is spooled", pipe: true, rewind: true, wantSpool: true},
		{caseName: "regular file with rewind is read directly", rewind: true},
	}
	for _, tc := range testCases {
		t.Run(tc.caseName, func(t *testing.T) {
			filename := "testdata/store.json"
			if tc.pipe {
				filename = pipePath(t, filename)
			}
			in, err := newInputFiles([]string{filename}, tc.rewind).nextReader()
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = in.Close() }()

			if _, isFile := in.(*os.File); isFile == tc.wantSpool {
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
		pipe     string
	}{
		{
			caseName: "regular file",
			args:     []string{"tq", "-O", "{out}", ".store.book[0]", "testdata/store.json"},
		}, {
			caseName: "non-seekable file with -U",
			args:     []string{"tq", "-U", "-O", "{out}", ".store.book[0]"},
			pipe:     "testdata/store.json",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.caseName, func(t *testing.T) {
			outPath := filepath.Join(t.TempDir(), "out.json")
			args := make([]string, len(tc.args))
			for i, a := range tc.args {
				args[i] = strings.ReplaceAll(a, "{out}", outPath)
			}
			if tc.pipe != "" {
				args = append(args, pipePath(t, tc.pipe))
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
			caseName: "yaml with numeric keys in place is not duplicated",
			src:      "testdata/numeric-keys.yaml",
			args:     []string{"-U", "."},
			golden:   "testdata/numeric-keys-out.yaml",
		}, {
			caseName: "broken json stream in place leaves the file untouched",
			src:      "testdata/broken-json-stream",
			args:     []string{"-U", ".a"},
			golden:   "testdata/broken-json-stream",
			errstr:   "invalid character 'o' in literal false (expecting 'a'); yaml: line 1: did not find expected <document start>",
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
		},
	}
	stdinOrg := os.Stdin
	defer func() { os.Stdin = stdinOrg }()

	for _, tc := range testCases {
		t.Run(tc.caseName, func(t *testing.T) {
			dir := t.TempDir()
			target := filepath.Join(dir, filepath.Base(tc.src))
			targets := []string{target}
			if tc.twoFiles {
				targets = append(targets, filepath.Join(dir, "second-"+filepath.Base(tc.src)))
			}
			src, err := os.ReadFile(tc.src)
			if err != nil {
				t.Fatalf("ReadFile src: %v", err)
			}
			for _, f := range targets {
				if err := os.WriteFile(f, src, 0o644); err != nil {
					t.Fatalf("WriteFile: %v", err)
				}
			}

			buf := new(bytes.Buffer)
			r := newTestRunner(buf)
			defer r.close()

			args := slices.Concat([]string{"tq"}, tc.args, targets)
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
			if tc.errstr != "" {
				want := "failed to evaluate " + target + ": " + tc.errstr
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
			}
		})
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
