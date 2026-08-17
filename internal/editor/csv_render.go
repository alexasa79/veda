package editor

import (
	"encoding/csv"
	"io"
	"path/filepath"
	"strconv"
	"strings"
	"unicode/utf8"
	"veda/internal/buffer"
	"veda/internal/commands"
	"veda/internal/theme"
	"veda/internal/ui"
)

const csvColumnGap = 2

type csvRenderCache struct {
	generation uint64
	table      csvTable
}

type csvTable struct {
	rows   []csvRow
	widths []int
}

type csvRow struct {
	raw    string
	fields []csvField
	valid  bool
}

type csvField struct {
	value         string
	start         int
	end           int
	sourceOffsets []int
}

type csvFieldRange struct {
	start int
	end   int
}

func (e *Editor) csvPresentationActive() bool {
	buf := e.CurrentBuffer()
	if buf == nil || buf.IsInternal || len(buf.DiffHints) > 0 || e.mode == commands.InsertMode {
		return false
	}
	return strings.EqualFold(filepath.Ext(buf.FilePath), ".csv")
}

func (e *Editor) csvTable() csvTable {
	buf := e.CurrentBuffer()
	if cached, ok := e.csvRenderCaches[buf]; ok && cached.generation == buf.Generation {
		return cached.table
	}

	table := parseCSVTable(buf.Rope.String())
	e.csvRenderCaches[buf] = csvRenderCache{generation: buf.Generation, table: table}
	return table
}

func parseCSVTable(content string) csvTable {
	lines := strings.Split(content, "\n")
	table := csvTable{rows: make([]csvRow, len(lines))}

	for lineNum, line := range lines {
		row := parseCSVRow(line)
		table.rows[lineNum] = row
		if !row.valid {
			continue
		}
		for column, field := range row.fields {
			for len(table.widths) <= column {
				table.widths = append(table.widths, 0)
			}
			width := csvTextWidth(field.value)
			if width > table.widths[column] {
				table.widths[column] = width
			}
		}
	}

	return table
}

func parseCSVRow(line string) csvRow {
	line = strings.TrimSuffix(line, "\r")
	if line == "" {
		return csvRow{raw: line, valid: true, fields: []csvField{{}}}
	}

	reader := csv.NewReader(strings.NewReader(line))
	reader.FieldsPerRecord = -1
	values, err := reader.Read()
	if err != nil {
		return csvRow{raw: line}
	}
	if _, err := reader.Read(); err != io.EOF {
		return csvRow{raw: line}
	}

	ranges, ok := csvFieldRanges(line)
	if !ok || len(ranges) != len(values) {
		return csvRow{raw: line}
	}

	fields := make([]csvField, len(values))
	for i, value := range values {
		r := ranges[i]
		offsets := csvSourceOffsets(line[r.start:r.end], r.start)
		if len(offsets) != utf8.RuneCountInString(value) {
			return csvRow{raw: line}
		}
		fields[i] = csvField{
			value:         value,
			start:         r.start,
			end:           r.end,
			sourceOffsets: offsets,
		}
	}
	return csvRow{raw: line, fields: fields, valid: true}
}

func csvFieldRanges(line string) ([]csvFieldRange, bool) {
	var ranges []csvFieldRange
	for start := 0; ; {
		end := start
		if start < len(line) && line[start] == '"' {
			end++
			closed := false
			for end < len(line) {
				if line[end] != '"' {
					end++
					continue
				}
				if end+1 < len(line) && line[end+1] == '"' {
					end += 2
					continue
				}
				end++
				closed = true
				break
			}
			if !closed || (end < len(line) && line[end] != ',') {
				return nil, false
			}
		} else {
			for end < len(line) && line[end] != ',' {
				end++
			}
		}

		ranges = append(ranges, csvFieldRange{start: start, end: end})
		if end == len(line) {
			return ranges, true
		}
		start = end + 1
		if start == len(line) {
			return append(ranges, csvFieldRange{start: start, end: start}), true
		}
	}
}

