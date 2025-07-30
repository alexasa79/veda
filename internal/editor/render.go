package editor

import (
	"fmt"
	"strings"
	"veda/internal/buffer"
	"veda/internal/commands"
	"veda/internal/config"
	"veda/internal/syntax"
	"veda/internal/theme"
	"veda/internal/ui"
)

// drawText draws text with optional search highlighting, selection, and whitespace visualization
// Returns the number of screen columns consumed
// segmentStart is the character position in the original line where this text segment starts
// trailingStart is the character position in the original line where trailing spaces begin (-1 if none)
func (e *Editor) drawText(x, y int, style ui.Style, text string, textOffset int, searchMatches []int, lineNum int, segmentStart int, trailingStart int) int {
	searchStyle := e.config.Theme.GetStyle(e.config.Theme.SearchHighlight)
	selectionStyle := e.config.Theme.GetStyle(e.config.Theme.Selection)
	flashStyle := e.config.Theme.GetStyle(e.config.Theme.FlashHighlight)
	whitespaceStyle := e.whitespaceStyle.style
	screenX := x
	charPos := textOffset

	// Get selection range once for efficiency
	selStart, selEnd, hasSelection := e.getSelectionRange()

	// Calculate line start position once if we need it for flash highlighting
	var lineStartPos int
	if lineNum >= 0 && e.hasFlashHighlight() {
		lineStartIter := buffer.NewBufferIteratorAtLine(e.buffer, lineNum)
		lineStartPos = lineStartIter.Position()
	}

	// Convert to slice of runes for position tracking
	runes := []rune(text)
	for i, r := range runes {
		// Check if this character is part of a search match
		useSearchStyle := e.isInSearchMatch(charPos, searchMatches)
		// Check if this character is part of the selection
		inSelection := hasSelection && charPos >= selStart && charPos < selEnd
		// Check if this character is part of the flash highlight
		var inFlash bool
		if lineNum >= 0 && e.hasFlashHighlight() {
			charPosInLine := charPos - lineStartPos
			inFlash = e.isInFlashRange(lineNum, charPosInLine)
		}

		currentStyle := style
		if inFlash {
			// Flash takes highest precedence
			currentStyle = theme.ApplyBackgroundStyle(style, flashStyle)
		} else if inSelection {
			// Selection takes precedence over search
			currentStyle = theme.ApplyBackgroundStyle(style, selectionStyle)
		} else if useSearchStyle {
			// Use search highlight style but preserve the foreground color
			currentStyle = theme.ApplyBackgroundStyle(style, searchStyle)
		}

		// Check if we should visualize this character
		absolutePos := segmentStart + i
		
		if r == '\t' && e.whitespaceStyle.showTabs {
			// Draw tab arrow with whitespace style, but preserve background from current style
			_, bg, _ := currentStyle.Decompose()
			tabStyle := whitespaceStyle.Background(bg)
			e.screen.SetContent(screenX, y, tabChar, nil, tabStyle)
			screenX++
			// Fill the rest with spaces using current style
			for j := 1; j < 4; j++ {
				e.screen.SetContent(screenX, y, ' ', nil, currentStyle)
				screenX++
			}
		} else if r == '\t' {
			// Tab without visualization
			for j := 0; j < 4; j++ {
				e.screen.SetContent(screenX, y, ' ', nil, currentStyle)
				screenX++
			}
		} else if r == ' ' && trailingStart >= 0 && absolutePos >= trailingStart && e.whitespaceStyle.showTrailingSpaces {
			// Draw trailing space dot with whitespace style, but preserve background
			_, bg, _ := currentStyle.Decompose()
			dotStyle := whitespaceStyle.Background(bg)
			e.screen.SetContent(screenX, y, spaceChar, nil, dotStyle)
			screenX++
		} else {
			// Normal character
			e.screen.SetContent(screenX, y, r, nil, currentStyle)
			screenX++
		}
		charPos++
	}
	return screenX - x
}

// drawLine draws a single line using the iterator
func (e *Editor) drawLine(iter *buffer.BufferIterator, startX int, screenY int) {
	e.drawLineInternal(iter, startX, screenY, nil)
}

// drawContextLine draws a single line with context background
func (e *Editor) drawContextLine(iter *buffer.BufferIterator, startX int, screenY int, contextBg ui.Color) {
	e.drawLineInternal(iter, startX, screenY, &contextBg)
}

