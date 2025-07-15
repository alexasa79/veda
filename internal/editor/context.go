package editor

import (
	"veda/internal/proxy"
)

// getContextLineNumbers returns the line numbers for context display
// Returns a slice of line numbers from folding ranges that contain the cursor
// viewStartLine and viewEndLine define the currently visible range
func getContextLineNumbers(ranges []proxy.FoldingRange, cursorLine int, viewStartLine int, viewEndLine int, maxLines int) []int {
	// Find all ranges that contain the cursor line
	var lineNumbers []int
	
	for _, r := range ranges {
		if cursorLine >= r.StartLine && cursorLine <= r.EndLine {
			// Only include context lines that are not currently visible
			if r.StartLine < viewStartLine || r.StartLine > viewEndLine {
				lineNumbers = append(lineNumbers, r.StartLine)
			}
		}
	}
	
	// Limit to maxLines, keeping the most specific (deepest) context
	if len(lineNumbers) > maxLines {
		// Keep the last maxLines entries
		lineNumbers = lineNumbers[len(lineNumbers)-maxLines:]
	}
	
	return lineNumbers
}