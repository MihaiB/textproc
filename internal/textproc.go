// Package internal contains textproc internals.
package internal

import (
	"github.com/MihaiB/textproc/v3"
	"strings"
	"testing"
)

// CheckRuneChannel checks the entire content of a rune channel.
func CheckRuneChannel(t *testing.T, runeCh <-chan rune, content string) {
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

// CheckTokenChannel checks the entire content of a token channel.
func CheckTokenChannel(t *testing.T, tokenCh <-chan []rune, strings []string) {
	for _, wantS := range strings {
		if gotR, ok := <-tokenCh; !ok {
			t.Fatalf("Token channel closed early, expected %#v",
				wantS)
		} else if gotS := string(gotR); gotS != wantS {
			t.Fatalf("Want %#v got %#v", wantS, gotS)
		}
	}

	if gotR, ok := <-tokenCh; ok {
		gotS := string(gotR)
		t.Fatalf("Unexpected additional token %#v", gotS)
	}
}

// CheckErrorChannel checks that an error channel delivers an expected value
// and is then closed.
func CheckErrorChannel(t *testing.T, errCh <-chan error, want error) {
	if got, ok := <-errCh; !ok {
		t.Fatal("Error channel closed early, expected", want)
	} else if got != want {
		t.Fatal("Want", want, "got", got)
	}

	if got, ok := <-errCh; ok {
		t.Fatal("Unexpected additional error:", got)
	}
}

// RuneProcessorTestCases maps input to corresponding output and error
// for a RuneProcessor.
type RuneProcessorTestCases = map[string]*struct {
	String string
	Error  error
}

// CheckRuneProcessor checks the RuneProcessor on the test cases.
func CheckRuneProcessor(t *testing.T, processor textproc.RuneProcessor, testcases RuneProcessorTestCases) {
	for in, want := range testcases {
		runeCh, errCh := processor(textproc.ReadRunes(strings.NewReader(in)))
		CheckRuneChannel(t, runeCh, want.String)
		CheckErrorChannel(t, errCh, want.Error)
	}
}

// TokenizerTestCases maps input to corresponding output and error
// for a Tokenizer.
type TokenizerTestCases = map[string]*struct {
	Strings []string
	Error   error
}

// CheckTokenizer checks the Tokenizer on the test cases.
func CheckTokenizer(t *testing.T, tokenizer textproc.Tokenizer, testcases TokenizerTestCases) {
	for in, want := range testcases {
		tokenCh, errCh := tokenizer(textproc.ReadRunes(strings.NewReader(in)))
		CheckTokenChannel(t, tokenCh, want.Strings)
		CheckErrorChannel(t, errCh, want.Error)
	}
}