// findSearchMatches finds all occurrences of the search pattern in a line
func (e *Editor) findSearchMatches(line string) []int {
	if e.searchPattern == "" {
		return nil
	}

	var matches []int
	start := 0
	for {
		index := strings.Index(line[start:], e.searchPattern)
		if index == -1 {
			break
		}
		matches = append(matches, start+index)
		start = start + index + 1
	}
	return matches
}

// isInSearchMatch checks if a position is within a search match
func (e *Editor) isInSearchMatch(pos int, matches []int) bool {
	if len(matches) == 0 || e.searchPattern == "" {
		return false
	}

	patternLen := len(e.searchPattern)
	for _, matchStart := range matches {
		if pos >= matchStart && pos < matchStart+patternLen {
			return true
		}
	}
	return false
}

// drawLineInternal draws a single line using the iterator with an optional background color override
func (e *Editor) drawLineInternal(iter *buffer.BufferIterator, startX int, screenY int, bgOverride *ui.Color) {
	originalLine := iter.Line()  // Keep the original line for whitespace detection
	lineIndex := iter.LineNumber()
	normalStyle := e.config.Theme.GetStyle(e.config.Theme.Normal)
	isCurrentLine := lineIndex == e.buffer.CursorIterator.LineNumber()

	// Find search matches in this line
	searchMatches := e.findSearchMatches(originalLine)

	// Determine which background to use
	var lineBg ui.Color
	var hasBg bool
	if bgOverride != nil {
		// Use the provided background override
		lineBg = *bgOverride
		normalStyle = normalStyle.Background(lineBg)
		hasBg = true
	} else if isCurrentLine {
		// Use current line background
		currentLineStyle := e.config.Theme.GetStyle(e.config.Theme.CurrentLine)
		_, lineBg, _ = currentLineStyle.Decompose()
		normalStyle = normalStyle.Background(lineBg)
		hasBg = true
	}

	// Get the absolute offset of the start of this line
	lineStartOffset := iter.Position()

	// Find trailing spaces position for the whole line BEFORE any modifications
	trailingStart := -1
	if e.whitespaceStyle.showTrailingSpaces {
		trailingStart = detectTrailingSpaces(originalLine)
	}

	// If no highlighter is set up, just draw normally
	if e.highlighter == nil || !e.buffer.HasFile() {
		x := e.drawText(startX, screenY, normalStyle, originalLine, lineStartOffset, searchMatches, lineIndex, 0, trailingStart)
		// Fill rest of line if we have background
		if hasBg {
			width, _ := e.screen.Size()
			for i := x; i < width; i++ {
				e.screen.SetContent(i, screenY, ' ', nil, normalStyle)
			}
		}
		return
	}

	// Get syntax tokens for this line using iterator position
	tokens := e.getLineTokens(iter)

	if len(tokens) == 0 {
		// No tokens, draw with normal style
		x := e.drawText(startX, screenY, normalStyle, originalLine, lineStartOffset, searchMatches, lineIndex, 0, trailingStart)
		// Fill rest of line if we have background
		if hasBg {
			width, _ := e.screen.Size()
			for i := x; i < width; i++ {
				e.screen.SetContent(i, screenY, ' ', nil, normalStyle)
			}
		}
		return
	}

	// Draw with syntax highlighting
	x := startX
	lastEnd := 0

	for _, token := range tokens {
		// Draw any text before this token with normal style
		if token.Start > lastEnd && token.Start <= len(originalLine) {
			endPos := token.Start
			if endPos > len(originalLine) {
				endPos = len(originalLine)
			}
			prefix := originalLine[lastEnd:endPos]
			x += e.drawText(x, screenY, normalStyle, prefix, lineStartOffset + lastEnd, searchMatches, lineIndex, lastEnd, trailingStart)
		}

		// Draw the token with its style
		if token.Start < len(originalLine) && token.End > token.Start {
			endPos := token.End
			if endPos > len(originalLine) {
				endPos = len(originalLine)
			}
			tokenText := originalLine[token.Start:endPos]

			tokenStyle := e.config.Theme.GetStyle(syntax.GetStyleForToken(token.TokenType, e.config.Theme))
			if bgOverride != nil {
				// Override background with the provided override
				tokenStyle = tokenStyle.Background(*bgOverride)
			} else if isCurrentLine {
				// Override background for current line
				tokenStyle = tokenStyle.Background(lineBg)
			}

			x += e.drawText(x, screenY, tokenStyle, tokenText, lineStartOffset + token.Start, searchMatches, lineIndex, token.Start, trailingStart)
			lastEnd = endPos
		}
	}

	// Draw any remaining text with normal style
	if lastEnd < len(originalLine) {
		suffix := originalLine[lastEnd:]
		x += e.drawText(x, screenY, normalStyle, suffix, lineStartOffset + lastEnd, searchMatches, lineIndex, lastEnd, trailingStart)
	}
	
	// Fill rest of line if we have background
	if hasBg {
		width, _ := e.screen.Size()
		for i := x; i < width; i++ {
			e.screen.SetContent(i, screenY, ' ', nil, normalStyle)
		}
	}
}

