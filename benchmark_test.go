package textproc_test

import (
	"bufio"
	"github.com/MihaiB/textproc/v3"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode"
)

func generateRune() rune {
	return rune(rand.Intn(int(unicode.MaxRune) + 1))
}

func generateRunes(n int) string {
	sb := &strings.Builder{}
	for ; n > 0; n-- {
		sb.WriteRune(generateRune())
	}
	return sb.String()
}

func generateSpace() rune {
	spaces := []rune{' ', '\t'}
	return spaces[rand.Intn(len(spaces))]
}

func generateSpaces(n int) string {
	sb := &strings.Builder{}
	for ; n > 0; n-- {
		sb.WriteRune(generateSpace())
	}
	return sb.String()
}

func generateLine() string {
	sb := &strings.Builder{}
	for items := rand.Intn(19) + 1; items > 0; items-- {
		sb.WriteString(generateRunes(rand.Intn(19) + 1))
		sb.WriteString(generateSpaces(rand.Intn(19) + 1))
	}
	sb.WriteRune('\n')
	return sb.String()
}

func generateParagraph() string {
	sb := &strings.Builder{}
	for n := rand.Intn(19) + 1; n > 0; n-- {
		sb.WriteString(generateLine())
	}
	return sb.String()
}

func generateText() string {
	sb := &strings.Builder{}
	for pars := 1000 + rand.Intn(2001); pars > 0; pars-- {
		sb.WriteString(generateParagraph())
		for newlines := rand.Intn(19) + 1; newlines > 0; newlines-- {
			sb.WriteRune('\n')
		}
	}
	return sb.String()
}

func BenchmarkInternalGenerateText(b *testing.B) {
	for i := 0; i < b.N; i++ {
		generateText()
	}
}

type fileProcFunc = func(inFileName, outFileName string) error

func testPassthroughFileProcFunc(t *testing.T, fn fileProcFunc) {
	dir, err := os.MkdirTemp("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.RemoveAll(dir); err != nil {
			t.Fatal(err)
		}
	}()

	inFileName := filepath.Join(dir, "in")
	outFileName := filepath.Join(dir, "out")
	err = os.WriteFile(inFileName, []byte(benchmarkText), 0644)
	if err != nil {
		t.Fatal(err)
	}

	if err = fn(inFileName, outFileName); err != nil {
		t.Fatal(err)
	}
	outBytes, err := os.ReadFile(outFileName)
	if err != nil {
		t.Fatal(err)
	}
	outString := string(outBytes)
	if outString != benchmarkText {
		t.Fatal("the content of the output file is incorrect")
	}
}

func benchmarkFileProcFunc(b *testing.B, fn fileProcFunc) {
	dir, err := os.MkdirTemp("", "")
	if err != nil {
		b.Fatal(err)
	}
	defer func() {
		if err := os.RemoveAll(dir); err != nil {
			b.Fatal(err)
		}
	}()

	inFileName := filepath.Join(dir, "in")
	outFileName := filepath.Join(dir, "out")
	err = os.WriteFile(inFileName, []byte(benchmarkText), 0644)
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if err = fn(inFileName, outFileName); err != nil {
			b.Fatal(err)
		}
	}
}

func pipeFile(inFileName, outFileName string) (err error) {
	inFile, err := os.Open(inFileName)
	if err != nil {
		return
	}
	defer func() {
		if closeErr := inFile.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
	}()
	bufR := bufio.NewReader(inFile)

	outFile, err := os.Create(outFileName)
	if err != nil {
		return
	}
	defer func() {
		if closeErr := outFile.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
	}()
	bufW := bufio.NewWriter(outFile)
	defer func() {
		if flushErr := bufW.Flush(); flushErr != nil && err == nil {
			err = flushErr
		}
	}()

	if _, err = io.Copy(bufW, bufR); err != nil {
		return
	}
	return
}

func TestPipeFile(t *testing.T) {
	testPassthroughFileProcFunc(t, pipeFile)
}

func BenchmarkFileBaselinePipe(b *testing.B) {
	benchmarkFileProcFunc(b, pipeFile)
}

func getFileProcFunc(runeProcs ...textproc.RuneProcessor) fileProcFunc {
	return func(inFileName, outFileName string) (err error) {
		inFile, err := os.Open(inFileName)
		if err != nil {
			return
		}
		defer func() {
			closeErr := inFile.Close()
			if closeErr != nil && err == nil {
				err = closeErr
			}
		}()

		outFile, err := os.Create(outFileName)
		if err != nil {
			return
		}
		defer func() {
			closeErr := outFile.Close()
			if closeErr != nil && err == nil {
				err = closeErr
			}
		}()
		bufW := bufio.NewWriter(outFile)
		defer func() {
			flushErr := bufW.Flush()
			if flushErr != nil && err == nil {
				err = flushErr
			}
		}()

		runeCh, errCh := textproc.ReadRunes(inFile)
		for _, rp := range runeProcs {
			runeCh, errCh = rp(runeCh, errCh)
		}
		for r := range runeCh {
			if _, err = bufW.WriteRune(r); err != nil {
				return
			}
		}
		return <-errCh
	}
}

func TestGetFileProcFunc(t *testing.T) {
	testPassthroughFileProcFunc(t, getFileProcFunc())
}

func BenchmarkFileNoProcessing(b *testing.B) {
	benchmarkFileProcFunc(b, getFileProcFunc())
}

func BenchmarkFileLF(b *testing.B) {
	benchmarkFileProcFunc(b, getFileProcFunc(
		textproc.ConvertLineTerminatorsToLF))
}

func BenchmarkFileTidy(b *testing.B) {
	benchmarkFileProcFunc(b, getFileProcFunc(
		textproc.ConvertLineTerminatorsToLF,
		textproc.TrimLFTrailingWhiteSpace,
		textproc.TrimLeadingEmptyLFLines,
		textproc.TrimTrailingEmptyLFLines,
		textproc.EnsureFinalLFIfNonEmpty))
}

func BenchmarkFileSortLinesI(b *testing.B) {
	benchmarkFileProcFunc(b, getFileProcFunc(textproc.SortLFLinesI))
}

func BenchmarkFileSortParagraphsI(b *testing.B) {
	benchmarkFileProcFunc(b, getFileProcFunc(textproc.SortLFParagraphsI))
}
