// Textproc processes text.
package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"github.com/MihaiB/textproc/v3"
	"io"
	"os"
	"sort"
	"strings"
)

var errNoProgramName = errors.New("no program name (os.Args empty)")

type catalogueEntry struct {
	runeProc textproc.RuneProcessor
	doc      string
}

func chainRuneProcessors(runeProcs ...textproc.RuneProcessor) textproc.RuneProcessor {
	return func(runeCh <-chan rune, errCh <-chan error) (
		<-chan rune, <-chan error) {
		for _, p := range runeProcs {
			runeCh, errCh = p(runeCh, errCh)
		}
		return runeCh, errCh
	}
}

var normChain = []string{"lf", "trail", "trimlf", "nelf"}

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
	"trimlf": {chainRuneProcessors(textproc.TrimLeadingEmptyLFLines,
		textproc.TrimTrailingEmptyLFLines),
		"Trim leading and trailing empty lines (LF end of line)"},
}

func init() {
	// Avoid initialization loop for catalogue chain processors

	chainCatalogueKeys := func(keys []string) textproc.RuneProcessor {
		var runeProcs []textproc.RuneProcessor
		for _, key := range keys {
			runeProcs = append(runeProcs, catalogue[key].runeProc)
		}
		return chainRuneProcessors(runeProcs...)
	}

	catalogue["norm"].runeProc = chainCatalogueKeys(normChain)
}

var catalogueKeys = func() []string {
	var keys []string
	for key := range catalogue {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}()

type cmdArgs struct {
	runeProcs []textproc.RuneProcessor
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
		args.runeProcs = append(args.runeProcs, entry.runeProc)
	}
	return args, nil
}

func write(runeCh <-chan rune, errCh <-chan error, ioW io.Writer) (err error) {
	bufW := bufio.NewWriter(ioW)
	defer func() {
		if flushErr := bufW.Flush(); flushErr != nil && err == nil {
			err = flushErr
		}
	}()

	for r := range runeCh {
		if _, err = bufW.WriteRune(r); err != nil {
			return
		}
	}

	return <-errCh
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

	runeCh, errCh := chainRuneProcessors(args.runeProcs...)(
		textproc.ReadRunes(os.Stdin))

	if err = write(runeCh, errCh, os.Stdout); err != nil {
		errExit(err)
	}
}