// drawContext draws the context lines at the top of the editor
func (e *Editor) drawContext() {
	if len(e.contextCache) == 0 {
		return
	}

	lineNumberStyle := e.config.Theme.GetStyle(e.config.Theme.LineNumber)
	contextBgStyle := e.config.Theme.GetStyle(e.config.Theme.Context)

	// Use cached line number width from buffer
	lineNumberWidth := e.buffer.LineNumberWidth

	// Get screen width
	width, _ := e.screen.Size()

	// Draw context lines with line numbers and syntax highlighting
	for i, lineNum := range e.contextCache {
		// Create an iterator at the context line
		iter := buffer.NewBufferIteratorAtLine(e.buffer, lineNum)
		if !iter.IsEOF() {
			// Fill the entire line with the context background
			for x := 0; x < width; x++ {
				e.screen.SetContent(x, i, ' ', nil, contextBgStyle)
			}

			// Draw line number with context background
			lineNumStr := fmt.Sprintf("%*d", lineNumberWidth-1, lineNum+1) // Line numbers are 1-based
			// Create a new style with context background
			_, contextBg, _ := contextBgStyle.Decompose()
			lineNumWithContextBg := lineNumberStyle.Background(contextBg)
			e.drawText(0, i, lineNumWithContextBg, lineNumStr, 0, nil, lineNum, 0, -1)

			// Draw separator with context background
			e.screen.SetContent(lineNumberWidth-1, i, ' ', nil, lineNumWithContextBg)

			// Draw the line content with syntax highlighting but context background
			e.drawContextLine(iter, lineNumberWidth, i, contextBg)
		}
	}
}

func (e *Editor) drawEditor() {
	lineNumberStyle := e.config.Theme.GetStyle(e.config.Theme.LineNumber)
	normalStyle := e.config.Theme.GetStyle(e.config.Theme.Normal)

	// Use cached line number width from buffer
	lineNumberWidth := e.buffer.LineNumberWidth

	// Clear the entire screen once
	e.screen.Clear()

	// Update context if needed (must be done before getContentStartY)
	e.updateContext()

	// Get screen dimensions
	width, _ := e.screen.Size()
	visibleHeight := e.getVisibleHeight()
	contentStartY := e.getContentStartY()

	// Start from the view iterator
	iter := e.buffer.ViewIterator
	
	for i := 0; i < visibleHeight; i++ {
		screenY := contentStartY + i
		// First, fill the entire line with the normal background
		for x := 0; x < width; x++ {
			e.screen.SetContent(x, screenY, ' ', nil, normalStyle)
		}

		// Special case: Empty buffer should still show line 1
		if i == 0 && iter.IsEOF() && e.buffer.Rope.Length() == 0 {
			// Draw line number 1 for empty buffer
			lineNumStr := fmt.Sprintf("%*d", lineNumberWidth-1, 1)
			e.drawText(0, screenY, lineNumberStyle, lineNumStr, 0, nil, 0, 0, -1)

			// Draw separator after line number
			e.screen.SetContent(lineNumberWidth-1, screenY, ' ', nil, lineNumberStyle)
			// No content to draw, but this represents the single empty line
		} else if !iter.IsEOF() {
			lineIndex := iter.LineNumber()
			lineNum := lineIndex + 1 // Line numbers start at 1

			// Draw line number
			lineNumStr := fmt.Sprintf("%*d", lineNumberWidth-1, lineNum)
			e.drawText(0, screenY, lineNumberStyle, lineNumStr, 0, nil, lineIndex, 0, -1)

			// Draw separator after line number
			e.screen.SetContent(lineNumberWidth-1, screenY, ' ', nil, lineNumberStyle)

			// Draw the actual line content using iterator
			e.drawLine(iter, lineNumberWidth, screenY)

			// Move to next line
			iter = iter.Next()
		} else {
			// For lines beyond the buffer, show tilde or fill line number area
			for x := 0; x < lineNumberWidth; x++ {
				e.screen.SetContent(x, screenY, ' ', nil, lineNumberStyle)
			}
		}
	}
}

