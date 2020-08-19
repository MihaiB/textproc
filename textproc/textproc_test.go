package main

import (
	"github.com/MihaiB/textproc/v2"
	"io"
	"strings"
	"testing"
)

func TestWrite(t *testing.T) {
	for _, tc := range []*struct {
		runes []rune
		err   error
	}{
		{[]rune{}, io.EOF},
		{nil, textproc.ErrInvalidUTF8},
		{[]rune("ø🚲🛥ô🐁"), io.EOF},
		{[]rune("∀ 𝒸: 𝒶≥𝒷."), io.ErrUnexpectedEOF},
	} {
		runeCh := make(chan rune)
		errCh := make(chan error)
		go func() {
			for _, r := range tc.runes {
				runeCh <- r
			}
			close(runeCh)
			errCh <- tc.err
			close(errCh)
		}()

		builder := &strings.Builder{}

		gotErr := write(runeCh, errCh, builder)
		wantErr := tc.err
		if tc.err == io.EOF {
			wantErr = nil
		}
		if gotErr != wantErr {
			t.Fatal("Want", wantErr, "got", gotErr)
		}

		gotStr := builder.String()
		wantStr := string(tc.runes)
		if gotStr != wantStr {
			t.Fatal("Want", wantStr, "got", gotStr)
		}
	}
}
