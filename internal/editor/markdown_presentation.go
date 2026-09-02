package editor

import (
	"strings"
	"unicode/utf8"
	"veda/internal/buffer"
	"veda/internal/commands"
)

type markdownRenderCache struct {
	generation uint64
	lines      []markdownPresentedLine
}

type markdownPresentedLine struct {
	source       string
	hidden       []bool
	replacements map[int]rune
	tableGrid    map[int]bool
}

func (e *Editor) markdownPresentationActive() bool {
	buf := e.CurrentBuffer()
	return buf != nil && !buf.IsInternal && len(buf.DiffHints) == 0 && e.mode != commands.InsertMode && e.isMarkdownFile()
}

func (e *Editor) markdownPresentedLine(lineNum int) *markdownPresentedLine {
	if !e.markdownPresentationActive() {
		return nil
	}
	buf := e.CurrentBuffer()
	cache, ok := e.markdownRenderCaches[buf]
	if !ok || cache.generation != buf.Generation {
		cache = buildMarkdownRenderCache(buf)
		e.markdownRenderCaches[buf] = cache
	}
	if lineNum < 0 || lineNum >= len(cache.lines) {
		return nil
	}
	return &cache.lines[lineNum]
}

func buildMarkdownRenderCache(buf *buffer.Buffer) markdownRenderCache {
	lines := make([]markdownPresentedLine, buf.LineCount)
	sourceLines := make([]string, buf.LineCount)
	for lineNum := range sourceLines {
		source := buffer.NewBufferIteratorAtLine(buf, lineNum).Line()
		sourceLines[lineNum] = source
		lines[lineNum] = markdownPresentedLine{
			source:       source,
			hidden:       make([]bool, len(source)),
			replacements: make(map[int]rune),
			tableGrid:    make(map[int]bool),
		}
		lines[lineNum].hideLinkMarkup()
	}

	for lineNum, tableLine := range markdownTablePresentationLines(sourceLines) {
		line := &lines[lineNum]
		// Keep table widths stable when a link hides its markup. The hidden source
		// cells become spaces inside tables instead of collapsing the column.
		for i, hidden := range line.hidden {
			if hidden {
				line.hidden[i] = false
				line.replacements[i] = ' '
			}
		}
		line.applyTableGlyphs(tableLine)
	}

	return markdownRenderCache{generation: buf.Generation, lines: lines}
}

func (line *markdownPresentedLine) hideLinkMarkup() {
	for _, match := range markdownLinkRegex.FindAllStringSubmatchIndex(line.source, -1) {
		if len(match) < 6 || match[2] < 0 || match[3] < 0 {
			continue
		}
		start := match[0]
		if start > 0 && line.source[start-1] == '!' {
			start--
		}
		line.hide(start, match[2])
		line.hide(match[3], match[1])
	}
}

func (line *markdownPresentedLine) hide(start, end int) {
	start = max(0, start)
	end = min(end, len(line.hidden))
	for i := start; i < end; i++ {
		line.hidden[i] = true
	}
}

type markdownTablePresentationLine struct {
	separator bool
	bottom    bool
}

func markdownTablePresentationLines(lines []string) map[int]markdownTablePresentationLine {
	result := make(map[int]markdownTablePresentationLine)
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
		if !ok || len(header.cells) != len(delimiter.cells) {
			continue
		}

		columns := len(header.cells)
		result[delimiterLine-1] = markdownTablePresentationLine{}
		result[delimiterLine] = markdownTablePresentationLine{separator: true}
		endLine := delimiterLine
		for endLine+1 < len(lines) {
			row, ok := parseMarkdownTableRow(lines[endLine+1])
			if !ok {
				break
			}
			endLine++
			_, separator := markdownTableRowSeparator(row, columns)
			result[endLine] = markdownTablePresentationLine{separator: separator}
		}
		if last := result[endLine]; last.separator {
			last.bottom = true
			result[endLine] = last
		}
		delimiterLine = endLine
	}
	return result
}

