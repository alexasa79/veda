package buffer

import (
	"path/filepath"
	"strings"
	"unicode"
	"veda/internal/config"
)

// AutoIndentInfo contains information about how to indent the next line
type AutoIndentInfo struct {
	Column    int    // The column position to indent to (0-based)
	IndentStr string // The actual string to use for indentation
}

// CalculateAutoIndent determines the indentation for a new line based on the current line
// and cursor position. It implements the following rules:
//  1. If cursor is between curly braces {} or after an open brace at end of line,
//     indent one level deeper
//  2. If current line appears to be a continuation line (has extra indentation beyond base level),
//     maintain that same indentation
//  3. If there are unbalanced open parentheses and characters after the last open paren,
//     indent to the column after the last open paren
//  4. If there are unbalanced open parentheses but no characters after,
//     indent to current line's indentation level plus one indent unit
//  5. Otherwise (balanced parens or no parens), maintain the same indentation level
func (b *Buffer) CalculateAutoIndent(line string, cursorCol int) AutoIndentInfo {
	// Special case: For empty lines, determine indentation from block context
	if strings.TrimSpace(line) == "" {
		return b.calculateIndentForEmptyLine()
	}

	// Get current line's base indentation
	baseIndentInfo := b.GetLineIndentInfo(line)

	// Python-specific: if line ends with ':' (block opener), indent one level
	if isPythonFile(b.FilePath) {
		if endsWithColonBeforeCursor(line, cursorCol) {
			indentLevel := baseIndentInfo.Level + 1
			indentStr := strings.Repeat(b.GetIndentationString(), indentLevel)
			visualCol := len(indentStr)
			if b.Indentation.Type == config.IndentTabs {
				visualCol = indentLevel * 8
			}
			return AutoIndentInfo{Column: visualCol, IndentStr: indentStr}
		}
	}

	// Rule 1: Check for cursor between braces or after open brace
	if shouldIndentAfterBrace(line, cursorCol) {
		// Add one more indent level to current indentation
		indentLevel := baseIndentInfo.Level + 1
		indentStr := strings.Repeat(b.GetIndentationString(), indentLevel)

		// Calculate visual column for this indentation
		visualCol := len(indentStr)
		if b.Indentation.Type == config.IndentTabs {
			// Each tab counts as 8 visual columns
			visualCol = indentLevel * 8
		}

		return AutoIndentInfo{
			Column:    visualCol,
			IndentStr: indentStr,
		}
	}

	// Count delimiter balance up to cursor first (before continuation line check)
	// In Normal mode, the cursor sits on the last character; if it's an opening
	// delimiter, include it in the analysis so newline after it indents.
	analyzeCol := cursorCol
	if analyzeCol >= 0 && analyzeCol < len(line) {
		ch := line[analyzeCol]
		// Include '(' or '[' under the cursor only when it's the last character
		// (Normal mode cursor on opener at EOL). Avoid including when cursor is
		// before an opener in the middle of text.
		if (ch == '(' || ch == '[') && analyzeCol == len(line)-1 {
			analyzeCol++
		}
	}
	balance, lastOpenCol, hasCharsAfter := analyzeDelimiters(line, analyzeCol)

	// Check if this line appears to be a continuation line
	// A continuation line has unusual indentation that doesn't match standard indent levels
	actualIndent := baseIndentInfo.Spaces + (baseIndentInfo.Tabs * 8) // Visual columns

	// If the indentation is not a multiple of the indent width, it's likely a continuation line
	// (e.g., 9 or 10 spaces when indent width is 4)
	// BUT: if parentheses are balanced AND we're positioned after what looks like a statement
	// terminator (not a continuation character like comma), we should return to block level instead
	atEndOfStatement := balance == 0 && isAfterStatementEndCharacter(line, cursorCol)
	if actualIndent > 0 && actualIndent%b.Indentation.Width != 0 && !atEndOfStatement {
		// Maintain the exact same indentation as the current line (only if not at end of statement)
		return AutoIndentInfo{
			Column:    actualIndent,
			IndentStr: baseIndentInfo.ActualPrefix,
		}
	}

	// Rule 2: Unbalanced delimiters with content after - align to position after last open
	if balance > 0 && lastOpenCol >= 0 && hasCharsAfter {
		// We want to align to the column position right after the open paren
		// First calculate the visual column of the open paren itself
		parenCol := calculateVisualColumn(line, lastOpenCol)
		// Then add 1 to get the position after the paren
		visualCol := parenCol + 1

		// Create indentation string using the buffer's settings
		// We'll try to use the same indentation style as the file
		indentStr := b.createIndentStringForColumn(visualCol)

		return AutoIndentInfo{
			Column:    visualCol,
			IndentStr: indentStr,
		}
	}

	// Rule 3: Unbalanced delimiters but no content after - add one indent level
	if balance > 0 {
		// Add one more indent level to current indentation
		indentLevel := baseIndentInfo.Level + 1
		indentStr := strings.Repeat(b.GetIndentationString(), indentLevel)

		// Calculate visual column for this indentation
		visualCol := len(indentStr)
		if b.Indentation.Type == config.IndentTabs {
			// Each tab counts as 8 visual columns
			visualCol = indentLevel * 8
		}

		return AutoIndentInfo{
			Column:    visualCol,
			IndentStr: indentStr,
		}
	}

	// Rule 4: Balanced delimiters - return to block-level indentation
	// When parentheses are balanced, we're likely at the end of a statement/expression
	// and should return to the containing block's indentation level
	if balance == 0 {
		actualIndent := baseIndentInfo.Spaces + (baseIndentInfo.Tabs * 8)

		// Determine base level from the line that introduced the paren/bracket continuation
		curLine := b.CursorIterator.LineNumber()
		// But avoid overriding brace-based indentation: if the last non-space
		// character on the current line before cursor is a closing brace '}',
		// we're finishing a block — do not use paren continuation logic here.
		useParenBase := true
		if cursorCol > 0 && cursorCol <= len(line) {
			i := cursorCol - 1
			if i >= len(line) {
				i = len(line) - 1
			}
			for ; i >= 0; i-- {
				if !unicode.IsSpace(rune(line[i])) {
					if line[i] == '}' {
						useParenBase = false
					}
					break
				}
			}
		}

		if useParenBase {
			if baseFromOpen, ok := b.findBaseLevelFromOpenDelimiter(curLine); ok {
				indentStr := strings.Repeat(b.GetIndentationString(), baseFromOpen)
				visualCol := baseFromOpen * b.Indentation.Width
				return AutoIndentInfo{Column: visualCol, IndentStr: indentStr}
			}
		}

		// Fallbacks when no open delimiter context is detected
		baseLevel := b.findPreviousNonContinuationLevel(curLine - 1)
		if actualIndent > 0 && actualIndent%b.Indentation.Width != 0 {
			indentStr := strings.Repeat(b.GetIndentationString(), baseLevel)
			visualCol := baseLevel * b.Indentation.Width
			return AutoIndentInfo{Column: visualCol, IndentStr: indentStr}
		}
		if baseLevel < baseIndentInfo.Level && b.hasParenContinuationSinceBase(curLine-1, baseLevel) {
			indentStr := strings.Repeat(b.GetIndentationString(), baseLevel)
			visualCol := baseLevel * b.Indentation.Width
			return AutoIndentInfo{Column: visualCol, IndentStr: indentStr}
		}
		// Otherwise maintain current standard indent level (fall through)
	}

	// Python safeguard: even if earlier checks missed, indent after ':'
	if isPythonFile(b.FilePath) && endsWithColonBeforeCursor(line, cursorCol) {
		indentLevel := baseIndentInfo.Level + 1
		indentStr := strings.Repeat(b.GetIndentationString(), indentLevel)
		visualCol := len(indentStr)
		if b.Indentation.Type == config.IndentTabs {
			visualCol = indentLevel * 8
		}
		return AutoIndentInfo{Column: visualCol, IndentStr: indentStr}
	}

	// Rule 5: Default - maintain same indentation level
	indentStr := strings.Repeat(b.GetIndentationString(), baseIndentInfo.Level)

	// Calculate visual column for this indentation
	visualCol := len(indentStr)
	if b.Indentation.Type == config.IndentTabs {
		// Each tab counts as 8 visual columns
		visualCol = baseIndentInfo.Level * 8
	}

	return AutoIndentInfo{
		Column:    visualCol,
		IndentStr: indentStr,
	}
}

