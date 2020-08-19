package textproc

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