func (line *markdownPresentedLine) applyTableGlyphs(tableLine markdownTablePresentationLine) {
	pipes := markdownTablePipes(line.source, 0)
	if !tableLine.separator {
		for _, pipe := range pipes {
			line.replacements[pipe] = '┃'
			line.tableGrid[pipe] = true
		}
		return
	}

	contentStart := len(line.source) - len(strings.TrimLeft(line.source, " \t"))
	contentEnd := len(strings.TrimRight(line.source, " \t"))
	pipeSet := make(map[int]bool, len(pipes))
	for _, pipe := range pipes {
		pipeSet[pipe] = true
	}
	for pos := contentStart; pos < contentEnd; {
		_, size := utf8.DecodeRuneInString(line.source[pos:])
		if pipeSet[pos] {
			if tableLine.bottom {
				switch pos {
				case contentStart:
					line.replacements[pos] = '┗'
				case contentEnd - 1:
					line.replacements[pos] = '┛'
				default:
					line.replacements[pos] = '┻'
				}
			} else {
				switch pos {
				case contentStart:
					line.replacements[pos] = '┣'
				case contentEnd - 1:
					line.replacements[pos] = '┫'
				default:
					line.replacements[pos] = '╋'
				}
			}
		} else {
			line.replacements[pos] = '━'
		}
		line.tableGrid[pos] = true
		pos += size
	}
}

func (line *markdownPresentedLine) presentedRune(sourceByte int, sourceRune rune) (rune, bool) {
	if sourceByte >= 0 && sourceByte < len(line.hidden) && line.hidden[sourceByte] {
		return 0, false
	}
	if replacement, ok := line.replacements[sourceByte]; ok {
		return replacement, true
	}
	return sourceRune, true
}

func (line *markdownPresentedLine) displayWidth(start, end int) int {
	start = max(0, start)
	end = min(end, len(line.source))
	width := 0
	for pos := start; pos < end; {
		r, size := utf8.DecodeRuneInString(line.source[pos:])
		if presented, visible := line.presentedRune(pos, r); visible {
			width += wrapRuneWidth(presented)
		}
		pos += size
	}
	return width
}

func (line *markdownPresentedLine) sourceOffsetAtDisplayColumn(start, end, target int) int {
	start = max(0, start)
	end = min(end, len(line.source))
	width := 0
	for pos := start; pos < end; {
		r, size := utf8.DecodeRuneInString(line.source[pos:])
		presented, visible := line.presentedRune(pos, r)
		if !visible {
			pos += size
			continue
		}
		if target <= width {
			return pos
		}
		runeWidth := wrapRuneWidth(presented)
		if target < width+runeWidth {
			return pos
		}
		width += runeWidth
		pos += size
	}
	return end
}

func (line *markdownPresentedLine) displayAndBoundaries() (string, map[int]int) {
	var display strings.Builder
	boundaries := map[int]int{0: 0}
	for pos := 0; pos < len(line.source); {
		r, size := utf8.DecodeRuneInString(line.source[pos:])
		presented, visible := line.presentedRune(pos, r)
		if !visible {
			pos += size
			boundaries[display.Len()] = pos
			continue
		}
		boundaries[display.Len()] = pos
		display.WriteRune(presented)
		pos += size
		boundaries[display.Len()] = pos
	}
	boundaries[display.Len()] = len(line.source)
	return display.String(), boundaries
}

func (line *markdownPresentedLine) wrap(wc *WrapCalculator, logicalLineNum int) []WrappedLine {
	display, boundaries := line.displayAndBoundaries()
	if display == "" {
		return []WrappedLine{{
			LogicalLineNum: logicalLineNum,
			StartChar:      0,
			EndChar:        len(line.source),
			IsFirstLine:    true,
			Text:           line.source,
		}}
	}
	displayLines := wc.WrapLine(display, logicalLineNum)
	wrapped := make([]WrappedLine, len(displayLines))
	for i, displayLine := range displayLines {
		start := boundaries[displayLine.StartChar]
		end := boundaries[displayLine.EndChar]
		wrapped[i] = WrappedLine{
			LogicalLineNum: logicalLineNum,
			VisualLineNum:  displayLine.VisualLineNum,
			StartChar:      start,
			EndChar:        end,
			IsFirstLine:    displayLine.IsFirstLine,
			Text:           line.source[start:end],
		}
	}
	return wrapped
}
