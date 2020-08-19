package main

var (
	normChain = []string{"lf", "trail", "trimlf", "nelf"}

	catalogue = map[string]*procEntry{
		"sortli": {textproc.SortLFLinesI, "Sort lines case-insensitive (LF end of line)"},
		"sortpi": {textproc.SortLFParagraphsI, "Sort paragraphs case-insensitive (LF end of line)"},
		"trail":  {textproc.TrimLFTrailingSpace, "Remove trailing whitespace (LF end of line)"},
		"trimlf": {func(r textproc.Reader) textproc.Reader {
			return textproc.TrimTrailingEmptyLFLines(textproc.TrimLeadingLF(r))
		}, "Trim leading and trailing empty lines (LF end of line)"},
	}
)