func (e *Editor) drawStatusBar() {
	width, height := e.screen.Size()
	statusBarStyle := e.config.Theme.GetStyle(e.config.Theme.StatusBar)
	statusBarStyle.Attrs |= ui.AttrBold // Make status bar text bold by default
	statusBarY := e.getStatusBarY()

	// Debug: ensure statusBarY is valid
	if statusBarY < 0 || statusBarY >= height {
		LL.Error("drawStatusBar: Invalid statusBarY=%d (height=%d)", statusBarY, height)
		return
	}

	// Fill status bar background
	for i := 0; i < width; i++ {
		e.screen.SetContent(i, statusBarY, ' ', nil, statusBarStyle)
	}

	if e.mode == commands.CommandMode {
		// Draw the command line widget
		e.commandLine.Draw(e.screen)
	} else if e.mode == commands.SearchMode {
		// Draw the search line widget
		e.searchLine.Draw(e.screen)
	} else {
		// Status bar content
		// Left side: mode and filename
		x := 0
		
		// Draw mode with inverted colors for highlighting
		modeStyle := statusBarStyle
		modeStyle.Fg, modeStyle.Bg = modeStyle.Bg, modeStyle.Fg // Invert colors
		
		modeText := " NORMAL "
		if e.mode == commands.InsertMode {
			modeText = " INSERT "
		} else if e.mode == commands.VisualMode {
			modeText = " VISUAL "
		}
		x += e.drawText(x, statusBarY, modeStyle, modeText, 0, nil, -1, 0, -1)
		x += e.drawText(x, statusBarY, statusBarStyle, " ", 0, nil, -1, 0, -1) // Space after mode

		// Show filename and LSP indicator
		filename := e.buffer.FileName()
		x += e.drawText(x, statusBarY, statusBarStyle, filename, 0, nil, -1, 0, -1)

		// Add LSP indicator if LSP is available
		if e.buffer.HasLSP {
			x += e.drawText(x, statusBarY, statusBarStyle, " ", 0, nil, -1, 0, -1)
		}

		// Add connection status or spinner if async operations are in progress
		if e.connectionStatus != "" || e.hasAsyncOperations() {
			// Use inverted colors for spinner/status
			spinnerStyle := statusBarStyle
			spinnerStyle.Fg, spinnerStyle.Bg = spinnerStyle.Bg, spinnerStyle.Fg
			
			x += e.drawText(x, statusBarY, statusBarStyle, "  ", 0, nil, -1, 0, -1) // Spacing
			
			if e.connectionStatus != "" {
				statusText := fmt.Sprintf(" %s %s ", e.getSpinnerChar(), e.connectionStatus)
				x += e.drawText(x, statusBarY, spinnerStyle, statusText, 0, nil, -1, 0, -1)
			} else {
				statusText := fmt.Sprintf(" %s ", e.getSpinnerChar())
				x += e.drawText(x, statusBarY, spinnerStyle, statusText, 0, nil, -1, 0, -1)
			}
		}

		// Right side: indentation and line:col

		// Use visual column position for display (accounting for tabs)
		visualCol := 1
		if !e.buffer.CursorIterator.IsEOF() {
			line := e.buffer.CursorIterator.Line()
			if e.buffer.CursorX <= len(line) {
				visualCol = e.visualWidth(line[:e.buffer.CursorX]) + 1
			}
		}

		// Combine indentation info and line:col
		var indentInfo string
		if e.buffer.Indentation.Type == config.IndentTabs {
			indentInfo = "Tabs"
		} else {
			indentInfo = fmt.Sprintf("Spaces:%d", e.buffer.Indentation.Width)
		}

		// Get current line indentation info
		lineIndent := e.getCurrentLineIndentInfo(e.buffer.CursorIterator)
		var lineIndentDisplay string
		if lineIndent.Level > 0 {
			lineIndentDisplay = fmt.Sprintf("L%d", lineIndent.Level)
			if lineIndent.IsContinuation {
				lineIndentDisplay += "+C"
			}
		} else {
			lineIndentDisplay = "L0"
		}

		// Get current line length
		currentLine := ""
		if !e.buffer.CursorIterator.IsEOF() {
			currentLine = e.buffer.CursorIterator.Line()
		}
		lineLength := len(currentLine)
		
		// Get subdued style for total counts (without bold)
		subduedStyle := e.config.Theme.GetStyle(e.config.Theme.StatusBarSubdued)
		
		// Build line info with mixed styles
		// Format: "40/65:1/16 [1234]"
		currentLineNum := e.buffer.CursorIterator.LineNumber() + 1
		totalLines := e.buffer.LineCount
		
		// Calculate the full right info for positioning
		fullRightInfo := fmt.Sprintf("%s  %s  %d/%d:%d/%d [%d]  ", 
			indentInfo, lineIndentDisplay, 
			currentLineNum, totalLines,
			visualCol, lineLength,
			e.getAbsolutePosition())
		
		// Calculate position for right-aligned text
		rightX := width - len(fullRightInfo)

		if rightX > x { // Only draw if there's space
			// Draw each part separately with appropriate styles
			x = rightX
			x += e.drawText(x, statusBarY, statusBarStyle, indentInfo+"  ", 0, nil, -1, 0, -1)
			x += e.drawText(x, statusBarY, statusBarStyle, lineIndentDisplay+"  ", 0, nil, -1, 0, -1)
			
			// Line info with mixed styles
			x += e.drawText(x, statusBarY, statusBarStyle, fmt.Sprintf("%d", currentLineNum), 0, nil, -1, 0, -1)
			x += e.drawText(x, statusBarY, subduedStyle, fmt.Sprintf("/%d", totalLines), 0, nil, -1, 0, -1)
			x += e.drawText(x, statusBarY, statusBarStyle, fmt.Sprintf(":%d", visualCol), 0, nil, -1, 0, -1)
			x += e.drawText(x, statusBarY, subduedStyle, fmt.Sprintf("/%d", lineLength), 0, nil, -1, 0, -1)
			x += e.drawText(x, statusBarY, statusBarStyle, fmt.Sprintf(" [%d]  ", e.getAbsolutePosition()), 0, nil, -1, 0, -1)
		}
	}
}

