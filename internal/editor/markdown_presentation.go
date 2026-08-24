package editor

import (
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"
	"veda/internal/buffer"
	"veda/internal/commands"
	"veda/internal/syntax"
	"veda/internal/theme"
	"veda/internal/ui"
)

type markdownLineProjection struct {
	Text          string
	SourceOffsets []int
}

type markdownImageSize struct {
	width  int
	height int
}

func (e *Editor) markdownPresentationEnabledForBuffer(buf *buffer.Buffer) bool {
	return e.mode != commands.InsertMode && buf != nil && buf.Highlighter != nil &&
		buf.Highlighter.GetCurrentLanguage() == "markdown" && buf.Highlighter.GetTree() != nil
}

func (e *Editor) projectMarkdownLine(buf *buffer.Buffer, line string, lineStart int) markdownLineProjection {
	ranges := buf.Highlighter.MarkdownConcealRangesForLine(lineStart, lineStart+len(line))
	if len(ranges) == 0 {
		return markdownLineProjection{Text: line}
	}
	text := make([]byte, 0, len(line))
	offsets := make([]int, 0, len(line))
	rangeIndex := 0
	for source := 0; source < len(line); source++ {
		for rangeIndex < len(ranges) && source >= ranges[rangeIndex].End {
			rangeIndex++
		}
		if rangeIndex < len(ranges) && source >= ranges[rangeIndex].Start {
			continue
		}
		text = append(text, line[source])
		offsets = append(offsets, source)
	}
	return markdownLineProjection{Text: string(text), SourceOffsets: offsets}
}

func (e *Editor) wrapLineForDisplay(wc *WrapCalculator, buf *buffer.Buffer, line string, lineStart, lineNumber int) []WrappedLine {
	if !e.markdownPresentationEnabledForBuffer(buf) {
		return wc.WrapLine(line, lineNumber)
	}
	if image, ok := buf.Highlighter.MarkdownImageForLine(lineStart, lineStart+len(line)); ok {
		if imagePath, ok := e.resolveMarkdownImagePath(buf, image.Destination); ok {
			images := e.screen.(KittyImageScreen)
			cellPixelWidth, cellPixelHeight := images.CellPixelSize()
			if wrapped, ok := e.markdownImageWrappedLines(buf, line, lineNumber, imagePath, wc.availableWidth, cellPixelWidth, cellPixelHeight); ok {
				return wrapped
			}
		}
	}

	projection := e.projectMarkdownLine(buf, line, lineStart)
	if projection.SourceOffsets == nil {
		return wc.WrapLine(line, lineNumber)
	}
	wrapped := wc.WrapLine(projection.Text, lineNumber)
	for i := range wrapped {
		displayStart := wrapped[i].StartChar
		displayEnd := wrapped[i].EndChar
		wrapped[i].StartChar = projection.sourceBoundary(displayStart, len(line))
		wrapped[i].EndChar = projection.sourceBoundary(displayEnd, len(line))
		wrapped[i].SourceOffsets = make([]int, displayEnd-displayStart)
		copy(wrapped[i].SourceOffsets, projection.SourceOffsets[displayStart:displayEnd])
	}
	if len(wrapped) > 0 {
		wrapped[0].StartChar = 0
		wrapped[len(wrapped)-1].EndChar = len(line)
		for i := 1; i < len(wrapped); i++ {
			wrapped[i-1].EndChar = wrapped[i].StartChar
		}
	}
	return wrapped
}

func (e *Editor) resolveMarkdownImagePath(buf *buffer.Buffer, destination string) (string, bool) {
	images, ok := e.screen.(KittyImageScreen)
	if !ok || !images.SupportsKittyImages() {
		return "", false
	}

	destination = strings.TrimSpace(destination)
	destination = strings.TrimPrefix(destination, "<")
	destination = strings.TrimSuffix(destination, ">")
	if destination == "" || strings.Contains(destination, "://") || strings.ToLower(filepath.Ext(destination)) != ".png" {
		return "", false
	}

	path := destination
	if !filepath.IsAbs(path) {
		markdownPath := buf.RealPath
		if markdownPath == "" {
			markdownPath = buf.FilePath
		}
		if _, _, remote := parsePathComponents(markdownPath); remote {
			return "", false
		}
		path = filepath.Join(filepath.Dir(markdownPath), path)
	}
	absPath, err := filepath.Abs(filepath.Clean(path))
	if err != nil {
		return "", false
	}
	return absPath, true
}

