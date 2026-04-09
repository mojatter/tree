package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"text/template"

	"github.com/mojatter/io2"
	"github.com/mojatter/tree"
	"github.com/spf13/pflag"
	"go.yaml.in/yaml/v3"
	"golang.org/x/term"
)

const (
	cmd          = "tq"
	desc         = cmd + " is a command-line JSON/YAML processor."
	usage        = cmd + " [flags] [query] ([file...])"
	examplesText = `Examples:
  % echo '{"colors": ["red", "green", "blue"]}' | tq '.colors[0]'
  "red"

  % echo '{"users":[{"id":1,"name":"one"},{"id":2,"name":"two"}]}' | tq -x -t '{{.id}}: {{.name}}' '.users'
  1: one
  2: two

  % echo '{}' | tq -e '.colors = ["red", "green"]' -e '.colors += "blue"' .
  {
    "colors": [
      "red",
      "green",
      "blue"
    ]
  }
`
	filenameStdin = "-"
)

type decodeError struct {
	err error
}

func (e *decodeError) Error() string {
	return e.err.Error()
}

func isDecodeError(err error) bool {
	_, ok := err.(*decodeError)
	return ok
}

type inputFiles struct {
	filenames []string
	off       int
	filename  string
}

func newInputFiles(filenames []string) *inputFiles {
	return &inputFiles{filenames: filenames}
}

func (f *inputFiles) nextReader() (io.ReadSeekCloser, error) {
	if f.off >= len(f.filenames) {
		return nil, io.EOF
	}
	f.filename = f.filenames[f.off]
	f.off++
	if f.filename == filenameStdin {
		return newStdinReader()
	}
	return os.Open(f.filename)
}

func newStdinReader() (io.ReadSeekCloser, error) {
	tmp, err := os.CreateTemp("", "*.tq.tmp")
	if err != nil {
		return nil, err
	}
	r := io2.DelegateReadSeekCloser(tmp)
	r.CloseFunc = func() error {
		_ = tmp.Close()
		return os.Remove(tmp.Name())
	}
	if _, err := io.Copy(tmp, os.Stdin); err != nil {
		_ = r.Close()
		return nil, err
	}
	if _, err := r.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}
	return r, nil
}

type runner struct {
	flagSet      *pflag.FlagSet
	isVersion    bool
	isHelp       bool
	isExpand     bool
	isSlurp      bool
	isRaw        bool
	isInplace    bool
	isColor      bool
	isInputJSON  bool
	isInputYAML  bool
	isOutputJSON bool
	isOutputYAML bool
	outputFile   string
	tmplText     string
	inputFormat  string
	outputFormat string
	editExprs    []string

	tmpl             *template.Template
	stderr           io.Writer
	out              io.WriteCloser
	guessFormat      string
	outputYAMLCalled int
	slurpResults     tree.Array
}

func newRunner() *runner {
	return &runner{
		stderr: os.Stderr,
		out:    io2.NopWriteCloser(os.Stdout),
	}
}

