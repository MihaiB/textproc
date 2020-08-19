// Textproc processes text.
package main

import (
	"fmt"
	"github.com/MihaiB/textproc/v2"
	"io"
	"os"
	"unicode/utf8"
)

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
	runeCh, errCh := textproc.Read(os.Stdin)

	if err := write(runeCh, errCh, os.Stdout); err != nil {
		errExit(err)
	}
}
