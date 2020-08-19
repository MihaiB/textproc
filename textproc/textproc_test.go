package main

import (
	"github.com/MihaiB/textproc/v2"
	"io"
	"strings"
	"testing"
)

func readAll(c <-chan rune) string {
	var runes []rune
	for {
		r, ok := <-c
		if !ok {
			return string(runes)
		}
		runes = append(runes, r)
	}
}

func checkError(t *testing.T, c <-chan error, err error) {
	if got, ok := <-c; !ok {
		t.Fatal("Error channel closed early, expected", err)
	} else if got != err {
		t.Fatal("Want", err, "got", got)
	}

	if got, ok := <-c; ok {
		t.Fatal("Unexpected additional error", got)
	}
}

func checkCatalogueEntry(t *testing.T, name string, inOut map[string]string) {
	for in, out := range inOut {
		runeCh, errCh := textproc.Read(strings.NewReader(in))
		if catEntry, ok := catalogue[name]; !ok {
			t.Fatal("Unknown processor:", name)
		} else {
			runeCh = catEntry.processor(runeCh)
		}

		got := readAll(runeCh)
		if got != out {
			t.Fatalf("Want %#v got %#v", out, got)
		}
		checkError(t, errCh, io.EOF)
	}
}

func TestCatalogueKeys(t *testing.T) {
	if len(catalogueKeys) != len(catalogue) {
		t.Fatal("Want", len(catalogue), "got", len(catalogueKeys))
	}

	unique := map[string]struct{}{}
	for _, k := range catalogueKeys {
		if _, ok := catalogue[k]; !ok {
			t.Fatal("Key", k, "not in catalogue")
		}
		if _, ok := unique[k]; ok {
			t.Fatal("Key", k, "not unique")
		}
		unique[k] = struct{}{}
	}

	for i := 0; i < len(catalogueKeys)-1; i++ {
		a, b := catalogueKeys[i], catalogueKeys[i+1]
		if a >= b {
			t.Fatal("catalogueKeys not sorted or not unique:",
				a, "≥", b)
		}
	}
}

func TestNorm(t *testing.T) {
	inOut := map[string]string{
		"":      "",
		" \t":   "",
		"a \rb": "a\nb\n",
	}
	checkCatalogueEntry(t, "norm", inOut)
}

func TestParseArgsNoPrgName(t *testing.T) {
	if args, err := parseArgs(nil); args != nil || err != errNoProgramName {
		t.Error("Want", nil, errNoProgramName, "got", args, err)
	}
}

func TestParseArgsUnknownProcessor(t *testing.T) {
	args, err := parseArgs([]string{"cmd", "lf", "myproc", "lf"})
	wantMsg := "unknown processor: myproc"
	if err == nil || err.Error() != wantMsg {
		t.Error("Want", wantMsg, "got", err)
	}
	if args != nil {
		t.Error("Want", nil, "got", args)
	}
}

func TestParseArgsProcs(t *testing.T) {
	for _, tc := range []struct {
		osArgs        []string
		processorsLen int
	}{
		{[]string{"cmd"}, 0},
		{[]string{"cmd", "lf"}, 1},
		{[]string{"cmd", "lf", "lf"}, 2},
		// TODO
		//{[]string{"cmd", "lf", "sortpi", "lf"}, 3},
		{[]string{"cmd", "norm"}, 1},
	} {
		args, err := parseArgs(tc.osArgs)
		if err != nil {
			t.Fatal("Want", nil, "got", err)
		}
		if len(args.processors) != tc.processorsLen {
			t.Fatal("Want", tc.processorsLen,
				"got", len(args.processors))
		}
	}
}

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
