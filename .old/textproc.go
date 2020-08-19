package textproc

import (
	"sort"
)

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

type lfLineContent struct {
	r   Reader
	err error
}

func (r *lfLineContent) ReadToken() ([]rune, error) {
	if r.err != nil {
		return nil, r.err
	}
	var token []rune
	for {
		var ch rune
		ch, r.err = r.r.Read()
		if r.err != nil {
			r.r = nil
			// Discard the partial line on error.
			// Otherwise the caller can't distinguish it
			// from a complete line ending in '\n' or EOF.
			if r.err == io.EOF && len(token) > 0 {
				return token, nil
			}
			return r.ReadToken()
		}
		if ch == '\n' {
			return token, nil
		}
		token = append(token, ch)
	}
}

// LFLineContent returns a new TokenReader reading the content of lines from r,
// excluding the line terminator "\n".
func LFLineContent(r Reader) TokenReader {
	return &lfLineContent{r: r}
}

type lfParagraphContent struct {
	r   TokenReader
	err error
}

func (r *lfParagraphContent) ReadToken() ([]rune, error) {
	if r.err != nil {
		return nil, r.err
	}

	var par []rune
	for {
		var line []rune
		line, r.err = r.r.ReadToken()
		if r.err != nil {
			r.r = nil
			// Discard the partial paragraph on error.
			// Otherwise the caller can't tell it is incomplete.
			if r.err == io.EOF && len(par) > 0 {
				return par, nil
			}
			return r.ReadToken()
		}
		if len(line) == 0 {
			if len(par) > 0 {
				return par, nil
			}
			continue
		}
		if len(par) > 0 {
			par = append(par, '\n')
		}
		par = append(par, line...)
	}
}

// LFParagraphContent returns a new TokenReader reading
// the content of paragraphs from r, excluding the final line terminator.
// A paragraph consists of adjacent non-empty lines terminated by "\n".
func LFParagraphContent(r Reader) TokenReader {
	return &lfParagraphContent{r: LFLineContent(r)}
}

// SortLFParagraphsI returns a new Reader which reads
// the content of all paragraphs from r using LFParagraphContent,
// sorts them in case-insensitive order,
// joins them with "\n\n" and adds "\n" after the last paragraph.
func SortLFParagraphsI(r Reader) Reader {
	parContents, err := ReadAllTokens(LFParagraphContent(r))
	sortTokensI(parContents)

	parSep := []rune{'\n', '\n'}
	result := make([][]rune, 2*len(parContents))
	for i, parContent := range parContents {
		result[2*i] = parContent
		result[2*i+1] = parSep
	}
	if len(parContents) > 0 {
		result[2*len(parContents)-1] = []rune{'\n'}
	}

	return NewReaderFromTokenReader(NewTokenReaderFromTokensErr(result, err))
}

// SortLFLinesI returns a new Reader which reads
// the content of all lines from r using LFLineContent,
// sorts them in case-insensitive order and appends "\n" after each.
func SortLFLinesI(r Reader) Reader {
	lines, err := ReadAllTokens(LFLineContent(r))
	sortTokensI(lines)
	tokens := make([][]rune, 2*len(lines))
	lineTerm := []rune{'\n'}
	for i := range lines {
		tokens[2*i] = lines[i]
		tokens[2*i+1] = lineTerm
	}
	return NewReaderFromTokenReader(NewTokenReaderFromTokensErr(tokens, err))
}
