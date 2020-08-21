// Package textproc provides text processing.
//
// On a pair of channels (chan dataType, chan error)
// all data is transmitted then the data channel is closed
// then a single error is transmitted then the error channel is closed.
// The nil error indicates success.
// Any non-nil error (including io.EOF) indicates failure.
package textproc

import (
	"bufio"
	"errors"
	"io"
	"sort"
	"unicode"
	"unicode/utf8"
)

// ErrInvalidUTF8 is the error returned when the input is not valid UTF-8.
var ErrInvalidUTF8 = errors.New("invalid UTF-8")

const runeErrorSize = len(string(utf8.RuneError))

// A RuneProcessor consumes and produces runes.
type RuneProcessor = func(runeIn <-chan rune, errIn <-chan error) (
	runeOut <-chan rune, errOut <-chan error)

// A Tokenizer consumes runes and produces tokens.
type Tokenizer = func(runeIn <-chan rune, errIn <-chan error) (
	tokenOut <-chan []rune, errOut <-chan error)

type tokenLowercaseT struct {
	token     []rune
	lowercase string
}

func getTokenLowercaseT(token []rune) *tokenLowercaseT {
	lowerRunes := make([]rune, len(token))
	for i := range token {
		lowerRunes[i] = unicode.ToLower(token[i])
	}
	return &tokenLowercaseT{token, string(lowerRunes)}
}

func sortTokensI(tokens [][]rune) {
	lowercaseTokens := make([]*tokenLowercaseT, len(tokens))
	for i := range tokens {
		lowercaseTokens[i] = getTokenLowercaseT(tokens[i])
	}
	sort.SliceStable(lowercaseTokens, func(i, j int) bool {
		return lowercaseTokens[i].lowercase < lowercaseTokens[j].lowercase
	})
	for i := range lowercaseTokens {
		tokens[i] = lowercaseTokens[i].token
	}
}

func readRune(reader io.RuneReader) (rune, error) {
	r, size, err := reader.ReadRune()
	if err == nil && r == utf8.RuneError && size != runeErrorSize {
		err = ErrInvalidUTF8
	}
	return r, err
}

// ReadRunes reads the runes from r.
// It fails with ErrInvalidUTF8 if the input is not valid UTF-8.
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
	runeOut := make(chan rune)

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
	}()

	return runeOut, errIn
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
	runeOut := make(chan rune)

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
	}()

	return runeOut, errIn
}

// TrimLeadingEmptyLFLines removes empty lines at the start of the input.
// Lines are terminated by "\n".
func TrimLeadingEmptyLFLines(runeIn <-chan rune, errIn <-chan error) (
	<-chan rune, <-chan error) {
	runeOut := make(chan rune)

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
	}()

	return runeOut, errIn
}

// TrimTrailingEmptyLFLines removes empty lines at the end of the input.
// Lines are terminated by "\n".
func TrimTrailingEmptyLFLines(runeIn <-chan rune, errIn <-chan error) (
	<-chan rune, <-chan error) {
	runeOut := make(chan rune)

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
	}()

	return runeOut, errIn
}

// ReadLFLineContent reads the content of each line.
// The content does not include the line terminator.
// Lines are terminated by "\n".
func ReadLFLineContent(runeIn <-chan rune, errIn <-chan error) (
	<-chan []rune, <-chan error) {
	tokenOut, errOut := make(chan []rune), make(chan error)

	go func() {
		var crt []rune

		for r := range runeIn {
			if r == '\n' {
				tokenOut <- crt
				crt = nil
				continue
			}

			crt = append(crt, r)
		}

		err := <-errIn
		if err == nil && len(crt) > 0 {
			tokenOut <- crt
		}
		close(tokenOut)
		errOut <- err
		close(errOut)
	}()

	return tokenOut, errOut
}

// SortLFLinesI reads the content of all lines using ReadLFLineContent,
// sorts the items in case-insensitive order and adds "\n" after each.
func SortLFLinesI(runeIn <-chan rune, errIn <-chan error) (
	<-chan rune, <-chan error) {
	lineIn, errIn := ReadLFLineContent(runeIn, errIn)
	runeOut := make(chan rune)

	go func() {
		var lines [][]rune
		for line := range lineIn {
			lines = append(lines, line)
		}
		sortTokensI(lines)

		for _, line := range lines {
			for _, char := range line {
				runeOut <- char
			}
			runeOut <- '\n'
		}
		close(runeOut)
	}()

	return runeOut, errIn
}

// ReadLFParagraphContent reads the content of each paragraph.
// The content does not include the line terminator
// of the paragraph's last line.
//
// A paragraph consists of adjacent non-empty lines.
// Lines are terminated by "\n".
func ReadLFParagraphContent(runeIn <-chan rune, errIn <-chan error) (
	<-chan []rune, <-chan error) {
	lineIn, errIn := ReadLFLineContent(runeIn, errIn)
	tokenOut, errOut := make(chan []rune), make(chan error)

	go func() {
		var par []rune

		for line := range lineIn {
			if len(line) != 0 {
				if len(par) > 0 {
					par = append(par, '\n')
				}
				par = append(par, line...)
				continue
			}

			if len(par) != 0 {
				tokenOut <- par
				par = nil
			}
		}

		err := <-errIn
		if err == nil && len(par) != 0 {
			tokenOut <- par
		}
		close(tokenOut)
		errOut <- err
		close(errOut)
	}()

	return tokenOut, errOut
}

// SortLFParagraphsI reads the content of all paragraphs
// using ReadLFParagraphContent,
// sorts the items in case-insensitive order, joins them with "\n\n"
// and adds "\n" after the last one.
func SortLFParagraphsI(runeIn <-chan rune, errIn <-chan error) (
	<-chan rune, <-chan error) {
	parIn, errIn := ReadLFParagraphContent(runeIn, errIn)
	runeOut := make(chan rune)

	go func() {
		var paragraphs [][]rune
		for par := range parIn {
			paragraphs = append(paragraphs, par)
		}
		sortTokensI(paragraphs)

		for i, par := range paragraphs {
			if i > 0 {
				runeOut <- '\n'
			}
			for _, char := range par {
				runeOut <- char
			}
			runeOut <- '\n'
		}
		close(runeOut)
	}()

	return runeOut, errIn
}
