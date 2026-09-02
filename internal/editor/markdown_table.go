package editor

import (
	"strings"
	"unicode/utf8"
	"veda/internal/buffer"
)

type markdownTableAlignment int

const (
	markdownTableAlignNone markdownTableAlignment = iota
	markdownTableAlignLeft
	markdownTableAlignRight
	markdownTableAlignCenter
)

type markdownTableCell struct {
	text         string
	segmentStart int
	segmentEnd   int
	contentStart int
	contentEnd   int
}

type markdownTableRow struct {
	indent string
	cells  []markdownTableCell
}

type markdownTable struct {
	startLine  int
	endLine    int
	rows       []markdownTableRow
	alignments []markdownTableAlignment
	valid      bool
}

func parseMarkdownTableRow(line string) (markdownTableRow, bool) {
	contentStart := len(line) - len(strings.TrimLeft(line, " \t"))
	contentEnd := len(strings.TrimRight(line, " \t"))
	if contentStart >= contentEnd {
		return markdownTableRow{}, false
	}

	pipes := markdownTablePipes(line[contentStart:contentEnd], contentStart)
	if len(pipes) == 0 {
		return markdownTableRow{}, false
	}

	leadingPipe := pipes[0] == contentStart
	trailingPipe := pipes[len(pipes)-1] == contentEnd-1
	segmentStart := contentStart
	pipeIndex := 0
	if leadingPipe {
		segmentStart++
		pipeIndex++
	}

	var cells []markdownTableCell
	for ; pipeIndex < len(pipes); pipeIndex++ {
		pipe := pipes[pipeIndex]
		cells = append(cells, newMarkdownTableCell(line, segmentStart, pipe))
		segmentStart = pipe + 1
	}
	if !trailingPipe {
		cells = append(cells, newMarkdownTableCell(line, segmentStart, contentEnd))
	}
	if len(cells) == 0 {
		return markdownTableRow{}, false
	}

	return markdownTableRow{indent: line[:contentStart], cells: cells}, true
}

func newMarkdownTableCell(line string, start, end int) markdownTableCell {
	raw := line[start:end]
	left := len(raw) - len(strings.TrimLeft(raw, " \t"))
	right := len(strings.TrimRight(raw, " \t"))
	if left > right {
		right = left
	}
	return markdownTableCell{
		text:         raw[left:right],
		segmentStart: start,
		segmentEnd:   end,
		contentStart: start + left,
		contentEnd:   start + right,
	}
}

func markdownTablePipes(line string, lineOffset int) []int {
	var pipes []int
	codeTicks := 0
	for i := 0; i < len(line); {
		switch line[i] {
		case '\\':
			if codeTicks == 0 && i+1 < len(line) {
				_, size := utf8.DecodeRuneInString(line[i+1:])
				i += 1 + size
				continue
			}
		case '`':
			run := 1
			for i+run < len(line) && line[i+run] == '`' {
				run++
			}
			if codeTicks == run {
				codeTicks = 0
			} else if codeTicks == 0 && hasClosingBacktickRun(line, i+run, run) {
				codeTicks = run
			}
			i += run
			continue
		case '|':
			if codeTicks == 0 {
				pipes = append(pipes, lineOffset+i)
			}
		}
		_, size := utf8.DecodeRuneInString(line[i:])
		i += size
	}
	return pipes
}

func hasClosingBacktickRun(line string, start, ticks int) bool {
	for i := start; i < len(line); {
		if line[i] != '`' {
			_, size := utf8.DecodeRuneInString(line[i:])
			i += size
			continue
		}
		run := 1
		for i+run < len(line) && line[i+run] == '`' {
			run++
		}
		if run == ticks {
			return true
		}
		i += run
	}
	return false
}

func parseMarkdownTableDelimiter(row markdownTableRow) ([]markdownTableAlignment, bool) {
	alignments := make([]markdownTableAlignment, len(row.cells))
	for i, cell := range row.cells {
		text := cell.text
		left := strings.HasPrefix(text, ":")
		right := strings.HasSuffix(text, ":")
		start := 0
		end := len(text)
		if left {
			start++
		}
		if right {
			end--
		}
		if start > end {
			return nil, false
		}
		text = text[start:end]
		if len(text) < 3 || strings.Trim(text, "-") != "" {
			return nil, false
		}
		switch {
		case left && right:
			alignments[i] = markdownTableAlignCenter
		case left:
			alignments[i] = markdownTableAlignLeft
		case right:
			alignments[i] = markdownTableAlignRight
		}
	}
	return alignments, true
}

