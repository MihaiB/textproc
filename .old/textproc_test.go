package textproc_test

import (
	"errors"
	"unicode/utf8"
)

func checkTokenReader(t *testing.T, r textproc.TokenReader,
	tokens [][]rune, err error) {
	for _, token := range tokens {
		gotToken, gotErr := r.ReadToken()
		if string(gotToken) != string(token) || gotErr != nil {
			t.Fatal("Want", token, nil, "got", gotToken, gotErr)
		}
	}
	const errCalls = 3
	for i := 0; i < errCalls; i++ {
		gotToken, gotErr := r.ReadToken()
		if len(gotToken) > 0 || gotErr != err {
			t.Fatal("Want empty token", err,
				"got", gotToken, gotErr)
		}
	}
}

func copyRunes(runes []rune) []rune {
	return append([]rune{}, runes...)
}

func copyTokens(tokens [][]rune) (result [][]rune) {
	for _, token := range tokens {
		result = append(result, copyRunes(token))
	}
	return
}

func TestNewIoReader(t *testing.T) {
	s := "🧐🚣🙊😱😜😎👽" + string([]rune{utf8.MaxRune})
	r := textproc.NewIoReader(textproc.NewReader(strings.NewReader(s)))
	buf := make([]byte, 3)
	var result []byte
	for {
		n, err := r.Read(buf)
		result = append(result, buf[:n]...)
		if n < len(buf) && len(result) != len(s) {
			t.Fatal("Incomplete intermediate read:", n,
				"bytes, want", len(buf))
		}
		if err != nil {
			if err != io.EOF {
				t.Fatal("want", io.EOF, "got", err)
			}
			break
		}
	}
	if string(result) != s {
		t.Fatal("want", s, "got", result)
	}

	for i := 0; i < 3; i++ {
		n, err := r.Read(buf)
		if n != 0 && err != io.EOF {
			t.Fatal("want", 0, io.EOF, "got", n, err)
		}
	}
}

func TestNewReaderFromRuneErrChanPanic(t *testing.T) {
	runeCh := make(chan rune)
	runes := []rune{'a', 'ê'}
	go textproc.SendRunesAndClose(copyRunes(runes), runeCh)

	errCh := make(chan error)
	go close(errCh)

	r := textproc.NewReaderFromRuneErrChan(runeCh, errCh)

	defer func() {
		gotI := recover()
		if gotS, ok := gotI.(string); !ok {
			t.Fatal("Not a string:", gotI)
		} else {
			want := "textproc: nil NewReaderFromRuneErrChan err"
			if gotS != want {
				t.Fatal("Want", want, "got", gotS)
			}
		}
	}()

	checkReader(t, r, runes, errors.New("dummy ignored value"))
}

func TestNewReaderFromRuneErrChan(t *testing.T) {
	for _, want := range []struct {
		runes []rune
		err   error
	}{
		{nil, io.EOF},
		{nil, textproc.ErrInvalidUTF8},
		{[]rune{'¡', 0, '⸘'}, io.EOF},
	} {
		runeCh := make(chan rune)
		go textproc.SendRunesAndClose(copyRunes(want.runes), runeCh)

		errCh := make(chan error)
		go func() {
			errCh <- want.err
		}()

		r := textproc.NewReaderFromRuneErrChan(runeCh, errCh)
		checkReader(t, r, want.runes, want.err)
	}
}

func TestNewTokenReaderFromTokenErrChanPanic(t *testing.T) {
	tokenCh := make(chan []rune)
	tokens := [][]rune{[]rune("Hi"), nil, []rune("✍")}
	go textproc.SendTokensAndClose(copyTokens(tokens), tokenCh)

	errCh := make(chan error)
	go close(errCh)

	r := textproc.NewTokenReaderFromTokenErrChan(tokenCh, errCh)

	defer func() {
		gotI := recover()
		if gotS, ok := gotI.(string); !ok {
			t.Fatal("Not a string:", gotI)
		} else {
			want := "textproc: nil NewTokenReaderFromTokenErrChan err"
			if gotS != want {
				t.Fatal("Want", want, "got", gotS)
			}
		}
	}()
	checkTokenReader(t, r, tokens, errors.New("dummy ignored value"))
}

func TestNewTokenReaderFromTokenErrChan(t *testing.T) {
	for _, want := range []struct {
		tokens [][]rune
		err    error
	}{
		{nil, io.EOF},
		{[][]rune{}, textproc.ErrInvalidUTF8},
		{[][]rune{{'𝄢'}, nil, []rune("𝓍÷𝓎")}, io.EOF},
	} {
		tokenCh := make(chan []rune)
		go textproc.SendTokensAndClose(copyTokens(want.tokens), tokenCh)

		errCh := make(chan error)
		go func() {
			errCh <- want.err
		}()

		r := textproc.NewTokenReaderFromTokenErrChan(tokenCh, errCh)
		checkTokenReader(t, r, want.tokens, want.err)
	}
}

