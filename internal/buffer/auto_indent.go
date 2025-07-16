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
// 1. If there are unbalanced open parentheses and characters after the last open paren,
//    indent to the column after the last open paren
// 2. If there are unbalanced open parentheses but no characters after,
//    indent to current line's indentation level plus one indent unit
// 3. Otherwise (balanced parens or no parens), maintain the same indentation level
func (b *Buffer) CalculateAutoIndent(line string, cursorCol int) AutoIndentInfo {
	// Get current line's base indentation
	baseIndentInfo := b.GetLineIndentInfo(line)
	
	// Count parentheses balance up to cursor
	balance, lastOpenCol, hasCharsAfter := analyzeParentheses(line, cursorCol)
	
	// Rule 1: Unbalanced parens with characters after last open paren
	if balance > 0 && hasCharsAfter {
		// Find the visual column of the character after the last open paren
		// We need to account for tabs when calculating visual columns
		visualCol := calculateVisualColumn(line, lastOpenCol+1)
		
		// Create indentation string using the buffer's settings
		// We'll try to use the same indentation style as the file
		indentStr := b.createIndentStringForColumn(visualCol)
		
		return AutoIndentInfo{
			Column:    visualCol,
			IndentStr: indentStr,
		}
	}
	
	// Rule 2: Unbalanced parens with no characters after
	if balance > 0 && !hasCharsAfter {
		// Indent to current level plus one
		newLevel := baseIndentInfo.Level + 1
		indentStr := strings.Repeat(b.GetIndentationString(), newLevel)
		
		// Calculate visual column for this indentation
		visualCol := len(indentStr)
		if b.Indentation.Type == config.IndentTabs {
			// Each tab counts as 8 visual columns
			visualCol = newLevel * 8
		}
		
		return AutoIndentInfo{
			Column:    visualCol,
			IndentStr: indentStr,
		}
	}
	
	// Rule 3: Balanced parens or no parens - maintain same indentation
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