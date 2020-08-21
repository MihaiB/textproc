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
	"unicode"
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

// EnsureFinalLFIfNonEmpty ensures non-empty content ends with "\n".
func EnsureFinalLFIfNonEmpty(runeIn <-chan rune, errIn <-chan error) (
	<-chan rune, <-chan error) {
	runeOut, errOut := make(chan rune), make(chan error)

	go func() {
		last := '\n'

		for r := range runeIn {
			runeOut <- r
			last = r
		}

		err := <-errIn
		if err == nil && last != '\n' {
			runeOut <- '\n'
		}
		close(runeOut)

		errOut <- err
		close(errOut)
	}()

	return runeOut, errOut
}

// TrimLFTrailingWhiteSpace removes white space at the end of lines.
// Lines are terminated by "\n".
func TrimLFTrailingWhiteSpace(runeIn <-chan rune, errIn <-chan error) (
	<-chan rune, <-chan error) {
	runeOut, errOut := make(chan rune), make(chan error)

	go func() {
		var spaces []rune
		for r := range runeIn {
			if r == '\n' {
				spaces = nil
				runeOut <- r
				continue
			}

			if unicode.IsSpace(r) {
				spaces = append(spaces, r)
				continue
			}

			for _, space := range spaces {
				runeOut <- space
			}
			spaces = nil
			runeOut <- r
		}

		close(runeOut)
		errOut <- <-errIn
		close(errOut)
	}()

	return runeOut, errOut
}

// TrimLeadingEmptyLFLines removes empty lines at the start of the input.
// Lines are terminated by "\n".
func TrimLeadingEmptyLFLines(runeIn <-chan rune, errIn <-chan error) (
	<-chan rune, <-chan error) {
	runeOut, errOut := make(chan rune), make(chan error)

	go func() {
		skipping := true
		for r := range runeIn {
			if skipping {
				if r == '\n' {
					continue
				}
				skipping = false
			}
			runeOut <- r
		}
		close(runeOut)
		errOut <- <-errIn
		close(errOut)
	}()

	return runeOut, errOut
}

// TrimTrailingEmptyLFLines removes empty lines at the end of the input.
// Lines are terminated by "\n".
func TrimTrailingEmptyLFLines(runeIn <-chan rune, errIn <-chan error) (
	<-chan rune, <-chan error) {
	runeOut, errOut := make(chan rune), make(chan error)

	go func() {
		atLineStart := true
		pendingNewlines := 0

		for r := range runeIn {
			if atLineStart && r == '\n' {
				pendingNewlines++
				continue
			}

			for ; pendingNewlines > 0; pendingNewlines-- {
				runeOut <- '\n'
			}
			runeOut <- r
			atLineStart = r == '\n'
		}

		close(runeOut)
		errOut <- <-errIn
		close(errOut)
	}()

	return runeOut, errOut
}
