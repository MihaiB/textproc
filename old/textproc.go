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

// SendRunes sends the runes on the channel.
func SendRunes(runes []rune, ch chan<- rune) {
	for _, r := range runes {
		ch <- r
	}
}

// SendRunesAndClose sends the runes on the channel then closes it.
func SendRunesAndClose(runes []rune, ch chan<- rune) {
	SendRunes(runes, ch)
	close(ch)
}

// SendTokens sends the tokens on the channel.
func SendTokens(tokens [][]rune, ch chan<- []rune) {
	for _, t := range tokens {
		ch <- t
	}
}

// SendTokensAndClose sends the tokens on the channel then closes it.
func SendTokensAndClose(tokens [][]rune, ch chan<- []rune) {
	SendTokens(tokens, ch)
	close(ch)
}

// SendErrorAndClose sends the error on the channel then closes it.
func SendErrorAndClose(err error, ch chan<- error) {
	ch <- err
	close(ch)
}

type readerFromRuneErrChan struct {
	runeCh      <-chan rune
	errCh       <-chan error
	errReceived bool
	err         error
}

func (r *readerFromRuneErrChan) Read() (rune, error) {
	if rn, ok := <-r.runeCh; ok {
		return rn, nil
	}
	if !r.errReceived {
		r.err = <-r.errCh
		r.errReceived = true
	}
	if r.err == nil {
		panic("textproc: nil NewReaderFromRuneErrChan err")
	}
	return 0, r.err
}

// NewReaderFromRuneErrChan returns a new Reader
// which reads all the runes then one error and panics if the error is nil.
func NewReaderFromRuneErrChan(runeCh <-chan rune, errCh <-chan error) Reader {
	return &readerFromRuneErrChan{runeCh: runeCh, errCh: errCh}
}

type tokenReaderFromTokenErrChan struct {
	tokenCh     <-chan []rune
	errCh       <-chan error
	errReceived bool
	err         error
}

func (r *tokenReaderFromTokenErrChan) ReadToken() ([]rune, error) {
	if token, ok := <-r.tokenCh; ok {
		return token, nil
	}
	if !r.errReceived {
		r.err = <-r.errCh
		r.errReceived = true
	}
	if r.err == nil {
		panic("textproc: nil NewTokenReaderFromTokenErrChan err")
	}
	return nil, r.err
}

// NewTokenReaderFromTokenErrChan returns a new TokenReader
// which reads all the tokens then one error and panics if the error is nil.
func NewTokenReaderFromTokenErrChan(tokenCh <-chan []rune,
	errCh <-chan error) TokenReader {
	return &tokenReaderFromTokenErrChan{tokenCh: tokenCh, errCh: errCh}
}

// NewTokenReaderFromTokensErr returns a new TokenReader which reads the tokens
// then returns err or panics if err is nil.
func NewTokenReaderFromTokensErr(tokens [][]rune, err error) TokenReader {
	tokenCh := make(chan []rune)
	go SendTokensAndClose(tokens, tokenCh)

	errCh := make(chan error)
	go SendErrorAndClose(err, errCh)

	return NewTokenReaderFromTokenErrChan(tokenCh, errCh)
}

