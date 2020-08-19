package textproc_test

import (
	"github.com/MihaiB/textproc/v2"
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

func checkProcessor(t *testing.T, p textproc.Processor,
	inOut map[string]string) {
	for in, out := range inOut {
		dry, errCh := textproc.Read(strings.NewReader(in))
		wet := p(dry)
		checkChannels(t, wet, []rune(out), errCh, io.EOF)
	}
}

func TestRead(t *testing.T) {
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
		runeCh, errCh := textproc.Read(strings.NewReader(s))
		checkChannels(t, runeCh, want.runes, errCh, want.err)
	}
}

func TestProcessorTypeMatch(*testing.T) {
	for range []textproc.Processor{
		textproc.ConvertLineTerminatorsToLF,
		textproc.EnsureFinalLFIfNonEmpty,
		textproc.TrimLFTrailingSpaces,
		textproc.TrimLeadingEmptyLFLines,
		textproc.TrimTrailingEmptyLFLines,
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
	checkProcessor(t, textproc.ConvertLineTerminatorsToLF, inOut)
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
	checkProcessor(t, textproc.EnsureFinalLFIfNonEmpty, inOut)
}

func TestTrimLFTrailingSpaces(t *testing.T) {
	inOut := map[string]string{
		"":                                   "",
		" @":                                 " @",
		"\nT\t\r\n\n sp  \n\tmix \tz \t\r\n": "\nT\n\n sp\n\tmix \tz\n",
		"no final LF \t":                     "no final LF",
	}
	checkProcessor(t, textproc.TrimLFTrailingSpaces, inOut)
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
	checkProcessor(t, textproc.TrimLeadingEmptyLFLines, inOut)
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
	checkProcessor(t, textproc.TrimTrailingEmptyLFLines, inOut)
}
