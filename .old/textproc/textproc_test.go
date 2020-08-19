package main

func checkProc(t *testing.T, name string, in textproc.Reader,
	want []rune, err error) {
	catEntry, ok := catalogue[name]
	if !ok {
		t.Fatal("Unknown processor:", name)
	}

	r := catEntry.proc(in)
	for _, rn := range want {
		got, gotErr := r.Read()
		if got != rn || gotErr != nil {
			t.Fatal("Want", rn, nil, "got", got, gotErr)
		}
	}
	for i := 0; i < 3; i++ {
		got, gotErr := r.Read()
		if got != 0 || gotErr != err {
			t.Fatal("Want", 0, err, "got", got, gotErr)
		}
	}
}

func checkProcStr(t *testing.T, name, in, out string) {
	r := textproc.NewReader(strings.NewReader(in))
	checkProc(t, name, r, []rune(out), io.EOF)
}

func TestNormChain(t *testing.T) {
	for _, name := range normChain {
		if _, ok := catalogue[name]; !ok {
			t.Error("Unknown processor:", name)
		}
	}
}

func TestNorm(t *testing.T) {
	for _, tc := range []struct{ in, want string }{
		{"", ""},
		{" \t", ""},
		{"a \rb", "a\nb\n"},
	} {
		checkProcStr(t, "norm", tc.in, tc.want)
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
			t.Fatal("catalogueKeys not sorted (or not unique)",
				a, "≥", b)
		}
	}
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
		osArgs  []string
		procLen int
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
		if len(args.procs) != tc.procLen {
			t.Fatal("Want", tc.procLen, "got", len(args.procs))
		}
	}
}
