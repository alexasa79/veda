package editor

import (
	"strings"
	"unicode"
	"veda/internal/buffer"
	"veda/internal/commands"
)

const defaultWrapWidth = 80

type wrapContext int

const (
	wrapContextPlain wrapContext = iota
	wrapContextMarkdownList
	wrapContextCBlockComment
	wrapContextCLineComment
	wrapContextPythonComment
	wrapContextMarkdownHeading
)

type paragraphInfo struct {
	startLine   int
	endLine     int
	lines       []string
	context     wrapContext
	indentation string
	prefix      string
}

func detectWrapContext(lines []string, startLine int) (wrapContext, string, string) {
	if len(lines) == 0 || startLine >= len(lines) {
		LL.Trace("detectWrapContext: No lines or invalid startLine, returning plain context")
		return wrapContextPlain, "", ""
	}

	firstLine := lines[startLine]
	trimmed := strings.TrimSpace(firstLine)
	LL.Trace("detectWrapContext: Analyzing line %d: '%s'", startLine, firstLine)

	// Check for markdown list
	if isMarkdownListItem(trimmed) {
		indent := strings.Index(firstLine, strings.TrimSpace(firstLine))
		listPrefix := extractListPrefix(trimmed)
		extraIndent := strings.Repeat(" ", len(listPrefix))
		return wrapContextMarkdownList, strings.Repeat(" ", indent), extraIndent
	}

	// Check for C-style line comment
	if strings.HasPrefix(trimmed, "//") {
		indent := strings.Index(firstLine, "//")
		return wrapContextCLineComment, strings.Repeat(" ", indent), "// "
	}

	// Check for Python comment vs markdown heading
	// If it's a single # followed by space and then lowercase letter, it's likely a Python comment
	// Check for # at beginning - could be markdown heading or Python comment
	if strings.HasPrefix(trimmed, "#") {
		indent := strings.Index(firstLine, "#")

		// Multiple # characters (like ### or ####) - definitely markdown
		if len(trimmed) > 1 && trimmed[1] == '#' {
			return wrapContextMarkdownHeading, "", ""
		}

		// All # characters - markdown heading separator
		if strings.Count(trimmed, "#") == len(trimmed) {
			return wrapContextMarkdownHeading, "", ""
		}

		// # with indentation - Python comment
		if indent > 0 {
			return wrapContextPythonComment, strings.Repeat(" ", indent), "# "
		}

		// # at start of line with space - check first character after "# "
		if len(trimmed) > 2 && trimmed[1] == ' ' {
			firstChar := trimmed[2]
			if unicode.IsLower(rune(firstChar)) {
				// Lowercase after "# " - treat as Python comment
				return wrapContextPythonComment, "", "# "
			}
		}

		// Default: treat as markdown heading
		return wrapContextMarkdownHeading, "", ""
	}

	// Check for C-style block comment
	if strings.Contains(firstLine, "/*") || strings.Contains(firstLine, "*") {
		// Find the column where the actual text starts
		for i, line := range lines[startLine:] {
			if i+startLine > startLine && strings.TrimSpace(line) == "" {
				break
			}
			trimmedLine := strings.TrimLeft(line, " \t")
			if strings.HasPrefix(trimmedLine, "*") && len(trimmedLine) > 1 {
				// Find position after "* "
				starPos := strings.Index(line, "*")
				textStart := starPos + 1
				for textStart < len(line) && line[textStart] == ' ' {
					textStart++
				}
				return wrapContextCBlockComment, strings.Repeat(" ", textStart), ""
			}
		}
		// Fallback for /* on the same line as content
		if strings.Contains(firstLine, "/*") {
			pos := strings.Index(firstLine, "/*") + 2
			for pos < len(firstLine) && firstLine[pos] == ' ' {
				pos++
			}
			return wrapContextCBlockComment, strings.Repeat(" ", pos), ""
		}
	}

	// Default to plain text
	indent := 0
	for i, ch := range firstLine {
		if !unicode.IsSpace(ch) {
			indent = i
			break
		}
	}
	return wrapContextPlain, strings.Repeat(" ", indent), ""
}

