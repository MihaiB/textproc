package textproc_test

import (
	"math/rand"
	"testing"
	"time"
	"unicode/utf8"
)

var benchmarkText string

func init() {
	rand.Seed(time.Now().Unix())
	benchmarkText = generateText()
}

func TestBenchmarkText(t *testing.T) {
	if !utf8.ValidString(benchmarkText) {
		t.Fatal("not valid UTF-8")
	}

	minLineRunes := 1*(1+1) + 1
	maxLineRunes := 19*(19+19) + 1

	minParRunes := 1 * minLineRunes
	maxParRunes := 19 * maxLineRunes

	minTextRunes := 1000 * (minParRunes + 1)
	maxTextRunes := 3000 * (maxParRunes + 19)

	minTextBytes := minTextRunes
	maxTextBytes := utf8.UTFMax * maxTextRunes

	textBytes := len(benchmarkText)
	if textBytes < minTextBytes || textBytes > maxTextBytes {
		t.Fatal("Bad text length", textBytes, "want between",
			minTextBytes, "and", maxTextBytes)
	}
}
