// Package textproc provides text processing.
package textproc

import (
	"bufio"
	"errors"
	"io"
	"sort"
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

type textLowercaseT struct {
	text      []rune
	lowercase string
}

func getTextLowercaseT(text []rune) *textLowercaseT {
	lowerRunes := make([]rune, len(text))
	for i := range text {
		lowerRunes[i] = unicode.ToLower(text[i])
	}
	return &textLowercaseT{text, string(lowerRunes)}
}

func sortTextsI(texts [][]rune) {
	lowercaseTexts := make([]*textLowercaseT, len(texts))
	for i := range texts {
		lowercaseTexts[i] = getTextLowercaseT(texts[i])
	}
	sort.SliceStable(lowercaseTexts, func(i, j int) bool {
		return lowercaseTexts[i].lowercase < lowercaseTexts[j].lowercase
	})
	for i := range lowercaseTexts {
		texts[i] = lowercaseTexts[i].text
	}
}

// ConvertLineTerminatorsToLF converts "\r" and "\r\n" to "\n".
func ConvertLineTerminatorsToLF(in <-chan rune) <-chan rune {
	out := make(chan rune)

	go func() {
		skipNextLF := false

		for r := range in {
			if skipNextLF && r == '\n' {
				skipNextLF = false
				continue
			}
			if r == '\r' {
				out <- '\n'
				skipNextLF = true
			} else {
				out <- r
				skipNextLF = false
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

		for r := range in {
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

// TrimLFTrailingWhiteSpace removes white space at the end of lines.
// Lines are terminated by "\n".
func TrimLFTrailingWhiteSpace(in <-chan rune) <-chan rune {
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

// getLFLineContent returns the content of all lines
// excluding the line terminator "\n".
func getLFLineContent(in <-chan rune) [][]rune {
	var texts [][]rune
	var crt []rune

	for r := range in {
		if r == '\n' {
			texts = append(texts, crt)
			crt = nil
			continue
		}

		crt = append(crt, r)
	}

	if len(crt) > 0 {
		texts = append(texts, crt)
	}

	return texts
}

// SortLFLinesI reads the content of all lines
// excluding the line terminator "\n",
// sorts them in case-insensitive order and appends "\n" after each.
func SortLFLinesI(in <-chan rune) <-chan rune {
	out := make(chan rune)

	go func() {
		lines := getLFLineContent(in)
		sortTextsI(lines)

		for _, line := range lines {
			for _, r := range line {
				out <- r
			}
			out <- '\n'
		}
		close(out)
	}()

	return out
}
