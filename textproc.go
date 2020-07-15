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

// Reader reads runes.
type Reader interface {
	Read() (rune, error)
}

// TokenReader reads tokens.
// It does not keep a reference to the returned token's uderlying storage.
// If err != nil callers must discard the returned token.
type TokenReader interface {
	ReadToken() (token []rune, err error)
}

type runeDecoder struct {
	r   io.RuneReader
	err error
}

func (dec *runeDecoder) Read() (rune, error) {
	if dec.err != nil {
		return 0, dec.err
	}

	r, size, err := dec.r.ReadRune()
	if err == nil && r == utf8.RuneError && size != runeErrorSize {
		err = ErrInvalidUTF8
	}
	if err != nil {
		dec.err = err
		dec.r = nil
		return dec.Read()
	}
	return r, nil
}

// NewReader returns a new Reader reading from r.
// The new Reader returns ErrInvalidUTF8 if the input is not valid UTF-8.
func NewReader(r io.Reader) Reader {
	return &runeDecoder{r: bufio.NewReader(r)}
}

type runeEncoder struct {
	r                Reader
	err              error
	buf              [utf8.UTFMax]byte
	bufStart, bufEnd int // first and after-last indexes in buf
}

func (enc *runeEncoder) Read(p []byte) (int, error) {
	written := 0
	for {
		if enc.bufStart < enc.bufEnd {
			if len(p) == 0 {
				return written, nil
			}
			n := copy(p, enc.buf[enc.bufStart:enc.bufEnd])
			enc.bufStart += n
			p = p[n:]
			written += n
			continue
		}
		if enc.err != nil {
			return written, enc.err
		}

		var ch rune
		ch, enc.err = enc.r.Read()
		if enc.err != nil {
			enc.r = nil
			continue
		}
		enc.bufStart = 0
		enc.bufEnd = utf8.EncodeRune(enc.buf[:], ch)
	}
}

// NewIoReader returns a new io.Reader reading from r.
func NewIoReader(r Reader) io.Reader {
	return &runeEncoder{r: r}
}

type lfLines struct {
	r      Reader
	loaded bool // ch and err are the next, unused result from r.Read()
	ch     rune
	err    error
}

func (r *lfLines) load() {
	r.ch, r.err = r.r.Read()
	if r.err != nil {
		r.r = nil
	}
	r.loaded = true
}

func (r *lfLines) Read() (rune, error) {
	if !r.loaded {
		r.load()
	}
	if r.err != nil {
		return 0, r.err
	}
	if r.ch != '\r' {
		r.loaded = false
		return r.ch, nil
	}

	r.load()
	if r.err == nil && r.ch == '\n' {
		r.loaded = false
	}
	return '\n', nil
}