func TestNewTokenReaderFromTokensErrPanic(t *testing.T) {
	defer func() {
		gotI := recover()
		if gotS, ok := gotI.(string); !ok {
			t.Fatal("Not a string:", gotI)
		} else {
			want := "textproc: nil NewTokenReaderFromTokenErrChan err"
			if gotS != want {
				t.Fatal("Want", want, "got", gotS)
			}
		}
	}()

	tokens := [][]rune{[]rune("hi")}
	r := textproc.NewTokenReaderFromTokensErr(copyTokens(tokens), nil)
	checkTokenReader(t, r, tokens, errors.New("dummy ignored value"))
}

func TestNewTokenReaderFromTokensErr(t *testing.T) {
	for _, tokens := range [][][]rune{
		nil,
		{[]rune(""), []rune("Hej"), []rune("världen")},
	} {
		err := errors.New("new error value")

		r := textproc.NewTokenReaderFromTokensErr(copyTokens(tokens), err)
		checkTokenReader(t, r, tokens, err)
	}
}

func TestNewReaderFromTokenReader(t *testing.T) {
	for _, tokens := range [][][]rune{
		nil,
		{[]rune("Êô"), []rune(""), nil, []rune("∮≡")},
		{[]rune("ab"), nil, nil, []rune{}},
	} {
		var want []rune
		for _, token := range tokens {
			want = append(want, token...)
		}

		err := errors.New("new error value")

		tr := textproc.NewTokenReaderFromTokensErr(copyTokens(tokens), err)
		r := textproc.NewReaderFromTokenReader(tr)
		checkReader(t, r, want, err)
	}
}

func TestReadAllTokens(t *testing.T) {
	for s, want := range map[string]struct {
		tokens [][]rune
		err    error
	}{
		"":        {nil, io.EOF},
		"»\n[}\n": {[][]rune{{'»'}, {'[', '}'}}, io.EOF},
	} {
		lineContentR := textproc.LFLineContent(textproc.NewReader(strings.NewReader(s)))
		gotTokens, gotErr := textproc.ReadAllTokens(lineContentR)
		if gotErr != want.err {
			t.Fatal("want", want.err, "got", gotErr)
		}
		if len(gotTokens) != len(want.tokens) {
			t.Fatal("want", len(want.tokens), "tokens, got",
				len(gotTokens))
		}
		for i, gotToken := range gotTokens {
			wantToken := want.tokens[i]
			if string(gotToken) != string(wantToken) {
				t.Fatal("want", wantToken, "got", gotToken)
			}
		}
	}
}

func TestLFLines(t *testing.T) {
	for s, want := range map[string]*struct {
		runes []rune
		err   error
	}{
		"":                  {nil, io.EOF},
		"\ra\r\rb\r\nc\n\r": {[]rune("\na\n\nb\nc\n\n"), io.EOF},
		"•\r\r\n\r≡":        {[]rune("•\n\n\n≡"), io.EOF},
	} {
		r := textproc.LFLines(textproc.NewReader(strings.NewReader(s)))
		checkReader(t, r, want.runes, want.err)
	}
}

func TestLFLineContent(t *testing.T) {
	for s, want := range map[string]*struct {
		tokens [][]rune
		err    error
	}{
		"":          {nil, io.EOF},
		"α":         {[][]rune{[]rune("α")}, io.EOF},
		"\r\nβè\n":  {[][]rune{[]rune("\r"), []rune("βè")}, io.EOF},
		"\n\nz":     {[][]rune{nil, nil, []rune("z")}, io.EOF},
		"ζ\nξ\xffa": {[][]rune{{'ζ'}}, textproc.ErrInvalidUTF8},
	} {
		textprocReader := textproc.NewReader(strings.NewReader(s))
		r := textproc.LFLineContent(textprocReader)
		checkTokenReader(t, r, want.tokens, want.err)
	}
}

func TestLFParagraphContent(t *testing.T) {
	for s, want := range map[string]*struct {
		tokens [][]rune
		err    error
	}{
		"": {nil, io.EOF},
		"a\r\nb\n \nc\n\nd": {[][]rune{
			[]rune("a\r\nb\n \nc"),
			[]rune("d")}, io.EOF},
		"\n\nδσ\n\n\n":  {[][]rune{[]rune("δσ")}, io.EOF},
		"ø\n\nb\nc\xff": {[][]rune{[]rune("ø")}, textproc.ErrInvalidUTF8},
	} {
		textprocReader := textproc.NewReader(strings.NewReader(s))
		r := textproc.LFParagraphContent(textprocReader)
		checkTokenReader(t, r, want.tokens, want.err)
	}
}

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

