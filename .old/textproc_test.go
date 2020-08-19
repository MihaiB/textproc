package textproc_test

func TestSortLFParagraphsI(t *testing.T) {
	for s, want := range map[string]*struct {
		runes []rune
		err   error
	}{
		"":       {nil, io.EOF},
		"\n\n\n": {nil, io.EOF},
		"Par1":   {[]rune("Par1\n"), io.EOF},
		"Hi\n👽\n\nalien\n\n\nspace": {
			[]rune("alien\n\nHi\n👽\n\nspace\n"), io.EOF},
		"NEON\n\nargon\n\nradon\nxenon\n\nHg\nHe\xffa": {
			[]rune("argon\n\nNEON\n\nradon\nxenon\n"),
			textproc.ErrInvalidUTF8},
	} {
		r := textproc.SortLFParagraphsI(textproc.NewReader(
			strings.NewReader(s)))
		checkReader(t, r, want.runes, want.err)
	}
}
