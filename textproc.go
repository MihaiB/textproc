// Package textproc provides text processing.
package textproc

import (
	"bufio"
	"errors"
	"io"
	"unicode/utf8"
)

const runeErrorSize = len(string(utf8.RuneError))

// ErrInvalidUTF8 is the error returned when the input is not valid UTF-8.
var ErrInvalidUTF8 = errors.New("Invalid UTF-8")

func readRune(reader io.RuneReader) (rune, error) {
	r, size, err := reader.ReadRune()
	if err == nil && r == utf8.RuneError && size != runeErrorSize {
		err = ErrInvalidUTF8
	}
	return r, err
}

// Read returns two channels.
// All runes read from r as UTF-8 are sent, then the rune channel is closed,
// then the error from r is sent, then the error channel is closed.
func Read(r io.Reader) (<-chan rune, <-chan error) {
	runeCh := make(chan rune)
	errCh := make(chan error)

	go func() {
		runeReader := bufio.NewReader(r)
		for {
			char, err := readRune(runeReader)
			if err != nil {
				close(runeCh)
				errCh <- err
				close(errCh)
				return
			}
			runeCh <- char
		}
	}()

	return runeCh, errCh
}