func TestTrimLFTrailingSpace(t *testing.T) {
	for s, want := range map[string]*struct {
		runes []rune
		err   error
	}{
		"":         {nil, io.EOF},
		"\xff3":    {nil, textproc.ErrInvalidUTF8},
		" \r\xff3": {nil, textproc.ErrInvalidUTF8},
		" @\xff3":  {[]rune(" @"), textproc.ErrInvalidUTF8},
		"\nT\t\r\n\n sp  \n\tmix \tz \t\r\n": {
			[]rune("\nT\n\n sp\n\tmix \tz\n"), io.EOF},
		"no final LF \t": {[]rune("no final LF"), io.EOF},
	} {
		r := textproc.TrimLFTrailingSpace(textproc.NewReader(
			strings.NewReader(s)))
		checkReader(t, r, want.runes, want.err)
	}
}

func TestNonEmptyFinalLF(t *testing.T) {
	for s, want := range map[string]*struct {
		runes []rune
		err   error
	}{
		"":            {nil, io.EOF},
		"a":           {[]rune("a\n"), io.EOF},
		"z\n":         {[]rune("z\n"), io.EOF},
		"a\xff1":      {[]rune("a"), textproc.ErrInvalidUTF8},
		"\nQ":         {[]rune("\nQ\n"), io.EOF},
		"One\nTwo\r":  {[]rune("One\nTwo\r\n"), io.EOF},
		"1\n2\n3\n\n": {[]rune("1\n2\n3\n\n"), io.EOF},
	} {
		r := textproc.NonEmptyFinalLF(textproc.NewReader(
			strings.NewReader(s)))
		checkReader(t, r, want.runes, want.err)
	}
}

func TestTrimLeadingLF(t *testing.T) {
	for s, want := range map[string]*struct {
		runes []rune
		err   error
	}{
		"":              {nil, io.EOF},
		"\n":            {nil, io.EOF},
		"\n\n\n":        {nil, io.EOF},
		"\n\nwy\x80z":   {[]rune("wy"), textproc.ErrInvalidUTF8},
		"ab\nc":         {[]rune("ab\nc"), io.EOF},
		"\n\nij\n\nk\n": {[]rune("ij\n\nk\n"), io.EOF},
	} {
		r := textproc.TrimLeadingLF(textproc.NewReader(
			strings.NewReader(s)))
		checkReader(t, r, want.runes, want.err)
	}
}

func TestTrimTrailingEmptyLFLines(t *testing.T) {
	for s, want := range map[string]*struct {
		runes []rune
		err   error
	}{
		"":                 {nil, io.EOF},
		"\n":               {nil, io.EOF},
		"\n\n":             {nil, io.EOF},
		"\n\n\n":           {nil, io.EOF},
		"\n\n\n\r":         {[]rune("\n\n\n\r"), io.EOF},
		"\n\n\nwz":         {[]rune("\n\n\nwz"), io.EOF},
		"a\n\n\n":          {[]rune("a\n"), io.EOF},
		"\n\na\n\nb":       {[]rune("\n\na\n\nb"), io.EOF},
		"x\n\ny\n\n":       {[]rune("x\n\ny\n"), io.EOF},
		"x\n\ny\n":         {[]rune("x\n\ny\n"), io.EOF},
		"a\n\nb\n\n\n\xcc": {[]rune("a\n\nb\n"), textproc.ErrInvalidUTF8},
		"a\n\nbc\xcc":      {[]rune("a\n\nbc"), textproc.ErrInvalidUTF8},
	} {
		r := textproc.TrimTrailingEmptyLFLines(textproc.NewReader(
			strings.NewReader(s)))
		checkReader(t, r, want.runes, want.err)
	}
}

func TestSortLFLinesI(t *testing.T) {
	for s, want := range map[string]struct {
		runes []rune
		err   error
	}{
		"":                       {nil, io.EOF},
		"Q\n\na\nrrr":            {[]rune("\na\nQ\nrrr\n"), io.EOF},
		"second\nfirst\nno\xcc.": {[]rune("first\nsecond\n"), textproc.ErrInvalidUTF8},
		"Bb\nbB\nBB\na\n":        {[]rune("a\nBb\nbB\nBB\n"), io.EOF},
	} {
		r := textproc.SortLFLinesI(textproc.NewReader(strings.NewReader(s)))
		checkReader(t, r, want.runes, want.err)
	}
}