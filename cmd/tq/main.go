package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
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

type yamlDecodeError struct {
	err error
}

func (e *yamlDecodeError) Error() string {
	return e.err.Error()
}

func isYAMLDecodeError(err error) bool {
	_, ok := err.(*yamlDecodeError)
	return ok
}

type inputFiles struct {
	filenames []string
	off       int
	filename  string
	// rewind spools non-seekable files so that format auto-detection can rewind them.
	rewind bool
}

func newInputFiles(filenames []string, rewind bool) *inputFiles {
	return &inputFiles{filenames: filenames, rewind: rewind}
}

func (f *inputFiles) nextReader() (io.ReadSeekCloser, error) {
	if f.off >= len(f.filenames) {
		return nil, io.EOF
	}
	f.filename = f.filenames[f.off]
	f.off++
	if f.filename == filenameStdin {
		return spool(os.Stdin)
	}
	file, err := os.Open(f.filename)
	if err != nil {
		return nil, err
	}
	if !f.rewind {
		return file, nil
	}
	if _, err := file.Seek(0, io.SeekCurrent); err == nil {
		return file, nil
	}
	defer func() { _ = file.Close() }()

	return spool(file)
}

// spool copies src to a temp file so that format detection can rewind it.
func spool(src io.Reader) (io.ReadSeekCloser, error) {
	tmp, err := os.CreateTemp("", "*.tq.tmp")
	if err != nil {
		return nil, err
	}
	r := io2.DelegateReadSeekCloser(tmp)
	r.CloseFunc = func() error {
		_ = tmp.Close()
		return os.Remove(tmp.Name())
	}
	if _, err := io.Copy(tmp, src); err != nil {
		_ = r.Close()
		return nil, err
	}
	if _, err := r.Seek(0, io.SeekStart); err != nil {
		_ = r.Close()
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
	s.StringVarP(&r.inputFormat, "input-format", "i", "", "input format (json or yaml, default: json if the input starts with { or [, else json then yaml)")
	s.StringVarP(&r.outputFormat, "output-format", "o", "", "output format (json or yaml, default: input format)")
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
	if r.isInplaceWrite() {
		if err := checkInplaceTargets(filenames); err != nil {
			return err
		}
	}
	return r.evaluateInputFiles(newInputFiles(filenames, r.needsRewind()))
}

// checkInplaceTargets rejects -U targets that are not regular or not readable before any of them is read.
func checkInplaceTargets(filenames []string) error {
	for _, filename := range filenames {
		if filename == filenameStdin {
			continue
		}
		fi, err := os.Stat(filename)
		if err != nil {
			return err
		}
		if !fi.Mode().IsRegular() {
			return fmt.Errorf("cannot update %s in place: not a regular file", filename)
		}
		f, err := os.Open(filename) //nolint:gosec // opening the user-specified input file is intentional
		if err != nil {
			return err
		}
		_ = f.Close()
	}
	return nil
}

// isInplaceWrite reports whether -U writes results back to the input files.
func (r *runner) isInplaceWrite() bool {
	return r.outputFile == "" && r.isInplace && !r.isSlurp
}

func (r *runner) evaluateInputFiles(f *inputFiles) error {
	for {
		in, err := f.nextReader()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		if err := r.evaluateInputFile(in, f.filename); err != nil {
			return err
		}
	}
}

// evaluateInputFile evaluates one input and releases its descriptors before returning.
func (r *runner) evaluateInputFile(in io.ReadSeekCloser, filename string) error {
	defer func() { _ = in.Close() }()

	var inplace *inplaceFile
	if r.isInplaceWrite() && filename != filenameStdin {
		var err error
		inplace, err = createInplaceFile(filename)
		if err != nil {
			return fmt.Errorf("failed to write %s: %w", filename, err)
		}
		defer inplace.discard()

		prevOut, prevCalled := r.out, r.outputYAMLCalled
		r.out = inplace
		r.outputYAMLCalled = 0
		defer func() { r.out, r.outputYAMLCalled = prevOut, prevCalled }()
	}
	err := r.evaluate(in)
	if inplace != nil && inplace.writeErr != nil {
		return fmt.Errorf("failed to write %s: %w", filename, inplace.writeErr)
	}
	if err != nil {
		if filename == filenameStdin {
			filename = "STDIN"
		}
		return fmt.Errorf("failed to evaluate %s: %w", filename, err)
	}
	if inplace != nil {
		// Windows cannot rename over a file that is still open.
		_ = in.Close()
		if err := inplace.commit(); err != nil {
			return fmt.Errorf("failed to write %s: %w", filename, err)
		}
	}
	return nil
}

// inplaceFile is a temp file next to a -U target that replaces the target on commit.
type inplaceFile struct {
	file      *os.File
	target    string
	mode      os.FileMode
	writeErr  error
	committed bool
}

// Write writes to the temp file and records the first error, so that it is reported as a write-back failure.
func (f *inplaceFile) Write(p []byte) (int, error) {
	n, err := f.file.Write(p)
	if err != nil && f.writeErr == nil {
		f.writeErr = err
	}
	return n, err
}

// Close closes the temp file.
func (f *inplaceFile) Close() error {
	return f.file.Close()
}

// createInplaceFile creates the temp file beside the file that filename, or the symlink it names, points to.
func createInplaceFile(filename string) (*inplaceFile, error) {
	target, err := filepath.EvalSymlinks(filename)
	if err != nil {
		return nil, err
	}
	fi, err := os.Stat(target)
	if err != nil {
		return nil, err
	}
	// A fixed name keeps the temp file within NAME_MAX whatever the target's name.
	tmp, err := os.CreateTemp(filepath.Dir(target), ".tq-*.tmp")
	if err != nil {
		return nil, err
	}
	return &inplaceFile{file: tmp, target: target, mode: fi.Mode().Perm()}, nil
}

// commit sets the target's permission bits on the temp file, syncs and closes it, then renames it over the target.
func (f *inplaceFile) commit() error {
	if err := f.file.Chmod(f.mode); err != nil {
		return err
	}
	if err := f.file.Sync(); err != nil {
		return err
	}
	if err := f.file.Close(); err != nil {
		return err
	}
	if err := os.Rename(f.file.Name(), f.target); err != nil {
		return err
	}
	f.committed = true
	return nil
}

// discard removes the temp file unless it was committed.
func (f *inplaceFile) discard() {
	if f.committed {
		return
	}
	_ = f.file.Close()
	_ = os.Remove(f.file.Name())
}

// fixedInputFormat returns "json" or "yaml" when the input format is given, otherwise "".
func (r *runner) fixedInputFormat() string {
	switch {
	case r.inputFormat == "json" || r.isInputJSON:
		return "json"
	case r.inputFormat == "yaml" || r.isInputYAML:
		return "yaml"
	}
	return ""
}

// needsRewind reports whether evaluate auto-detects the format and so rewinds the input.
func (r *runner) needsRewind() bool {
	return r.fixedInputFormat() == ""
}

func (r *runner) evaluate(in io.ReadSeekCloser) error {
	switch r.fixedInputFormat() {
	case "json":
		return r.evaluateJSON(in)
	case "yaml":
		return r.evaluateYAML(in)
	}
	jsonErr := detectJSON(in)
	if _, err := in.Seek(0, io.SeekStart); err != nil {
		return err
	}
	if jsonErr == nil {
		return r.evaluateJSON(in)
	}
	if err := r.evaluateYAML(in); err != nil {
		if isYAMLDecodeError(err) {
			return fmt.Errorf("%w; %w", jsonErr, err)
		}
		return err
	}
	return nil
}

// detectJSON returns nil when the input is read as JSON: it reaches a { or [ before any syntax error, or holds only JSON scalars.
func detectJSON(in io.Reader) error {
	dec := json.NewDecoder(in)
	dec.UseNumber()
	for {
		t, err := dec.Token()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		if _, ok := t.(json.Delim); ok {
			return nil
		}
	}
}

func (r *runner) evaluateJSON(in io.Reader) error {
	dec := json.NewDecoder(in)
	for dec.More() {
		n, err := tree.DecodeJSON(dec)
		if err != nil {
			return err
		}
		r.guessFormat = "json"
		if err := r.evaluateNode(n); err != nil {
			return err
		}
	}
	// More stops silently at a stray } or ], so reject anything but EOF.
	if _, err := dec.Token(); err != io.EOF {
		return err
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
			return &yamlDecodeError{err}
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
	if r.effectiveOutputFormat() == "yaml" {
		return r.outputYAML(node)
	}
	return r.outputJSON(node)
}

// effectiveOutputFormat resolves the output format: -o, then -J, then -Y, then the input format.
func (r *runner) effectiveOutputFormat() string {
	switch {
	case r.outputFormat == "json" || r.outputFormat == "yaml":
		return r.outputFormat
	case r.isOutputJSON:
		return "json"
	case r.isOutputYAML:
		return "yaml"
	}
	return r.guessFormat
}

func (r *runner) outputYAML(n tree.Node) error {
	if r.outputYAMLCalled > 0 {
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
