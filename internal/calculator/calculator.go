package calculator

import (
	"fmt"
	"strings"
)

// Result represents the result of a calculation
type Result struct {
	Expression string  // The valid expression found
	Value      float64 // The calculated value
	StartPos   int     // Start position in the line
	EndPos     int     // End position in the line
}

// FindExpressionFromPosition finds a valid expression ending at the given position
// It works backwards from endPos to find the longest valid expression
func FindExpressionFromPosition(line string, endPos int) (*Result, error) {
	if endPos > len(line) {
		endPos = len(line)
	}
	if endPos <= 0 {
		return nil, fmt.Errorf("invalid position")
	}

	// Track the best (longest) valid expression found
	var bestResult *Result
	
	// Try progressively longer substrings from right to left
	for startPos := endPos - 1; startPos >= 0; startPos-- {
		// Get the substring to test
		testExpr := strings.TrimSpace(line[startPos:endPos])
		if testExpr == "" {
			continue
		}

		// Check for stopping characters at the start position
		ch := line[startPos]
		
		// Check for arrow symbol (→) in the substring
		// If the substring contains an arrow, only consider the part after the last arrow
		if idx := strings.LastIndex(line[startPos:endPos], "→"); idx != -1 {
			// Found an arrow in the substring
			// Get the part after the arrow
			afterArrowStart := startPos + idx + 3 // 3 bytes for "→"
			if afterArrowStart < endPos {
				afterArrow := strings.TrimSpace(line[afterArrowStart:endPos])
				if afterArrow != "" {
					parser := NewParser(afterArrow, afterArrowStart)
					value, err := parser.Parse()
					if err == nil && (bestResult == nil || len(afterArrow) > len(bestResult.Expression)) {
						// Calculate actual start position in the original line
						actualStart := afterArrowStart
						// Skip leading whitespace to get actual expression start
						for actualStart < endPos && (line[actualStart] == ' ' || line[actualStart] == '\t') {
							actualStart++
						}
						bestResult = &Result{
							Expression: afterArrow,
							Value:      value,
							StartPos:   actualStart,
							EndPos:     endPos,
						}
					}
				}
			}
			// Don't break - we found a result after the arrow, but don't continue past the arrow
			break
		}
		
		// Check for other delimiter characters
		if ch == '=' || ch == ';' || ch == ',' || ch == '[' || ch == ']' || 
		   ch == '{' || ch == '}' || ch == '"' || ch == '\'' || ch == '`' {
			// Found delimiter, can't include it in expression
			break
		}

		// Try to parse and evaluate this expression
		parser := NewParser(testExpr, startPos)
		value, err := parser.Parse()
		
		// If parsing succeeded, update best result (we want the longest valid expression)
		if err == nil {
			// Adjust start position to account for trimmed whitespace
			actualStartPos := startPos
			for actualStartPos < endPos && (line[actualStartPos] == ' ' || line[actualStartPos] == '\t') {
				actualStartPos++
			}
			bestResult = &Result{
				Expression: testExpr,
				Value:      value,
				StartPos:   actualStartPos,
				EndPos:     endPos,
			}
			// Continue searching for a longer expression
		}
	}

	// Also try from position 0 if we haven't already
	if endPos > 0 {
		testExpr := strings.TrimSpace(line[0:endPos])
		if testExpr != "" {
			parser := NewParser(testExpr, 0)
			value, err := parser.Parse()
			if err == nil {
				// Only use this if it's longer than what we already found
				if bestResult == nil || len(testExpr) > len(bestResult.Expression) {
					// Adjust start position to account for trimmed whitespace
					actualStartPos := 0
					for actualStartPos < endPos && (line[actualStartPos] == ' ' || line[actualStartPos] == '\t') {
						actualStartPos++
					}
					bestResult = &Result{
						Expression: testExpr,
						Value:      value,
						StartPos:   actualStartPos,
						EndPos:     endPos,
					}
				}
			}
		}
	}

	if bestResult != nil {
		return bestResult, nil
	}

	return nil, fmt.Errorf("no valid expression found")
}

// EvaluateExpression evaluates a mathematical expression and returns the result
func EvaluateExpression(expr string) (float64, error) {
	parser := NewParser(expr, 0)
	return parser.Parse()
}

// FormatResult formats a calculation result as a string
func FormatResult(value float64) string {
	// Check if the value is an integer
	if value == float64(int64(value)) {
		return fmt.Sprintf("%d", int64(value))
	}
	// Otherwise format as a float with reasonable precision
	// Remove trailing zeros
	s := fmt.Sprintf("%.10f", value)
	s = strings.TrimRight(s, "0")
	s = strings.TrimRight(s, ".")
	return s
}

// ProcessLineAppend processes a line and appends the calculation result
func ProcessLineAppend(line string, cursorPos int) (string, error) {
	// Find expression from cursor position going backward
	result, err := FindExpressionFromPosition(line, cursorPos)
	if err != nil {
		return line, err
	}

	// Format the result
	formattedResult := FormatResult(result.Value)
	
	// Create the new line with the result appended
	newLine := line[:result.EndPos] + " → " + formattedResult
	if result.EndPos < len(line) {
		newLine += line[result.EndPos:]
	}
	
	return newLine, nil
}

// ProcessLineReplace processes a line and replaces the expression with the result
func ProcessLineReplace(line string, cursorPos int) (string, int, error) {
	// Find expression from cursor position going backward
	result, err := FindExpressionFromPosition(line, cursorPos)
	if err != nil {
		return line, cursorPos, err
	}

	// Format the result
	formattedResult := FormatResult(result.Value)
	
	// Check if there's a space before the expression position
	// If so, we want to preserve it in the output
	if result.StartPos > 0 && line[result.StartPos-1] == ' ' {
		formattedResult = " " + formattedResult
		// Adjust to include the space in the replacement
		result.StartPos--
	}
	
	// Create the new line with the expression replaced
	newLine := line[:result.StartPos] + formattedResult
	newCursorPos := result.StartPos + len(formattedResult)
	if result.EndPos < len(line) {
		newLine += line[result.EndPos:]
	}
	
	return newLine, newCursorPos, nil
}