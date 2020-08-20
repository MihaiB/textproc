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