func isMarkdownListItem(line string) bool {
	// Check for numbered list (1. 2. etc)
	if len(line) > 2 {
		for i, ch := range line {
			if unicode.IsDigit(ch) {
				continue
			}
			if (ch == '.' || ch == ')') && i > 0 && i+1 < len(line) && line[i+1] == ' ' {
				return true
			}
			break
		}
	}

	// Check for bullet lists
	if len(line) >= 2 {
		if (line[0] == '-' || line[0] == '*' || line[0] == '+') && line[1] == ' ' {
			return true
		}
	}

	return false
}

func extractListPrefix(line string) string {
	// Extract "1. " or "- " etc
	for i, ch := range line {
		if unicode.IsDigit(ch) {
			continue
		}
		if (ch == '.' || ch == ')') && i > 0 && i+1 < len(line) && line[i+1] == ' ' {
			return line[:i+2]
		}
		if (ch == '-' || ch == '*' || ch == '+') && i == 0 && len(line) > 1 && line[1] == ' ' {
			return line[:2]
		}
		break
	}
	return ""
}

func findParagraphBounds(lines []string, cursorLine int) (int, int) {
	startLine := cursorLine
	endLine := cursorLine

	LL.Trace("findParagraphBounds: Starting at cursorLine=%d", cursorLine)
	if cursorLine < len(lines) {
		LL.Trace("findParagraphBounds: Current line content: '%s'", lines[cursorLine])
	}

	// Find start of paragraph (previous empty line, markdown heading, or start of file)
	for startLine > 0 {
		prevLine := strings.TrimSpace(lines[startLine-1])
		currentLine := strings.TrimSpace(lines[startLine])
		LL.Trace("findParagraphBounds: Checking line %d: '%s'", startLine-1, lines[startLine-1])

		// Stop at empty line
		if prevLine == "" {
			break
		}

		// Stop at markdown heading (line starting with #)
		if strings.HasPrefix(prevLine, "#") {
			break
		}

		// If we're currently on a list item line, do not cross into previous item
		if isMarkdownListItem(currentLine) {
			break
		}

		// If the previous line is a list item, it marks the start of this item
		if isMarkdownListItem(prevLine) {
			startLine--
			break
		}

		startLine--
	}

	// Find end of paragraph (next empty line, markdown heading, or end of file)
	for endLine < len(lines)-1 {
		nextLine := strings.TrimSpace(lines[endLine+1])
		LL.Trace("findParagraphBounds: Checking line %d: '%s'", endLine+1, lines[endLine+1])

		// Stop at empty line
		if nextLine == "" {
			break
		}

		// Stop at markdown heading (line starting with #)
		if strings.HasPrefix(nextLine, "#") {
			break
		}

		// Stop at markdown list item boundary. Even if the current line is a
		// continuation line (not itself a list item), the next list item marks
		// the end of this paragraph.
		if isMarkdownListItem(nextLine) {
			break
		}

		endLine++
	}

	LL.Trace("findParagraphBounds: Found bounds: startLine=%d, endLine=%d", startLine, endLine)
	return startLine, endLine
}

func wrapText(text string, width int, indent string, prefix string) []string {
	// Ensure we have reasonable minimum width for wrapping
	minWidth := len(indent) + len(prefix) + 10 // At least 10 chars for text
	if width < minWidth {
		width = minWidth
	}

	words := strings.Fields(text)
	if len(words) == 0 {
		return []string{indent + prefix}
	}

	var lines []string
	currentLine := indent + prefix

	for i, word := range words {
		if i == 0 {
			currentLine += word
		} else if len(currentLine)+1+len(word) <= width {
			currentLine += " " + word
		} else {
			lines = append(lines, currentLine)
			currentLine = indent + prefix + word
		}
	}

	if currentLine != indent+prefix {
		lines = append(lines, currentLine)
	}

	return lines
}

