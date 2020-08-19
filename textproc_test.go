package textproc_test

import (
	"github.com/MihaiB/textproc/v2"
	"io"
	"strings"
	"testing"
)

func checkChannels(t *testing.T, runeCh <-chan rune, runes []rune, errCh <-chan error, err error) {
	for _, r := range runes {
		if got, ok := <-runeCh; !ok {
			t.Fatal("Rune channel closed early, expected", r)
		} else if got != r {
			t.Fatal("Want", r, "got", got)
		}
	}

	if got, ok := <-runeCh; ok {
		t.Fatal("Unexpected additional rune:", got)
	}

	if got, ok := <-errCh; !ok {
		t.Fatal("Error channel closed early, expected", err)
	} else if got != err {
		t.Fatal("Want", err, "got", got)
	}

	if got, ok := <-errCh; ok {
		t.Fatal("Unexpected additional error:", got)
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