// createIndentStringForColumn creates an indentation string to reach the target visual column
// using the buffer's indentation settings
func (b *Buffer) createIndentStringForColumn(targetCol int) string {
	if b.Indentation.Type == config.IndentTabs {
		// Use as many tabs as possible, then spaces for the remainder
		numTabs := targetCol / 8
		numSpaces := targetCol % 8
		return strings.Repeat("\t", numTabs) + strings.Repeat(" ", numSpaces)
	} else {
		// For space indentation, just use spaces
		return strings.Repeat(" ", targetCol)
	}
}

// analyzeParentheses analyzes parentheses in a line up to the given column
// Returns:
// - balance: number of unmatched open parentheses
// - lastOpenCol: column position of the last unbalanced open parenthesis (-1 if none)
// - hasCharsAfter: whether there are non-whitespace characters after the last unbalanced open paren
func analyzeDelimiters(line string, upToCol int) (balance int, lastOpenCol int, hasCharsAfter bool) {
	lastOpenCol = -1
	balance = 0

	// Ensure we don't go beyond the line length
	if upToCol > len(line) {
		upToCol = len(line)
	}

	// Track open delimiter positions for alignment: only parentheses and brackets
	// Braces are handled by dedicated brace rules and should not drive alignment.
	openStack := []rune{}
	openPos := []int{}

	// Count delimiters up to cursor
	for i := 0; i < upToCol; i++ {
		ch := rune(line[i])
		switch ch {
		case '(', '[': // exclude '{'
			openStack = append(openStack, ch)
			openPos = append(openPos, i)
		case ')', ']': // exclude '}'
			if len(openStack) > 0 {
				// Pop matching if types correspond; otherwise still pop conservatively
				openStack = openStack[:len(openStack)-1]
				openPos = openPos[:len(openPos)-1]
			}
		}
	}

	// The number of unmatched open delimiters
	balance = len(openStack)

	// Get the last unbalanced open delimiter position
	if len(openPos) > 0 {
		lastOpenCol = openPos[len(openPos)-1]

		// Check if there are characters after this open paren
		if lastOpenCol < len(line)-1 {
			for i := lastOpenCol + 1; i < len(line); i++ {
				if !unicode.IsSpace(rune(line[i])) {
					hasCharsAfter = true
					break
				}
			}
		}
	}

	return balance, lastOpenCol, hasCharsAfter
}

