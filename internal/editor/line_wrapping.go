package editor

import (
	"unicode/utf8"
	"veda/internal/buffer"
	"veda/internal/commands"
	"veda/internal/syntax"
)

type wrappedLinesCacheKey struct {
	buffer                    *buffer.Buffer
	bufferGeneration          uint64
	filePath                  string
	realPath                  string
	isInternal                bool
	highlighter               *syntax.Highlighter
	highlighterRevision       uint64
	viewLine                  int
	width                     int
	gutterWidth               int
	visibleHeight             int
	mode                      commands.Mode
	kittyImages               bool
	cellPixelWidth            int
	cellPixelHeight           int
	pairedBuffer              *buffer.Buffer
	pairedBufferGeneration    uint64
	pairedFilePath            string
	pairedRealPath            string
	pairedIsInternal          bool
	pairedHighlighter         *syntax.Highlighter
	pairedHighlighterRevision uint64
	pairedWidth               int
	pairedGutterWidth         int
}

type wrappedLinesCacheEntry struct {
	key   wrappedLinesCacheKey
	lines []WrappedLine
}

// WrappedLine represents a single visual line that may be part of a wrapped logical line
type WrappedLine struct {
	LogicalLineNum  int    // The actual line number in the buffer (0-based)
	VisualLineNum   int    // The visual line number on screen (0-based)
	StartChar       int    // Starting source byte position in the logical line
	EndChar         int    // Ending source byte position in the logical line (exclusive)
	IsFirstLine     bool   // True if this is the first visual line of a logical line
	Text            string // The text content of this visual line
	SourceOffsets   []int  // Display-byte to source-byte mapping; nil for literal text
	IsKittyImageRow bool   // True for every visual row reserved by a Kitty image
	KittyImageKey   string // Stable placement identity for a rendered Markdown image
	KittyImagePath  string // Local PNG path; set only on the first reserved image row
	KittyImageCols  int
	KittyImageRows  int
}

// WrapCalculator handles line wrapping calculations
type WrapCalculator struct {
	screenWidth    int
	gutterWidth    int // Width used by line numbers and separator
	availableWidth int // Width available for text
}

// NewWrapCalculator creates a new wrap calculator
func NewWrapCalculator(screenWidth, gutterWidth int) *WrapCalculator {
	return &WrapCalculator{
		screenWidth:    screenWidth,
		gutterWidth:    gutterWidth,
		availableWidth: screenWidth - gutterWidth,
	}
}

func wrapRuneWidth(r rune) int {
	if r == '\t' {
		return 4
	}
	width := runeDisplayWidth(r)
	if width < 1 {
		return 1
	}
	return width
}

func wrapTextWidth(text string) int {
	width := 0
	for _, r := range text {
		width += wrapRuneWidth(r)
	}
	return width
}

// WrapLine wraps a single logical line into multiple visual lines
func (wc *WrapCalculator) WrapLine(line string, logicalLineNum int) []WrappedLine {
	if wc.availableWidth <= 0 {
		// No space for text, return empty line
		return []WrappedLine{{
			LogicalLineNum: logicalLineNum,
			VisualLineNum:  0,
			StartChar:      0,
			EndChar:        0,
			IsFirstLine:    true,
			Text:           "",
		}}
	}

	// If line fits within available width, return as single line
	if wrapTextWidth(line) <= wc.availableWidth {
		return []WrappedLine{{
			LogicalLineNum: logicalLineNum,
			VisualLineNum:  0,
			StartChar:      0,
			EndChar:        len(line),
			IsFirstLine:    true,
			Text:           line,
		}}
	}

	// Line needs wrapping
	var wrapped []WrappedLine
	visualLineNum := 0
	pos := 0

	for pos < len(line) {
		// Find wrap point
		wrapPoint := wc.findWrapPoint(line, pos)

		// Create wrapped line
		wrappedLine := WrappedLine{
			LogicalLineNum: logicalLineNum,
			VisualLineNum:  visualLineNum,
			StartChar:      pos,
			EndChar:        wrapPoint,
			IsFirstLine:    visualLineNum == 0,
			Text:           line[pos:wrapPoint],
		}

		wrapped = append(wrapped, wrappedLine)
		pos = wrapPoint
		visualLineNum++

		// Skip any single space at wrap point to avoid leading space on wrapped line
		if pos < len(line) && line[pos] == ' ' {
			pos++
		}
	}

	// Handle empty wrapped lines (shouldn't happen but defensive)
	if len(wrapped) == 0 {
		return []WrappedLine{{
			LogicalLineNum: logicalLineNum,
			VisualLineNum:  0,
			StartChar:      0,
			EndChar:        0,
			IsFirstLine:    true,
			Text:           "",
		}}
	}

	return wrapped
}