func csvSourceOffsets(raw string, lineStart int) []int {
	if len(raw) < 2 || raw[0] != '"' || raw[len(raw)-1] != '"' {
		return runeSourceOffsets(raw, lineStart)
	}

	var offsets []int
	for i := 1; i < len(raw)-1; {
		if raw[i] == '"' && i+1 < len(raw)-1 && raw[i+1] == '"' {
			offsets = append(offsets, lineStart+i)
			i += 2
			continue
		}
		_, size := utf8.DecodeRuneInString(raw[i:])
		offsets = append(offsets, lineStart+i)
		i += size
	}
	return offsets
}

func runeSourceOffsets(text string, start int) []int {
	offsets := make([]int, 0, utf8.RuneCountInString(text))
	for i := range text {
		if utf8.RuneStart(text[i]) {
			offsets = append(offsets, start+i)
		}
	}
	return offsets
}

func csvTextWidth(text string) int {
	width := 0
	for _, r := range text {
		if r == '\t' {
			width += 4
		} else {
			width++
		}
	}
	return width
}

func (t csvTable) row(lineNum int) csvRow {
	if lineNum < 0 || lineNum >= len(t.rows) {
		return csvRow{}
	}
	return t.rows[lineNum]
}

func (t csvTable) displayColumn(lineNum, sourceOffset int) int {
	row := t.row(lineNum)
	if !row.valid {
		return csvTextWidth(row.raw[:min(max(sourceOffset, 0), len(row.raw))])
	}

	x := 0
	for column, field := range row.fields {
		if sourceOffset < field.start {
			return x
		}
		if sourceOffset < field.end {
			fieldX := 0
			runes := []rune(field.value)
			for i, offset := range field.sourceOffsets {
				if sourceOffset <= offset {
					return x + fieldX
				}
				fieldX += csvRuneWidth(runes[i])
			}
			return x + csvTextWidth(field.value)
		}
		if sourceOffset == field.end {
			return x + t.widths[column]
		}
		x += t.widths[column] + csvColumnGap
	}
	return x
}

func (t csvTable) sourceOffset(lineNum, displayColumn int) int {
	row := t.row(lineNum)
	if !row.valid {
		return bytePositionAtCSVWidth(row.raw, displayColumn)
	}

	x := 0
	for column, field := range row.fields {
		columnEnd := x + t.widths[column]
		if displayColumn <= columnEnd {
			fieldX := 0
			for i, r := range []rune(field.value) {
				width := csvRuneWidth(r)
				if displayColumn < x+fieldX+width {
					return field.sourceOffsets[i]
				}
				fieldX += width
			}
			return field.end
		}
		x = columnEnd + csvColumnGap
	}
	return len(row.raw)
}

func bytePositionAtCSVWidth(text string, target int) int {
	width := 0
	for i, r := range text {
		if width+csvRuneWidth(r) > target {
			return i
		}
		width += csvRuneWidth(r)
	}
	return len(text)
}

func csvRuneWidth(r rune) int {
	if r == '\t' {
		return 4
	}
	return 1
}

func (e *Editor) calculateCSVLines() []WrappedLine {
	buf := e.CurrentBuffer()
	start := buf.ViewIterator.LineNumber()
	visible := e.getVisibleHeight()
	lines := make([]WrappedLine, 0, visible)
	iter := buffer.NewBufferIteratorAtLine(buf, start)

	for len(lines) < visible && !iter.IsEOF() {
		line := iter.Line()
		lines = append(lines, WrappedLine{
			LogicalLineNum: iter.LineNumber(),
			VisualLineNum:  len(lines),
			StartChar:      0,
			EndChar:        len(line),
			IsFirstLine:    true,
			Text:           line,
		})
		iter = iter.Next()
	}
	if len(lines) == 0 && buf.Rope.Length() == 0 {
		lines = append(lines, WrappedLine{LogicalLineNum: 0, IsFirstLine: true})
	}
	if len(lines) < visible && buf.Rope.Length() > 0 && buf.Rope.ByteAt(buf.Rope.Length()-1) == '\n' {
		lines = append(lines, WrappedLine{
			LogicalLineNum: buf.LineCount - 1,
			VisualLineNum:  len(lines),
			IsFirstLine:    true,
		})
	}
	return lines
}

