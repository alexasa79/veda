package editor

import (
	"veda/internal/buffer"
)

// WrappedLine represents a single visual line that may be part of a wrapped logical line
type WrappedLine struct {
	LogicalLineNum int    // The actual line number in the buffer (0-based)
	VisualLineNum  int    // The visual line number on screen (0-based)
	StartChar      int    // Starting character position in the logical line
	EndChar        int    // Ending character position in the logical line (exclusive)
	IsFirstLine    bool   // True if this is the first visual line of a logical line
	Text           string // The text content of this visual line
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
	if len(line) <= wc.availableWidth {
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

	// Maximum position we can go to
	maxPos := start + wc.availableWidth
	if maxPos >= len(line) {
		return len(line)
	}

	// Look backwards from maxPos to find a word boundary
	bestPos := maxPos
	foundSpace := false

	for pos := maxPos; pos > start; pos-- {
		if pos < len(line) && line[pos] == ' ' {
			// Found a space, this is a good wrap point
			bestPos = pos
			foundSpace = true
			break
		}
		
		// Check if we're at a word boundary (transition between word and non-word)
		if pos > start && pos < len(line) {
			prevIsWord := isWordChar(rune(line[pos-1]))
			currIsWord := isWordChar(rune(line[pos]))
			if prevIsWord != currIsWord {
				bestPos = pos
				break
			}
		}
	}

	// If we didn't find any good wrap point, just break at the maximum position
	if !foundSpace && bestPos == maxPos {
		return maxPos
	}

	return bestPos
}

// CalculateWrappedLines calculates all wrapped lines for the visible portion of the buffer
func (e *Editor) CalculateWrappedLines() []WrappedLine {
	width, _ := e.screen.Size()
	gutterWidth := e.buffer.LineNumberWidth
	
	wc := NewWrapCalculator(width, gutterWidth)
	
	var wrappedLines []WrappedLine
	visibleHeight := e.getVisibleHeight()
	
	// Start from the view iterator
	iter := buffer.NewBufferIteratorAtLine(e.buffer, e.buffer.ViewIterator.LineNumber())
	visualLineCount := 0
	
	// Process lines until we fill the visible area
	for !iter.IsEOF() && visualLineCount < visibleHeight {
		logicalLineNum := iter.LineNumber()
		line := iter.Line()
		
		// Wrap this line
		wrapped := wc.WrapLine(line, logicalLineNum)
		
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
	if len(wrappedLines) == 0 && e.buffer.Rope.Length() == 0 {
		wrappedLines = append(wrappedLines, WrappedLine{
			LogicalLineNum: 0,
			VisualLineNum:  0,
			StartChar:      0,
			EndChar:        0,
			IsFirstLine:    true,
			Text:           "",
		})
	}
	
	return wrappedLines
}

// GetCursorVisualPosition calculates the visual position of the cursor with line wrapping
// Returns the visual line and column position relative to the current view
func (e *Editor) GetCursorVisualPosition() (visualLine, visualCol int) {
	// Calculate all visible wrapped lines
	wrappedLines := e.CalculateWrappedLines()
	
	cursorLogicalLine := e.buffer.CursorIterator.LineNumber()
	cursorX := e.buffer.CursorX
	
	// Find which wrapped line contains the cursor
	for i, wl := range wrappedLines {
		if wl.LogicalLineNum == cursorLogicalLine {
			// Check if cursor is in this wrapped segment
			if cursorX >= wl.StartChar && cursorX < wl.EndChar {
				visualLine = i
				// Calculate visual column accounting for tabs
				lineSegment := wl.Text[:cursorX - wl.StartChar]
				visualCol = e.visualWidth(lineSegment) + e.buffer.LineNumberWidth
				return
			}
			// Special case for cursor exactly at the end of a wrapped segment
			if cursorX == wl.EndChar {
				// Check if this is the last segment of this logical line
				isLastSegment := i == len(wrappedLines)-1 || wrappedLines[i+1].LogicalLineNum != cursorLogicalLine
				if isLastSegment {
					// Cursor at end of line
					visualLine = i
					// Calculate visual column accounting for tabs
					visualCol = e.visualWidth(wl.Text) + e.buffer.LineNumberWidth
					return
				}
			}
		}
	}
	
	// If cursor wasn't found in wrapped lines, it might be beyond the visible area
	// or at the end of a line
	for i := len(wrappedLines) - 1; i >= 0; i-- {
		wl := wrappedLines[i]
		if wl.LogicalLineNum == cursorLogicalLine {
			// Place cursor at end of this wrapped line
			visualLine = i
			// Calculate visual column accounting for tabs
			visualCol = e.visualWidth(wl.Text) + e.buffer.LineNumberWidth
			return
		}
	}
	
	// Default case - cursor is beyond visible area
	// Return the count of wrapped lines to indicate we're off-screen
	return len(wrappedLines), e.buffer.LineNumberWidth
}

// MapVisualToLogical maps a visual line position to logical line and character position
func (e *Editor) MapVisualToLogical(visualLine, visualCol int) (logicalLine, charPos int) {
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
	textCol := visualCol - e.buffer.LineNumberWidth
	if textCol < 0 {
		textCol = 0
	}
	
	charPos = wl.StartChar + textCol
	if charPos > wl.EndChar {
		charPos = wl.EndChar
	}
	
	return
}