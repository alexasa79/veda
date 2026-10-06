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
	source string
	hidden []bool
	table  *markdownTableDisplay
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
			source: source,
			hidden: make([]bool, len(source)),
		}
		lines[lineNum].hideLinkMarkup()
	}

	alignMarkdownTables(sourceLines, lines)

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

func (line *markdownPresentedLine) presentedRune(sourceByte int, sourceRune rune) (rune, bool) {
	if sourceByte >= 0 && sourceByte < len(line.hidden) && line.hidden[sourceByte] {
		return 0, false
	}
	return sourceRune, true
}

func (line *markdownPresentedLine) displayWidth(start, end int) int {
	start = max(0, start)
	end = min(end, len(line.source))
	if line.table != nil {
		if start == 0 {
			return line.table.columns[end]
		}
		return line.table.columns[end] - line.table.columns[start]
	}
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
	if line.table != nil {
		for _, glyph := range line.table.glyphs {
			if target < glyph.column+wrapRuneWidth(glyph.r) {
				return glyph.source
			}
		}
		return end
	}
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
	if line.table != nil {
		return []WrappedLine{{LogicalLineNum: logicalLineNum, StartChar: 0, EndChar: len(line.source), IsFirstLine: true, Text: line.source}}
	}
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