func (e *Editor) rewrapParagraph() error {
	// If no rulers configured, use default
	if len(e.config.VerticalRulers) == 0 {
		return e.rewrapParagraphWithWidth(defaultWrapWidth)
	}

	// Get current cursor position
	currentLine := e.CurrentBuffer().CursorIterator.LineNumber()
	currentChar := e.CurrentBuffer().CursorX

	// Determine the ruler index to use
	var rulerIndex int
	if currentLine == e.lastRewrapLine && currentChar == e.lastRewrapChar {
		// Same position as last rewrap - cycle to next ruler
		rulerIndex = (e.lastRewrapIndex + 1) % len(e.config.VerticalRulers)
	} else {
		// Different position - start at first ruler
		rulerIndex = 0
	}

	// Get the width from the ruler at this index
	width := e.config.VerticalRulers[rulerIndex]

	// Perform the rewrap
	if err := e.rewrapParagraphWithWidth(width); err != nil {
		return err
	}

	// Update tracking AFTER rewrap to the new cursor position so a repeated
	// Alt+Q at the same physical location cycles correctly even if cursorX shifts
	// due to reflow.
	e.lastRewrapLine = e.CurrentBuffer().CursorIterator.LineNumber()
	e.lastRewrapChar = e.CurrentBuffer().CursorX
	e.lastRewrapIndex = rulerIndex

	return nil
}

// rewrapParagraphCmd is the command wrapper for rewrapParagraph
func (e *Editor) rewrapParagraphCmd(args []string) error {
	// If invoked during insert mode, create an undo boundary so the rewrap
	// is undoable separately from the ongoing insert session. We do this by
	// ending the current insert session, executing the rewrap as its own
	// history entry, and then immediately starting a new insert session.
	if e.mode == commands.InsertMode && e.CurrentBuffer() != nil && e.CurrentBuffer().History != nil {
		// End current insert session (no-op if empty)
		e.CurrentBuffer().History.EndInsertSession()
		// Perform the rewrap as a standalone history unit
		if err := e.rewrapParagraph(); err != nil {
			return err
		}
		// Resume a fresh insert session so subsequent typing is grouped
		e.CurrentBuffer().History.StartInsertSession("i")
		return nil
	}
	return e.rewrapParagraph()
}

