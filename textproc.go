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

// A Tokenizer emits tokens.
type Tokenizer = func(<-chan rune) <-chan []rune

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

// EmitLFLineContent emits the content of each line
// (excluding the line terminator "\n") as a token.
func EmitLFLineContent(in <-chan rune) <-chan []rune {
	out := make(chan []rune)

	go func() {
		var crt []rune

		for r := range in {
			if r == '\n' {
				out <- crt
				crt = nil
				continue
			}

			crt = append(crt, r)
		}

		if len(crt) > 0 {
			out <- crt
		}
		close(out)
	}()

	return out
}

// SortLFLinesI reads the content of all lines
// excluding the line terminator "\n",
// sorts that content in case-insensitive order
// and adds "\n" after each item.
func SortLFLinesI(in <-chan rune) <-chan rune {
	out := make(chan rune)

	go func() {
		var lines [][]rune
		for line := range EmitLFLineContent(in) {
			lines = append(lines, line)
		}
		sortTextsI(lines)

		for _, line := range lines {
			for _, char := range line {
				out <- char
			}
			out <- '\n'
		}
		close(out)
	}()

	return out
}

// EmitLFParagraphContent emits the content of each paragraph
// (excluding the line terminator of the paragraph's last line) as a token.
//
// A paragraph consists of adjacent non-empty lines.
// Lines are terminated by "\n".
func EmitLFParagraphContent(in <-chan rune) <-chan []rune {
	out := make(chan []rune)

	go func() {
		var par []rune

		for line := range EmitLFLineContent(in) {
			if len(line) != 0 {
				if len(par) > 0 {
					par = append(par, '\n')
				}
				par = append(par, line...)
				continue
			}

			if len(par) != 0 {
				out <- par
				par = nil
			}
		}

		if len(par) != 0 {
			out <- par
		}
		close(out)
	}()

	return out
}

// SortLFParagraphsI reads the content of all paragraphs
// excluding the line terminator of a paragraph's last line,
// sorts that content in case-insensitive order,
// joins the items with "\n\n" and adds "\n" after the last item.
//
// A paragraph consists of adjacent non-empty lines.
// Lines are terminated by "\n".
func SortLFParagraphsI(in <-chan rune) <-chan rune {
	out := make(chan rune)

	go func() {
		var paragraphs [][]rune
		for par := range EmitLFParagraphContent(in) {
			paragraphs = append(paragraphs, par)
		}
		sortTextsI(paragraphs)

		for i, par := range paragraphs {
			if i > 0 {
				out <- '\n'
			}
			for _, char := range par {
				out <- char
			}
			out <- '\n'
		}
		close(out)
	}()

	return out
}
