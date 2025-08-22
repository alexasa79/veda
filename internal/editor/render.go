package editor

import (
	"fmt"
	"strings"
	"veda/internal/buffer"
	"veda/internal/commands"
	"veda/internal/config"
	"veda/internal/proxy"
	"veda/internal/syntax"
	"veda/internal/theme"
	"veda/internal/ui"
)

// drawGutterGitIndicator renders git added/modified indicator in the gutter for a given wrapped line.
// - Renders a space with background color corresponding to the change type
// - Deleted indicators are intentionally not rendered
func (e *Editor) drawGutterGitIndicator(wl WrappedLine, lineNum, digitsWidth, lineNumberWidth, screenY int, lineNumberStyle ui.Style) {
	if e.buffer.GitChanges == nil {
		return
	}

	isAdded := false
	isModified := false

	// Added/Modified are 1-based
	for _, a := range e.buffer.GitChanges.Added {
		if a == lineNum {
			isAdded = true
			break
		}
	}
	if !isAdded {
		for _, m := range e.buffer.GitChanges.Modified {
			if m == lineNum {
				isModified = true
				break
			}
		}
	}

	// Deleted lines: do not render any indicator

	// Leftmost gutter column (immediately after digits) is used for git indicators
	indicatorX := digitsWidth
	if indicatorX >= lineNumberWidth {
		return
	}

	// Only draw when added or modified
	if !(isAdded || isModified) {
		return
	}

	// Use distinct glyphs for added vs modified
	var ch rune
	indicatorStyle := lineNumberStyle
	_, lnBg, _ := lineNumberStyle.Decompose()
	if isAdded {
		ch = '+'
		indicatorStyle = e.config.Theme.GetStyle(e.config.Theme.GitAdded).Background(lnBg)
	} else {
		ch = '≋'
		indicatorStyle = e.config.Theme.GetStyle(e.config.Theme.GitModified).Background(lnBg)
	}
	// Make the indicator bold for visibility
	indicatorStyle.Attrs |= ui.AttrBold

	e.screen.SetContent(indicatorX, screenY, ch, nil, indicatorStyle)
}

