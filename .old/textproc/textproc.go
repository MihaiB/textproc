package main

var (
	catalogue = map[string]*procEntry{
		"sortpi": {textproc.SortLFParagraphsI, "Sort paragraphs case-insensitive (LF end of line)"},
	}
)