// LFLines returns a new Reader reading from r,
// converting "\r" and "\r\n" to "\n".
func LFLines(r Reader) Reader {
	return &lfLines{r: r}
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

type sortLfParagraphsI struct {
	r         Reader
	processed bool
	result    [][]rune
	err       error
}

func (r *sortLfParagraphsI) process() {
	var parContents [][]rune
	parContents, r.err = ReadAllTokens(LFParagraphContent(r.r))
	r.r = nil
	sortTokensI(parContents)

	parSep := []rune{'\n', '\n'}
	result := make([][]rune, 2*len(parContents))
	for i, parContent := range parContents {
		result[2*i] = parContent
		result[2*i+1] = parSep
	}
	if len(parContents) > 0 {
		result[2*len(parContents)-1] = []rune{'\n'}
		// only set r.result to a non-nil value if its length is > 0
		r.result = result
	}

	r.processed = true
}

func (r *sortLfParagraphsI) readOne() rune {
	val := r.result[0][0]
	r.result[0] = r.result[0][1:]
	if len(r.result[0]) == 0 {
		r.result = r.result[1:]
		if len(r.result) == 0 {
			r.result = nil
		}
	}
	return val
}

func (r *sortLfParagraphsI) Read() (rune, error) {
	if !r.processed {
		r.process()
	}
	if r.result != nil {
		return r.readOne(), nil
	}
	return 0, r.err
}

// SortLFParagraphsI returns a new Reader which reads
// the content of all paragraphs from r using LFParagraphContent,
// sorts them in case-insensitive order,
// joins them with "\n\n" and adds "\n" after the last paragraph.
func SortLFParagraphsI(r Reader) Reader {
	return &sortLfParagraphsI{r: r}
}

type trimLfTrailingSpace struct {
	r    Reader
	err  error
	next []rune
}

func (r *trimLfTrailingSpace) Read() (rune, error) {
	if r.err != nil {
		return 0, r.err
	}

	if r.next != nil {
		val := r.next[0]
		r.next = r.next[1:]
		if len(r.next) == 0 {
			r.next = nil
		}
		return val, nil
	}

	var pending []rune
	for {
		var val rune
		if val, r.err = r.r.Read(); r.err != nil {
			r.r = nil
			return r.Read()
		}
		if val == '\n' {
			return val, nil
		}
		if unicode.IsSpace(val) {
			pending = append(pending, val)
			continue
		}

		if pending == nil {
			return val, nil
		}
		r.next = append(pending, val)
		return r.Read()
	}
}

// TrimLFTrailingSpace returns a new Reader which reads from r
// and removes white space at the end of lines. Lines are terminated by "\n".
func TrimLFTrailingSpace(r Reader) Reader {
	return &trimLfTrailingSpace{r: r}
}

type nonEmptyFinalLF struct {
	r        Reader
	err      error
	nonEmpty bool
	finalLF  bool
}

func (r *nonEmptyFinalLF) Read() (rune, error) {
	if r.err != nil {
		return 0, r.err
	}

	var val rune
	val, r.err = r.r.Read()
	if r.err != nil {
		r.r = nil
		if r.err == io.EOF && r.nonEmpty && !r.finalLF {
			r.finalLF = true
			return '\n', nil
		}
		return r.Read()
	}
	r.nonEmpty = true
	r.finalLF = val == '\n'
	return val, nil
}

// NonEmptyFinalLF returns a new Reader which reads from r
// and ensures non-empty content ends with "\n".
func NonEmptyFinalLF(r Reader) Reader {
	return &nonEmptyFinalLF{r: r}
}

type trimLeadingLF struct {
	r           Reader
	err         error
	afterPrefix bool
}

func (r *trimLeadingLF) Read() (rune, error) {
	for r.err == nil {
		var val rune
		val, r.err = r.r.Read()
		if r.err != nil {
			r.r = nil
			continue
		}
		if !r.afterPrefix {
			if val == '\n' {
				continue
			}
			r.afterPrefix = true
		}
		return val, nil
	}
	return 0, r.err
}

// TrimLeadingLF returns a new Reader which reads from r
// and removes "\n" characters at the start of the input.
func TrimLeadingLF(r Reader) Reader {
	return &trimLeadingLF{r: r}
}

type trimTrailingEmptyLFLines struct {
	r          Reader
	err        error
	withinLine bool
	next       []rune
}

func (r *trimTrailingEmptyLFLines) Read() (rune, error) {
	if r.err != nil {
		return 0, r.err
	}

	if r.next != nil {
		val := r.next[0]
		r.next = r.next[1:]
		if len(r.next) == 0 {
			r.next = nil
		}
		return val, nil
	}

	var val rune
	if val, r.err = r.r.Read(); r.err != nil {
		r.r = nil
		return r.Read()
	}
	if r.withinLine {
		r.withinLine = val != '\n'
		return val, nil
	}

	var pending []rune
	for {
		pending = append(pending, val)
		r.withinLine = val != '\n'
		if r.withinLine {
			r.next = pending
			return r.Read()
		}
		if val, r.err = r.r.Read(); r.err != nil {
			r.r = nil
			return r.Read()
		}
	}
}

// TrimTrailingEmptyLFLines returns a new Reader which reads from r
// and removes empty lines at the end of the input.
// Lines are terminated by "\n".
func TrimTrailingEmptyLFLines(r Reader) Reader {
	return &trimTrailingEmptyLFLines{r: r}
}

// ReadAllTokens reads tokens from r until it encounters an error
// and returns the tokens from all but the last call
// and the error from the last call.
func ReadAllTokens(r TokenReader) (tokens [][]rune, err error) {
	for {
		var token []rune
		if token, err = r.ReadToken(); err != nil {
			return
		}
		tokens = append(tokens, token)
	}
}

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
