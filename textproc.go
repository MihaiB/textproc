// Package textproc provides text processing.
package textproc

import (
	"bufio"
	"errors"
	"io"
	"unicode"
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

// A Processor processes runes.
type Processor = func(<-chan rune) <-chan rune

// ConvertLineTerminatorsToLF converts "\r" and "\r\n" to "\n".
func ConvertLineTerminatorsToLF(in <-chan rune) <-chan rune {
	out := make(chan rune)

	go func() {
		for prev, crt := '\n', '\n'; ; prev = crt {
			var ok bool
			if crt, ok = <-in; !ok {
				break
			}

			if prev == '\r' && crt == '\n' {
				continue
			}
			if crt == '\r' {
				out <- '\n'
			} else {
				out <- crt
			}
		}
		close(out)
	}()

	return out
}

// EnsureFinalLFIfNonEmpty ensures non-empty content ends with "\n".
func EnsureFinalLFIfNonEmpty(in <-chan rune) <-chan rune {
	out := make(chan rune)

	go func() {
		last := '\n'

		for {
			r, ok := <-in
			if !ok {
				break
			}
			out <- r
			last = r
		}

		if last != '\n' {
			out <- '\n'
		}
		close(out)
	}()

	return out
}

// TrimLFTrailingSpaces removes white space at the end of lines.
// Lines are terminated by "\n".
func TrimLFTrailingSpaces(in <-chan rune) <-chan rune {
	out := make(chan rune)

	go func() {
		var spaces []rune
		for r := range in {
			if r == '\n' {
				spaces = nil
				out <- r
				continue
			}

			if unicode.IsSpace(r) {
				spaces = append(spaces, r)
				continue
			}

			for _, space := range spaces {
				out <- space
			}
			spaces = nil
			out <- r
		}

		close(out)
	}()

	return out
}

// TrimLeadingEmptyLFLines removes empty lines at the start of the input.
// Lines are terminated by "\n".
func TrimLeadingEmptyLFLines(in <-chan rune) <-chan rune {
	out := make(chan rune)

	go func() {
		skipping := true
		for r := range in {
			if skipping {
				if r == '\n' {
					continue
				}
				skipping = false
			}
			out <- r
		}
		close(out)
	}()

	return out
}

// TrimTrailingEmptyLFLines removes empty lines at the end of the input.
// Lines are terminated by "\n".
func TrimTrailingEmptyLFLines(in <-chan rune) <-chan rune {
	out := make(chan rune)

	go func() {
		atLineStart := true
		pendingNewlines := 0

		for r := range in {
			if atLineStart && r == '\n' {
				pendingNewlines++
				continue
			}

			for ; pendingNewlines > 0; pendingNewlines-- {
				out <- '\n'
			}
			out <- r
			atLineStart = r == '\n'
		}

		close(out)
	}()

	return out
}