func (e *Editor) csvCursorVisualPosition() (int, int) {
	buf := e.CurrentBuffer()
	line := buf.CursorIterator.LineNumber()
	visualLine := line - buf.ViewIterator.LineNumber()
	if visualLine < 0 {
		visualLine = 0
	}
	return visualLine, buf.LineNumberWidth + e.csvTable().displayColumn(line, buf.CursorX)
}

func (e *Editor) renderCSVPane() {
	buf := e.CurrentBuffer()
	table := e.csvTable()
	width, _ := e.screen.Size()
	contentStartY := e.getContentStartY()
	visibleHeight := e.getVisibleHeight()
	normalStyle := e.config.Theme.GetStyle(e.config.Theme.Normal)
	lineNumberStyle := e.config.Theme.GetStyle(e.config.Theme.LineNumber)

	e.screen.Clear()
	e.buildCSVVirtualCursorCells(table, contentStartY)

	iter := buffer.NewBufferIteratorAtLine(buf, buf.ViewIterator.LineNumber())
	drawn := 0
	for drawn < visibleHeight && !iter.IsEOF() {
		y := contentStartY + drawn
		for x := 0; x < width; x++ {
			e.screen.SetContent(x, y, ' ', nil, normalStyle)
		}
		e.drawCSVLineNumber(iter.LineNumber(), y)
		e.drawCSVRow(table, iter.LineNumber(), iter.Position(), y)
		drawn++
		iter = iter.Next()
	}
	if drawn == 0 && buf.Rope.Length() == 0 {
		y := contentStartY
		for x := 0; x < width; x++ {
			e.screen.SetContent(x, y, ' ', nil, normalStyle)
		}
		e.drawCSVLineNumber(0, y)
		e.drawCSVRow(table, 0, 0, y)
		drawn++
	}
	if drawn < visibleHeight && buf.Rope.Length() > 0 && buf.Rope.ByteAt(buf.Rope.Length()-1) == '\n' {
		y := contentStartY + drawn
		for x := 0; x < width; x++ {
			e.screen.SetContent(x, y, ' ', nil, normalStyle)
		}
		lineNum := buf.LineCount - 1
		e.drawCSVLineNumber(lineNum, y)
		e.drawCSVRow(table, lineNum, buf.Rope.Length(), y)
		drawn++
	}

	for ; drawn < visibleHeight; drawn++ {
		y := contentStartY + drawn
		for x := 0; x < width; x++ {
			e.screen.SetContent(x, y, ' ', nil, normalStyle)
		}
		for x := 0; x < buf.LineNumberWidth; x++ {
			e.screen.SetContent(x, y, '~', nil, lineNumberStyle)
		}
	}
}

func (e *Editor) drawCSVLineNumber(lineNum, y int) {
	buf := e.CurrentBuffer()
	digitsWidth := max(1, buf.LineNumberWidth-3)
	displayNum := lineNum + 1
	if lineNum != buf.CursorIterator.LineNumber() {
		difference := lineNum - buf.CursorIterator.LineNumber()
		if difference < 0 {
			difference = -difference
		}
		displayNum = difference
	}

	style := e.config.Theme.GetStyle(e.config.Theme.LineNumber)
	if lineNum == buf.CursorIterator.LineNumber() {
		style = e.config.Theme.GetStyle(e.config.Theme.CurrentLineNumber)
	}
	for x := 0; x < buf.LineNumberWidth; x++ {
		e.screen.SetContent(x, y, ' ', nil, style)
	}
	e.drawText(0, y, style, leftPadNumber(displayNum, digitsWidth), 0, nil, lineNum, 0, -1)

	wl := WrappedLine{LogicalLineNum: lineNum, IsFirstLine: true}
	e.drawGutterGitIndicator(wl, lineNum+1, digitsWidth, buf.LineNumberWidth, y, style)
	e.drawGutterDiagnosticsIndicator(wl, lineNum+1, digitsWidth, buf.LineNumberWidth, y, style)
}

func leftPadNumber(number, width int) string {
	text := strconv.Itoa(number)
	if len(text) >= width {
		return text
	}
	return strings.Repeat(" ", width-len(text)) + text
}