func findMarkdownTable(lines []string, cursorLine int) (markdownTable, bool) {
	for delimiterLine := 1; delimiterLine < len(lines); delimiterLine++ {
		delimiter, ok := parseMarkdownTableRow(lines[delimiterLine])
		if !ok {
			continue
		}
		alignments, ok := parseMarkdownTableDelimiter(delimiter)
		if !ok || len(alignments) < 2 {
			continue
		}
		header, ok := parseMarkdownTableRow(lines[delimiterLine-1])
		if !ok {
			continue
		}

		rows := []markdownTableRow{header, delimiter}
		endLine := delimiterLine
		for endLine+1 < len(lines) {
			row, ok := parseMarkdownTableRow(lines[endLine+1])
			if !ok {
				break
			}
			rows = append(rows, row)
			endLine++
		}
		startLine := delimiterLine - 1
		if cursorLine < startLine || cursorLine > endLine {
			continue
		}

		valid := len(header.cells) == len(delimiter.cells)
		for _, row := range rows[2:] {
			if len(row.cells) > len(header.cells) {
				valid = false
			}
		}
		return markdownTable{
			startLine:  startLine,
			endLine:    endLine,
			rows:       rows,
			alignments: alignments,
			valid:      valid,
		}, true
	}
	return markdownTable{}, false
}

func formatMarkdownTable(table markdownTable) ([]string, [][]int) {
	widths := make([]int, len(table.rows[0].cells))
	for column, alignment := range table.alignments {
		widths[column] = markdownTableMinimumWidth(alignment)
	}
	for rowIndex, row := range table.rows {
		if _, separator := markdownTableRowSeparator(row, len(widths)); rowIndex == 1 || separator {
			continue
		}
		for column, cell := range row.cells {
			widths[column] = max(widths[column], markdownTableTextWidth(cell.text))
		}
	}

	lines := make([]string, len(table.rows))
	contentStarts := make([][]int, len(table.rows))
	indent := table.rows[0].indent
	for rowIndex, row := range table.rows {
		var line strings.Builder
		line.WriteString(indent)
		line.WriteString("|")
		contentStarts[rowIndex] = make([]int, len(widths))
		separatorAlignments, separator := markdownTableRowSeparator(row, len(widths))
		for column := range widths {
			text := ""
			if column < len(row.cells) {
				text = row.cells[column].text
			}
			leftPadding := 0
			if rowIndex == 1 || separator {
				alignment := table.alignments[column]
				if separator {
					alignment = separatorAlignments[column]
				}
				text = markdownTableDelimiter(widths[column], alignment)
			} else {
				padding := widths[column] - markdownTableTextWidth(text)
				switch table.alignments[column] {
				case markdownTableAlignRight:
					leftPadding = padding
				case markdownTableAlignCenter:
					leftPadding = padding / 2
				}
			}
			line.WriteString(" ")
			line.WriteString(strings.Repeat(" ", leftPadding))
			contentStarts[rowIndex][column] = line.Len()
			line.WriteString(text)
			line.WriteString(strings.Repeat(" ", widths[column]-markdownTableTextWidth(text)-leftPadding))
			line.WriteString(" |")
		}
		lines[rowIndex] = line.String()
	}
	return lines, contentStarts
}

func markdownTableRowSeparator(row markdownTableRow, columns int) ([]markdownTableAlignment, bool) {
	if len(row.cells) != columns {
		return nil, false
	}
	alignments, ok := parseMarkdownTableDelimiter(row)
	return alignments, ok
}

func markdownTableMinimumWidth(alignment markdownTableAlignment) int {
	switch alignment {
	case markdownTableAlignLeft, markdownTableAlignRight:
		return 4
	case markdownTableAlignCenter:
		return 5
	default:
		return 3
	}
}

