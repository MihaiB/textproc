package main

var (
	catalogue = map[string]*procEntry{
		"sortli": {textproc.SortLFLinesI, "Sort lines case-insensitive (LF end of line)"},
		"sortpi": {textproc.SortLFParagraphsI, "Sort paragraphs case-insensitive (LF end of line)"},
	}
)
