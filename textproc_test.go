package textproc_test

import (
	"github.com/MihaiB/textproc/v3"
	"strings"
	"testing"
)

func checkRuneChan(t *testing.T, runeCh <-chan rune, content string) {
	for _, wantR := range []rune(content) {
		wantS := string([]rune{wantR})
		if gotR, ok := <-runeCh; !ok {
			t.Fatalf("Rune channel closed early, expected %#v",
				wantS)
		} else if gotS := string([]rune{gotR}); gotR != wantR {
			t.Fatalf("Want %#v got %#v", wantS, gotS)
		}
	}

	if gotR, ok := <-runeCh; ok {
		gotS := string([]rune{gotR})
		t.Fatalf("Unexpected additional rune %#v", gotS)
	}
}

func checkErrChan(t *testing.T, errCh <-chan error, want error) {
	if got, ok := <-errCh; !ok {
		t.Fatal("Error channel closed early, expected", want)
	} else if got != want {
		t.Fatal("Want", want, "got", got)
	}

	if got, ok := <-errCh; ok {
		t.Fatal("Unexpected additional error:", got)
	}
}

type runeProcessorTestCases = map[string]*struct {
	str string
	err error
}

func checkRuneProcessor(t *testing.T, processor textproc.RuneProcessor,
	testcases runeProcessorTestCases) {
	for in, want := range testcases {
		runeCh, errCh := textproc.ReadRunes(strings.NewReader(in))
		runeCh, errCh = processor(runeCh, errCh)
		checkRuneChan(t, runeCh, want.str)
		checkErrChan(t, errCh, want.err)
	}
}

func TestReadRunes(t *testing.T) {
	for in, want := range map[string]*struct {
		str string
		err error
	}{
		"":            {"", nil},
		"\x80a":       {"", textproc.ErrInvalidUTF8},
		"a•🧐/":        {"a•🧐/", nil},
		"@\uFFFD\t":   {"@\uFFFD\t", nil},
		"=•\xf0\x9f!": {"=•", textproc.ErrInvalidUTF8},
	} {
		runeCh, errCh := textproc.ReadRunes(strings.NewReader(in))
		checkRuneChan(t, runeCh, want.str)
		checkErrChan(t, errCh, want.err)
	}
}

func TestConvertLineTerminatorsToLF(t *testing.T) {
	testcases := runeProcessorTestCases{
		"":                  {"", nil},
		"\ra\r\rb\r\nc\n\r": {"\na\n\nb\nc\n\n", nil},
		"•\r\r\n\r≡":        {"•\n\n\n≡", nil},
		"\r\r\r\r\r":        {"\n\n\n\n\n", nil},
		"⏎\r\xaa\r\n":       {"⏎\n", textproc.ErrInvalidUTF8},
	}
	checkRuneProcessor(t, textproc.ConvertLineTerminatorsToLF, testcases)
}

func TestEnsureFinalLFIfNonEmpty(t *testing.T) {
	testcases := runeProcessorTestCases{
		"":            {"", nil},
		"a":           {"a\n", nil},
		"z\n":         {"z\n", nil},
		"a\xff1":      {"a", textproc.ErrInvalidUTF8},
		"\nQ":         {"\nQ\n", nil},
		"One\nTwo\r":  {"One\nTwo\r\n", nil},
		"1\n2\n3\n\n": {"1\n2\n3\n\n", nil},
	}
	checkRuneProcessor(t, textproc.EnsureFinalLFIfNonEmpty, testcases)
}

func TestTrimLFTrailingWhiteSpace(t *testing.T) {
	testcases := runeProcessorTestCases{
		"":          {"", nil},
		"\xff3":     {"", textproc.ErrInvalidUTF8},
		" \r\xff3":  {"", textproc.ErrInvalidUTF8},
		" @\xff3":   {" @", textproc.ErrInvalidUTF8},
		" @  \xff3": {" @", textproc.ErrInvalidUTF8},
		"\nT\t\r\n\n sp  \n\tmix \tz \t\r\n": {
			"\nT\n\n sp\n\tmix \tz\n", nil},
		"no final LF \t": {"no final LF", nil},
	}
	checkRuneProcessor(t, textproc.TrimLFTrailingWhiteSpace, testcases)
}

func TestTrimLeadingEmptyLFLines(t *testing.T) {
	testcases := runeProcessorTestCases{
		"":              {"", nil},
		"\n":            {"", nil},
		"\n\n\n":        {"", nil},
		"\n\nwy\x80z":   {"wy", textproc.ErrInvalidUTF8},
		"ab\nc":         {"ab\nc", nil},
		"\n\nij\n\nk\n": {"ij\n\nk\n", nil},
	}
	checkRuneProcessor(t, textproc.TrimLeadingEmptyLFLines, testcases)
}

func TestTrimTrailingEmptyLFLines(t *testing.T) {
	testcases := runeProcessorTestCases{
		"":                 {"", nil},
		"\n":               {"", nil},
		"\n\n":             {"", nil},
		"\n\n\n":           {"", nil},
		"\n\n\n\r":         {"\n\n\n\r", nil},
		"\n\n\nwz":         {"\n\n\nwz", nil},
		"a\n\n\n":          {"a\n", nil},
		"\n\na\n\nb":       {"\n\na\n\nb", nil},
		"x\n\ny\n\n":       {"x\n\ny\n", nil},
		"x\n\ny\n":         {"x\n\ny\n", nil},
		"a\n\nb\n\n\n\xcc": {"a\n\nb\n", textproc.ErrInvalidUTF8},
		"a\n\nbc\xcc":      {"a\n\nbc", textproc.ErrInvalidUTF8},
	}
	checkRuneProcessor(t, textproc.TrimTrailingEmptyLFLines, testcases)
}