// analyzeParentheses is kept for backward-compatibility with existing tests.
// It now delegates to analyzeDelimiters which handles (), [], {} uniformly.
func analyzeParentheses(line string, upToCol int) (int, int, bool) {
	return analyzeDelimiters(line, upToCol)
}

func isPythonFile(path string) bool {
	if path == "" {
		return false
	}
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".py" || ext == ".pyw"
}

// endsWithColonBeforeCursor returns true if the last non-space before cursor (or end of line)
// is a ':' character (used for Python block starters)
func endsWithColonBeforeCursor(line string, cursorCol int) bool {
	if cursorCol > len(line) {
		cursorCol = len(line)
	}
	// If cursor is on a ':' character, consider it a block opener
	if cursorCol >= 0 && cursorCol < len(line) && line[cursorCol] == ':' {
		return true
	}
	// Also consider the character immediately before the cursor
	if cursorCol > 0 && line[cursorCol-1] == ':' {
		return true
	}
	// Otherwise, scan backward from just before the cursor for the last non-space
	i := cursorCol - 1
	if i >= len(line) {
		i = len(line) - 1
	}
	for ; i >= 0; i-- {
		if !unicode.IsSpace(rune(line[i])) {
			return line[i] == ':'
		}
	}
	return false
}

// calculateVisualColumn converts a byte position to visual column, accounting for tabs
func calculateVisualColumn(line string, bytePos int) int {
	if bytePos > len(line) {
		bytePos = len(line)
	}

	visualCol := 0
	for i := 0; i < bytePos; i++ {
		if line[i] == '\t' {
			// Tab advances to next multiple of 8
			visualCol = (visualCol/8 + 1) * 8
		} else {
			visualCol++
		}
	}

	return visualCol
}

