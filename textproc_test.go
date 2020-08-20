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