const markdownImageMaxRows = 12

func (e *Editor) markdownImageWrappedLines(buf *buffer.Buffer, line string, lineNumber int, imagePath string, availableWidth, cellPixelWidth, cellPixelHeight int) ([]WrappedLine, bool) {
	size, ok := e.markdownImageSize(imagePath)
	if !ok {
		return nil, false
	}
	columns, rows := fitMarkdownImageToCells(size.width, size.height, availableWidth, markdownImageMaxRows, cellPixelWidth, cellPixelHeight)

	wrapped := make([]WrappedLine, rows)
	for row := range wrapped {
		wrapped[row] = WrappedLine{
			LogicalLineNum:  lineNumber,
			VisualLineNum:   row,
			StartChar:       len(line),
			EndChar:         len(line),
			IsFirstLine:     row == 0,
			IsKittyImageRow: true,
		}
	}
	wrapped[0].StartChar = 0
	wrapped[0].KittyImageKey = fmt.Sprintf("%s:%d", buf.FilePath, lineNumber)
	wrapped[0].KittyImagePath = imagePath
	wrapped[0].KittyImageCols = columns
	wrapped[0].KittyImageRows = rows
	return wrapped, true
}

func (e *Editor) markdownImageSize(imagePath string) (markdownImageSize, bool) {
	if size, ok := e.markdownImageSizes[imagePath]; ok {
		return size, true
	}
	file, err := os.Open(imagePath)
	if err != nil {
		return markdownImageSize{}, false
	}
	defer file.Close()
	config, err := png.DecodeConfig(file)
	if err != nil {
		return markdownImageSize{}, false
	}
	if e.markdownImageSizes == nil {
		e.markdownImageSizes = make(map[string]markdownImageSize)
	}
	size := markdownImageSize{width: config.Width, height: config.Height}
	e.markdownImageSizes[imagePath] = size
	return size, true
}

func fitMarkdownImageToCells(pixelWidth, pixelHeight, maxColumns, maxRows, cellPixelWidth, cellPixelHeight int) (columns, rows int) {
	if pixelWidth < 1 {
		pixelWidth = 1
	}
	if pixelHeight < 1 {
		pixelHeight = 1
	}
	if maxColumns < 1 {
		maxColumns = 1
	}
	if maxRows < 1 {
		maxRows = 1
	}
	if cellPixelWidth < 1 {
		cellPixelWidth = 8
	}
	if cellPixelHeight < 1 {
		cellPixelHeight = 16
	}

	columns = (pixelWidth + cellPixelWidth - 1) / cellPixelWidth
	rows = (pixelHeight + cellPixelHeight - 1) / cellPixelHeight
	if columns < 1 {
		columns = 1
	}
	if rows < 1 {
		rows = 1
	}
	if columns <= maxColumns && rows <= maxRows {
		return columns, rows
	}

	widthScale := int64(maxColumns) * int64(cellPixelWidth) * int64(pixelHeight)
	heightScale := int64(maxRows) * int64(cellPixelHeight) * int64(pixelWidth)
	if widthScale <= heightScale {
		columns = maxColumns
		rows = ceilRatio(int64(pixelHeight)*int64(columns)*int64(cellPixelWidth), int64(pixelWidth)*int64(cellPixelHeight))
		if rows > maxRows {
			rows = maxRows
		}
	} else {
		rows = maxRows
		columns = ceilRatio(int64(pixelWidth)*int64(rows)*int64(cellPixelHeight), int64(pixelHeight)*int64(cellPixelWidth))
		if columns > maxColumns {
			columns = maxColumns
		}
	}
	return columns, rows
}

func ceilRatio(numerator, denominator int64) int {
	return int((numerator + denominator - 1) / denominator)
}

func (p markdownLineProjection) sourceBoundary(displayByte, lineLength int) int {
	if displayByte <= 0 {
		return 0
	}
	if displayByte >= len(p.SourceOffsets) {
		return lineLength
	}
	return p.SourceOffsets[displayByte]
}