func (e *Editor) drawCSVRow(table csvTable, lineNum, lineStart, y int) {
	row := table.row(lineNum)
	startX := e.CurrentBuffer().LineNumberWidth
	header := lineNum == 0
	if row.raw == "" {
		return
	}
	if !row.valid {
		e.drawText(startX, y, e.csvCellStyle(0, startX, y, header), row.raw, lineStart, e.findSearchMatches(row.raw), lineNum, 0, -1)
		return
	}

	searchMatches := e.findSearchMatches(row.raw)
	x := startX
	for column, field := range row.fields {
		cellWidth := table.widths[column] + csvColumnGap
		for cellX := x; cellX < x+cellWidth; cellX++ {
			e.screen.SetContent(cellX, y, ' ', nil, e.csvCellStyle(column, cellX, y, header))
		}

		fieldX := x
		for i, r := range []rune(field.value) {
			sourceX := field.sourceOffsets[i]
			style := e.csvSourceStyle(column, sourceX, lineStart, searchMatches, fieldX, y, header)
			if r == '\t' {
				for j := 0; j < 4; j++ {
					e.screen.SetContent(fieldX, y, ' ', nil, style)
					fieldX++
				}
			} else {
				e.screen.SetContent(fieldX, y, r, nil, style)
				fieldX++
			}
		}
		x += cellWidth
	}
}

func (e *Editor) csvCellStyle(column, x, y int, header bool) ui.Style {
	style := e.config.Theme.GetStyle(e.config.Theme.Normal)
	_, normalBg, _ := style.Decompose()
	if header {
		style = style.Background(csvHeaderBackground(normalBg))
		style.Attrs |= ui.AttrBold
	} else {
		style = style.Background(csvColumnBackground(normalBg, column))
	}
	if e.hasVirtualCursorCell(x, y) {
		style = theme.ApplyBackgroundStyle(style, e.config.Theme.GetStyle(e.config.Theme.VirtualCursor))
	}
	return style
}

func (e *Editor) csvSourceStyle(column, sourceX, lineStart int, searchMatches []int, x, y int, header bool) ui.Style {
	style := e.csvCellStyle(column, x, y, header)
	selStart, selEnd, hasSelection := e.getSelectionRange()
	absolute := lineStart + sourceX
	if hasSelection && absolute >= selStart && absolute < selEnd {
		return theme.ApplyBackgroundStyle(style, e.config.Theme.GetStyle(e.config.Theme.Selection))
	}
	if e.isInSearchMatchInLine(sourceX, searchMatches) {
		return theme.ApplyBackgroundStyle(style, e.config.Theme.GetStyle(e.config.Theme.SearchHighlight))
	}
	return style
}

func csvColumnBackground(normal ui.Color, column int) ui.Color {
	tint := ui.NewRGBColor(129, 140, 248)
	if column%2 == 1 {
		tint = ui.NewRGBColor(14, 165, 233)
	}
	if normal == ui.ColorDefault {
		return tint
	}
	nr, ng, nb := normal.RGB()
	tr, tg, tb := tint.RGB()
	return ui.NewRGBColor((nr*9+tr)/10, (ng*9+tg)/10, (nb*9+tb)/10)
}

func csvHeaderBackground(normal ui.Color) ui.Color {
	tint := ui.NewRGBColor(239, 68, 68)
	if normal == ui.ColorDefault {
		return tint
	}
	nr, ng, nb := normal.RGB()
	tr, tg, tb := tint.RGB()
	return ui.NewRGBColor((nr*9+tr)/10, (ng*9+tg)/10, (nb*9+tb)/10)
}

func (e *Editor) buildCSVVirtualCursorCells(table csvTable, contentStartY int) {
	e.virtualCursorCells = make(map[int]map[int]bool)
	viewLine := e.CurrentBuffer().ViewIterator.LineNumber()
	for _, cursor := range e.GetVirtualCursors() {
		y := contentStartY + cursor.Y - viewLine
		x := e.CurrentBuffer().LineNumberWidth + table.displayColumn(cursor.Y, cursor.X)
		if e.virtualCursorCells[y] == nil {
			e.virtualCursorCells[y] = make(map[int]bool)
		}
		e.virtualCursorCells[y][x] = true
	}
}
