package textproc

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