// shouldIndentAfterBrace checks if we should add extra indentation due to braces
// Returns true if:
// - Cursor is between { and } on the same line
// - Cursor is at the end of a line that ends with {
func shouldIndentAfterBrace(line string, cursorCol int) bool {
	// Ensure we don't go beyond the line length
	if cursorCol > len(line) {
		cursorCol = len(line)
	}

	// Check if cursor is between { and }
	if cursorCol > 0 && cursorCol < len(line) {
		// Look for { before cursor and } after cursor
		hasOpenBrace := false
		for i := cursorCol - 1; i >= 0; i-- {
			if line[i] == '{' {
				hasOpenBrace = true
				break
			} else if !unicode.IsSpace(rune(line[i])) {
				// Found non-space character before finding {
				break
			}
		}

		if hasOpenBrace {
			// Check if there's a } after cursor
			for i := cursorCol; i < len(line); i++ {
				if line[i] == '}' {
					return true
				} else if !unicode.IsSpace(rune(line[i])) {
					// Found non-space character before finding }
					break
				}
			}
		}
	}

	// Check if cursor is at end of line that ends with {
	if cursorCol == len(line) && len(line) > 0 {
		// Find the last non-whitespace character
		for i := len(line) - 1; i >= 0; i-- {
			if !unicode.IsSpace(rune(line[i])) {
				return line[i] == '{'
			}
		}
	}

	return false
}

// calculateIndentForEmptyLine determines the appropriate indentation for an empty line
// by analyzing the surrounding block context in the buffer
func (b *Buffer) calculateIndentForEmptyLine() AutoIndentInfo {
	// We need to find the current cursor position in the buffer to analyze context
	if b.CursorIterator == nil {
		// Fallback to no indentation if we can't determine context
		return AutoIndentInfo{Column: 0, IndentStr: ""}
	}

	currentLineNum := b.CursorIterator.LineNumber()

	// Look backward from current position to find the most recent non-empty line
	// and analyze the block structure
	braceBalance := 0
	baseIndentLevel := 0

	// Start from the line before current and go backward
	for lineNum := currentLineNum - 1; lineNum >= 0; lineNum-- {
		iter := NewBufferIteratorAtLine(b, lineNum)
		if iter.IsEOF() {
			continue
		}

		rawLine := iter.Line()
		lineContent := strings.TrimSpace(rawLine)
		if lineContent == "" {
			continue // Skip empty lines
		}

		// This is the first non-empty line we found going backward
		// Analyze its brace content and indentation
		lineIndentInfo := b.GetLineIndentInfo(iter.Line())

		// Python: if previous significant line ends with ':', increase indent
		if isPythonFile(b.FilePath) && endsWithColonBeforeCursor(rawLine, len(rawLine)) {
			baseIndentLevel = lineIndentInfo.Level + 1
		} else {
			// Count braces in this line
			openBraces := strings.Count(lineContent, "{")
			closeBraces := strings.Count(lineContent, "}")
			braceBalance = openBraces - closeBraces

			// The base indent level should be this line's level plus any brace changes
			baseIndentLevel = lineIndentInfo.Level
			if braceBalance > 0 {
				// There are more opening braces, so we should be one level deeper
				baseIndentLevel += braceBalance
			}
		}

		break // We found our context line
	}

	// Create the indentation
	indentStr := strings.Repeat(b.GetIndentationString(), baseIndentLevel)
	visualCol := len(indentStr)
	if b.Indentation.Type == config.IndentTabs {
		visualCol = baseIndentLevel * 8
	}

	return AutoIndentInfo{
		Column:    visualCol,
		IndentStr: indentStr,
	}
}

// findPreviousNonContinuationLevel scans backward from the given line (inclusive)
// to find the nearest non-empty line whose indentation is a standard level
// (i.e., not a continuation alignment). Returns that line's Level, or 0 if none.
func (b *Buffer) findPreviousNonContinuationLevel(fromLine int) int {
	if fromLine >= b.LineCount {
		fromLine = b.LineCount - 1
	}
	for lineNum := fromLine; lineNum >= 0; lineNum-- {
		it := NewBufferIteratorAtLine(b, lineNum)
		if it.IsEOF() {
			continue
		}
		raw := it.Line()
		if strings.TrimSpace(raw) == "" {
			continue
		}
		ii := b.GetLineIndentInfo(raw)
		if !ii.IsContinuation {
			return ii.Level
		}
	}
	return 0
}

