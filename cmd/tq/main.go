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
	flagSet       *pflag.FlagSet
	isVersion     bool
	isHelp        bool
	isExpand      bool
	isSlurp       bool
	isRaw         bool
	isInplace     bool
	isColor       bool
	isInputJSON   bool
	isInputYAML   bool
	isOutputJSON  bool
	isOutputYAML  bool
	outputFile    string
	tmplText      string
	inputFormat   string
	outputFormat  string
	editExprs     []string
	mergeStrategy string

	tmpl             *template.Template
	stderr           io.Writer
	out              io.WriteCloser
	guessFormat      string
	outputYAMLCalled int
	slurpResults     tree.Array
	mergeOption      tree.MergeOption
	queryExpr        string
}

// looksLikeQuery reports whether s looks like a tree query expression
// rather than a file path. Used by --merge mode to decide whether the
// first positional argument is a query or a file. tree queries always
// begin with "." (path) or "[" (filter / array index).
func looksLikeQuery(s string) bool {
	return strings.HasPrefix(s, ".") || strings.HasPrefix(s, "[")
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
	s.StringVarP(&r.mergeStrategy, "merge", "m", "", "merge inputs (optional strategy: default, override, replace, append, slurp; combine with comma)")
	s.Lookup("merge").NoOptDefVal = "default"
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
	if r.isHelp || (r.flagSet.Arg(0) == "" && len(r.editExprs) == 0 && r.mergeStrategy == "") {
		r.flagSet.Usage()
		return nil
	}
	if r.mergeStrategy != "" {
		opt, err := tree.MergeOptionFromString(r.mergeStrategy)
		if err != nil {
			return fmt.Errorf("--merge: %w", err)
		}
		r.mergeOption = opt
		if r.isInplace {
			return errors.New("--merge cannot be combined with --inplace")
		}
		if r.isSlurp {
			return errors.New("--merge cannot be combined with --slurp")
		}
	}
	if r.tmplText != "" {
		tmpl, err := template.New("").Parse(r.tmplText)
		if err != nil {
			return err
		}
		r.tmpl = tmpl
	}

	var filenames []string
	posArgs := r.flagSet.Args()
	if r.mergeStrategy != "" && len(posArgs) > 0 && !looksLikeQuery(posArgs[0]) {
		// In --merge mode, accept the conventional "tq --merge a.yaml b.yaml"
		// (no query). Treat all positionals as files when the first one
		// doesn't look like a tree query.
		filenames = posArgs
	} else {
		if len(posArgs) > 0 {
			r.queryExpr = posArgs[0]
		}
		if len(posArgs) > 1 {
			filenames = posArgs[1:]
		}
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
	if r.mergeStrategy != "" {
		return r.evaluateMergeFiles(newInputFiles(filenames))
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

// evaluateMergeFiles reads every doc from every input file and folds
// them with tree.Merge in left-to-right order, then runs the usual
// edit/query/output pipeline once on the merged result. With multiple
// docs in the same file (e.g. YAML "---" separators) the docs are
// merged in the order they appear, before merging into the next file.
func (r *runner) evaluateMergeFiles(f *inputFiles) error {
	var acc tree.Node
	accSet := false
	handle := func(n tree.Node) error {
		if !accSet {
			acc = n
			accSet = true
			return nil
		}
		acc = tree.Merge(acc, n, r.mergeOption)
		return nil
	}
	for {
		in, err := f.nextReader()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		filename := f.filename
		err = r.parseStream(in, handle)
		_ = in.Close()
		if err != nil {
			if filename == filenameStdin {
				filename = "STDIN"
			}
			return fmt.Errorf("failed to evaluate %s: %w", filename, err)
		}
	}
	if !accSet {
		return nil
	}
	return r.evaluateNode(acc)
}

func (r *runner) evaluate(in io.ReadSeekCloser) error {
	if err := r.parseStream(in, r.evaluateNode); err != nil {
		return err
	}
	if len(r.slurpResults) > 0 {
		defer func() { r.slurpResults = nil }()

		return r.output(r.slurpResults)
	}
	return nil
}

// parseStream reads in, dispatches to the JSON or YAML parser based on
// the input-format flags (or by trying both when neither is forced),
// and invokes handle for every parsed node. Decoding the input as one
// format and falling back to the other is preserved here so callers
// (evaluate, evaluateMergeFiles) get the same auto-detection.
func (r *runner) parseStream(in io.ReadSeekCloser, handle func(tree.Node) error) error {
	if r.inputFormat == "json" || r.isInputJSON {
		return r.parseJSON(in, handle)
	}
	if r.inputFormat == "yaml" || r.isInputYAML {
		return r.parseYAML(in, handle)
	}
	fns := []func(io.Reader, func(tree.Node) error) error{
		r.parseJSON,
		r.parseYAML,
	}
	var errs []string
	for _, fn := range fns {
		if _, err := in.Seek(0, io.SeekStart); err != nil {
			return err
		}
		if err := fn(in, handle); err != nil {
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

func (r *runner) parseJSON(in io.Reader, handle func(tree.Node) error) error {
	dec := json.NewDecoder(in)
	for dec.More() {
		n, err := tree.DecodeJSON(dec)
		if err != nil {
			return &decodeError{err}
		}
		r.guessFormat = "json"
		if err := handle(n); err != nil {
			return err
		}
	}
	return nil
}

func (r *runner) parseYAML(in io.Reader, handle func(tree.Node) error) error {
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
		if err := handle(n); err != nil {
			return err
		}
	}
	return nil
}

func (r *runner) evaluateNode(node tree.Node) error {
	for _, expr := range r.editExprs {
		if err := tree.Edit(&node, expr); err != nil {
			return err
		}
	}
	expr := r.queryExpr
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
