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
					// Tokenize and extract preprocessor directives
					tokens, _, _ := TokenizeAndExtractPreprocessors(afterArrow, afterArrowStart)
					parser := NewParserFromTokens(tokens)
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
		// Note: Don't treat comma as delimiter since it can be part of numbers (1,000)
		if ch == '=' || ch == ';' || ch == '[' || ch == ']' || 
		   ch == '{' || ch == '}' || ch == '"' || ch == '\'' || ch == '`' {
			// Found delimiter, can't include it in expression
			break
		}

		// Try to parse and evaluate this expression with preprocessing
		// First tokenize and extract preprocessor directives
		tokens, _, _ := TokenizeAndExtractPreprocessors(testExpr, startPos)
		
		// Parse using the tokens (which have directives removed)
		parser := NewParserFromTokens(tokens)
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
			// Tokenize and extract preprocessor directives
			tokens, _, _ := TokenizeAndExtractPreprocessors(testExpr, 0)
			parser := NewParserFromTokens(tokens)
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
	
	// Save result to $? variable
	parser.variables.SetLastResult(result)
	
	// Convert to float64 for backward compatibility
	f64, _ := result.Float64()
	return f64, nil
}

// EvaluateExpressionWithResult evaluates an expression and returns a CalculationResult with formatting info
func EvaluateExpressionWithResult(expr string) (*CalculationResult, error) {
	// Tokenize and extract preprocessor directives
	tokens, directive, hasDirective := TokenizeAndExtractPreprocessors(expr, 0)
	
	// Parse and evaluate the expression
	parser := NewParserFromTokens(tokens)
	value, err := parser.Parse()
	if err != nil {
		return nil, err
	}
	
	// Save result to $? variable
	parser.variables.SetLastResult(value)
	
	// Create result with appropriate formatting
	result := NewCalculationResult(value)
	if hasDirective {
		if directive.Precision >= 0 {
			result.SetPrecision(directive.Precision)
		}
		if directive.Format != FormatDecimal {
			result.SetFormat(directive.Format, directive.BitWidth)
		}
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

	// Check if we need to include preprocessor directives before or after the expression
	fullExpr := exprInfo.Expression
	
	// Helper function to check if a number is power of 2
	isPowerOfTwo := func(n int) bool {
		return n > 0 && (n&(n-1)) == 0
	}
	
	// Check for preprocessor directives right before the found expression
	// Check for "pre", "hex", "bin", "oct", "dec"
	for _, directive := range []string{"pre", "hex", "bin", "oct", "dec"} {
		dirLen := len(directive)
		if exprInfo.StartPos >= dirLen {
			checkStart := exprInfo.StartPos - dirLen
			if checkStart >= 0 && line[checkStart:exprInfo.StartPos] == directive {
				// This is a directive followed by number, need to find the actual expression
				if checkStart > 0 && directive == "pre" {
					// For "pre", look for expression before it
					exprBeforeDirective, err := FindExpressionFromPosition(line, checkStart)
					if err == nil {
						fullExpr = exprBeforeDirective.Expression + " " + directive + exprInfo.Expression
						break
					}
				} else if directive != "pre" {
					// For hex/bin/oct, this might be a bit width, look for expression before
					if checkStart > 0 {
						exprBeforeDirective, err := FindExpressionFromPosition(line, checkStart)
						if err == nil {
							// Check if the found expression is just a number that could be bit width
							num, err := strconv.Atoi(exprInfo.Expression)
							if err == nil && isPowerOfTwo(num) && num >= 8 && num <= 128 {
								// It's a bit width
								fullExpr = exprBeforeDirective.Expression + " " + directive + exprInfo.Expression
							} else {
								// Not a bit width, the directive applies to the expression after
								fullExpr = directive + " " + exprInfo.Expression
							}
							break
						}
					}
				}
			}
		}
	}
	
	// Check if there's a directive AFTER the expression
	// Look for patterns like "15 hex", "255 hex16", "7 bin", "15 dec"
	if exprInfo.EndPos < len(line) && !strings.Contains(fullExpr, "pre") && 
	   !strings.Contains(fullExpr, "hex") && !strings.Contains(fullExpr, "bin") && 
	   !strings.Contains(fullExpr, "oct") && !strings.Contains(fullExpr, "dec") {
		// Look forward for directive patterns
		searchEnd := exprInfo.EndPos + 10
		if searchEnd > len(line) {
			searchEnd = len(line)
		}
		afterStr := line[exprInfo.EndPos:searchEnd]
		trimmed := strings.TrimSpace(afterStr)
		
		// Check for directive patterns
		for _, directive := range []string{"pre", "hex", "bin", "oct", "dec"} {
			if strings.HasPrefix(trimmed, directive) {
				rest := strings.TrimPrefix(trimmed, directive)
				// Check if rest starts with a number (bit width)
				// Note: "dec" doesn't support bit width
				if rest != "" && len(rest) > 0 && directive != "dec" {
					// Try to extract number at the beginning
					numStr := ""
					for i, ch := range rest {
						if ch >= '0' && ch <= '9' {
							numStr += string(ch)
						} else {
							break
						}
						if i > 3 { // Reasonable limit for bit width
							break
						}
					}
					if numStr != "" {
						fullExpr = exprInfo.Expression + " " + directive + numStr
					} else {
						fullExpr = exprInfo.Expression + " " + directive
					}
				} else {
					// Just the directive without number
					fullExpr = exprInfo.Expression + " " + directive
				}
				break
			}
		}
	}
	
	// Also check if there's a directive pattern before the expression
	// Look for patterns like "pre20 2/3", "hex 15", "dec 3.14" where the expression comes after
	if exprInfo.StartPos > 0 && !strings.Contains(fullExpr, "pre") && 
	   !strings.Contains(fullExpr, "hex") && !strings.Contains(fullExpr, "bin") && 
	   !strings.Contains(fullExpr, "oct") && !strings.Contains(fullExpr, "dec") {
		// Look backwards for directive patterns
		searchStart := 0
		if exprInfo.StartPos > 10 {
			searchStart = exprInfo.StartPos - 10
		}
		beforeStr := line[searchStart:exprInfo.StartPos]
		trimmed := strings.TrimSpace(beforeStr)
		
		// Check for directive patterns
		for _, directive := range []string{"pre", "hex", "bin", "oct", "dec"} {
			if strings.HasPrefix(trimmed, directive) {
				rest := strings.TrimPrefix(trimmed, directive)
				// Check if rest is a number (could be bit width or precision)
				if rest != "" {
					if _, err := strconv.Atoi(rest); err == nil {
						// Include the directive in the expression
						fullExpr = directive + rest + " " + exprInfo.Expression
						break
					}
				} else {
					// Just the directive without number
					fullExpr = directive + " " + exprInfo.Expression
					break
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