func (e *Editor) rewrapParagraphWithWidth(width int) error {
	LL.Trace("rewrapParagraphWithWidth: Starting with width=%d", width)

	if e.CurrentBuffer() == nil {
		LL.Trace("rewrapParagraphWithWidth: buffer is nil, returning")
		return nil
	}

	// Save iterator positions before any buffer modifications
	cursorLine := e.CurrentBuffer().CursorIterator.LineNumber()
	cursorX := e.CurrentBuffer().CursorX
	viewLine := e.CurrentBuffer().ViewIterator.LineNumber()

	LL.Trace("rewrapParagraphWithWidth: cursorLine=%d, cursorX=%d, viewLine=%d", cursorLine, cursorX, viewLine)

	// Get all lines from buffer
	lines := make([]string, e.CurrentBuffer().LineCount)
	for i := 0; i < e.CurrentBuffer().LineCount; i++ {
		iter := buffer.NewBufferIteratorAtLine(e.CurrentBuffer(), i)
		lines[i] = iter.Line()
	}

	LL.Trace("rewrapParagraphWithWidth: buffer has %d lines", len(lines))

	if cursorLine >= len(lines) {
		cursorLine = len(lines) - 1
	}

	// Initial naive bounds
	startLine, endLine := findParagraphBounds(lines, cursorLine)
	// Detect context near the cursor line (more accurate for comments)
	context, indentation, prefix := detectWrapContext(lines, cursorLine)

	// Recompute bounds strictly by comment context to avoid including adjacent code
	if context == wrapContextCLineComment || context == wrapContextPythonComment {
		isCommentLine := func(s string) bool {
			trimmed := strings.TrimSpace(s)
			if context == wrapContextCLineComment {
				return strings.HasPrefix(trimmed, "//")
			}
			return strings.HasPrefix(trimmed, "#")
		}
		// Start from cursor line and expand only across contiguous comment lines
		startLine, endLine = cursorLine, cursorLine
		for startLine > 0 && isCommentLine(lines[startLine-1]) {
			startLine--
		}
		for endLine < len(lines)-1 && isCommentLine(lines[endLine+1]) {
			endLine++
		}
		// Re-detect indent/prefix at the new start line
		_, indentation, prefix = detectWrapContext(lines, startLine)
	} else if context == wrapContextCBlockComment {
		// Find exact block boundaries: up to '/*' and down to '*/'
		openLine := cursorLine
		for openLine > 0 && !strings.Contains(lines[openLine], "/*") {
			openLine--
		}
		// If we didn't land on an opener, scan forward until we find one
		for openLine < len(lines) && !strings.Contains(lines[openLine], "/*") {
			openLine++
		}
		if openLine >= 0 && openLine < len(lines) {
			closeLine := openLine
			for closeLine < len(lines) && !strings.Contains(lines[closeLine], "*/") {
				closeLine++
			}
			if closeLine < len(lines) {
				startLine, endLine = openLine, closeLine
			}
		}
		// For block comments indentation is computed later from opener position
	}
	LL.Trace("rewrapParagraphWithWidth: paragraph bounds (adjusted): startLine=%d, endLine=%d", startLine, endLine)
	LL.Trace("rewrapParagraphWithWidth: context=%d, indentation='%s', prefix='%s'", context, indentation, prefix)

	// Handle markdown heading - don't rewrap
	if context == wrapContextMarkdownHeading {
		LL.Trace("rewrapParagraphWithWidth: detected markdown heading, skipping rewrap")
		return nil
	}

	// Calculate cursor position within the paragraph text (if cursor is in paragraph)
	var cursorOffsetInParagraph int = -1
	if cursorLine >= startLine && cursorLine <= endLine {
		// Calculate how many characters come before the cursor in the unwrapped text
		cursorOffsetInParagraph = 0
		for i := startLine; i < cursorLine; i++ {
			line := lines[i]
			trimmed := strings.TrimSpace(line)
			if trimmed == "" {
				continue
			}
			// Remove comment prefixes to get actual text length
			switch context {
			case wrapContextCLineComment:
				trimmed = strings.TrimPrefix(trimmed, "//")
				trimmed = strings.TrimSpace(trimmed)
			case wrapContextPythonComment:
				trimmed = strings.TrimPrefix(trimmed, "#")
				trimmed = strings.TrimSpace(trimmed)
			case wrapContextCBlockComment:
				if strings.HasPrefix(trimmed, "*") && !strings.HasPrefix(trimmed, "*/") {
					trimmed = strings.TrimPrefix(trimmed, "*")
					trimmed = strings.TrimSpace(trimmed)
				} else if strings.HasPrefix(trimmed, "/*") {
					trimmed = strings.TrimPrefix(trimmed, "/*")
					trimmed = strings.TrimSpace(trimmed)
				}
				// Always check for and remove trailing */ (for any line in a C block comment)
				if strings.HasSuffix(trimmed, "*/") {
					trimmed = strings.TrimSuffix(trimmed, "*/")
					trimmed = strings.TrimSpace(trimmed)
				}
			case wrapContextMarkdownList:
				if i == startLine {
					listPrefix := extractListPrefix(trimmed)
					trimmed = strings.TrimPrefix(trimmed, listPrefix)
				}
			}
			cursorOffsetInParagraph += len(trimmed) + 1 // +1 for space between words
		}

		// Add position within current line
		currentLineTrimmed := strings.TrimSpace(lines[cursorLine])
		switch context {
		case wrapContextCLineComment:
			if strings.HasPrefix(currentLineTrimmed, "//") {
				prefixLen := strings.Index(lines[cursorLine], "//") + 2
				// Skip spaces after //
				for prefixLen < len(lines[cursorLine]) && lines[cursorLine][prefixLen] == ' ' {
					prefixLen++
				}
				if cursorX > prefixLen {
					cursorOffsetInParagraph += cursorX - prefixLen
				}
			}
		case wrapContextPythonComment:
			if strings.HasPrefix(currentLineTrimmed, "#") {
				prefixLen := strings.Index(lines[cursorLine], "#") + 1
				// Skip spaces after #
				for prefixLen < len(lines[cursorLine]) && lines[cursorLine][prefixLen] == ' ' {
					prefixLen++
				}
				if cursorX > prefixLen {
					cursorOffsetInParagraph += cursorX - prefixLen
				}
			}
		case wrapContextCBlockComment:
			if strings.Contains(lines[cursorLine], "/*") {
				prefixLen := strings.Index(lines[cursorLine], "/*") + 2
				// Skip spaces after /*
				for prefixLen < len(lines[cursorLine]) && lines[cursorLine][prefixLen] == ' ' {
					prefixLen++
				}
				if cursorX > prefixLen {
					cursorOffsetInParagraph += cursorX - prefixLen
				}
			}
		default:
			// For plain text, calculate from the first non-space character
			firstNonSpace := 0
			for firstNonSpace < len(lines[cursorLine]) && lines[cursorLine][firstNonSpace] == ' ' {
				firstNonSpace++
			}
			if cursorX > firstNonSpace {
				cursorOffsetInParagraph += cursorX - firstNonSpace
			}
		}
	}

	// Extract paragraph text
	var paragraphText strings.Builder
	for i := startLine; i <= endLine; i++ {
		line := lines[i]
		LL.Trace("rewrapParagraphWithWidth: Processing line %d: '%s'", i, line)

		// Remove existing prefix/indentation
		trimmed := strings.TrimSpace(line)

		// Skip empty lines within the paragraph
		if trimmed == "" {
			LL.Trace("rewrapParagraphWithWidth: Skipping empty line %d", i)
			continue
		}

		// Remove comment prefixes
		switch context {
		case wrapContextCLineComment:
			trimmed = strings.TrimPrefix(trimmed, "//")
			trimmed = strings.TrimSpace(trimmed)
		case wrapContextPythonComment:
			trimmed = strings.TrimPrefix(trimmed, "#")
			trimmed = strings.TrimSpace(trimmed)
		case wrapContextCBlockComment:
			if strings.HasPrefix(trimmed, "*") && !strings.HasPrefix(trimmed, "*/") {
				trimmed = strings.TrimPrefix(trimmed, "*")
				trimmed = strings.TrimSpace(trimmed)
			} else if strings.HasPrefix(trimmed, "/*") {
				trimmed = strings.TrimPrefix(trimmed, "/*")
				trimmed = strings.TrimSpace(trimmed)
			}
			// Always check for and remove trailing */ (for any line in a C block comment)
			if strings.HasSuffix(trimmed, "*/") {
				trimmed = strings.TrimSuffix(trimmed, "*/")
				trimmed = strings.TrimSpace(trimmed)
			}
		case wrapContextMarkdownList:
			if i == startLine {
				// First line - remove list marker
				listPrefix := extractListPrefix(trimmed)
				trimmed = strings.TrimPrefix(trimmed, listPrefix)
			}
		}

		LL.Trace("rewrapParagraphWithWidth: After processing, trimmed text: '%s'", trimmed)

		if paragraphText.Len() > 0 {
			paragraphText.WriteString(" ")
		}
		paragraphText.WriteString(trimmed)
	}

	LL.Trace("rewrapParagraphWithWidth: Final paragraph text: '%s'", paragraphText.String())

	// Now prepare wrapped output and replacement region based on context
	wrappedLines := []string{}
	if context == wrapContextCBlockComment {
		// Handle C block comments
		openIdx := strings.Index(lines[startLine], "/*")
		closeIdx := strings.Index(lines[endLine], "*/")
		if openIdx < 0 || closeIdx < 0 {
			// Fallback: do nothing if malformed
			return nil
		}

		// Check if this is a single-line block comment
		if startLine == endLine {
			// Single-line block comment: /* text */
			line := lines[startLine]
			beforeComment := line[:openIdx]
			afterComment := ""
			if closeIdx+2 < len(line) {
				afterComment = line[closeIdx+2:]
			}

			// For inline C block comments, use a simpler approach:
			// Use wrapText with appropriate prefix/indentation, then post-process to add /* */ formatting
			indentStr := beforeComment

			// First, try to see if it fits on a single line
			singleLine := indentStr + "/* " + paragraphText.String() + " */" + afterComment
			if len(singleLine) <= width {
				// Fits on single line - keep it as is
				wrappedLines = []string{singleLine}
			} else {
				// Needs multiple lines - wrap the text content and add comment formatting
				continuationIndent := indentStr + strings.Repeat(" ", len("/* "))

				// Wrap the text content with continuation indent
				textLines := wrapText(paragraphText.String(), width, continuationIndent, "")

				// Post-process to add /* and */ formatting
				wrappedLines = make([]string, len(textLines))
				for i, line := range textLines {
					if i == 0 {
						// First line: replace continuation indent with "/* "
						textPart := strings.TrimPrefix(line, continuationIndent)
						wrappedLines[i] = indentStr + "/* " + textPart
					} else if i == len(textLines)-1 {
						// Last line: add " */" at the end
						wrappedLines[i] = line + " */" + afterComment
					} else {
						// Middle lines: keep as is (they already have the right indentation)
						wrappedLines[i] = line
					}
				}
			}
		} else {
			// Multi-line block comment
			// Handle inline-style multi-line C comments (where /* has content after it)
			// Extract the complete comment text and rewrap it as inline style
			beforeComment := lines[startLine][:openIdx]
			afterComment := ""

			// Check if there's content after */ on the closing line
			closeIdx := strings.Index(lines[endLine], "*/")
			if closeIdx >= 0 && closeIdx+2 < len(lines[endLine]) {
				afterComment = lines[endLine][closeIdx+2:]
			}

			// Use the same inline wrapping logic as single-line comments
			indentStr := beforeComment

			// First, try to see if it fits on a single line
			singleLine := indentStr + "/* " + paragraphText.String() + " */" + afterComment
			if len(singleLine) <= width {
				// Fits on single line - collapse to single line
				wrappedLines = []string{singleLine}
			} else {
				// Needs multiple lines - wrap the text content and add comment formatting
				continuationIndent := indentStr + strings.Repeat(" ", len("/* "))

				// Wrap the text content with continuation indent
				textLines := wrapText(paragraphText.String(), width, continuationIndent, "")

				// Post-process to add /* and */ formatting
				wrappedLines = make([]string, len(textLines))
				for i, line := range textLines {
					if i == 0 {
						// First line: replace continuation indent with "/* "
						textPart := strings.TrimPrefix(line, continuationIndent)
						wrappedLines[i] = indentStr + "/* " + textPart
					} else if i == len(textLines)-1 {
						// Last line: add " */" at the end
						wrappedLines[i] = line + " */" + afterComment
					} else {
						// Middle lines: keep as is (they already have the right indentation)
						wrappedLines[i] = line
					}
				}
			}
		}
	} else {
		// Non-block contexts use existing wrap behavior
		wrappedLines = wrapText(paragraphText.String(), width, indentation, prefix)
		LL.Trace("rewrapParagraphWithWidth: Wrapped into %d lines", len(wrappedLines))
		if context == wrapContextMarkdownList && len(wrappedLines) > 0 {
			firstLineTrimmed := strings.TrimSpace(lines[startLine])
			listPrefix := extractListPrefix(firstLineTrimmed)
			wrappedLines[0] = indentation + listPrefix + strings.TrimPrefix(wrappedLines[0], indentation+prefix)
		}
	}

	// Calculate the byte positions for replacement (full lines for all Stage 1 contexts)
	startIter := buffer.NewBufferIteratorAtLine(e.CurrentBuffer(), startLine)
	startPos := startIter.Position()
	endIter := buffer.NewBufferIteratorAtLine(e.CurrentBuffer(), endLine)
	endPos := endIter.LineEnd()

	// Build new content with newlines
	var newContent strings.Builder
	for i, line := range wrappedLines {
		if i > 0 {
			newContent.WriteString("\n")
		}
		newContent.WriteString(line)
	}

	// Save cursor position before change
	oldCursorX := e.CurrentBuffer().CursorX
	oldCursorY := e.CurrentBuffer().CursorIterator.LineNumber()

	// Create a single history entry for the whole operation
	oldLinesCopy := make([]string, endLine-startLine+1)
	copy(oldLinesCopy, lines[startLine:endLine+1])

	entry := &RewrapHistoryEntry{
		buffer:     e.CurrentBuffer(),
		startLine:  startLine,
		endLine:    endLine,
		oldLines:   oldLinesCopy,
		newLines:   wrappedLines,
		oldCursorX: oldCursorX,
		oldCursorY: oldCursorY,
		startPos:   startPos,
		endPos:     endPos,
		newContent: newContent.String(),
	}

	// Execute through history
	if e.CurrentBuffer().History != nil {
		if executed, err := e.CurrentBuffer().History.Execute(e.CurrentBuffer(), entry); err != nil {
			return err
		} else if executed != nil {
			e.lastChange = executed
		}
	} else {
		// If no history, execute directly
		entry.Execute(e.CurrentBuffer())
	}

	// Update cursor position
	if cursorLine >= startLine && cursorLine <= endLine && cursorOffsetInParagraph >= 0 {
		// Cursor was in the paragraph - find its new position
		// Walk through wrapped lines to find where the cursor should be
		remainingOffset := cursorOffsetInParagraph
		newCursorLine := startLine
		newCursorX := len(indentation) + len(prefix)

		for i, wrappedLine := range wrappedLines {
			// Get the actual text content (without indentation/prefix)
			lineContent := wrappedLine
			if i == 0 && context == wrapContextMarkdownList {
				// First line of markdown list has the list prefix
				listPrefix := extractListPrefix(strings.TrimSpace(lines[startLine]))
				lineContent = strings.TrimPrefix(wrappedLine, indentation+listPrefix)
			} else {
				lineContent = strings.TrimPrefix(wrappedLine, indentation+prefix)
			}

			lineContent = strings.TrimSpace(lineContent)

			if remainingOffset <= len(lineContent) {
				// Cursor is on this line
				newCursorLine = startLine + i
				if i == 0 && context == wrapContextMarkdownList {
					listPrefix := extractListPrefix(strings.TrimSpace(lines[startLine]))
					newCursorX = len(indentation) + len(listPrefix) + remainingOffset
				} else {
					newCursorX = len(indentation) + len(prefix) + remainingOffset
				}
				break
			}

			// Account for the space between lines
			remainingOffset -= len(lineContent) + 1
			if remainingOffset < 0 {
				// Cursor was in the space between lines, put it at start of next line
				newCursorLine = startLine + i + 1
				newCursorX = len(indentation) + len(prefix)
				break
			}
		}

		// Ensure we don't go past the end of the wrapped paragraph
		if newCursorLine >= startLine+len(wrappedLines) {
			newCursorLine = startLine + len(wrappedLines) - 1
			e.CurrentBuffer().CursorIterator = buffer.NewBufferIteratorAtLine(e.CurrentBuffer(), newCursorLine)
			e.CurrentBuffer().CursorX = len(e.CurrentBuffer().CursorIterator.Line())
		} else {
			e.CurrentBuffer().CursorIterator = buffer.NewBufferIteratorAtLine(e.CurrentBuffer(), newCursorLine)
			e.CurrentBuffer().CursorX = newCursorX
		}
	} else if cursorLine > endLine {
		// Adjust cursor if paragraph size changed
		lineDiff := len(wrappedLines) - (endLine - startLine + 1)
		newLineNum := cursorLine + lineDiff
		if newLineNum < e.CurrentBuffer().LineCount {
			e.CurrentBuffer().CursorIterator = buffer.NewBufferIteratorAtLine(e.CurrentBuffer(), newLineNum)
		} else {
			e.CurrentBuffer().CursorIterator = buffer.NewBufferIteratorAtLine(e.CurrentBuffer(), e.CurrentBuffer().LineCount-1)
		}
	} else {
		// Cursor was above the paragraph, recreate iterator at same position
		// (it may have been invalidated by buffer changes)
		e.CurrentBuffer().CursorIterator = buffer.NewBufferIteratorAtLine(e.CurrentBuffer(), cursorLine)
		// Restore the original cursor X position
		e.CurrentBuffer().CursorX = cursorX
	}

	// Update view iterator to handle line count changes
	if viewLine > endLine {
		// Adjust view if it was below the paragraph
		lineDiff := len(wrappedLines) - (endLine - startLine + 1)
		newViewLine := viewLine + lineDiff
		if newViewLine >= e.CurrentBuffer().LineCount {
			newViewLine = e.CurrentBuffer().LineCount - 1
		}
		if newViewLine < 0 {
			newViewLine = 0
		}
		e.CurrentBuffer().ViewIterator = buffer.NewBufferIteratorAtLine(e.CurrentBuffer(), newViewLine)
	} else {
		// Recreate view iterator at same position (it may have been invalidated)
		if viewLine >= e.CurrentBuffer().LineCount {
			viewLine = e.CurrentBuffer().LineCount - 1
		}
		if viewLine < 0 {
			viewLine = 0
		}
		e.CurrentBuffer().ViewIterator = buffer.NewBufferIteratorAtLine(e.CurrentBuffer(), viewLine)
	}

	// Update syntax highlighting (skip during paste mode)
	if !e.inPasteMode {
		e.updateSyntaxHighlighting()
	}

	return nil
}