// findWrapPoint finds the best position to wrap the line
func (wc *WrapCalculator) findWrapPoint(line string, start int) int {
	if start >= len(line) {
		return len(line)
	}

	// Find the last byte position that fits in the available terminal cells.
	maxPos := start
	width := 0
	for maxPos < len(line) {
		r, size := utf8.DecodeRuneInString(line[maxPos:])
		runeWidth := wrapRuneWidth(r)
		if width+runeWidth > wc.availableWidth {
			break
		}
		width += runeWidth
		maxPos += size
	}
	if maxPos == start {
		_, size := utf8.DecodeRuneInString(line[start:])
		return start + size
	}
	if maxPos >= len(line) {
		return len(line)
	}

	// Look backwards from maxPos to find a word boundary
	for pos := maxPos; pos > start; pos = prevRuneStart(line, pos) {
		currentRune, _ := utf8.DecodeRuneInString(line[pos:])
		if currentRune == ' ' {
			// Found a space, this is a good wrap point
			return pos
		}

		// Check if we're at a word boundary (transition between word and non-word)
		previousPos := prevRuneStart(line, pos)
		previousRune, _ := utf8.DecodeRuneInString(line[previousPos:pos])
		if pos < len(line) {
			prevIsWord := isWordChar(previousRune)
			currIsWord := isWordChar(currentRune)
			if prevIsWord != currIsWord {
				return pos
			}
		}
	}

	return maxPos
}

func (e *Editor) wrapScreenWidth() int {
	width, _ := e.screen.Size()
	if _, isWindowScreen := e.screen.(*windowScreen); !isWindowScreen {
		if e.activeWindow >= 0 && e.activeWindow < len(e.windows) && e.windows[e.activeWindow] != nil {
			width = e.windows[e.activeWindow].Width
			if _, isDiff := e.windows[e.activeWindow].View.(*DiffBufferView); isDiff {
				width = width / 2
			}
		}
	}
	return width
}