// drawGutterDiagnosticsIndicator renders a diagnostics icon in the second gutter column
func (e *Editor) drawGutterDiagnosticsIndicator(wl WrappedLine, lineNum, digitsWidth, lineNumberWidth, screenY int, lineNumberStyle ui.Style) {
	if e.buffer.Diagnostics == nil || len(e.buffer.Diagnostics) == 0 {
		return
	}

	diag, ok := e.buffer.GetDiagnosticForLine(lineNum)
	if !ok {
		return
	}

	// Diagnostics in middle gutter column: digits + 1
	diagX := digitsWidth + 1
	if diagX >= lineNumberWidth {
		return
	}

	var style ui.Style
	_, lnBg, _ := lineNumberStyle.Decompose()
	switch diag.Severity {
	case "error":
		style = e.config.Theme.GetStyle(e.config.Theme.DiagnosticError).Background(lnBg)
	case "warning":
		style = e.config.Theme.GetStyle(e.config.Theme.DiagnosticWarning).Background(lnBg)
	default:
		// Skip info/hint for now
		return
	}
	e.screen.SetContent(diagX, screenY, '●', nil, style)
}

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
	if lineNum >= 0 && e.buffer != nil && e.hasFlashHighlight() {
		lineStartIter := buffer.NewBufferIteratorAtLine(e.buffer, lineNum)
		lineStartPos = lineStartIter.Position()
	}

	// Convert to slice of runes for position tracking
	runes := []rune(text)
	// Fetch diagnostic for this line (1-based)
	var diagForLine *proxy.Diagnostic
	var hasDiag bool
	if lineNum >= 0 && e.buffer != nil && len(e.buffer.Diagnostics) > 0 {
		if d, ok := e.buffer.GetDiagnosticForLine(lineNum + 1); ok {
			diagForLine = d
			hasDiag = true
		}
	}

	for i, r := range runes {
		// Check if this character is part of a search match
		// searchMatches contains positions relative to the line, not absolute positions
		charPosInLine := segmentStart + i
		useSearchStyle := e.isInSearchMatchInLine(charPosInLine, searchMatches)
		// Check if this character is part of the selection
		// Only apply selection style if we're not drawing line numbers (x parameter is 0 for line numbers)
		inSelection := hasSelection && charPos >= selStart && charPos < selEnd && x > 0
		// Check if this character is part of the flash highlight
		var inFlash bool
		if lineNum >= 0 && e.hasFlashHighlight() {
			charPosInLine := charPos - lineStartPos
			inFlash = e.isInFlashRange(lineNum, charPosInLine)
		}

		currentStyle := style

		// Determine if this position is within a diagnostic range for this line
		useDiagStyle := false
		var diagStyle ui.Style
		if hasDiag && (diagForLine.Severity == "error" || diagForLine.Severity == "warning") {
			// Compare with line-relative position
			posInLine := segmentStart + i
			startChar := diagForLine.StartChar
			endChar := diagForLine.EndChar
			if endChar <= startChar {
				endChar = startChar + 1 // ensure at least one column highlighted
			}
			if posInLine >= startChar && posInLine < endChar {
				useDiagStyle = true
				if diagForLine.Severity == "error" {
					diagStyle = e.config.Theme.GetStyle(e.config.Theme.DiagnosticError)
				} else {
					diagStyle = e.config.Theme.GetStyle(e.config.Theme.DiagnosticWarning)
				}
			}
		}

		// Check if this position has a virtual cursor
		hasVirtualCursor := false
		// Virtual cursors store the logical column position (CursorX)
		// We need to compare with the actual position in the line
		// Only check for virtual cursors if we're rendering in the text area (not line numbers)
		if e.buffer != nil && screenX >= e.buffer.LineNumberWidth && lineNum >= 0 {
			for _, vc := range e.virtualCursors {
				if vc.Y == lineNum && vc.X == charPosInLine {
					hasVirtualCursor = true
					break
				}
			}
		}

		// Note: Do not apply vertical ruler background on characters themselves.
		// Ruler background is rendered in background fill phases to avoid per-character
		// style variation within tokens (important for multi-window consistency tests).

		if inFlash {
			// Flash takes highest precedence
			currentStyle = theme.ApplyBackgroundStyle(style, flashStyle)
		} else if inSelection {
			// Selection takes precedence over search
			currentStyle = theme.ApplyBackgroundStyle(style, selectionStyle)
		} else if hasVirtualCursor {
			// Virtual cursor gets a distinct background color
			virtualCursorStyle := e.config.Theme.GetStyle(e.config.Theme.VirtualCursor)
			currentStyle = theme.ApplyBackgroundStyle(style, virtualCursorStyle)
		} else if useDiagStyle {
			// Apply diagnostic background for error/warning range
			currentStyle = theme.ApplyBackgroundStyle(style, diagStyle)
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

// isInSearchMatchInLine checks if a line-relative position is within a search match
// pos is the character position within the line (0-based)
// matches contains line-relative match positions
func (e *Editor) isInSearchMatchInLine(pos int, matches []int) bool {
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
	originalLine := iter.Line() // Keep the original line for whitespace detection
	lineIndex := iter.LineNumber()
	normalStyle := e.config.Theme.GetStyle(e.config.Theme.Normal)
	isCurrentLine := e.buffer != nil && lineIndex == e.buffer.CursorIterator.LineNumber()

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

	// If this is a terminal/internal buffer, render using terminal emulator styles (regardless of mode)
	if e.buffer != nil && e.buffer.IsInternal {
		// Determine terminal row corresponding to this buffer line
		ts := e.terminalSessions[e.buffer.FilePath]
		if ts != nil && ts.term != nil {
			width, _ := e.screen.Size()
			// Draw characters with per-cell styles; iterate runes to avoid breaking UTF-8
			col := 0
			for _, ch := range originalLine {
				if startX+col >= width {
					break
				}
				// Adjust for banner occupying first buffer line
				emuRow := lineIndex
				if ts.Banner != "" {
					emuRow = lineIndex - 1
				}
				var fg, bg ui.Color
				var attrs ui.AttrMask
				if emuRow >= 0 {
					// emuRow indexes into combined scrollback + grid
					fg, bg, attrs = ts.term.StyleAtGlobal(col, emuRow)
				} else {
					fg, bg, attrs = ui.ColorDefault, ui.ColorDefault, ui.AttrNone
				}
				// Replace any zero-width or control rendering remnants with spaces visually
				if ch == 0 {
					ch = ' '
				}
				// Upgrade basic 8-color foreground to bright when bold is set (common terminal behavior)
				if attrs&ui.AttrBold != 0 && fg >= ui.ColorBlack && fg <= ui.ColorWhite {
					fg = ui.Color(int32(fg) + 8)
				}
				style := normalStyle
				if fg != ui.ColorDefault {
					style = style.Foreground(fg)
				}
				if bg != ui.ColorDefault {
					style = style.Background(bg)
				}
				if attrs != ui.AttrNone {
					// Merge attributes into style
					fg0, bg0, _ := style.Decompose()
					style = ui.Style{Fg: fg0, Bg: bg0, Attrs: attrs}
				}
				e.screen.SetContent(startX+col, screenY, ch, nil, style)
				col++
			}
			// Fill remainder of line if needed
			return
		}
	}

	// Try to get syntax tokens first, even with background overrides
	var x int

	// Get syntax tokens for this line using iterator position
    // Always use the buffer's own highlighter
    var tokens []syntax.Token
    if e.buffer != nil && e.buffer.Highlighter != nil {
        tokens = e.buffer.Highlighter.GetTokensForLine(iter.Position(), iter.LineEnd())
    }

    // If we have syntax tokens and a file-backed buffer, use syntax highlighting path
    if len(tokens) > 0 && e.buffer != nil && e.buffer.HasFile() {
		// Use syntax highlighting (this will be handled below)
    } else if e.buffer == nil || !e.buffer.HasFile() {
		x = startX
		x += e.drawText(startX, screenY, normalStyle, originalLine, lineStartOffset, searchMatches, lineIndex, 0, trailingStart)
		// Fill rest of line if we have background
		if hasBg {
			width, _ := e.screen.Size()
			virtualCursorStyle := e.config.Theme.GetStyle(e.config.Theme.VirtualCursor)

			for i := x; i < width; i++ {
				fillStyle := normalStyle

				// Check if this position has a virtual cursor beyond the end of line
				hasVirtualCursor := false
				if lineIndex >= 0 {
					// Calculate the character position in the line for this screen position
					textColumn := i - e.buffer.LineNumberWidth
					charPosInLine := textColumn
					for _, vc := range e.virtualCursors {
						if vc.Y == lineIndex && vc.X == charPosInLine {
							hasVirtualCursor = true
							break
						}
					}
				}

				// Apply virtual cursor style if needed
				if hasVirtualCursor {
					fillStyle = theme.ApplyBackgroundStyle(normalStyle, virtualCursorStyle)
				}

				e.screen.SetContent(i, screenY, ' ', nil, fillStyle)
			}
		}
	} else {
		// Even if no background fill, we still need to render virtual cursors beyond end of line
		width, _ := e.screen.Size()
		virtualCursorStyle := e.config.Theme.GetStyle(e.config.Theme.VirtualCursor)

		for i := x; i < width; i++ {
			// Check if this position has a virtual cursor beyond the end of line
			hasVirtualCursor := false
			if lineIndex >= 0 {
				// Calculate the character position in the line for this screen position
				textColumn := i - e.buffer.LineNumberWidth
				for _, vc := range e.virtualCursors {
					if vc.Y == lineIndex && vc.X == textColumn {
						hasVirtualCursor = true
						break
					}
				}
			}

			// Render virtual cursor if needed
			if hasVirtualCursor {
				fillStyle := theme.ApplyBackgroundStyle(normalStyle, virtualCursorStyle)
				e.screen.SetContent(i, screenY, ' ', nil, fillStyle)
			}
		}
	}

	// Handle syntax highlighting path
    if len(tokens) > 0 && e.buffer != nil && e.buffer.HasFile() {
		// Draw with syntax highlighting
		x = startX
		lastEnd := 0

		for _, token := range tokens {
			// Draw any text before this token with normal style
			if token.Start > lastEnd && token.Start <= len(originalLine) {
				endPos := token.Start
				if endPos > len(originalLine) {
					endPos = len(originalLine)
				}
				prefix := originalLine[lastEnd:endPos]
				x += e.drawText(x, screenY, normalStyle, prefix, lineStartOffset+lastEnd, searchMatches, lineIndex, lastEnd, trailingStart)
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

				x += e.drawText(x, screenY, tokenStyle, tokenText, lineStartOffset+token.Start, searchMatches, lineIndex, token.Start, trailingStart)
				lastEnd = endPos
			}
		}

		// Draw any remaining text with normal style
		if lastEnd < len(originalLine) {
			suffix := originalLine[lastEnd:]
			x += e.drawText(x, screenY, normalStyle, suffix, lineStartOffset+lastEnd, searchMatches, lineIndex, lastEnd, trailingStart)
		}
	} else {
		// Use standard rendering for files without syntax highlighting
		// No tokens, draw with normal style
		x = startX
		x += e.drawText(startX, screenY, normalStyle, originalLine, lineStartOffset, searchMatches, lineIndex, 0, trailingStart)
	}

	// Fill rest of line with background if needed
	if hasBg {
		width, _ := e.screen.Size()
		virtualCursorStyle := e.config.Theme.GetStyle(e.config.Theme.VirtualCursor)

		for i := x; i < width; i++ {
			fillStyle := normalStyle

			// Check if this position has a virtual cursor beyond the end of line
			hasVirtualCursor := false
			if lineIndex >= 0 {
				// Calculate the character position in the line for this screen position
				textColumn := i - e.buffer.LineNumberWidth
				charPosInLine := textColumn
				for _, vc := range e.virtualCursors {
					if vc.Y == lineIndex && vc.X == charPosInLine {
						hasVirtualCursor = true
						break
					}
				}
			}

			// Apply virtual cursor style if needed
			if hasVirtualCursor {
				fillStyle = theme.ApplyBackgroundStyle(normalStyle, virtualCursorStyle)
			}

			e.screen.SetContent(i, screenY, ' ', nil, fillStyle)
		}
	} else {
		// Even if no background fill, we still need to render virtual cursors beyond end of line
		width, _ := e.screen.Size()
		virtualCursorStyle := e.config.Theme.GetStyle(e.config.Theme.VirtualCursor)

		for i := x; i < width; i++ {
			// Check if this position has a virtual cursor beyond the end of line
			hasVirtualCursor := false
			if lineIndex >= 0 {
				// Calculate the character position in the line for this screen position
				textColumn := i - e.buffer.LineNumberWidth
				for _, vc := range e.virtualCursors {
					if vc.Y == lineIndex && vc.X == textColumn {
						hasVirtualCursor = true
						break
					}
				}
			}

			// Render virtual cursor if needed
			if hasVirtualCursor {
				fillStyle := theme.ApplyBackgroundStyle(normalStyle, virtualCursorStyle)
				e.screen.SetContent(i, screenY, ' ', nil, fillStyle)
			}
		}
	}
}

// drawWrappedLine draws a wrapped line segment with proper handling of syntax highlighting
func (e *Editor) drawWrappedLine(iter *buffer.BufferIterator, wl WrappedLine, startX int, screenY int) {
	// Get the full line for syntax highlighting
	originalLine := iter.Line()
	lineIndex := iter.LineNumber()
	normalStyle := e.config.Theme.GetStyle(e.config.Theme.Normal)
	isCurrentLine := e.buffer != nil && lineIndex == e.buffer.CursorIterator.LineNumber()

	// Find search matches in the full line
	searchMatches := e.findSearchMatches(originalLine)

	// Determine background style
	var lineBg ui.Color
	var hasBg bool
	if isCurrentLine {
		currentLineStyle := e.config.Theme.GetStyle(e.config.Theme.CurrentLine)
		_, lineBg, _ = currentLineStyle.Decompose()
		normalStyle = normalStyle.Background(lineBg)
		hasBg = true
	}

	// Get the absolute offset of the start of this line
	lineStartOffset := iter.Position()

	// Defensive clamping of wrapped segment bounds against actual line length
	lineLen := len(originalLine)
	segStart := wl.StartChar
	segEnd := wl.EndChar
	if segStart < 0 {
		segStart = 0
	}
	if segStart > lineLen {
		segStart = lineLen
	}
	if segEnd < segStart {
		segEnd = segStart
	}
	if segEnd > lineLen {
		segEnd = lineLen
	}

	// Find trailing spaces position for the whole line
	trailingStart := -1
	if e.whitespaceStyle.showTrailingSpaces {
		trailingStart = detectTrailingSpaces(originalLine)
	}

	// Terminal/internal buffers: draw using terminal emulator per-cell styles (regardless of mode)
	if e.buffer != nil && e.buffer.IsInternal {
		if ts := e.terminalSessions[e.buffer.FilePath]; ts != nil && ts.term != nil {
			// Draw only the wrapped segment using terminal styles; iterate runes to preserve UTF-8
			x := startX
			col := 0
			for _, ch := range wl.Text {
				emuRow := lineIndex
				if ts.Banner != "" {
					emuRow = lineIndex - 1
				}
				var fg, bg ui.Color
				var attrs ui.AttrMask
				if emuRow >= 0 {
					fg, bg, attrs = ts.term.StyleAtGlobal(col, emuRow)
				} else {
					fg, bg, attrs = ui.ColorDefault, ui.ColorDefault, ui.AttrNone
				}
				if ch == 0 {
					ch = ' '
				}
				// Upgrade basic 8-color foreground to bright when bold is set (common terminal behavior)
				if attrs&ui.AttrBold != 0 && fg >= ui.ColorBlack && fg <= ui.ColorWhite {
					fg = ui.Color(int32(fg) + 8)
				}
				style := normalStyle
				if fg != ui.ColorDefault {
					style = style.Foreground(fg)
				}
				if bg != ui.ColorDefault {
					style = style.Background(bg)
				}
				if attrs != ui.AttrNone {
					fg0, bg0, _ := style.Decompose()
					style = ui.Style{Fg: fg0, Bg: bg0, Attrs: attrs}
				}
				// Determine display width to keep emulator columns aligned (e.g., emoji)
				w := runeDisplayWidth(ch)
				if w < 1 {
					w = 1
				}
				e.screen.SetContent(x, screenY, ch, nil, style)
				if w == 2 {
					// Paint the trailing cell for double-width glyphs with same style
					e.screen.SetContent(x+1, screenY, ' ', nil, style)
				}
				x += w
				col += w
			}
			return
		}
	}

    // If no highlighter, draw without syntax highlighting
    if e.buffer == nil || e.buffer.Highlighter == nil || !e.buffer.HasFile() {
		x := startX
		// Draw the wrapped segment
		segmentOffset := lineStartOffset + segStart
		// Clamp wl.Text usage to the clamped segment range
		textSegment := originalLine[segStart:segEnd]
		x += e.drawText(startX, screenY, normalStyle, textSegment, segmentOffset, searchMatches, lineIndex, segStart, trailingStart)

		// Fill rest of line if we have background
		if hasBg {
			width, _ := e.screen.Size()
			virtualCursorStyle := e.config.Theme.GetStyle(e.config.Theme.VirtualCursor)

			for i := x; i < width; i++ {
				fillStyle := normalStyle

				// Check if this position has a virtual cursor beyond the end of line
				hasVirtualCursor := false
				if lineIndex >= 0 {
					// Calculate the character position in the line for this screen position
					textColumn := i - e.buffer.LineNumberWidth
					charPosInLine := textColumn
					for _, vc := range e.virtualCursors {
						if vc.Y == lineIndex && vc.X == charPosInLine {
							hasVirtualCursor = true
							break
						}
					}
				}

				// Apply virtual cursor style if needed
				if hasVirtualCursor {
					fillStyle = theme.ApplyBackgroundStyle(normalStyle, virtualCursorStyle)
				}

				e.screen.SetContent(i, screenY, ' ', nil, fillStyle)
			}
		} else {
			// Even if no background fill, we still need to render virtual cursors beyond end of line
			width, _ := e.screen.Size()
			virtualCursorStyle := e.config.Theme.GetStyle(e.config.Theme.VirtualCursor)

			for i := x; i < width; i++ {
				// Check if this position has a virtual cursor beyond the end of line
				hasVirtualCursor := false
				if lineIndex >= 0 {
					// Calculate the character position in the line for this screen position
					textColumn := i - e.buffer.LineNumberWidth
					for _, vc := range e.virtualCursors {
						if vc.Y == lineIndex && vc.X == textColumn {
							hasVirtualCursor = true
							break
						}
					}
				}

				// Render virtual cursor if needed
				if hasVirtualCursor {
					fillStyle := theme.ApplyBackgroundStyle(normalStyle, virtualCursorStyle)
					e.screen.SetContent(i, screenY, ' ', nil, fillStyle)
				}
			}
		}
		return
	}

	// Get syntax tokens for the full line
    var tokens []syntax.Token
    if e.buffer != nil && e.buffer.Highlighter != nil {
        tokens = e.buffer.Highlighter.GetTokensForLine(iter.Position(), iter.LineEnd())
    }

	if len(tokens) == 0 {
		// No tokens, draw with normal style
		x := startX
		segmentOffset := lineStartOffset + segStart
		textSegment := originalLine[segStart:segEnd]
		x += e.drawText(startX, screenY, normalStyle, textSegment, segmentOffset, searchMatches, lineIndex, segStart, trailingStart)

		// Fill rest of line
		if hasBg {
			width, _ := e.screen.Size()
			virtualCursorStyle := e.config.Theme.GetStyle(e.config.Theme.VirtualCursor)

			for i := x; i < width; i++ {
				fillStyle := normalStyle

				// Check if this position has a virtual cursor beyond the end of line
				hasVirtualCursor := false
				if lineIndex >= 0 {
					// Calculate the character position in the line for this screen position
					textColumn := i - e.buffer.LineNumberWidth
					charPosInLine := textColumn
					for _, vc := range e.virtualCursors {
						if vc.Y == lineIndex && vc.X == charPosInLine {
							hasVirtualCursor = true
							break
						}
					}
				}

				// Apply virtual cursor style if needed
				if hasVirtualCursor {
					fillStyle = theme.ApplyBackgroundStyle(normalStyle, virtualCursorStyle)
				}

				e.screen.SetContent(i, screenY, ' ', nil, fillStyle)
			}
		} else {
			// Even if no background fill, we still need to render virtual cursors beyond end of line
			width, _ := e.screen.Size()
			virtualCursorStyle := e.config.Theme.GetStyle(e.config.Theme.VirtualCursor)

			for i := x; i < width; i++ {
				// Check if this position has a virtual cursor beyond the end of line
				hasVirtualCursor := false
				if lineIndex >= 0 {
					// Calculate the character position in the line for this screen position
					textColumn := i - e.buffer.LineNumberWidth
					for _, vc := range e.virtualCursors {
						if vc.Y == lineIndex && vc.X == textColumn {
							hasVirtualCursor = true
							break
						}
					}
				}

				// Render virtual cursor if needed
				if hasVirtualCursor {
					fillStyle := theme.ApplyBackgroundStyle(normalStyle, virtualCursorStyle)
					e.screen.SetContent(i, screenY, ' ', nil, fillStyle)
				}
			}
		}
		return
	}

	// Draw with syntax highlighting, but only for the wrapped segment
	x := startX
	pos := segStart

	for _, token := range tokens {
		// Skip tokens that end before our segment
		if token.End <= segStart {
			continue
		}

		// Stop if token starts after our segment
		if token.Start >= segEnd {
			break
		}

		// Draw any text before this token with normal style
		if token.Start > pos && pos < segEnd {
			endPos := token.Start
			if endPos > segEnd {
				endPos = segEnd
			}
			if endPos > pos {
				text := originalLine[pos:endPos]
				x += e.drawText(x, screenY, normalStyle, text, lineStartOffset+pos, searchMatches, lineIndex, pos, trailingStart)
				pos = endPos
			}
		}

		// Draw the token portion that's in our segment
		if token.Start < segEnd && token.End > pos {
			tokenStart := token.Start
			if tokenStart < pos {
				tokenStart = pos
			}
			tokenEnd := token.End
			if tokenEnd > segEnd {
				tokenEnd = segEnd
			}

			if tokenEnd > tokenStart {
				tokenText := originalLine[tokenStart:tokenEnd]
				tokenStyle := e.config.Theme.GetStyle(syntax.GetStyleForToken(token.TokenType, e.config.Theme))
				if isCurrentLine {
					tokenStyle = tokenStyle.Background(lineBg)
				}

				x += e.drawText(x, screenY, tokenStyle, tokenText, lineStartOffset+tokenStart, searchMatches, lineIndex, tokenStart, trailingStart)
				pos = tokenEnd
			}
		}
	}

	// Draw any remaining text with normal style
	if pos < segEnd {
		text := originalLine[pos:segEnd]
		x += e.drawText(x, screenY, normalStyle, text, lineStartOffset+pos, searchMatches, lineIndex, pos, trailingStart)
	}

	// Fill rest of line if we have background
	if hasBg {
		width, _ := e.screen.Size()
		rulerStyle := e.config.Theme.GetStyle(e.config.Theme.VerticalRuler)
		virtualCursorStyle := e.config.Theme.GetStyle(e.config.Theme.VirtualCursor)

		for i := x; i < width; i++ {
			fillStyle := normalStyle

			// Check if this column is a ruler position
			textColumn := i - e.buffer.LineNumberWidth
			for _, rulerPos := range e.config.VerticalRulers {
				if textColumn == rulerPos {
					fillStyle = theme.ApplyBackgroundStyle(normalStyle, rulerStyle)
					break
				}
			}

			// Check if this position has a virtual cursor beyond the end of line
			hasVirtualCursor := false
			if lineIndex >= 0 {
				// Calculate the character position in the line for this screen position
				charPosInLine := textColumn
				for _, vc := range e.virtualCursors {
					if vc.Y == lineIndex && vc.X == charPosInLine {
						hasVirtualCursor = true
						break
					}
				}

				// Special-case: zero-length diagnostic at end-of-line (e.g., missing semicolon)
				// When a diagnostic range starts and ends at EOL, there are no characters to paint.
				// Paint the single cell immediately after the last character of the final wrapped segment.
				if e.buffer != nil && len(e.buffer.Diagnostics) > 0 && wl.EndChar == len(originalLine) {
					if diag, ok := e.buffer.GetDiagnosticForLine(lineIndex + 1); ok {
						if (diag.Severity == "error" || diag.Severity == "warning") && diag.EndChar <= diag.StartChar && diag.StartChar >= len(originalLine) {
							// Use the current x position which is just after the drawn content of this segment
							width, _ := e.screen.Size()
							if x >= startX && x < width {
								var diagStyle ui.Style
								if diag.Severity == "error" {
									diagStyle = e.config.Theme.GetStyle(e.config.Theme.DiagnosticError)
								} else {
									diagStyle = e.config.Theme.GetStyle(e.config.Theme.DiagnosticWarning)
								}
								markerStyle := theme.ApplyBackgroundStyle(normalStyle, diagStyle)
								e.screen.SetContent(x, screenY, ' ', nil, markerStyle)
							}
						}
					}
				}
			}

			// Apply virtual cursor style if needed
			if hasVirtualCursor {
				fillStyle = theme.ApplyBackgroundStyle(normalStyle, virtualCursorStyle)
			}

			e.screen.SetContent(i, screenY, ' ', nil, fillStyle)
		}
	} else {
		// Even if no background fill, we still need to render virtual cursors beyond end of line
		width, _ := e.screen.Size()
		virtualCursorStyle := e.config.Theme.GetStyle(e.config.Theme.VirtualCursor)

		for i := x; i < width; i++ {
			// Check if this position has a virtual cursor beyond the end of line
			hasVirtualCursor := false
			if lineIndex >= 0 {
				// Calculate the character position in the line for this screen position
				textColumn := i - e.buffer.LineNumberWidth
				for _, vc := range e.virtualCursors {
					if vc.Y == lineIndex && vc.X == textColumn {
						hasVirtualCursor = true
						break
					}
				}
			}

			// Render virtual cursor if needed
			if hasVirtualCursor {
				fillStyle := theme.ApplyBackgroundStyle(normalStyle, virtualCursorStyle)
				e.screen.SetContent(i, screenY, ' ', nil, fillStyle)
			}
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
			// Reserve 2 columns for indicators and 1 space padding
			digitsWidth := lineNumberWidth - 3
			if digitsWidth < 1 {
				digitsWidth = 1
			}
			lineNumStr := fmt.Sprintf("%*d", digitsWidth, lineNum+1) // Line numbers are 1-based
			// Create a new style with context background
			_, contextBg, _ := contextBgStyle.Decompose()
			lineNumWithContextBg := lineNumberStyle.Background(contextBg)
			e.drawText(0, i, lineNumWithContextBg, lineNumStr, 0, nil, lineNum, 0, -1)

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

	// Update context if needed
	e.updateContext()

	// Get screen dimensions
	width, _ := e.screen.Size()
	contentStartY := e.getContentStartY()

	// Calculate wrapped lines
	wrappedLines := e.CalculateWrappedLines()

	// Prepare iterator reuse across the entire visible region
	var cachedIter *buffer.BufferIterator
	cachedIterLine := -1
	if len(wrappedLines) > 0 {
		firstLogical := wrappedLines[0].LogicalLineNum
		cachedIter = buffer.NewBufferIteratorAtLine(e.buffer, firstLogical)
		cachedIterLine = cachedIter.LineNumber()
	}

	// Draw each wrapped line
	for i, wl := range wrappedLines {
		screenY := contentStartY + i

		// First, fill the entire line with the normal background
		rulerStyle := e.config.Theme.GetStyle(e.config.Theme.VerticalRuler)
		for x := 0; x < width; x++ {
			fillStyle := normalStyle
			textColumn := x - e.buffer.LineNumberWidth
			for _, rulerPos := range e.config.VerticalRulers {
				if textColumn == rulerPos {
					fillStyle = theme.ApplyBackgroundStyle(normalStyle, rulerStyle)
					break
				}
			}
			e.screen.SetContent(x, screenY, ' ', nil, fillStyle)
		}

		// Draw line number only for first visual line
		if wl.IsFirstLine {
			lineNum := wl.LogicalLineNum + 1 // Line numbers start at 1
			// Draw line number with standard line number style
			// Compute digit area width (exclude 2 indicator columns and 1 space padding)
			digitsWidth := lineNumberWidth - 3
			if digitsWidth < 1 {
				digitsWidth = 1
			}
			lineNumStr := fmt.Sprintf("%*d", digitsWidth, lineNum)
			// Fill entire line number area with the standard background
			for x := 0; x < lineNumberWidth; x++ {
				e.screen.SetContent(x, screenY, ' ', nil, lineNumberStyle)
			}
			// Draw the line number text starting at column 0
			e.drawText(0, screenY, lineNumberStyle, lineNumStr, 0, nil, wl.LogicalLineNum, 0, -1)
			// Draw 1 space padding after digits
			padX := digitsWidth
			if padX < lineNumberWidth {
				e.screen.SetContent(padX, screenY, ' ', nil, lineNumberStyle)
			}

			// Draw git change indicators in the first gutter column
			e.drawGutterGitIndicator(wl, lineNum, digitsWidth, lineNumberWidth, screenY, lineNumberStyle)
			// Draw diagnostics in the second gutter column
			e.drawGutterDiagnosticsIndicator(wl, lineNum, digitsWidth, lineNumberWidth, screenY, lineNumberStyle)
		} else {
			// Fill line number area with spaces for wrapped lines
			for x := 0; x < lineNumberWidth; x++ {
				e.screen.SetContent(x, screenY, ' ', nil, lineNumberStyle)
			}
		}

		// Get iterator for this logical line (reuse, advance with Next when needed)
		var iter *buffer.BufferIterator
		if cachedIter == nil {
			// No wrapped lines or initialization failed; fall back to direct creation
			iter = buffer.NewBufferIteratorAtLine(e.buffer, wl.LogicalLineNum)
			cachedIter = iter
			cachedIterLine = wl.LogicalLineNum
		} else {
			// Advance cached iterator to requested logical line
			for cachedIterLine < wl.LogicalLineNum {
				cachedIter = cachedIter.Next()
				cachedIterLine = cachedIter.LineNumber()
			}
			iter = cachedIter
		}
		if !iter.IsEOF() {
			// Draw the wrapped line segment
			e.drawWrappedLine(iter, wl, lineNumberWidth, screenY)
		}
	}

	// Fill remaining screen lines for this window region.
	// Previously we only drew tildes in the gutter and left the content area
	// untouched, which could leave stale cells when rendering into a sub-screen
	// (right window) after a full-screen draw. Paint the entire line background
	// first, then overlay the tildes in the gutter to match typical editor UX.
	visibleHeight := e.getVisibleHeight()
	// No vertical ruler application in this background pass to avoid token style skew
	for i := len(wrappedLines); i < visibleHeight; i++ {
		screenY := contentStartY + i
		// Fill the whole line with normal background only (no per-column rulers here)
		for x := 0; x < width; x++ {
			e.screen.SetContent(x, screenY, ' ', nil, normalStyle)
		}
		// Draw the tildes in the gutter area
		for x := 0; x < lineNumberWidth; x++ {
			e.screen.SetContent(x, screenY, '~', nil, lineNumberStyle)
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
		} else if e.mode == commands.VisualLineMode {
			modeText = " VISUAL LINE "
		} else if e.mode == commands.CursorMode {
			modeText = " CURSOR "
		} else if e.mode == commands.TerminalMode {
			modeText = " TERMINAL "
		} else if e.mode == commands.NormalMode && e.temporaryNormal {
			modeText = " NORMAL (i) "
		}
		x += e.drawText(x, statusBarY, modeStyle, modeText, 0, nil, -1, 0, -1)
		x += e.drawText(x, statusBarY, statusBarStyle, " ", 0, nil, -1, 0, -1) // Space after mode

		// Show filename and modified indicator (only if buffer exists)
		if e.buffer != nil {
			filename := e.buffer.FileName()
			x += e.drawText(x, statusBarY, statusBarStyle, filename, 0, nil, -1, 0, -1)

			// Add modified indicator if buffer is modified
			if e.buffer.IsModified {
				x += e.drawText(x, statusBarY, statusBarStyle, " [+]", 0, nil, -1, 0, -1)
			}
		} else {
			// Show empty window indicator
			x += e.drawText(x, statusBarY, statusBarStyle, "[empty window]", 0, nil, -1, 0, -1)
		}

		// Add git branch and change counts (only if buffer exists)
		if e.buffer != nil && e.buffer.GitBranch != "" {
			x += e.drawText(x, statusBarY, statusBarStyle, fmt.Sprintf("  %s", e.buffer.GitBranch), 0, nil, -1, 0, -1)

			// Count added and deleted lines
			addedCount := 0
			deletedCount := 0
			if e.buffer.GitChanges != nil {
				addedCount = len(e.buffer.GitChanges.Added) + len(e.buffer.GitChanges.Modified)
				deletedCount = len(e.buffer.GitChanges.Deleted)
			}

			if addedCount > 0 || deletedCount > 0 {
				x += e.drawText(x, statusBarY, statusBarStyle, fmt.Sprintf("  +%d-%d", addedCount, deletedCount), 0, nil, -1, 0, -1)
			}
		}

		// Add diagnostic counts if there are any (only if buffer exists)
		if e.buffer != nil && e.buffer.Diagnostics != nil && len(e.buffer.Diagnostics) > 0 {
			errorCount := 0
			warningCount := 0

			// Count errors and warnings
			for _, diag := range e.buffer.Diagnostics {
				switch diag.Severity {
				case "error":
					errorCount++
				case "warning":
					warningCount++
				}
			}

			// Show diagnostic counts with colored circles
			if errorCount > 0 || warningCount > 0 {
				x += e.drawText(x, statusBarY, statusBarStyle, " ", 0, nil, -1, 0, -1) // Space before diagnostics

				// Red circle for errors
				if errorCount > 0 {
					errorStyle := statusBarStyle
					errorStyle.Fg = ui.ColorRed
					x += e.drawText(x, statusBarY, errorStyle, "●", 0, nil, -1, 0, -1)
					x += e.drawText(x, statusBarY, statusBarStyle, fmt.Sprintf("%d", errorCount), 0, nil, -1, 0, -1)
				}

				// Yellow circle for warnings (with space if we have both)
				if warningCount > 0 {
					if errorCount > 0 {
						x += e.drawText(x, statusBarY, statusBarStyle, " ", 0, nil, -1, 0, -1) // Space between error and warning
					}
					warningStyle := statusBarStyle
					warningStyle.Fg = ui.ColorYellow
					x += e.drawText(x, statusBarY, warningStyle, "●", 0, nil, -1, 0, -1)
					x += e.drawText(x, statusBarY, statusBarStyle, fmt.Sprintf("%d", warningCount), 0, nil, -1, 0, -1)
				}
			}
		}

		// Add LSP indicator if LSP is available (only if buffer exists)
		if e.buffer != nil && e.buffer.HasLSP {
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

		// Add search indicator if there's an active search with matches
		if e.searchPattern != "" && len(e.searchMatches) > 0 {
			x += e.drawText(x, statusBarY, statusBarStyle, " ", 0, nil, -1, 0, -1) // Single space

			// Use SearchHighlight style for search indicator
			searchHighlightStyle := e.config.Theme.GetStyle(e.config.Theme.SearchHighlight)

			searchText := fmt.Sprintf(" /%s %d/%d ", e.searchPattern, e.currentMatch+1, len(e.searchMatches))
			x += e.drawText(x, statusBarY, searchHighlightStyle, searchText, 0, nil, -1, 0, -1)
		}

		// Add count prefix indicator if building a numeric prefix
		if e.countPrefix != "" {
			x += e.drawText(x, statusBarY, statusBarStyle, " ", 0, nil, -1, 0, -1) // Single space

			// Use inverted colors for count prefix
			countStyle := statusBarStyle
			countStyle.Fg, countStyle.Bg = countStyle.Bg, countStyle.Fg

			countText := fmt.Sprintf(" %s ", e.countPrefix)
			x += e.drawText(x, statusBarY, countStyle, countText, 0, nil, -1, 0, -1)
		}

		// Add references indicator if there's an active references buffer
		if e.currentReferencesBuffer != nil && len(e.currentReferencesBuffer.References) > 0 {
			x += e.drawText(x, statusBarY, statusBarStyle, " ", 0, nil, -1, 0, -1) // Single space

			// Use SearchHighlight style for references indicator (same style as search)
			referencesHighlightStyle := e.config.Theme.GetStyle(e.config.Theme.SearchHighlight)

			referencesText := fmt.Sprintf(" @%s %d/%d ", e.currentReferencesBuffer.SymbolName,
				e.currentReferencesBuffer.CurrentIndex+1, len(e.currentReferencesBuffer.References))
			x += e.drawText(x, statusBarY, referencesHighlightStyle, referencesText, 0, nil, -1, 0, -1)
		}

		// Right side: indentation and line:col (only if buffer exists)

		var rightSideInfo string
		if e.buffer != nil {
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

			// Build line info with mixed styles
			// Format: "40/65:1/16 [1234]"
			currentLineNum := e.buffer.CursorIterator.LineNumber() + 1
			totalLines := e.buffer.LineCount

			// Calculate the full right info for positioning
			rightSideInfo = fmt.Sprintf("%s  %s  %d/%d:%d/%d [%d]  ",
				indentInfo, lineIndentDisplay,
				currentLineNum, totalLines,
				visualCol, lineLength,
				e.getAbsolutePosition())
		} else {
			// Empty window - no position info
			rightSideInfo = ""
		}

		// Get subdued style for total counts (without bold)
		subduedStyle := e.config.Theme.GetStyle(e.config.Theme.StatusBarSubdued)

		// Calculate position for right-aligned text
		rightX := width - len(rightSideInfo)

		if rightX > x && rightSideInfo != "" { // Only draw if there's space and info to show
			// For empty windows, we already set rightSideInfo to "", so this won't draw
			if e.buffer != nil {
				// Draw each part separately with appropriate styles
				x = rightX
				indentInfo := ""
				if e.buffer.Indentation.Type == config.IndentTabs {
					indentInfo = "Tabs"
				} else {
					indentInfo = fmt.Sprintf("Spaces:%d", e.buffer.Indentation.Width)
				}

				lineIndent := e.getCurrentLineIndentInfo(e.buffer.CursorIterator)
				lineIndentDisplay := ""
				if lineIndent.Level > 0 {
					lineIndentDisplay = fmt.Sprintf("L%d", lineIndent.Level)
					if lineIndent.IsContinuation {
						lineIndentDisplay += "+C"
					}
				} else {
					lineIndentDisplay = "L0"
				}

				x += e.drawText(x, statusBarY, statusBarStyle, indentInfo+"  ", 0, nil, -1, 0, -1)
				x += e.drawText(x, statusBarY, statusBarStyle, lineIndentDisplay+"  ", 0, nil, -1, 0, -1)

				// Line info with mixed styles
				currentLineNum := e.buffer.CursorIterator.LineNumber() + 1
				totalLines := e.buffer.LineCount
				visualCol := 1
				if !e.buffer.CursorIterator.IsEOF() {
					line := e.buffer.CursorIterator.Line()
					if e.buffer.CursorX <= len(line) {
						visualCol = e.visualWidth(line[:e.buffer.CursorX]) + 1
					}
				}
				currentLine := ""
				if !e.buffer.CursorIterator.IsEOF() {
					currentLine = e.buffer.CursorIterator.Line()
				}
				lineLength := len(currentLine)

				x += e.drawText(x, statusBarY, statusBarStyle, fmt.Sprintf("%d", currentLineNum), 0, nil, -1, 0, -1)
				x += e.drawText(x, statusBarY, subduedStyle, fmt.Sprintf("/%d", totalLines), 0, nil, -1, 0, -1)
				x += e.drawText(x, statusBarY, statusBarStyle, fmt.Sprintf(":%d", visualCol), 0, nil, -1, 0, -1)
				x += e.drawText(x, statusBarY, subduedStyle, fmt.Sprintf("/%d", lineLength), 0, nil, -1, 0, -1)
				x += e.drawText(x, statusBarY, statusBarStyle, fmt.Sprintf(" [%d]  ", e.getAbsolutePosition()), 0, nil, -1, 0, -1)
			}
		}
	}
}

// drawErrorLine renders the error message above the status bar
func (e *Editor) drawErrorLine() {
	width, _ := e.screen.Size()

	// Transient info banner has highest priority and persists until next key press
	if e.transientInfoActive && e.transientInfoMessage != "" {
		bannerY := e.getStatusBarY() - 1
		style := e.config.Theme.GetStyle(e.config.Theme.NotificationInfo)
		for i := 0; i < width; i++ {
			e.screen.SetContent(i, bannerY, ' ', nil, style)
		}
		e.drawText(0, bannerY, style, " "+e.transientInfoMessage+" ", 0, nil, -1, 0, -1)
		return
	}

	// Show temporary banners on top (override diagnostics)
	if currentMsg, severity := e.errorManager.GetCurrentBanner(); currentMsg != "" {
		bannerY := e.getStatusBarY() - 1
		var style ui.Style
		switch severity {
		case "info":
			style = e.config.Theme.GetStyle(e.config.Theme.NotificationInfo)
		default:
			style = e.config.Theme.GetStyle(e.config.Theme.NotificationError)
		}
		for i := 0; i < width; i++ {
			e.screen.SetContent(i, bannerY, ' ', nil, style)
		}
		e.drawText(0, bannerY, style, " "+currentMsg+" ", 0, nil, -1, 0, -1)
		return
	}

	// Otherwise, show diagnostic for cursor line persistently (only if buffer exists)
	if e.buffer != nil {
		cursorLine := e.buffer.CursorIterator.LineNumber() + 1
		if diag, ok := e.buffer.GetDiagnosticForLine(cursorLine); ok {
			var style ui.Style
			switch diag.Severity {
			case "error":
				style = e.config.Theme.GetStyle(e.config.Theme.Normal)
				style.Bg = ui.ColorRed
				style.Fg = ui.ColorWhite
			case "warning":
				style = e.config.Theme.GetStyle(e.config.Theme.Normal)
				style.Bg = ui.ColorYellow
				style.Fg = ui.ColorBlack
			default:
				return
			}

			// Split diagnostic message by newlines and render each line
			message := strings.ReplaceAll(diag.Message, "\r", "") // Remove carriage returns
			allLines := strings.Split(message, "\n")

			// Filter out empty lines
			var lines []string
			for _, line := range allLines {
				cleanLine := strings.TrimSpace(line)
				if cleanLine != "" {
					lines = append(lines, cleanLine)
				}
			}

			// Limit to reasonable number of lines to avoid taking up too much screen space
			maxLines := 3
			if len(lines) > maxLines {
				lines = lines[:maxLines]
				// Add truncation indicator to last line
				if len(lines) > 0 {
					lines[len(lines)-1] = lines[len(lines)-1] + "..."
				}
			}

			// Render each line, starting from the bottom and working up
			statusBarY := e.getStatusBarY()
			for i, line := range lines {
				bannerY := statusBarY - len(lines) + i

				// Fill the entire line with background
				for j := 0; j < width; j++ {
					e.screen.SetContent(j, bannerY, ' ', nil, style)
				}

				// Clean the line by collapsing multiple spaces
				cleanLine := line
				for strings.Contains(cleanLine, "  ") {
					cleanLine = strings.ReplaceAll(cleanLine, "  ", " ")
				}

				e.drawText(0, bannerY, style, " "+cleanLine+" ", 0, nil, -1, 0, -1)
			}
		}
	}
}

// drawSignature renders the function signature above the status bar
func (e *Editor) drawSignature() {
	// Skip if no buffer
	if e.buffer == nil {
		return
	}

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
    if e.buffer != nil && e.buffer.Highlighter != nil {
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
    if e.buffer == nil || e.buffer.Highlighter == nil {
        e.drawSignatureWithBold(text, y, baseStyle, activeStart, activeEnd)
        return
    }
    tokens := e.buffer.Highlighter.GetTokensForString(text)
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

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// drawEmptyWindow renders an empty window with visible boundaries
func (e *Editor) drawEmptyWindow(screen Screen, isActive bool) {
	width, height := screen.Size()

	// Get styles for empty window
	normalStyle := e.config.Theme.GetStyle(e.config.Theme.Normal)
	borderStyle := e.config.Theme.GetStyle(e.config.Theme.LineNumber)

	// Fill the entire window area with normal background
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			screen.SetContent(x, y, ' ', nil, normalStyle)
		}
	}

	// Draw right border for window separation (vertical line)
	if width > 0 {
		for y := 0; y < height; y++ {
			screen.SetContent(width-1, y, '│', nil, borderStyle)
		}
	}

	// Draw a subtle indicator in the middle for empty window
	if width > 10 && height > 5 {
		centerX := width / 2
		centerY := height / 2

		// Draw a small empty window indicator
		text := "[empty]"
		startX := centerX - len(text)/2
		if startX >= 0 && startX+len(text) < width {
			for i, ch := range text {
				if startX+i < width-1 { // Don't overwrite the border
					screen.SetContent(startX+i, centerY, ch, nil, borderStyle)
				}
			}
		}
	}

	// Show cursor in center of empty window if it's active
	if isActive && width > 2 && height > 0 {
		centerX := width / 2
		centerY := height / 2
		// Position cursor to not overlap with border
		if centerX >= width-1 {
			centerX = width - 2
		}
		screen.ShowCursor(centerX, centerY)
	}
}