func (r *runner) initFlagSet(args []string) error {
	s := pflag.NewFlagSet(args[0], pflag.ExitOnError)
	r.flagSet = s

	s.SetOutput(r.stderr)
	s.BoolVarP(&r.isVersion, "version", "v", false, "print version")
	s.BoolVarP(&r.isHelp, "help", "h", false, "help for "+cmd)
	s.BoolVarP(&r.isExpand, "expand", "x", false, "expand results")
	s.BoolVarP(&r.isSlurp, "slurp", "s", false, "slurp all results into an array")
	s.BoolVarP(&r.isRaw, "raw", "r", false, "output raw strings")
	s.BoolVarP(&r.isInplace, "inplace", "U", false, "update files, inplace")
	s.BoolVarP(&r.isColor, "color", "c", false, "output with colors")
	s.BoolVarP(&r.isInputJSON, "input-json", "j", false, "alias --input-format json")
	s.BoolVarP(&r.isInputYAML, "input-yaml", "y", false, "alias --input-format yaml")
	s.BoolVarP(&r.isOutputJSON, "output-json", "J", false, "alias --output-format json")
	s.BoolVarP(&r.isOutputYAML, "output-yaml", "Y", false, "alias --output-format yaml")
	s.StringVarP(&r.outputFile, "output", "O", "", "output file")
	s.StringVarP(&r.tmplText, "template", "t", "", "golang text/template string")
	s.StringVarP(&r.inputFormat, "input-format", "i", "", "input format (json or yaml)")
	s.StringVarP(&r.outputFormat, "output-format", "o", "", "output format (json or yaml, default json)")
	s.StringArrayVarP(&r.editExprs, "edit", "e", nil, "edit expression")
	s.Usage = func() {
		_, _ = fmt.Fprintf(r.stderr, "%s\n\nUsage:\n  %s\n\n", desc, usage)
		_, _ = fmt.Fprintln(r.stderr, "Flags:")
		s.PrintDefaults()
		_, _ = fmt.Fprintf(r.stderr, "\n%s", examplesText)
	}
	return s.Parse(args[1:])
}

func (r *runner) close() {
	if r.out != nil {
		_ = r.out.Close()
		r.out = nil
	}
}

func (r *runner) run(args []string) error {
	defer r.close()

	if err := r.initFlagSet(args); err != nil {
		return err
	}
	if r.isVersion {
		_, _ = fmt.Fprintln(r.out, tree.VERSION)
		return nil
	}
	if r.isHelp || (r.flagSet.Arg(0) == "" && len(r.editExprs) == 0) {
		r.flagSet.Usage()
		return nil
	}
	if r.tmplText != "" {
		tmpl, err := template.New("").Parse(r.tmplText)
		if err != nil {
			return err
		}
		r.tmpl = tmpl
	}

	var filenames []string
	if args := r.flagSet.Args(); len(args) > 1 {
		filenames = args[1:]
	}
	if len(filenames) == 0 {
		if term.IsTerminal(0) {
			r.flagSet.Usage()
			return nil
		}
		filenames = []string{filenameStdin}
	}

	if r.outputFile != "" {
		out, err := os.Create(r.outputFile)
		if err != nil {
			return err
		}
		r.out = out
	}
	return r.evaluateInputFiles(newInputFiles(filenames))
}

func (r *runner) evaluateInputFiles(f *inputFiles) error {
	in, err := f.nextReader()
	if err != nil {
		if err == io.EOF {
			return nil
		}
		return err
	}
	defer func() { _ = in.Close() }()

	filename := f.filename
	var inplaceTmp *os.File
	if r.outputFile == "" && r.isInplace && !r.isSlurp && filename != filenameStdin {
		inplaceTmp, err = os.CreateTemp("", "*.tq.tmp")
		if err != nil {
			return err
		}
		r.out = inplaceTmp
		defer func() {
			_ = inplaceTmp.Close()
			_ = os.Remove(inplaceTmp.Name())
		}()
	}
	if err := r.evaluate(in); err != nil {
		if filename == filenameStdin {
			filename = "STDIN"
		}
		return fmt.Errorf("failed to evaluate %s: %w", filename, err)
	}
	if inplaceTmp != nil {
		if _, err := inplaceTmp.Seek(0, io.SeekStart); err != nil {
			return err
		}
		out, err := os.Create(filename) //nolint:gosec // inplace write to the user-specified input file is intentional
		if err != nil {
			return err
		}
		defer func() { _ = out.Close() }()
		if _, err := io.Copy(out, inplaceTmp); err != nil {
			return err
		}
	}
	return r.evaluateInputFiles(f)
}

