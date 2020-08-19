// Textproc processes text.
package main

import (
	"errors"
	"flag"
	"fmt"
	"github.com/MihaiB/textproc/v2"
	"io"
	"os"
	"sort"
	"strings"
	"unicode/utf8"
)

var errNoProgramName = errors.New("no program name (os.Args empty)")

type catalogueEntry struct {
	processor textproc.Processor
	doc       string
}

var normChain = []string{"lf", "trail", "trimlf", "nelf"}

func chainProcessors(processors ...textproc.Processor) textproc.Processor {
	return func(c <-chan rune) <-chan rune {
		for _, p := range processors {
			c = p(c)
		}
		return c
	}
}

var catalogue = map[string]*catalogueEntry{
	"lf": {textproc.ConvertLineTerminatorsToLF,
		"Convert line terminators to LF"},
	"nelf": {textproc.EnsureFinalLFIfNonEmpty,
		"Ensure non-empty content ends with LF"},
	"norm": {nil, fmt.Sprint("Normalize: ", strings.Join(normChain, " "))},
	"sortli": {textproc.SortLFLinesI,
		"Sort lines case-insensitive (LF end of line)"},
	"trail": {textproc.TrimLFTrailingWhiteSpace,
		"Remove trailing whitespace (LF end of line)"},
	"trimlf": {chainProcessors(textproc.TrimLeadingEmptyLFLines,
		textproc.TrimTrailingEmptyLFLines),
		"Trim leading and trailing empty lines (LF end of line)"},
}

func init() {
	// Avoid initialization loop for catalogue chain processors

	chainCatalogueKeys := func(keys []string) textproc.Processor {
		return func(c <-chan rune) <-chan rune {
			for _, key := range keys {
				c = catalogue[key].processor(c)
			}
			return c
		}
	}

	catalogue["norm"].processor = chainCatalogueKeys(normChain)
}

var catalogueKeys = func() []string {
	var keys []string
	for k := range catalogue {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}()

type cmdArgs struct {
	processors []textproc.Processor
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
		args.processors = append(args.processors, entry.processor)
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

func write(runeCh <-chan rune, errCh <-chan error, w io.Writer) error {
	b := make([]byte, utf8.UTFMax)
	for {
		r, ok := <-runeCh
		if !ok {
			break
		}
		n := utf8.EncodeRune(b, r)
		if _, err := w.Write(b[:n]); err != nil {
			return err
		}
	}

	err := <-errCh
	if err == io.EOF {
		err = nil
	}
	return err
}

func main() {
	args, err := parseArgs(os.Args)
	if err != nil {
		errExit(err)
	}

	runeCh, errCh := textproc.Read(os.Stdin)
	for _, processor := range args.processors {
		runeCh = processor(runeCh)
	}

	if err = write(runeCh, errCh, os.Stdout); err != nil {
		errExit(err)
	}
}
