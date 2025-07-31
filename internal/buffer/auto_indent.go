package buffer

import (
	"strings"
	"unicode"
	"veda/internal/config"
)

// AutoIndentInfo contains information about how to indent the next line
type AutoIndentInfo struct {
	Column      int    // The column position to indent to (0-based)
	IndentStr   string // The actual string to use for indentation
}

// CalculateAutoIndent determines the indentation for a new line based on the current line
// and cursor position. It implements the following rules:
// 1. If cursor is between curly braces {} or after an open brace at end of line,
//    indent one level deeper
// 2. If current line appears to be a continuation line (has extra indentation beyond base level),
//    maintain that same indentation
// 3. If there are unbalanced open parentheses and characters after the last open paren,
//    indent to the column after the last open paren
// 4. If there are unbalanced open parentheses but no characters after,
//    indent to current line's indentation level plus one indent unit
// 5. Otherwise (balanced parens or no parens), maintain the same indentation level
func (b *Buffer) CalculateAutoIndent(line string, cursorCol int) AutoIndentInfo {
	// Get current line's base indentation
	baseIndentInfo := b.GetLineIndentInfo(line)
	
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
	
	// Check if this line appears to be a continuation line
	// A continuation line has unusual indentation that doesn't match standard indent levels
	actualIndent := baseIndentInfo.Spaces + (baseIndentInfo.Tabs * 8) // Visual columns
	
	// If the indentation is not a multiple of the indent width, it's likely a continuation line
	// (e.g., 9 or 10 spaces when indent width is 4)
	if actualIndent > 0 && actualIndent % b.Indentation.Width != 0 {
		// Maintain the exact same indentation as the current line
		return AutoIndentInfo{
			Column:    actualIndent,
			IndentStr: baseIndentInfo.ActualPrefix,
		}
	}
	
	// Count parentheses balance up to cursor
	balance, lastOpenCol, hasCharsAfter := analyzeParentheses(line, cursorCol)
	
	// Rule 2: Unbalanced parens with content after - align to position after last open paren
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
	
	// Rule 3: Unbalanced parens but no content after - add one indent level
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
	
	// Rule 4: Balanced parens or no parens - maintain same indentation
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
func analyzeParentheses(line string, upToCol int) (balance int, lastOpenCol int, hasCharsAfter bool) {
	lastOpenCol = -1
	balance = 0
	
	// Ensure we don't go beyond the line length
	if upToCol > len(line) {
		upToCol = len(line)
	}
	
	// Track open parentheses positions
	openParens := []int{}
	
	// Count parentheses up to cursor
	for i := 0; i < upToCol; i++ {
		if line[i] == '(' {
			openParens = append(openParens, i)
		} else if line[i] == ')' {
			if len(openParens) > 0 {
				openParens = openParens[:len(openParens)-1]
			}
		}
	}
	
	// The number of unmatched open parens
	balance = len(openParens)
	
	// Get the last unbalanced open paren position
	if len(openParens) > 0 {
		lastOpenCol = openParens[len(openParens)-1]
		
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

// calculateVisualColumn converts a byte position to visual column, accounting for tabs
func calculateVisualColumn(line string, bytePos int) int {
	if bytePos > len(line) {
		bytePos = len(line)
	}
	
	visualCol := 0
	for i := 0; i < bytePos; i++ {
		if line[i] == '\t' {
			// Tab advances to next multiple of 8
			visualCol = (visualCol / 8 + 1) * 8
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