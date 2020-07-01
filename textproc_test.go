package textproc_test

import (
	"github.com/MihaiB/textproc"
	"io"
	"strings"
	"testing"
	"unicode/utf8"
)

func checkReader(t *testing.T, r textproc.Reader, runes []rune, err error) {
	for _, rn := range runes {
		got, gotErr := r.Read()
		if got != rn || gotErr != nil {
			t.Fatal("Want", rn, nil, "got", got, gotErr)
		}
	}
	for i := 0; i < 3; i++ {
		got, gotErr := r.Read()
		if got != 0 || gotErr != err {
			t.Fatal("Want", 0, err, "got", got, gotErr)
		}
	}
}

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

func TestNewReader(t *testing.T) {
	for s, want := range map[string]*struct {
		runes []rune
		err   error
	}{
		"":            {nil, io.EOF},
		"\x80a":       {nil, textproc.ErrInvalidUTF8},
		"a•🧐/":        {[]rune("a•🧐/"), io.EOF},
		"@\uFFFD\t":   {[]rune("@\uFFFD\t"), io.EOF},
		"=•\xf0\x9f!": {[]rune("=•"), textproc.ErrInvalidUTF8},
	} {
		r := textproc.NewReader(strings.NewReader(s))
		checkReader(t, r, want.runes, want.err)
	}
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
		"":         {nil, io.EOF},
		"α":        {[][]rune{[]rune("α")}, io.EOF},
		"\r\nβè\n": {[][]rune{[]rune("\r"), []rune("βè")}, io.EOF},
		"\n\nz":    {[][]rune{nil, nil, []rune("z")}, io.EOF},
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
		"\n\nδσ\n\n\n": {[][]rune{[]rune("δσ")}, io.EOF},
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
		"NEON\n\nargon\n\nradon\nxenon\xffa": {
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

func TestFinalLF(t *testing.T) {
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
		"":           {nil, io.EOF},
		"\n":         {nil, io.EOF},
		"\n\n":       {nil, io.EOF},
		"\n\n\n":     {nil, io.EOF},
		"\n\n\n\r":   {[]rune("\n\n\n\r"), io.EOF},
		"\n\n\nwz":   {[]rune("\n\n\nwz"), io.EOF},
		"a\n\n\n":    {[]rune("a\n"), io.EOF},
		"\n\na\n\nb": {[]rune("\n\na\n\nb"), io.EOF},
		"x\n\ny\n\n": {[]rune("x\n\ny\n"), io.EOF},
		"x\n\ny\n":   {[]rune("x\n\ny\n"), io.EOF},
	} {
		r := textproc.TrimTrailingEmptyLFLines(textproc.NewReader(
			strings.NewReader(s)))
		checkReader(t, r, want.runes, want.err)
	}
}