// drawSignature renders the function signature above the status bar
func (e *Editor) drawSignature() {
	LL.Trace("Drawing signature %v", e.buffer.CurrentSignature)
	if e.buffer.CurrentSignature == nil {
		return
	}

	width, _ := e.screen.Size()
	signatureY := e.getSignatureY()
	signatureStyle := e.config.Theme.GetStyle(e.config.Theme.Context)

	// Fill signature background
	e.fillLine(signatureY, signatureStyle)

	// Prepare display text
	sig := e.buffer.CurrentSignature
	displayText := e.formatSignatureText(sig)
	displayText = e.truncateText(displayText, width-2)

	LL.Debug("Drawing signature at Y=%d: %s", signatureY, displayText)

	// Find active parameter range
	activeStart, activeEnd := e.getActiveParameterRange(sig)

	// Draw the signature
	if e.highlighter != nil {
		e.drawSignatureWithHighlighting(displayText, signatureY, signatureStyle, activeStart, activeEnd)
	} else {
		e.drawSignatureWithBold(displayText, signatureY, signatureStyle, activeStart, activeEnd)
	}
}

// fillLine fills an entire line with the given style
func (e *Editor) fillLine(y int, style ui.Style) {
	width, _ := e.screen.Size()
	for i := 0; i < width; i++ {
		e.screen.SetContent(i, y, ' ', nil, style)
	}
}

// formatSignatureText formats the signature for display
func (e *Editor) formatSignatureText(sig *buffer.SignatureHelp) string {
	if len(sig.Parameters) > 0 && sig.ActiveParameter >= 0 && sig.ActiveParameter < len(sig.Parameters) {
		return fmt.Sprintf("%s  [%d/%d]", sig.Label, sig.ActiveParameter+1, len(sig.Parameters))
	}
	return sig.Label
}