// NewReaderFromTokenReader returns a new Reader reading from r.
func NewReaderFromTokenReader(r TokenReader) Reader {
	runeCh := make(chan rune)
	errCh := make(chan error)
	go func() {
		for {
			token, err := r.ReadToken()
			if err != nil {
				close(runeCh)
				SendErrorAndClose(err, errCh)
				return
			}
			SendRunes(token, runeCh)
		}
	}()

	return NewReaderFromRuneErrChan(runeCh, errCh)
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

func lfLines(reader Reader, runeCh chan<- rune, errCh chan<- error) {
	var r rune
	var err error

	load := func() {
		r, err = reader.Read()
	}

	load()

	for {
		if err != nil {
			close(runeCh)
			SendErrorAndClose(err, errCh)
			return
		}

		if r != '\r' {
			runeCh <- r
			load()
			continue
		}

		runeCh <- '\n'
		load()
		if err == nil && r == '\n' {
			load()
		}
	}
}

// LFLines returns a new Reader reading from r,
// converting "\r" and "\r\n" to "\n".
func LFLines(r Reader) Reader {
	runeCh := make(chan rune)
	errCh := make(chan error)
	go lfLines(r, runeCh, errCh)
	return NewReaderFromRuneErrChan(runeCh, errCh)
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

func trimLFTrailingSpace(reader Reader,
	runeCh chan<- rune, errCh chan<- error) {
	getNext := func() ([]rune, error) {
		var runes []rune
		for {
			r, err := reader.Read()
			if err != nil {
				return nil, err
			}
			if r == '\n' {
				return []rune{r}, nil
			}
			runes = append(runes, r)
			if !unicode.IsSpace(r) {
				return runes, nil
			}
		}
	}

	for {
		runes, err := getNext()
		if err != nil {
			close(runeCh)
			SendErrorAndClose(err, errCh)
			return
		}
		SendRunes(runes, runeCh)
	}
}

// TrimLFTrailingSpace returns a new Reader which reads from r
// and removes white space at the end of lines. Lines are terminated by "\n".
func TrimLFTrailingSpace(r Reader) Reader {
	runeCh := make(chan rune)
	errCh := make(chan error)
	go trimLFTrailingSpace(r, runeCh, errCh)
	return NewReaderFromRuneErrChan(runeCh, errCh)
}

func nonEmptyFinalLF(r Reader, runeCh chan<- rune, errCh chan<- error) {
	prev := '\n'
	char, err := r.Read()

	for err == nil {
		runeCh <- char
		prev = char
		char, err = r.Read()
	}

	if err == io.EOF && prev != '\n' {
		runeCh <- '\n'
	}
	close(runeCh)
	SendErrorAndClose(err, errCh)
}

// NonEmptyFinalLF returns a new Reader which reads from r
// and ensures non-empty content ends with "\n".
func NonEmptyFinalLF(r Reader) Reader {
	runeCh := make(chan rune)
	errCh := make(chan error)
	go nonEmptyFinalLF(r, runeCh, errCh)
	return NewReaderFromRuneErrChan(runeCh, errCh)
}

type trimLeadingLF struct {
	r           Reader
	passthrough bool
}

func (r *trimLeadingLF) Read() (rune, error) {
	if !r.passthrough {
		char, err := r.r.Read()
		for err == nil && char == '\n' {
			char, err = r.r.Read()
		}

		r.passthrough = true
		return char, err
	}

	return r.r.Read()
}

// TrimLeadingLF returns a new Reader which reads from r
// and removes "\n" characters at the start of the input.
func TrimLeadingLF(r Reader) Reader {
	return &trimLeadingLF{r: r}
}

func trimTrailingEmptyLFLines(r Reader,
	runeCh chan<- rune, errCh chan<- error) {
	prev := '\n'
	pendingNewlines := 0

	for {
		char, err := r.Read()
		if err != nil {
			close(runeCh)
			SendErrorAndClose(err, errCh)
			return
		}

		if prev != '\n' || char != '\n' {
			for ; pendingNewlines > 0; pendingNewlines-- {
				runeCh <- '\n'
			}
			runeCh <- char
			prev = char
			continue
		}

		pendingNewlines++
	}
}

// TrimTrailingEmptyLFLines returns a new Reader which reads from r
// and removes empty lines at the end of the input.
// Lines are terminated by "\n".
func TrimTrailingEmptyLFLines(r Reader) Reader {
	runeCh := make(chan rune)
	errCh := make(chan error)
	go trimTrailingEmptyLFLines(r, runeCh, errCh)
	return NewReaderFromRuneErrChan(runeCh, errCh)
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
