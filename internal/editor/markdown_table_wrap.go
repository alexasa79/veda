package editor

import (
	"unicode"
	"unicode/utf8"
)

type markdownCellFragment struct {
	text  string
	start int // Byte offset in the complete cell.
}

// Keep links and code spans together, including whitespace and pipes inside them.
func markdownCellWords(text string) []markdownCellFragment {
	var words []markdownCellFragment
	links := markdownLinkRegex.FindAllStringIndex(text, -1)
	linkEnds := make(map[int]int, len(links))
	for _, link := range links {
		linkEnds[link[0]] = link[1]
	}
	start := -1
	for pos := 0; pos < len(text); {
		r, size := utf8.DecodeRuneInString(text[pos:])
		if unicode.IsSpace(r) {
			if start >= 0 {
				words = append(words, markdownCellFragment{text[start:pos], start})
				start = -1
			}
			pos += size
			continue
		}
		if start < 0 {
			start = pos
		}
		if end, ok := linkEnds[pos]; ok {
			pos = end
			continue
		}
		if r == '`' {
			ticks := 1
			for pos+ticks < len(text) && text[pos+ticks] == '`' {
				ticks++
			}
			if hasClosingBacktickRun(text, pos+ticks, ticks) {
				end := pos + ticks
				for end < len(text) {
					if text[end] != '`' {
						end++
						continue
					}
					run := 1
					for end+run < len(text) && text[end+run] == '`' {
						run++
					}
					end += run
					if run == ticks {
						break
					}
				}
				pos = end
				continue
			}
			pos += ticks
			continue
		}
		pos += size
	}
	if start >= 0 {
		words = append(words, markdownCellFragment{text[start:], start})
	}
	return words
}

func wrapMarkdownCell(text string, width int) []markdownCellFragment {
	words := markdownCellWords(text)
	if len(words) == 0 {
		return []markdownCellFragment{{text: text}}
	}
	var fragments []markdownCellFragment
	start := words[0].start
	end := start + len(words[0].text)
	for _, word := range words[1:] {
		nextEnd := word.start + len(word.text)
		if markdownTableTextWidth(text[start:nextEnd]) > width {
			fragments = append(fragments, markdownCellFragment{text[start:end], start})
			start = word.start
		}
		end = nextEnd
	}
	return append(fragments, markdownCellFragment{text[start:end], start})
}

// Join only explicitly continued rows. Empty cells in ordinary rows remain distinct.
func markdownTableLogicalRows(table markdownTable, cursorRow, cursorX int) ([]markdownTableRow, int, int, int) {
	var rows []markdownTableRow
	logicalCursorRow, cursorColumn, cursorOffset := 0, 0, 0
	for rowIndex, row := range table.rows {
		join := rowIndex > 0 && table.rows[rowIndex-1].continues
		if !join {
			rows = append(rows, markdownTableRow{indent: row.indent, cells: make([]markdownTableCell, len(table.alignments))})
		}
		logical := &rows[len(rows)-1]
		column := len(row.cells) - 1
		for c, cell := range row.cells {
			if cursorX <= cell.segmentEnd {
				column = c
				break
			}
		}
		for c, cell := range row.cells {
			prefix := len(logical.cells[c].text)
			if join && prefix > 0 && cell.text != "" {
				logical.cells[c].text += " "
				prefix++
			}
			if rowIndex == cursorRow && c == column {
				logicalCursorRow = len(rows) - 1
				cursorColumn = c
				cursorOffset = prefix + min(max(cursorX-cell.contentStart, 0), len(cell.text))
			}
			logical.cells[c].text += cell.text
		}
	}
	return rows, logicalCursorRow, cursorColumn, cursorOffset
}

func markdownTableWrapWidths(table markdownTable, rows []markdownTableRow, target int) []int {
	widths := make([]int, len(table.alignments))
	minimums := make([]int, len(widths))
	for c, alignment := range table.alignments {
		minimums[c] = markdownTableMinimumWidth(alignment)
		widths[c] = minimums[c]
	}
	for _, row := range rows {
		if alignments, separator := markdownTableRowSeparator(row, len(widths)); separator {
			for c, alignment := range alignments {
				minimums[c] = max(minimums[c], markdownTableMinimumWidth(alignment))
				widths[c] = max(widths[c], minimums[c])
			}
			continue
		}
		for c, cell := range row.cells {
			widths[c] = max(widths[c], markdownTableTextWidth(cell.text))
			for _, word := range markdownCellWords(cell.text) {
				minimums[c] = max(minimums[c], markdownTableTextWidth(word.text))
			}
		}
	}
	total := markdownTableTextWidth(rows[0].indent) + 1 + 3*len(widths)
	for _, width := range widths {
		total += width
	}
	if total <= target {
		return widths
	}
	// Match the visible table width; normal mode hides continuation markers.
	for total > target {
		column := -1
		for c, width := range widths {
			if width > minimums[c] && (column < 0 || width > widths[column]) {
				column = c
			}
		}
		if column < 0 {
			break
		}
		widths[column]--
		total--
	}
	return widths
}

func wrapMarkdownTable(table markdownTable, width, cursorRow, cursorX int) ([]string, int, int) {
	rows, logicalCursorRow, cursorColumn, cursorOffset := markdownTableLogicalRows(table, cursorRow, cursorX)
	widths := markdownTableWrapWidths(table, rows, width)
	var output []markdownTableRow
	newCursorRow, newCursorOffset := 0, 0
	for rowIndex, row := range rows {
		if _, separator := markdownTableRowSeparator(row, len(widths)); separator {
			if rowIndex == logicalCursorRow {
				newCursorRow = len(output)
			}
			output = append(output, row)
			continue
		}
		fragments := make([][]markdownCellFragment, len(widths))
		height := 1
		for c, cell := range row.cells {
			fragments[c] = wrapMarkdownCell(cell.text, widths[c])
			height = max(height, len(fragments[c]))
		}
		if rowIndex == logicalCursorRow {
			parts := fragments[cursorColumn]
			part := len(parts) - 1
			for i, fragment := range parts {
				if cursorOffset <= fragment.start+len(fragment.text) {
					part = i
					break
				}
			}
			newCursorRow = len(output) + part
			newCursorOffset = min(max(cursorOffset-parts[part].start, 0), len(parts[part].text))
		}
		for line := 0; line < height; line++ {
			physical := markdownTableRow{indent: row.indent, cells: make([]markdownTableCell, len(widths)), continues: line+1 < height}
			for c, parts := range fragments {
				if line < len(parts) {
					physical.cells[c].text = parts[line].text
				}
			}
			output = append(output, physical)
		}
	}
	table.rows = output
	lines, starts := formatMarkdownTable(table, widths)
	return lines, newCursorRow, starts[newCursorRow][cursorColumn] + newCursorOffset
}
