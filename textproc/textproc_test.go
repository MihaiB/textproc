package main

import (
	"github.com/MihaiB/textproc/v3"
	"github.com/MihaiB/textproc/v3/internal"
	"io"
	"strings"
	"testing"
)

func TestCatalogueKeys(t *testing.T) {
	if len(catalogueKeys) != len(catalogue) {
		t.Fatal("Want", len(catalogue), "got", len(catalogueKeys))
	}

	unique := map[string]struct{}{}
	for _, k := range catalogueKeys {
		if k != strings.ToLower(k) {
			t.Fatal("Key", k, "is not lowercase.",
				"Keys should be lowercase",
				"because they are sorted",
				"in case-sensitive order",
				"using sort.Strings(…).")
		}

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
	testcases := internal.RuneProcessorTestCases{
		"":      {"", nil},
		" \t":   {"", nil},
		"a \rb": {"a\nb\n", nil},
	}
	internal.CheckRuneProcessor(t, catalogue["norm"].runeProc, testcases)
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
	for _, tc := range []*struct {
		osArgs        []string
		processorsLen int
	}{
		{[]string{"cmd"}, 0},
		{[]string{"cmd", "lf"}, 1},
		{[]string{"cmd", "lf", "lf"}, 2},
		{[]string{"cmd", "lf", "sortpi", "lf"}, 3},
		{[]string{"cmd", "norm"}, 1},
	} {
		args, err := parseArgs(tc.osArgs)
		if err != nil {
			t.Fatal("Want", nil, "got", err)
		}
		if len(args.runeProcs) != tc.processorsLen {
			t.Fatal("Want", tc.processorsLen,
				"got", len(args.runeProcs))
		}
	}
}

func TestWrite(t *testing.T) {
	for _, tc := range []*struct {
		str string
		err error
	}{
		{"", nil},
		{"", textproc.ErrInvalidUTF8},
		{"ø🚲🛥ô🐁", nil},
		{"∀ 𝒸: 𝒶≥𝒷.", io.ErrUnexpectedEOF},
	} {
		runeCh := make(chan rune)
		errCh := make(chan error)
		go func() {
			for _, r := range []rune(tc.str) {
				runeCh <- r
			}
			close(runeCh)
			errCh <- tc.err
			close(errCh)
		}()

		builder := &strings.Builder{}

		gotErr := write(runeCh, errCh, builder)
		if gotErr != tc.err {
			t.Fatal("Want", tc.err, "got", gotErr)
		}

		gotStr := builder.String()
		if gotStr != tc.str {
			t.Fatalf("Want %#v got %#v", tc.str, gotStr)
		}
	}
}
