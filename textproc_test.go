package textproc

import (
	"io"
	"strings"
	"testing"
)

func checkChannel(t *testing.T, runeCh <-chan rune, runes []rune) {
	for _, want := range runes {
		if got, ok := <-runeCh; !ok {
			t.Fatal("Rune channel closed early, expected", want)
		} else if got != want {
			t.Fatalf("Want %#v got %#v", string([]rune{want}),
				string([]rune{got}))
		}
	}

	if got, ok := <-runeCh; ok {
		t.Fatal("Unexpected additional rune:", got)
	}
}

func checkChannels(t *testing.T, runeCh <-chan rune, runes []rune, errCh <-chan error, err error) {
	checkChannel(t, runeCh, runes)

	if got, ok := <-errCh; !ok {
		t.Fatal("Error channel closed early, expected", err)
	} else if got != err {
		t.Fatal("Want", err, "got", got)
	}

	if got, ok := <-errCh; ok {
		t.Fatal("Unexpected additional error:", got)
	}
}

func checkProcessor(t *testing.T, p Processor, inOut map[string]string) {
	for in, out := range inOut {
		dry, errCh := Read(strings.NewReader(in))
		wet := p(dry)
		checkChannels(t, wet, []rune(out), errCh, io.EOF)
	}
}

func checkTexts(t *testing.T, gots [][]rune, wants []string) {
	if len(gots) != len(wants) {
		t.Fatal("Want", len(wants), "got", len(gots))
	}
	for i, want := range wants {
		got := string(gots[i])
		if got != want {
			t.Fatalf("Want %#v got %#v", want, got)
		}
	}
}

func TestRead(t *testing.T) {
	for s, want := range map[string]*struct {
		runes []rune
		err   error
	}{
		"":            {nil, io.EOF},
		"\x80a":       {nil, ErrInvalidUTF8},
		"a•🧐/":        {[]rune("a•🧐/"), io.EOF},
		"@\uFFFD\t":   {[]rune("@\uFFFD\t"), io.EOF},
		"=•\xf0\x9f!": {[]rune("=•"), ErrInvalidUTF8},
	} {
		runeCh, errCh := Read(strings.NewReader(s))
		checkChannels(t, runeCh, want.runes, errCh, want.err)
	}
}

func TestProcessorTypeMatch(*testing.T) {
	for range []Processor{
		ConvertLineTerminatorsToLF,
		EnsureFinalLFIfNonEmpty,
		TrimLFTrailingWhiteSpace,
		TrimLeadingEmptyLFLines,
		TrimTrailingEmptyLFLines,
		SortLFLinesI,
		SortLFParagraphsI,
	} {
	}
}

func TestConvertLineTerminatorsToLF(t *testing.T) {
	inOut := map[string]string{
		"":                  "",
		"\ra\r\rb\r\nc\n\r": "\na\n\nb\nc\n\n",
		"•\r\r\n\r≡":        "•\n\n\n≡",
		"\r\r\r\r\r":        "\n\n\n\n\n",
	}
	checkProcessor(t, ConvertLineTerminatorsToLF, inOut)
}

func TestEnsureFinalLFIfNonEmpty(t *testing.T) {
	inOut := map[string]string{
		"":            "",
		"a":           "a\n",
		"z\n":         "z\n",
		"\nQ":         "\nQ\n",
		"One\nTwo\r":  "One\nTwo\r\n",
		"1\n2\n3\n\n": "1\n2\n3\n\n",
	}
	checkProcessor(t, EnsureFinalLFIfNonEmpty, inOut)
}

func TestTrimLFTrailingWhiteSpace(t *testing.T) {
	inOut := map[string]string{
		"":                                   "",
		" @":                                 " @",
		"\nT\t\r\n\n sp  \n\tmix \tz \t\r\n": "\nT\n\n sp\n\tmix \tz\n",
		"no final LF \t":                     "no final LF",
	}
	checkProcessor(t, TrimLFTrailingWhiteSpace, inOut)
}

func TestTrimLeadingEmptyLFLines(t *testing.T) {
	inOut := map[string]string{
		"":              "",
		"\n":            "",
		"\n\n\n":        "",
		"\n\nwy-z":      "wy-z",
		"ab\nc":         "ab\nc",
		"\n\nij\n\nk\n": "ij\n\nk\n",
	}
	checkProcessor(t, TrimLeadingEmptyLFLines, inOut)
}

func TestTrimTrailingEmptyLFLines(t *testing.T) {
	inOut := map[string]string{
		"":             "",
		"\n":           "",
		"\n\n":         "",
		"\n\n\n":       "",
		"\n\n\n\r":     "\n\n\n\r",
		"\n\n\nwz":     "\n\n\nwz",
		"a\n\n\n":      "a\n",
		"\n\na\n\nb":   "\n\na\n\nb",
		"x\n\ny\n\n":   "x\n\ny\n",
		"x\n\ny\n":     "x\n\ny\n",
		"a\n\nb\n\n\n": "a\n\nb\n",
		"a\n\nbc":      "a\n\nbc",
	}
	checkProcessor(t, TrimTrailingEmptyLFLines, inOut)
}

func TestGetLFLineContent(t *testing.T) {
	for in, wants := range map[string][]string{
		"":         nil,
		"α":        {"α"},
		"\r\nβè\n": {"\r", "βè"},
		"\n\nz":    {"", "", "z"},
		"ζ\nξ a":   {"ζ", "ξ a"},
	} {
		c, _ := Read(strings.NewReader(in))
		texts := getLFLineContent(c)
		checkTexts(t, texts, wants)
	}
}

func TestSortLFLinesI(t *testing.T) {
	inOut := map[string]string{
		"":                       "",
		"Q\n\na\nrrr":            "\na\nQ\nrrr\n",
		"second\nfirst\nmiddle.": "first\nmiddle.\nsecond\n",
		"Bb\nbB\nBB\na\n":        "a\nBb\nbB\nBB\n",
		"bz\n\nA\n\n\nC":         "\n\n\nA\nbz\nC\n",
	}
	checkProcessor(t, SortLFLinesI, inOut)
}

func TestGetLFParagraphContent(t *testing.T) {
	for in, wants := range map[string][]string{
		"":                     nil,
		"a\r\nb\n \nc\n\nd":    {"a\r\nb\n \nc", "d"},
		"\n\nδσ\n\n\n":         {"δσ"},
		"\n\nδσ\n\n\n\nx\ny\n": {"δσ", "x\ny"},
		"ø\n\nb\nc\n":          {"ø", "b\nc"},
	} {
		c, _ := Read(strings.NewReader(in))
		texts := getLFParagraphContent(c)
		checkTexts(t, texts, wants)
	}
}

func TestSortLFParagraphsI(t *testing.T) {
	inOut := map[string]string{
		"":                          "",
		"\n\n\n":                    "",
		"Par1":                      "Par1\n",
		"Hi\n👽\n\nalien\n\n\nspace": "alien\n\nHi\n👽\n\nspace\n",
		"NEON\n\nargon\n\nradon\nxenon\n\n\n\nKr\nHe\n\n": "argon\n\nKr\nHe\n\nNEON\n\nradon\nxenon\n",
	}
	checkProcessor(t, SortLFParagraphsI, inOut)
}
