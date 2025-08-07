package calculator

import (
	"fmt"
	"strconv"
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
					bigValue, err := parser.Parse()
					if err == nil && (bestResult == nil || len(afterArrow) > len(bestResult.Expression)) {
						// Convert to float64 for Result struct
						value, _ := bigValue.Float64()
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
		bigValue, err := parser.Parse()
		
		// If parsing succeeded, update best result (we want the longest valid expression)
		if err == nil {
			// Convert to float64 for Result struct
			value, _ := bigValue.Float64()
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
			bigValue, err := parser.Parse()
			if err == nil {
				// Convert to float64 for Result struct
				value, _ := bigValue.Float64()
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
	// Tokenize and extract preprocessor directives
	tokens, _, _ := TokenizeAndExtractPreprocessors(expr, 0)
	
	// Parse and evaluate the expression
	parser := NewParserFromTokens(tokens)
	result, err := parser.Parse()
	if err != nil {
		return 0, err
	}
	
	// Convert to float64 for backward compatibility
	f64, _ := result.Float64()
	return f64, nil
}

// EvaluateExpressionWithResult evaluates an expression and returns a CalculationResult with formatting info
func EvaluateExpressionWithResult(expr string) (*CalculationResult, error) {
	// Tokenize and extract preprocessor directives
	tokens, precision, hasPrecision := TokenizeAndExtractPreprocessors(expr, 0)
	
	// Parse and evaluate the expression
	parser := NewParserFromTokens(tokens)
	value, err := parser.Parse()
	if err != nil {
		return nil, err
	}
	
	// Create result with appropriate precision
	result := NewCalculationResult(value)
	if hasPrecision {
		result.SetPrecision(precision)
	}
	
	return result, nil
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
	exprInfo, err := FindExpressionFromPosition(line, cursorPos)
	if err != nil {
		return line, err
	}

	// Check if we need to include "pre" directive before or after the expression
	fullExpr := exprInfo.Expression
	
	// Check if "pre" comes right before the found expression
	if exprInfo.StartPos >= 3 {
		checkStart := exprInfo.StartPos - 3
		if checkStart >= 0 && line[checkStart:exprInfo.StartPos] == "pre" {
			// This is "preNNN", we need to find the actual expression before "pre"
			if checkStart > 0 {
				exprBeforePre, err := FindExpressionFromPosition(line, checkStart)
				if err == nil {
					// Found a valid expression before "pre"
					fullExpr = exprBeforePre.Expression + " pre" + exprInfo.Expression
				}
			}
		}
	}
	
	// Also check if there's a "preNNN" pattern before the expression
	// Look for patterns like "pre20 2/3" where the expression is "2/3"
	if exprInfo.StartPos > 0 && !strings.Contains(fullExpr, "pre") {
		// Look backwards for "preNNN " pattern
		searchStart := 0
		if exprInfo.StartPos > 10 {
			searchStart = exprInfo.StartPos - 10
		}
		beforeStr := line[searchStart:exprInfo.StartPos]
		trimmed := strings.TrimSpace(beforeStr)
		
		// Check if it ends with "preNNN"
		if strings.HasPrefix(trimmed, "pre") {
			rest := strings.TrimPrefix(trimmed, "pre")
			// Check if rest is a number
			if rest != "" {
				if _, err := strconv.Atoi(rest); err == nil {
					// Include the pre directive in the expression
					fullExpr = "pre" + rest + " " + exprInfo.Expression
				}
			}
		}
	}

	// Evaluate with precision support
	result, err := EvaluateExpressionWithResult(fullExpr)
	if err != nil {
		return line, err
	}
	
	// Format the result
	formattedResult := result.String()
	
	// Create the new line with the result appended
	newLine := line[:exprInfo.EndPos] + " → " + formattedResult
	if exprInfo.EndPos < len(line) {
		newLine += line[exprInfo.EndPos:]
	}
	
	return newLine, nil
}

// ProcessLineReplace processes a line and replaces the expression with the result
func ProcessLineReplace(line string, cursorPos int) (string, int, error) {
	// Find expression from cursor position going backward
	exprInfo, err := FindExpressionFromPosition(line, cursorPos)
	if err != nil {
		return line, cursorPos, err
	}

	// Re-evaluate with precision support
	result, err := EvaluateExpressionWithResult(exprInfo.Expression)
	if err != nil {
		return line, cursorPos, err
	}
	
	// Format the result
	formattedResult := result.String()
	
	// Check if there's a space before the expression position
	// If so, we want to preserve it in the output
	if exprInfo.StartPos > 0 && line[exprInfo.StartPos-1] == ' ' {
		formattedResult = " " + formattedResult
		// Adjust to include the space in the replacement
		exprInfo.StartPos--
	}
	
	// Create the new line with the expression replaced
	newLine := line[:exprInfo.StartPos] + formattedResult
	newCursorPos := exprInfo.StartPos + len(formattedResult)
	if exprInfo.EndPos < len(line) {
		newLine += line[exprInfo.EndPos:]
	}
	
	return newLine, newCursorPos, nil
}