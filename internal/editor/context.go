package editor

import (
	"sort"
	"strings"
	"veda/internal/buffer"
	"veda/internal/proxy"
)

// findFunctionSignatureLine searches backwards from startLine to find the actual function signature
// This is needed because LSP folding ranges often start at the opening brace, not the function declaration
func findFunctionSignatureLine(buf *buffer.Buffer, startLine int) int {
	// First check if the start line itself is already a good signature
	iter := buffer.NewBufferIteratorAtLine(buf, startLine)
	if !iter.IsEOF() {
		line := strings.TrimSpace(iter.Line())
		// If the line contains 'func' or looks like a function declaration, use it
		if strings.Contains(line, "func ") || 
		   (strings.Contains(line, "(") && strings.Contains(line, ")") && !strings.HasPrefix(line, "{")) {
			return startLine
		}
	}
	
	// If the start line is just an opening brace, look backwards
	if !iter.IsEOF() && strings.TrimSpace(iter.Line()) == "{" {
		// Look at the previous line
		if startLine > 0 {
			prevIter := buffer.NewBufferIteratorAtLine(buf, startLine-1)
			if !prevIter.IsEOF() {
				prevLine := strings.TrimSpace(prevIter.Line())
				// Check if previous line is a function signature
				if strings.Contains(prevLine, "func ") ||
				   (strings.Contains(prevLine, "(") && strings.Contains(prevLine, ")")) {
					return startLine - 1
				}
			}
		}
	}
	
	// Otherwise, return the original line
	return startLine
}

// getContextLineNumbers returns the line numbers for context display
// Returns a slice of line numbers from folding ranges that contain the cursor
// viewStartLine and viewEndLine define the currently visible range
func getContextLineNumbers(buf *buffer.Buffer, ranges []proxy.FoldingRange, cursorLine int, viewStartLine int, viewEndLine int, maxLines int) []int {
	// Find all ranges that contain the cursor line
	var contextRanges []proxy.FoldingRange
	
	for _, r := range ranges {
		if cursorLine >= r.StartLine && cursorLine <= r.EndLine {
			contextRanges = append(contextRanges, r)
		}
	}
	
	// Convert to line numbers, finding actual function signatures
	// Use a map to deduplicate lines
	lineMap := make(map[int]bool)
	var lineNumbers []int
	
	for _, r := range contextRanges {
		// Try to find the actual function signature line
		contextLine := findFunctionSignatureLine(buf, r.StartLine)
		
		// Only include context lines that are not currently visible
		// and haven't been added yet
		if (contextLine < viewStartLine || contextLine > viewEndLine) && !lineMap[contextLine] {
			lineMap[contextLine] = true
			lineNumbers = append(lineNumbers, contextLine)
		}
	}
	
	// Sort line numbers to maintain order
	sort.Ints(lineNumbers)
	
	// Limit to maxLines, keeping the most specific (deepest) context
	if len(lineNumbers) > maxLines {
		// Keep the last maxLines entries
		lineNumbers = lineNumbers[len(lineNumbers)-maxLines:]
	}
	
	return lineNumbers
}