// CalculateWrappedLines calculates all wrapped lines for the visible portion of the buffer
func (e *Editor) CalculateWrappedLines() []WrappedLine {
	t0 := e.perfTracker.Start()

	// Handle empty windows
	currentBuf := e.CurrentBuffer()
	if currentBuf == nil {
		e.perfTracker.ReportSince("CalculateWrappedLines", t0)
		return []WrappedLine{}
	}

	width := e.wrapScreenWidth()
	gutterWidth := currentBuf.LineNumberWidth

	wc := NewWrapCalculator(width, gutterWidth)

	// For diff views, set up coordinated wrapping with the paired buffer
	// so both panes use the same number of visual rows per logical line.
	var pairedBuf *buffer.Buffer
	var pairedWC *WrapCalculator
	if e.activeWindow >= 0 && e.activeWindow < len(e.windows) && e.windows[e.activeWindow] != nil {
		if dv, ok := e.windows[e.activeWindow].View.(*DiffBufferView); ok {
			if currentBuf == dv.LeftBuffer {
				pairedBuf = dv.RightBuffer
			} else if currentBuf == dv.RightBuffer {
				pairedBuf = dv.LeftBuffer
			}
			if pairedBuf != nil {
				winW := e.windows[e.activeWindow].Width
				pairedPaneW := winW - width // other half
				if pairedPaneW < 1 {
					pairedPaneW = 1
				}
				pairedWC = NewWrapCalculator(pairedPaneW, pairedBuf.LineNumberWidth)
			}
		}
	}

	visibleHeight := e.getVisibleHeight()
	key := wrappedLinesCacheKey{
		buffer:                    currentBuf,
		bufferGeneration:          currentBuf.Generation,
		filePath:                  currentBuf.FilePath,
		realPath:                  currentBuf.RealPath,
		isInternal:                currentBuf.IsInternal,
		highlighter:               currentBuf.Highlighter,
		highlighterRevision:       highlighterRevision(currentBuf),
		viewLine:                  currentBuf.ViewIterator.LineNumber(),
		width:                     width,
		gutterWidth:               gutterWidth,
		visibleHeight:             visibleHeight,
		mode:                      e.mode,
		pairedBuffer:              pairedBuf,
		pairedHighlighterRevision: highlighterRevision(pairedBuf),
	}
	if pairedBuf != nil {
		key.pairedBufferGeneration = pairedBuf.Generation
		key.pairedFilePath = pairedBuf.FilePath
		key.pairedRealPath = pairedBuf.RealPath
		key.pairedIsInternal = pairedBuf.IsInternal
		key.pairedHighlighter = pairedBuf.Highlighter
		key.pairedWidth = pairedWC.screenWidth
		key.pairedGutterWidth = pairedWC.gutterWidth
	}
	if images, ok := e.screen.(KittyImageScreen); ok {
		key.kittyImages = images.SupportsKittyImages()
		key.cellPixelWidth, key.cellPixelHeight = images.CellPixelSize()
	}
	if e.wrappedLinesCache != nil && e.wrappedLinesCache.key == key {
		e.perfTracker.ReportSince("wrapping.CalculateWrappedLines", t0)
		return e.wrappedLinesCache.lines
	}

	var wrappedLines []WrappedLine

	// Start from the view iterator
	iter := buffer.NewBufferIteratorAtLine(currentBuf, currentBuf.ViewIterator.LineNumber())
	var pairedIter *buffer.BufferIterator
	if pairedBuf != nil {
		pairedIter = buffer.NewBufferIteratorAtLine(pairedBuf, currentBuf.ViewIterator.LineNumber())
	}
	visualLineCount := 0

	startViewLine := currentBuf.ViewIterator.LineNumber()
	lastLogical := startViewLine - 1
	// Process lines until we fill the visible area
	for !iter.IsEOF() && visualLineCount < visibleHeight {
		logicalLineNum := iter.LineNumber()
		lastLogical = logicalLineNum
		line := iter.Line()

		// For terminal/internal buffers, avoid editor-side wrapping to preserve UTF-8
		var wrapped []WrappedLine
		if currentBuf.IsInternal {
			wrapped = []WrappedLine{{
				LogicalLineNum: logicalLineNum,
				VisualLineNum:  visualLineCount,
				StartChar:      0,
				EndChar:        len(line),
				IsFirstLine:    true,
				Text:           line,
			}}
		} else {
			wrapped = e.wrapLineForDisplay(wc, currentBuf, line, iter.Position(), logicalLineNum)
		}

		// Coordinated wrapping: pad to match the paired pane's wrap count
		if pairedIter != nil && !pairedIter.IsEOF() && pairedIter.LineNumber() == logicalLineNum {
			pairedWrapped := e.wrapLineForDisplay(pairedWC, pairedBuf, pairedIter.Line(), pairedIter.Position(), logicalLineNum)
			for len(wrapped) < len(pairedWrapped) {
				wrapped = append(wrapped, WrappedLine{
					LogicalLineNum: logicalLineNum,
					StartChar:      len(line),
					EndChar:        len(line),
					IsFirstLine:    false,
					Text:           "",
				})
			}
			pairedIter = pairedIter.Next()
		}

		// Add wrapped lines up to visible height
		for _, wl := range wrapped {
			if visualLineCount >= visibleHeight {
				break
			}
			wrappedLines = append(wrappedLines, wl)
			visualLineCount++
		}

		iter = iter.Next()
	}

	// Handle empty buffer case
	if len(wrappedLines) == 0 && currentBuf.Rope.Length() == 0 {
		wrappedLines = append(wrappedLines, WrappedLine{
			LogicalLineNum: 0,
			VisualLineNum:  0,
			StartChar:      0,
			EndChar:        0,
			IsFirstLine:    true,
			Text:           "",
		})
	}

	// Render final empty line when buffer ends with a newline
	// Using newline-as-boundary semantics: "hello\n" has two lines, second is empty
	if visualLineCount < visibleHeight && currentBuf.Rope.Length() > 0 {
		lastByte := currentBuf.Rope.ByteAt(currentBuf.Rope.Length() - 1)
		if lastByte == '\n' {
			// The empty line immediately after the last logical line
			// Its logical line number is lastLogical+1, but if we didn't render any lines
			// and view starts at 0, then lastLogical may be -1; handle that.
			emptyLineNum := lastLogical + 1
			if emptyLineNum >= startViewLine && visualLineCount < visibleHeight {
				wrappedLines = append(wrappedLines, WrappedLine{
					LogicalLineNum: emptyLineNum,
					VisualLineNum:  visualLineCount,
					StartChar:      0,
					EndChar:        0,
					IsFirstLine:    true,
					Text:           "",
				})
			}
		}
	}

	e.wrappedLinesCache = &wrappedLinesCacheEntry{key: key, lines: wrappedLines}
	e.perfTracker.ReportSince("wrapping.CalculateWrappedLines", t0)
	return wrappedLines
}

