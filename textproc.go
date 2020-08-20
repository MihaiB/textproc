// Package textproc provides text processing.
//
// For a pairs of channels (chan dataType, chan error)
// first all data is transmitted then the data channel is closed
// then a single error is transmitted then the error channel is closed.
// The nil error indicates success.
// Any non-nil error (including io.EOF) indicates failure.
package textproc

import (
	"bufio"
	"errors"
	"io"
	"unicode/utf8"
)

// ErrInvalidUTF8 is the error returned when the input is not valid UTF-8.
var ErrInvalidUTF8 = errors.New("invalid UTF-8")

const runeErrorSize = len(string(utf8.RuneError))

// A RuneProcessor receives and sends runes.
type RuneProcessor = func(runeIn <-chan rune, errIn <-chan error) (
	runeOut <-chan rune, errOut <-chan error)

// A Tokenizer receives runes and sends tokens.
type Tokenizer = func(runeIn <-chan rune, errIn <-chan error) (
	tokenOut <-chan []rune, errOut <-chan error)

func readRune(reader io.RuneReader) (rune, error) {
	r, size, err := reader.ReadRune()
	if err == nil && r == utf8.RuneError && size != runeErrorSize {
		err = ErrInvalidUTF8
	}
	return r, err
}

// ReadRunes sends the runes from r.
// It sends ErrInvalidUTF8 if the input is not valid UTF-8.
func ReadRunes(r io.Reader) (<-chan rune, <-chan error) {
	runeOut, errOut := make(chan rune), make(chan error)

	go func() {
		runeReader := bufio.NewReader(r)
		for {
			char, err := readRune(runeReader)
			if err != nil {
				if err == io.EOF {
					err = nil
				}
				close(runeOut)
				errOut <- err
				close(errOut)
				return
			}
			runeOut <- char
		}
	}()

	return runeOut, errOut
}

// ConvertLineTerminatorsToLF converts "\r" and "\r\n" to "\n".
func ConvertLineTerminatorsToLF(runeIn <-chan rune, errIn <-chan error) (
	<-chan rune, <-chan error) {
	runeOut, errOut := make(chan rune), make(chan error)

	go func() {
		skipNextLF := false

		for r := range runeIn {
			if skipNextLF && r == '\n' {
				skipNextLF = false
				continue
			}
			if r == '\r' {
				runeOut <- '\n'
				skipNextLF = true
			} else {
				runeOut <- r
				skipNextLF = false
			}
		}

		close(runeOut)
		errOut <- <-errIn
		close(errOut)
	}()

	return runeOut, errOut
}