// hasParenContinuationSinceBase scans backward from fromLine down to the nearest
// non-continuation line at the given baseLevel and determines whether there is
// an open parenthesis/bracket context spanning those lines (i.e., more '(' or '['
// than ')' or ']'). It ignores braces.
func (b *Buffer) hasParenContinuationSinceBase(fromLine int, baseLevel int) bool {
	if fromLine >= b.LineCount {
		fromLine = b.LineCount - 1
	}
	balance := 0
	for ln := fromLine; ln >= 0; ln-- {
		it := NewBufferIteratorAtLine(b, ln)
		if it.IsEOF() {
			continue
		}
		raw := it.Line()
		if strings.TrimSpace(raw) == "" {
			continue
		}
		// Count () and [] only
		for i := 0; i < len(raw); i++ {
			switch raw[i] {
			case '(', '[':
				balance++
			case ')', ']':
				balance--
			}
		}
		ii := b.GetLineIndentInfo(raw)
		if !ii.IsContinuation && ii.Level == baseLevel {
			break
		}
	}
	return balance > 0
}

// findBaseLevelFromOpenDelimiter scans backward from curLine to find the line
// containing the unmatched opening '(' or '[' that starts a continuation block
// reaching the current line. Returns that line's indent level and true if found.
func (b *Buffer) findBaseLevelFromOpenDelimiter(curLine int) (int, bool) {
	if curLine >= b.LineCount {
		curLine = b.LineCount - 1
	}
	needParen := 0
	needBracket := 0
	for ln := curLine; ln >= 0; ln-- {
		it := NewBufferIteratorAtLine(b, ln)
		if it.IsEOF() {
			continue
		}
		raw := it.Line()
		if strings.TrimSpace(raw) == "" {
			continue
		}
		// Scan this line right-to-left counting delimiters (ignore braces)
		for i := len(raw) - 1; i >= 0; i-- {
			switch raw[i] {
			case ')':
				needParen++
			case ']':
				needBracket++
			case '(': // match a pending ')'
				if needParen > 0 {
					// This '(' closes a ')' from the right. If this resolves
					// all pending closes and it's on a previous line, this is
					// the true anchor for a multi-line continuation.
					needParen--
					if needParen == 0 && ln < curLine {
						ii := b.GetLineIndentInfo(raw)
						return ii.Level, true
					}
				} else {
					// Unmatched '(': anchor only if on a previous line
					if ln < curLine {
						ii := b.GetLineIndentInfo(raw)
						return ii.Level, true
					}
				}
			case '[': // match a pending ']'
				if needBracket > 0 {
					// This '[' closes a ']' from the right.
					needBracket--
					if needBracket == 0 && ln < curLine {
						ii := b.GetLineIndentInfo(raw)
						return ii.Level, true
					}
				} else {
					if ln < curLine {
						ii := b.GetLineIndentInfo(raw)
						return ii.Level, true
					}
				}
			}
		}
	}
	return 0, false
}

// isAfterStatementEndCharacter checks if the cursor is positioned after a character
// that typically ends a statement (as opposed to continuing it).
// Returns true for characters like ';', '}' that end statements.
// Returns false for characters like ',' that indicate continuation.
func isAfterStatementEndCharacter(line string, cursorCol int) bool {
	if cursorCol == 0 {
		return false
	}

	// Ensure we don't go beyond the line length
	searchStart := cursorCol - 1
	if searchStart >= len(line) {
		searchStart = len(line) - 1
	}

	// Look backward from cursor position to find the last non-whitespace character
	for i := searchStart; i >= 0; i-- {
		char := line[i]
		if !unicode.IsSpace(rune(char)) {
			// Check if this character typically ends a statement
			switch char {
			case ';': // Statement terminator in C/C++/Java/etc
				return true
			case '}': // Block terminator
				return true
			case ',': // Continuation character
				return false
			default:
				// For other characters, consider context
				// If we're at the end of the line, it might be end of statement
				// but for now, be conservative and assume continuation
				return false
			}
		}
	}

	return false
}