func highlighterRevision(buf *buffer.Buffer) uint64 {
	if buf == nil || buf.Highlighter == nil {
		return 0
	}
	return buf.Highlighter.Revision()
}

// GetCursorVisualPosition calculates the visual position of the cursor with line wrapping
// Returns the visual line and column position relative to the current view
func (e *Editor) GetCursorVisualPosition() (visualLine, visualCol int) {
	t0 := e.perfTracker.Start()

	// Handle empty windows
	if e.CurrentBuffer() == nil {
		e.perfTracker.ReportSince("GetCursorVisualPosition", t0)
		return 0, 0
	}

	// Calculate all visible wrapped lines
	wrappedLines := e.CalculateWrappedLines()
	LL.Trace("GetCursorVisualPosition: viewTop=%d visibleWrapped=%d cursor(line=%d,x=%d)",
		e.CurrentBuffer().ViewIterator.LineNumber(), len(wrappedLines), e.CurrentBuffer().CursorIterator.LineNumber(), e.CurrentBuffer().CursorX)

	cursorLogicalLine := e.CurrentBuffer().CursorIterator.LineNumber()
	cursorX := e.CurrentBuffer().CursorX

	// Find which wrapped line contains the cursor
	for i, wl := range wrappedLines {
		if wl.LogicalLineNum == cursorLogicalLine {
			// Check if cursor is in this wrapped segment
			if cursorX >= wl.StartChar && cursorX < wl.EndChar {
				visualLine = i
				// Calculate visual column accounting for tabs
				displayByte := wl.displayByteForSource(cursorX)
				if wl.SourceOffsets != nil && displayByte >= len(wl.Text) && len(wl.Text) > 0 {
					displayByte = lastRuneStart(wl.Text)
				}
				lineSegment := wl.Text[:displayByte]
				visualCol = e.visualWidth(lineSegment) + e.CurrentBuffer().LineNumberWidth
				return
			}
			// Special case for cursor exactly at the end of a wrapped segment
			if cursorX == wl.EndChar {
				// Check if this is the last segment of this logical line
				isLastSegment := i == len(wrappedLines)-1 || wrappedLines[i+1].LogicalLineNum != cursorLogicalLine
				if isLastSegment {
					// Cursor at end of line
					// Calculate visual column accounting for tabs
					visualCol = e.visualWidth(wl.Text) + e.CurrentBuffer().LineNumberWidth
					width := e.wrapScreenWidth()
					if visualCol == width {
						return i + 1, e.CurrentBuffer().LineNumberWidth
					}
					visualLine = i
					return
				}
			}
		}
	}

	// If cursor wasn't found in wrapped lines, it might be beyond the visible area
	// or at the end of a line
	// Special handling: cursor sitting on the empty logical line after a trailing newline.
	// In that case, place the cursor on the next visual row after the last visible segment
	// of the previous logical line, at the start of content.
	for i := len(wrappedLines) - 1; i >= 0; i-- {
		wl := wrappedLines[i]
		if wl.LogicalLineNum+1 == cursorLogicalLine && cursorX == 0 {
			// Cursor is on the empty line following wl.LogicalLineNum
			// Place the cursor on the next visual row, but never beyond the last
			// visible wrapped row to avoid drawing on the status bar.
			nextRow := i + 1
			if nextRow >= len(wrappedLines) {
				nextRow = len(wrappedLines) - 1
				if nextRow < 0 {
					nextRow = 0
				}
			}
			visualLine = nextRow
			visualCol = e.CurrentBuffer().LineNumberWidth
			LL.Trace("GetCursorVisualPosition: placing cursor on synthetic empty line after logical %d -> visualLine=%d, visualCol=%d", wl.LogicalLineNum, visualLine, visualCol)
			return
		}
	}

	// Fallbacks
	for i := len(wrappedLines) - 1; i >= 0; i-- {
		wl := wrappedLines[i]
		if wl.LogicalLineNum == cursorLogicalLine {
			// Place cursor at end of this wrapped line
			visualLine = i
			// Calculate visual column accounting for tabs
			visualCol = e.visualWidth(wl.Text) + e.CurrentBuffer().LineNumberWidth
			LL.Trace("GetCursorVisualPosition: found logical line at i=%d, placing at EOL visualCol=%d", i, visualCol)
			return
		}
	}

	// Default case - cursor is beyond visible area
	// Clamp to the last visible wrapped line to avoid placing cursor past window height
	if len(wrappedLines) == 0 {
		LL.Trace("GetCursorVisualPosition: no wrapped lines; default to top start-of-content")
		return 0, e.CurrentBuffer().LineNumberWidth
	}
	LL.Trace("GetCursorVisualPosition: cursor line not visible; default to last visible line start-of-content (i=%d)", len(wrappedLines)-1)
	v := len(wrappedLines) - 1
	c := e.CurrentBuffer().LineNumberWidth
	e.perfTracker.ReportSince("wrapping.GetCursorVisualPosition", t0)
	return v, c
}

// MapVisualToLogical maps a visual line position to logical line and character position
func (e *Editor) MapVisualToLogical(visualLine, visualCol int) (logicalLine, charPos int) {
	t0 := e.perfTracker.Start()
	// Calculate wrapped lines
	wrappedLines := e.CalculateWrappedLines()

	if visualLine >= len(wrappedLines) {
		// Beyond visible lines
		if len(wrappedLines) > 0 {
			last := wrappedLines[len(wrappedLines)-1]
			return last.LogicalLineNum, last.EndChar
		}
		return 0, 0
	}

	wl := wrappedLines[visualLine]
	logicalLine = wl.LogicalLineNum

	// Convert visual column to character position
	textCol := visualCol - e.CurrentBuffer().LineNumberWidth
	if textCol < 0 {
		textCol = 0
	}

	// Get the text segment for this wrapped line
	lineText := wl.Text

	// Convert visual column to byte position within the line segment
	byteOffsetInSegment := e.bytePositionFromVisualColumn(lineText, textCol)

	charPos = wl.sourceByteForDisplay(byteOffsetInSegment)
	if charPos > wl.EndChar {
		charPos = wl.EndChar
	}

	e.perfTracker.ReportSince("wrapping.MapVisualToLogical", t0)
	return
}