func (wl WrappedLine) displayByteForSource(sourceByte int) int {
	if wl.SourceOffsets == nil {
		position := sourceByte - wl.StartChar
		if position < 0 {
			return 0
		}
		if position > len(wl.Text) {
			return len(wl.Text)
		}
		return position
	}
	return sort.Search(len(wl.SourceOffsets), func(i int) bool {
		return wl.SourceOffsets[i] >= sourceByte
	})
}

func (wl WrappedLine) sourceByteForDisplay(displayByte int) int {
	if wl.SourceOffsets == nil {
		position := wl.StartChar + displayByte
		if position > wl.EndChar {
			return wl.EndChar
		}
		return position
	}
	if len(wl.SourceOffsets) == 0 {
		return wl.StartChar
	}
	if displayByte >= len(wl.SourceOffsets) {
		return wl.SourceOffsets[lastRuneStart(wl.Text)]
	}
	if displayByte < 0 {
		displayByte = 0
	}
	return wl.SourceOffsets[displayByte]
}

func (p markdownLineProjection) visibleRuneSourceOffsets() []int {
	visible := make([]int, 0, utf8.RuneCountInString(p.Text))
	for displayByte := range p.Text {
		sourceByte := displayByte
		if p.SourceOffsets != nil {
			sourceByte = p.SourceOffsets[displayByte]
		}
		visible = append(visible, sourceByte)
	}
	return visible
}

func (e *Editor) markdownVisibleRuneOffsets() []int {
	buf := e.CurrentBuffer()
	if !e.markdownPresentationEnabledForBuffer(buf) || buf.CursorIterator == nil || buf.CursorIterator.IsEOF() {
		return nil
	}
	line := buf.CursorIterator.Line()
	lineStart := buf.CursorIterator.Position()
	if image, ok := buf.Highlighter.MarkdownImageForLine(lineStart, lineStart+len(line)); ok {
		if _, ok := e.resolveMarkdownImagePath(buf, image.Destination); ok {
			return []int{0}
		}
	}
	projection := e.projectMarkdownLine(buf, line, lineStart)
	return projection.visibleRuneSourceOffsets()
}

func (e *Editor) moveMarkdownCursorHorizontally(direction int) bool {
	visible := e.markdownVisibleRuneOffsets()
	if len(visible) == 0 {
		return false
	}
	current := e.CurrentBuffer().CursorX
	currentIndex := sort.Search(len(visible), func(i int) bool { return visible[i] >= current })
	if currentIndex == len(visible) {
		currentIndex = len(visible) - 1
	}
	if direction > 0 && currentIndex < len(visible)-1 {
		currentIndex++
	} else if direction < 0 && currentIndex > 0 {
		currentIndex--
	}
	e.CurrentBuffer().CursorX = visible[currentIndex]
	return true
}

func (e *Editor) normalizeMarkdownCursorToVisible() {
	visible := e.markdownVisibleRuneOffsets()
	if len(visible) == 0 {
		return
	}
	current := e.CurrentBuffer().CursorX
	index := sort.Search(len(visible), func(i int) bool { return visible[i] >= current })
	if index == len(visible) {
		index--
	}
	e.CurrentBuffer().CursorX = visible[index]
}

