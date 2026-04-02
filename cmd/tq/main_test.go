package main

import (
	"bytes"
	"os"
	"testing"

	"github.com/mojatter/io2"
	"github.com/mojatter/tree"
)

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
		want     string
		errstr   string
	}{
		{
			caseName: "show usage",
			args:     []string{},
			want:     mustReadFileString("testdata/usage"),
		}, {
			caseName: "show help",
			args:     []string{"-h"},
			want:     mustReadFileString("testdata/usage"),
		}, {
			caseName: "show version",
			args:     []string{"-v"},
			want:     tree.VERSION + "\n",
		}, {
			caseName: "first book from stdin",
			stdin:    "testdata/store.json",
			args:     []string{".store.book[0]"},
			want:     mustReadFileString("testdata/book-0.json"),
		}, {
			caseName: "first book from stdin yaml",
			stdin:    "testdata/store.yaml",
			args:     []string{".store.book[0]"},
			want:     mustReadFileString("testdata/book-0.yaml"),
		}, {
			caseName: "first book from stdin with -i yaml",
			stdin:    "testdata/store.yaml",
			args:     []string{"-i", "yaml", ".store.book[0]"},
			want:     mustReadFileString("testdata/book-0.yaml"),
		}, {
			caseName: "first book from json",
			args:     []string{".store.book[0]", "testdata/store.json"},
			want:     mustReadFileString("testdata/book-0.json"),
		}, {
			caseName: "first book from json to yaml",
			args:     []string{"-o", "yaml", ".store.book[0]", "testdata/store.json"},
			want:     mustReadFileString("testdata/book-0.yaml"),
		}, {
			caseName: "range[1:3] books",
			args:     []string{".store.book[1:3]|", "testdata/store.json"},
			want:     mustReadFileString("testdata/book-1-3.json"),
		}, {
			caseName: "select books using tags",
			args:     []string{".store.book[.tags[.name == \"genre\" and .value == \"fiction\"].count() > 0]|", "testdata/store.json"},
			want:     mustReadFileString("testdata/book-1-3.json"),
		}, {
			caseName: "select books using tags omit operators",
			args:     []string{".store.book[.tags[.name == \"genre\" and .value == \"fiction\"]]|", "testdata/store.json"},
			want:     mustReadFileString("testdata/book-1-3.json"),
		}, {
			caseName: "single quote",
			args:     []string{".store.book[.tags[.name == 'genre' and .value == 'fiction']]|", "testdata/store.json"},
			want:     mustReadFileString("testdata/book-1-3.json"),
		}, {
			caseName: "expand books",
			stdin:    "testdata/store.json",
			args:     []string{"-x", ".store.book"},
			want:     mustReadFileString("testdata/book-x"),
		}, {
			caseName: "slurp books",
			stdin:    "testdata/store.json",
			args:     []string{"-s", ".store.book[]"},
			want:     mustReadFileString("testdata/book-s"),
		}, {
			caseName: "slurp books",
			stdin:    "testdata/book-x",
			args:     []string{"-s", "."},
			want:     mustReadFileString("testdata/book-s"),
		}, {
			caseName: "expand books",
			stdin:    "testdata/book-s",
			args:     []string{"-x", "."},
			want:     mustReadFileString("testdata/book-x"),
		}, {
			caseName: "template output",
			stdin:    "testdata/store.json",
			args: []string{
				"-t", "{{.title}},{{.author}},{{.category}},{{.price}}",
				".store.book[]",
			},
			want: mustReadFileString("testdata/book.csv"),
		}, {
			caseName: "output json with color",
			stdin:    "testdata/store.json",
			args:     []string{"-c", "."},
			want:     mustReadFileString("testdata/store-color.json"),
		}, {
			caseName: "output yaml with color",
			stdin:    "testdata/store.yaml",
			args:     []string{"-c", "."},
			want:     mustReadFileString("testdata/store-color.yaml"),
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
			want: mustReadFileString("testdata/book-0.json"),
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
				defer in.Close()
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
			if got := buf.String(); got != tc.want {
				t.Errorf("got %s; want %s", got, tc.want)
			}
		})
	}
}
