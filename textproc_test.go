package textproc_test

import (
	"github.com/MihaiB/textproc/v3"
	"github.com/MihaiB/textproc/v3/internal"
	"strings"
	"testing"
)

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
		internal.CheckRuneChannel(t, runeCh, want.str)
		internal.CheckErrorChannel(t, errCh, want.err)
	}
}

func TestConvertLineTerminatorsToLF(t *testing.T) {
	testcases := internal.RuneProcessorTestCases{
		"":                  {"", nil},
		"\ra\r\rb\r\nc\n\r": {"\na\n\nb\nc\n\n", nil},
		"•\r\r\n\r≡":        {"•\n\n\n≡", nil},
		"\r\r\r\r\r":        {"\n\n\n\n\n", nil},
		"⏎\r\xaa\r\n":       {"⏎\n", textproc.ErrInvalidUTF8},
	}
	internal.CheckRuneProcessor(t, textproc.ConvertLineTerminatorsToLF, testcases)
}

func TestEnsureFinalLFIfNonEmpty(t *testing.T) {
	testcases := internal.RuneProcessorTestCases{
		"":            {"", nil},
		"a":           {"a\n", nil},
		"z\n":         {"z\n", nil},
		"a\xff1":      {"a", textproc.ErrInvalidUTF8},
		"\nQ":         {"\nQ\n", nil},
		"One\nTwo\r":  {"One\nTwo\r\n", nil},
		"1\n2\n3\n\n": {"1\n2\n3\n\n", nil},
	}
	internal.CheckRuneProcessor(t, textproc.EnsureFinalLFIfNonEmpty, testcases)
}

func TestTrimLFTrailingWhiteSpace(t *testing.T) {
	testcases := internal.RuneProcessorTestCases{
		"":          {"", nil},
		"\xff3":     {"", textproc.ErrInvalidUTF8},
		" \r\xff3":  {"", textproc.ErrInvalidUTF8},
		" @\xff3":   {" @", textproc.ErrInvalidUTF8},
		" @  \xff3": {" @", textproc.ErrInvalidUTF8},
		"\nT\t\r\n\n sp  \n\tmix \tz \t\r\n": {
			"\nT\n\n sp\n\tmix \tz\n", nil},
		"no final LF \t": {"no final LF", nil},
	}
	internal.CheckRuneProcessor(t, textproc.TrimLFTrailingWhiteSpace, testcases)
}

func TestTrimLeadingEmptyLFLines(t *testing.T) {
	testcases := internal.RuneProcessorTestCases{
		"":              {"", nil},
		"\n":            {"", nil},
		"\n\n\n":        {"", nil},
		"\n\nwy\x80z":   {"wy", textproc.ErrInvalidUTF8},
		"ab\nc":         {"ab\nc", nil},
		"\n\nij\n\nk\n": {"ij\n\nk\n", nil},
	}
	internal.CheckRuneProcessor(t, textproc.TrimLeadingEmptyLFLines, testcases)
}

func TestTrimTrailingEmptyLFLines(t *testing.T) {
	testcases := internal.RuneProcessorTestCases{
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
	internal.CheckRuneProcessor(t, textproc.TrimTrailingEmptyLFLines, testcases)
}

func TestReadLFLineContent(t *testing.T) {
	testcases := internal.TokenizerTestCases{
		"":          {nil, nil},
		"α":         {[]string{"α"}, nil},
		"\r\nβè\n":  {[]string{"\r", "βè"}, nil},
		"\n\nz":     {[]string{"", "", "z"}, nil},
		"ζ\nξ\xffa": {[]string{"ζ"}, textproc.ErrInvalidUTF8},
	}
	internal.CheckTokenizer(t, textproc.ReadLFLineContent, testcases)
}

func TestSortLFLinesI(t *testing.T) {
	testcases := internal.RuneProcessorTestCases{
		"":                       {"", nil},
		"Q\n\na\nrrr":            {"\na\nQ\nrrr\n", nil},
		"second\nfirst\nno\xcc.": {"first\nsecond\n", textproc.ErrInvalidUTF8},
		"Bb\nbB\nBB\na\n":        {"a\nBb\nbB\nBB\n", nil},
		"bz\n\nA\n\n\nC":         {"\n\n\nA\nbz\nC\n", nil},
	}
	internal.CheckRuneProcessor(t, textproc.SortLFLinesI, testcases)
}

func TestReadLFParagraphContent(t *testing.T) {
	testcases := internal.TokenizerTestCases{
		"":                     {nil, nil},
		"a\r\nb\n \nc\n\nd":    {[]string{"a\r\nb\n \nc", "d"}, nil},
		"\n\nδσ\n\n\n":         {[]string{"δσ"}, nil},
		"\n\nδσ\n\n\n\nx\ny\n": {[]string{"δσ", "x\ny"}, nil},
		"ø\n\nb\nc\xff":        {[]string{"ø"}, textproc.ErrInvalidUTF8},
	}
	internal.CheckTokenizer(t, textproc.ReadLFParagraphContent, testcases)
}

func TestSortLFParagraphsI(t *testing.T) {
	testcases := internal.RuneProcessorTestCases{
		"":                          {"", nil},
		"\n\n\n":                    {"", nil},
		"Par1":                      {"Par1\n", nil},
		"Hi\n👽\n\nalien\n\n\nspace": {"alien\n\nHi\n👽\n\nspace\n", nil},
		"NEON\n\nargon\n\nradon\nxenon\n\n\n\nKr\nHe\n\n": {
			"argon\n\nKr\nHe\n\nNEON\n\nradon\nxenon\n", nil},
		"NEON\n\nargon\n\nradon\nxenon\n\nHg\nHe\xffa": {
			"argon\n\nNEON\n\nradon\nxenon\n",
			textproc.ErrInvalidUTF8},
	}
	internal.CheckRuneProcessor(t, textproc.SortLFParagraphsI, testcases)
}
