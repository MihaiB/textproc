// Textproc processes text.
package main

import (
	"errors"
	"flag"
	"fmt"
	"github.com/MihaiB/textproc"
	"io"
	"os"
	"sort"
	"strings"
)

type procFunc func(textproc.Reader) textproc.Reader
type procEntry struct {
	proc procFunc
	doc  string
}

var (
	errNoProgramName = errors.New("no program name (os.Args empty)")

	normChain = []string{"lf", "trail", "trimlf", "nelf"}

	catalogue = map[string]*procEntry{
		"lf":     {textproc.LFLines, "Convert line terminators to LF"},
		"sortpi": {textproc.SortLFParagraphsI, "Sort paragraphs case-insensitive (LF end of line)"},
		"trail":  {textproc.TrimLFTrailingSpace, "Remove trailing whitespace (LF end of line)"},
		"nelf":   {textproc.NonEmptyFinalLF, "Ensure non-empty content ends with LF"},
		"trimlf": {func(r textproc.Reader) textproc.Reader {
			return textproc.TrimTrailingEmptyLFLines(textproc.TrimLeadingLF(r))
		}, "Trim leading and trailing empty lines (LF end of line)"},
		"norm": {nil, fmt.Sprint("Normalize: ", strings.Join(normChain, " "))},
	}

	catalogueKeys = func() []string {
		var keys []string
		for k := range catalogue {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		return keys
	}()
)

func init() {
	// Avoid initialization loop for catalogue chain processors

	chainProcessors := func(names []string) procFunc {
		return func(r textproc.Reader) textproc.Reader {
			for _, name := range names {
				r = catalogue[name].proc(r)
			}
			return r
		}
	}

	catalogue["norm"].proc = chainProcessors(normChain)
}

type cmdArgs struct {
	procs []procFunc
}

func parseArgs(osArgs []string) (*cmdArgs, error) {
	if len(osArgs) == 0 {
		return nil, errNoProgramName
	}

	fs := flag.NewFlagSet(osArgs[0], flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprint(fs.Output(), "usage: ", fs.Name(), " [processors]\n")
		fmt.Fprint(fs.Output(), `
Process text from stdin to stdout.

processors:
`)
		for _, k := range catalogueKeys {
			fmt.Fprintf(fs.Output(), "\t%s\t%s\n",
				k, catalogue[k].doc)
		}
		// say ‘optional arguments:’ first, if there will be flags
		fs.PrintDefaults()
	}
	if err := fs.Parse(osArgs[1:]); err != nil {
		return nil, err
	}

	args := &cmdArgs{}
	for _, k := range fs.Args() {
		entry, ok := catalogue[k]
		if !ok {
			return nil, errors.New("unknown processor: " + k)
		}
		args.procs = append(args.procs, entry.proc)
	}
	return args, nil
}

func errExit(err error) {
	if len(os.Args) > 0 && os.Args[0] != "" {
		fmt.Fprint(os.Stderr, os.Args[0], ": ")
	}
	fmt.Fprintln(os.Stderr, "error:", err)
	os.Exit(1)
}

func main() {
	args, err := parseArgs(os.Args)
	if err != nil {
		errExit(err)
	}

	in := textproc.NewReader(os.Stdin)
	for _, proc := range args.procs {
		in = proc(in)
	}

	if _, err = io.Copy(os.Stdout, textproc.NewIoReader(in)); err != nil {
		errExit(err)
	}
}