func (e *Editor) drawProjectedMarkdownLine(iter *buffer.BufferIterator, wl WrappedLine, tokens []syntax.Token, startX, screenY int, normalStyle ui.Style, lineBg ui.Color, diffBg *ui.Color, hasBg bool, searchMatches []int, trailingStart int) int {
	x := startX
	if len(wl.Text) == 0 {
		return x
	}

	type textRun struct {
		displayStart int
		displayEnd   int
		sourceStart  int
		sourceEnd    int
		tokenType    syntax.TokenType
		hasToken     bool
	}

	var current textRun
	hasRun := false
	flush := func() {
		if !hasRun {
			return
		}
		style := normalStyle
		if current.hasToken {
			style = e.config.Theme.GetStyle(syntax.GetStyleForToken(current.tokenType, e.config.Theme))
			if wordBg := getWordLevelDiffBg(e, iter.LineNumber(), current.sourceStart, current.sourceEnd); wordBg != nil {
				style = style.Background(*wordBg)
			} else if diffBg != nil {
				style = style.Background(*diffBg)
			} else if hasBg {
				style = style.Background(lineBg)
			}
		}
		text := wl.Text[current.displayStart:current.displayEnd]
		x += e.drawText(x, screenY, style, text, iter.Position()+current.sourceStart, searchMatches,
			iter.LineNumber(), current.sourceStart, trailingStart)
		hasRun = false
	}

	for displayByte, r := range wl.Text {
		sourceStart := wl.SourceOffsets[displayByte]
		sourceEnd := sourceStart + utf8.RuneLen(r)
		tokenType, hasToken := mostSpecificMarkdownToken(tokens, sourceStart)
		displayEnd := displayByte + utf8.RuneLen(r)

		if hasRun && (current.hasToken != hasToken || current.tokenType != tokenType || current.sourceEnd != sourceStart) {
			flush()
		}
		if !hasRun {
			current = textRun{
				displayStart: displayByte,
				sourceStart:  sourceStart,
				tokenType:    tokenType,
				hasToken:     hasToken,
			}
			hasRun = true
		}
		current.displayEnd = displayEnd
		current.sourceEnd = sourceEnd
	}
	flush()
	return x
}

func mostSpecificMarkdownToken(tokens []syntax.Token, sourceByte int) (syntax.TokenType, bool) {
	bestLength := 0
	var best syntax.TokenType
	found := false
	for _, token := range tokens {
		if sourceByte < token.Start || sourceByte >= token.End {
			continue
		}
		length := token.End - token.Start
		if !found || length < bestLength {
			best = token.TokenType
			bestLength = length
			found = true
		}
	}
	return best, found
}

func (e *Editor) fillAfterProjectedMarkdown(x, startX, screenY int, wl WrappedLine, normalStyle ui.Style, lineBg ui.Color, diffBg *ui.Color, hasBg bool, originalLineLength, lineIndex int) {
	width, _ := e.screen.Size()
	virtualCursorStyle := e.config.Theme.GetStyle(e.config.Theme.VirtualCursor)
	if !hasBg {
		for column := x; column < width; column++ {
			if e.hasVirtualCursorCell(column, screenY) {
				style := theme.ApplyBackgroundStyle(normalStyle, virtualCursorStyle)
				e.screen.SetContent(column, screenY, ' ', nil, style)
			}
		}
		return
	}

	rulerStyle := e.config.Theme.GetStyle(e.config.Theme.VerticalRuler)
	for column := x; column < width; column++ {
		style := normalStyle
		if diffBg != nil {
			style = normalStyle.Background(*diffBg)
		} else {
			textColumn := column - e.CurrentBuffer().LineNumberWidth
			for _, rulerPos := range e.config.VerticalRulers {
				if textColumn == rulerPos {
					style = theme.ApplyBackgroundStyle(normalStyle, rulerStyle)
					break
				}
			}
		}

		if lineIndex >= 0 && len(e.CurrentBuffer().Diagnostics) > 0 && wl.EndChar == originalLineLength {
			if diagnostic, ok := e.CurrentBuffer().GetDiagnosticForLine(lineIndex + 1); ok &&
				(diagnostic.Severity == "error" || diagnostic.Severity == "warning") &&
				diagnostic.EndChar <= diagnostic.StartChar && diagnostic.StartChar >= originalLineLength &&
				x >= startX && x < width {
				diagnosticStyle := e.config.Theme.GetStyle(e.config.Theme.DiagnosticWarning)
				if diagnostic.Severity == "error" {
					diagnosticStyle = e.config.Theme.GetStyle(e.config.Theme.DiagnosticError)
				}
				e.screen.SetContent(x, screenY, ' ', nil, theme.ApplyBackgroundStyle(normalStyle, diagnosticStyle))
			}
		}

		if e.hasVirtualCursorCell(column, screenY) {
			style = theme.ApplyBackgroundStyle(normalStyle, virtualCursorStyle)
		}
		e.screen.SetContent(column, screenY, ' ', nil, style)
	}
}