// truncateText truncates text if it exceeds maxLen
func (e *Editor) truncateText(text string, maxLen int) string {
	if len(text) > maxLen {
		return text[:maxLen-3] + "..."
	}
	return text
}

// getActiveParameterRange returns the start and end positions of the active parameter
func (e *Editor) getActiveParameterRange(sig *buffer.SignatureHelp) (int, int) {
	if sig.ActiveParameter < 0 || sig.ActiveParameter >= len(sig.Parameters) {
		return -1, -1
	}

	param := sig.Parameters[sig.ActiveParameter]
	if param.LabelStart > 0 || param.LabelEnd > 0 {
		return param.LabelStart, param.LabelEnd
	}

	// Try to find parameter by string matching
	if param.Label != "" {
		if idx := strings.Index(sig.Label, param.Label); idx >= 0 {
			return idx, idx + len(param.Label)
		}
	}

	return -1, -1
}

// drawSignatureWithHighlighting draws the signature with syntax highlighting and bold active parameter
func (e *Editor) drawSignatureWithHighlighting(text string, y int, baseStyle ui.Style, activeStart, activeEnd int) {
	tokens := e.highlighter.GetTokensForString(text)
	if len(tokens) == 0 {
		e.drawSignatureWithBold(text, y, baseStyle, activeStart, activeEnd)
		return
	}

	x := 0
	lastEnd := 0

	for _, token := range tokens {
		// Draw text before token
		if token.Start > lastEnd {
			beforeText := text[lastEnd:token.Start]
			style := e.applyBoldIfInRange(baseStyle, lastEnd, token.Start, activeStart, activeEnd)
			x += e.drawText(x, y, style, beforeText, 0, nil, -1, 0, -1)
		}

		// Draw token
		if token.Start < len(text) {
			endPos := min(token.End, len(text))
			tokenText := text[token.Start:endPos]
			tokenStyle := e.config.Theme.GetStyle(syntax.GetStyleForToken(token.TokenType, e.config.Theme))
			tokenStyle = tokenStyle.Background(baseStyle.Bg)
			tokenStyle = e.applyBoldIfInRange(tokenStyle, token.Start, endPos, activeStart, activeEnd)
			x += e.drawText(x, y, tokenStyle, tokenText, 0, nil, -1, 0, -1)
			lastEnd = endPos
		}
	}

	// Draw remaining text
	if lastEnd < len(text) {
		remainingText := text[lastEnd:]
		style := e.applyBoldIfInRange(baseStyle, lastEnd, len(text), activeStart, activeEnd)
		e.drawText(x, y, style, remainingText, 0, nil, -1, 0, -1)
	}
}

// drawSignatureWithBold draws the signature with bold active parameter but no syntax highlighting
func (e *Editor) drawSignatureWithBold(text string, y int, style ui.Style, activeStart, activeEnd int) {
	if activeStart < 0 || activeEnd <= activeStart {
		e.drawText(0, y, style, text, 0, nil, -1, 0, -1)
		return
	}

	x := 0

	// Before active parameter
	if activeStart > 0 {
		x += e.drawText(x, y, style, text[:activeStart], 0, nil, -1, 0, -1)
	}

	// Active parameter (bold)
	if activeEnd <= len(text) {
		boldStyle := style
		boldStyle.Attrs |= ui.AttrBold
		x += e.drawText(x, y, boldStyle, text[activeStart:activeEnd], 0, nil, -1, 0, -1)
	}

	// After active parameter
	if activeEnd < len(text) {
		e.drawText(x, y, style, text[activeEnd:], 0, nil, -1, 0, -1)
	}
}

// applyBoldIfInRange applies bold attribute if the range overlaps with the active parameter
func (e *Editor) applyBoldIfInRange(style ui.Style, start, end, activeStart, activeEnd int) ui.Style {
	if activeStart >= 0 && e.rangesOverlap(start, end, activeStart, activeEnd) {
		style.Attrs |= ui.AttrBold
	}
	return style
}

// rangesOverlap checks if two ranges overlap
func (e *Editor) rangesOverlap(start1, end1, start2, end2 int) bool {
	return start1 < end2 && start2 < end1
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

