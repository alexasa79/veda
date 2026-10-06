package editor

import (
	"iter"
	"unicode/utf8"
)

// Display padding has a source boundary for hit testing, but is never inserted
// into the buffer. Real characters retain their original byte positions.
type markdownDisplayRune struct {
	r         rune
	source    int
	column    int
	grid      bool
	synthetic bool
}

type markdownTableDisplay struct {
	startLine int
	endLine   int
	glyphs    []markdownDisplayRune
	columns   []int
}

func markdownDisplaySeparator(table markdownTable, row int) bool {
	_, separator := markdownTableRowSeparator(table.rows[row], len(table.alignments))
	return separator && !table.rows[row].continues && (row == 0 || !table.rows[row-1].continues)
}

func alignMarkdownTables(source []string, lines []markdownPresentedLine) {
	for delimiter := 0; delimiter < len(source); delimiter++ {
		table, found := markdownTableAtDelimiter(source, delimiter)
		if !found {
			continue
		}
		delimiter = table.endLine
		if !table.valid {
			continue
		}
		widths := make([]int, len(table.alignments))
		continued := false
		for _, row := range table.rows {
			continued = continued || row.continues
		}
		for col, alignment := range table.alignments {
			widths[col] = markdownTableMinimumWidth(alignment)
		}
		for rowIndex, row := range table.rows {
			separator := markdownDisplaySeparator(table, rowIndex)
			if separator && !continued {
				continue
			}
			line := &lines[table.startLine+rowIndex]
			for col, cell := range row.cells {
				// Alt+Q encodes its chosen widths in the delimiter. Preserve
				// these for explicitly continued rows instead of shrinking them.
				widths[col] = max(widths[col], line.displayWidth(cell.contentStart, cell.contentEnd))
			}
		}
		for rowIndex, row := range table.rows {
			lines[table.startLine+rowIndex].buildTableDisplay(table, rowIndex, row, widths)
		}
	}
}

func (line *markdownPresentedLine) buildTableDisplay(table markdownTable, rowIndex int, row markdownTableRow, widths []int) {
	display := &markdownTableDisplay{startLine: table.startLine, endLine: table.endLine, columns: make([]int, len(line.source)+1)}
	for i := range display.columns {
		display.columns[i] = -1
	}
	column := 0
	add := func(r rune, source int, grid, synthetic bool) {
		display.glyphs = append(display.glyphs, markdownDisplayRune{r, source, column, grid, synthetic})
		if !synthetic {
			display.columns[source] = column
		}
		column += wrapRuneWidth(r)
	}
	for pos, r := range table.rows[0].indent {
		add(r, min(pos, len(row.indent)), false, pos >= len(row.indent))
	}
	separator := markdownDisplaySeparator(table, rowIndex)
	border := func(col int) rune {
		if !separator {
			return '┃'
		}
		if rowIndex == 0 {
			if col == 0 {
				return '┏'
			}
			if col == len(widths) {
				return '┓'
			}
			return '┳'
		}
		if rowIndex == len(table.rows)-1 {
			if col == 0 {
				return '┗'
			}
			if col == len(widths) {
				return '┛'
			}
			return '┻'
		}
		if col == 0 {
			return '┣'
		}
		if col == len(widths) {
			return '┫'
		}
		return '╋'
	}
	first := len(row.indent)
	add(border(0), first, true, line.source[first] != '|')
	for col, width := range widths {
		cell := markdownTableCell{segmentStart: len(line.source), segmentEnd: len(line.source), contentStart: len(line.source), contentEnd: len(line.source)}
		if col < len(row.cells) {
			cell = row.cells[col]
		}
		if separator {
			for i := 0; i < width+2; i++ {
				add('━', min(cell.segmentStart+i, max(cell.segmentStart, cell.segmentEnd-1)), true, false)
			}
		} else {
			padding := width - line.displayWidth(cell.contentStart, cell.contentEnd)
			left := 0
			switch table.alignments[col] {
			case markdownTableAlignRight:
				left = padding
			case markdownTableAlignCenter:
				left = padding / 2
			}
			for i := 0; i < left+1; i++ {
				add(' ', cell.contentStart, false, true)
			}
			for pos := cell.contentStart; pos < cell.contentEnd; {
				r, size := utf8.DecodeRuneInString(line.source[pos:])
				if presented, visible := line.presentedRune(pos, r); visible {
					add(presented, pos, false, false)
				}
				pos += size
			}
			for i := 0; i < padding-left+1; i++ {
				add(' ', cell.contentEnd, false, true)
			}
		}
		add(border(col+1), cell.segmentEnd, true, cell.segmentEnd == len(line.source))
	}
	// Continuation markers and other hidden source bytes map to the next
	// displayed boundary, just as hidden link markup does.
	display.columns[len(line.source)] = column
	for pos := len(line.source) - 1; pos >= 0; pos-- {
		if display.columns[pos] < 0 {
			display.columns[pos] = display.columns[pos+1]
		}
	}
	line.table = display
}

// Derive horizontal scrolling from the active cursor so all rows of its table
// stay aligned, without storing a separate scroll position.
func (e *Editor) markdownTableOffset(line *markdownPresentedLine) int {
	if line.table == nil {
		return 0
	}
	buf := e.CurrentBuffer()
	cursorLine := buf.CursorIterator.LineNumber()
	if cursorLine < line.table.startLine || cursorLine > line.table.endLine {
		return 0
	}
	cursor := e.markdownPresentedLine(cursorLine)
	column := cursor.displayWidth(0, buf.CursorX)
	available := max(1, e.wrapScreenWidth()-buf.LineNumberWidth)
	return max(0, column-max(0, available-2))
}

func (line *markdownPresentedLine) renderedRunes(text string, start int) iter.Seq[markdownDisplayRune] {
	return func(yield func(markdownDisplayRune) bool) {
		if line != nil && line.table != nil {
			end := start + len(text)
			for _, glyph := range line.table.glyphs {
				if glyph.source < start || glyph.source > end || (glyph.source == end && end != len(line.source)) {
					continue
				}
				if !yield(glyph) {
					return
				}
			}
			return
		}
		for pos, r := range text {
			if line != nil {
				var visible bool
				r, visible = line.presentedRune(start+pos, r)
				if !visible {
					continue
				}
			}
			if !yield(markdownDisplayRune{r: r, source: start + pos}) {
				return
			}
		}
	}
}