func (e *Editor) indentationForLine(line string) string {
	indent := 0
	for _, ch := range line {
		if !unicode.IsSpace(ch) {
			break
		}
		indent++
	}
	return line[:indent]
}

// RewrapHistoryEntry represents a paragraph rewrap operation for undo/redo
type RewrapHistoryEntry struct {
	buffer     *buffer.Buffer
	startLine  int
	endLine    int
	oldLines   []string
	newLines   []string
	oldCursorX int
	oldCursorY int
	startPos   int
	endPos     int
	newContent string
}

func (r *RewrapHistoryEntry) Execute(buf *buffer.Buffer) error {
	// Delete the old paragraph
	if r.endPos > r.startPos {
		buf.Delete(r.startPos, r.endPos)
	}

	// Insert the new content
	buf.Insert(r.startPos, r.newContent)

	return nil
}

func (r *RewrapHistoryEntry) Undo(buf *buffer.Buffer) error {
	// Replace new lines with old lines
	// Calculate positions
	numNewLines := len(r.newLines)
	if numNewLines == 0 {
		numNewLines = 1
	}

	startIter := buffer.NewBufferIteratorAtLine(buf, r.startLine)
	startPos := startIter.Position()

	// Find end position after the new content
	endLine := r.startLine + numNewLines - 1
	if endLine >= buf.LineCount {
		endLine = buf.LineCount - 1
	}
	endIter := buffer.NewBufferIteratorAtLine(buf, endLine)
	endPos := endIter.LineEnd()
	if endLine < buf.LineCount-1 {
		endPos++ // Include newline
	}

	// Delete new content
	if endPos > startPos {
		buf.Delete(startPos, endPos)
	}

	// Insert old content
	var oldContent strings.Builder
	for i, line := range r.oldLines {
		if i > 0 {
			oldContent.WriteString("\n")
		}
		oldContent.WriteString(line)
	}
	buf.Insert(startPos, oldContent.String())

	// Restore cursor position
	buf.CursorIterator = buffer.NewBufferIteratorAtLine(buf, r.oldCursorY)
	buf.CursorX = r.oldCursorX

	return nil
}

func (r *RewrapHistoryEntry) Description() string {
	return "Rewrap paragraph"
}