func markdownTableDelimiter(width int, alignment markdownTableAlignment) string {
	switch alignment {
	case markdownTableAlignLeft:
		return ":" + strings.Repeat("-", width-1)
	case markdownTableAlignRight:
		return strings.Repeat("-", width-1) + ":"
	case markdownTableAlignCenter:
		return ":" + strings.Repeat("-", width-2) + ":"
	default:
		return strings.Repeat("-", width)
	}
}

func markdownTableTextWidth(text string) int {
	width := 0
	for _, r := range text {
		width += wrapRuneWidth(r)
	}
	return width
}

func markdownTableCursorX(row markdownTableRow, oldX int, newStarts []int, newLine string) int {
	for column, cell := range row.cells {
		if oldX > cell.segmentEnd {
			continue
		}
		offset := oldX - cell.contentStart
		if offset < 0 {
			offset = 0
		}
		if offset > len(cell.text) {
			offset = len(cell.text)
		}
		return min(newStarts[column]+offset, len(newLine))
	}
	return min(oldX, len(newLine))
}

func (e *Editor) formatMarkdownTableAtCursor() (bool, error) {
	if e.CurrentBuffer() == nil || !e.isMarkdownFile() {
		return false, nil
	}

	buf := e.CurrentBuffer()
	lines := make([]string, buf.LineCount)
	for line := range lines {
		lines[line] = buffer.NewBufferIteratorAtLine(buf, line).Line()
	}
	cursorLine := buf.CursorIterator.LineNumber()
	table, found := findMarkdownTable(lines, cursorLine)
	if !found {
		return false, nil
	}
	if !table.valid {
		return true, nil
	}

	newLines, newStarts := formatMarkdownTable(table)
	oldLines := append([]string(nil), lines[table.startLine:table.endLine+1]...)
	if strings.Join(oldLines, "\n") == strings.Join(newLines, "\n") {
		return true, nil
	}

	rowIndex := cursorLine - table.startLine
	newCursorX := markdownTableCursorX(table.rows[rowIndex], buf.CursorX, newStarts[rowIndex], newLines[rowIndex])
	viewLine := buf.ViewIterator.LineNumber()
	entry := &markdownTableHistoryEntry{
		startLine:  table.startLine,
		oldLines:   oldLines,
		newLines:   newLines,
		oldCursorX: buf.CursorX,
		oldCursorY: cursorLine,
		newCursorX: newCursorX,
		newCursorY: cursorLine,
	}
	if buf.History != nil {
		executed, err := buf.History.Execute(buf, entry)
		if err != nil {
			return true, err
		}
		if executed != nil {
			e.lastChange = executed
		}
	} else if err := entry.Execute(buf); err != nil {
		return true, err
	}

	buf.ViewIterator = buffer.NewBufferIteratorAtLine(buf, min(viewLine, buf.LineCount-1))
	if !e.inPasteMode {
		e.updateSyntaxHighlighting()
	}
	return true, nil
}

type markdownTableHistoryEntry struct {
	startLine  int
	oldLines   []string
	newLines   []string
	oldCursorX int
	oldCursorY int
	newCursorX int
	newCursorY int
}

func (entry *markdownTableHistoryEntry) Execute(buf *buffer.Buffer) error {
	replaceMarkdownTableLines(buf, entry.startLine, len(entry.oldLines), entry.newLines)
	buf.CursorIterator = buffer.NewBufferIteratorAtLine(buf, entry.newCursorY)
	buf.CursorX = entry.newCursorX
	return nil
}

func (entry *markdownTableHistoryEntry) Undo(buf *buffer.Buffer) error {
	replaceMarkdownTableLines(buf, entry.startLine, len(entry.newLines), entry.oldLines)
	buf.CursorIterator = buffer.NewBufferIteratorAtLine(buf, entry.oldCursorY)
	buf.CursorX = entry.oldCursorX
	return nil
}

func (entry *markdownTableHistoryEntry) Description() string {
	return "Format Markdown table"
}

func replaceMarkdownTableLines(buf *buffer.Buffer, startLine, lineCount int, lines []string) {
	start := buffer.NewBufferIteratorAtLine(buf, startLine).Position()
	endLine := startLine + lineCount - 1
	end := buffer.NewBufferIteratorAtLine(buf, endLine).LineEnd()
	buf.Delete(start, end)
	buf.Insert(start, strings.Join(lines, "\n"))
}