func (r *runner) evaluate(in io.ReadSeekCloser) error {
	if r.inputFormat == "json" || r.isInputJSON {
		return r.evaluateJSON(in)
	}
	if r.inputFormat == "yaml" || r.isInputYAML {
		return r.evaluateYAML(in)
	}
	fns := []func(io.Reader) error{
		r.evaluateJSON,
		r.evaluateYAML,
	}
	var errs []string
	for _, fn := range fns {
		if _, err := in.Seek(0, io.SeekStart); err != nil {
			return err
		}
		if err := fn(in); err != nil {
			errs = append(errs, err.Error())
			if !isDecodeError(err) {
				break
			}
			continue
		}
		return nil
	}
	return errors.New(strings.Join(errs, "; "))
}

func (r *runner) evaluateJSON(in io.Reader) error {
	dec := json.NewDecoder(in)
	for dec.More() {
		n, err := tree.DecodeJSON(dec)
		if err != nil {
			return &decodeError{err}
		}
		r.guessFormat = "json"
		if err := r.evaluateNode(n); err != nil {
			return err
		}
	}
	if len(r.slurpResults) > 0 {
		defer func() { r.slurpResults = nil }()
		return r.output(r.slurpResults)
	}
	return nil
}

func (r *runner) evaluateYAML(in io.Reader) error {
	dec := yaml.NewDecoder(in)
	for {
		n, err := tree.DecodeYAML(dec)
		if err != nil {
			if err == io.EOF {
				break
			}
			return &decodeError{err}
		}
		r.guessFormat = "yaml"
		if err := r.evaluateNode(n); err != nil {
			return err
		}
	}
	if len(r.slurpResults) > 0 {
		defer func() { r.slurpResults = nil }()
		return r.output(r.slurpResults)
	}
	return nil
}

func (r *runner) evaluateNode(node tree.Node) error {
	for _, expr := range r.editExprs {
		if err := tree.Edit(&node, expr); err != nil {
			return err
		}
	}
	expr := r.flagSet.Arg(0)
	if expr == "" {
		expr = "."
	}
	results, err := tree.Find(node, expr)
	if err != nil {
		return err
	}
	if len(results) == 0 {
		return nil
	}
	if r.isSlurp {
		r.slurpResults = append(r.slurpResults, results...)
		return nil
	}
	if r.isExpand {
		cb := func(_ any, v tree.Node) error {
			return r.output(v)
		}
		for _, result := range results {
			if err := result.Each(cb); err != nil {
				return err
			}
		}
		return nil
	}
	for _, result := range results {
		if err := r.output(result); err != nil {
			return err
		}
	}
	return nil
}

func (r *runner) output(node tree.Node) error {
	if r.isRaw && node.Type().IsValue() {
		if _, err := fmt.Fprintln(r.out, node.Value().String()); err != nil {
			return err
		}
		return nil
	}
	if r.tmpl != nil {
		if err := r.tmpl.Execute(r.out, node); err != nil {
			return err
		}
		if _, err := fmt.Fprintln(r.out); err != nil {
			return err
		}
		return nil
	}
	if r.outputFormat == "yaml" || r.isOutputYAML || r.guessFormat == "yaml" {
		return r.outputYAML(node)
	}
	return r.outputJSON(node)
}

func (r *runner) outputYAML(n tree.Node) error {
	if r.outputYAMLCalled > 0 && !r.isInplace {
		if _, err := fmt.Fprintln(r.out, "---"); err != nil {
			return err
		}
	}
	r.outputYAMLCalled++
	if r.isColor {
		return tree.OutputColorYAML(r.out, n)
	}
	enc := yaml.NewEncoder(r.out)
	enc.SetIndent(2)
	if err := enc.Encode(n); err != nil {
		return err
	}
	return enc.Close()
}

func (r *runner) outputJSON(n tree.Node) error {
	if r.isColor {
		return tree.OutputColorJSON(r.out, n)
	}
	enc := json.NewEncoder(r.out)
	enc.SetIndent("", "  ")
	return enc.Encode(n)
}

func main() {
	r := newRunner()
	defer r.close()

	if err := r.run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
}
