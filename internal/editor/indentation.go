package editor

import (
	"fmt"
	"strings"
)

// IndentationType represents the type of indentation used
type IndentationType int

const (
	IndentSpaces IndentationType = iota
	IndentTabs
)

// IndentationInfo holds information about file indentation
type IndentationInfo struct {
	Type  IndentationType
	Width int // Number of spaces per indent level (for spaces) or 1 for tabs
}

// String returns a string representation for the status bar
func (i *IndentationInfo) String() string {
	if i.Type == IndentTabs {
		return "Tabs"
	}
	return fmt.Sprintf("Spaces:%d", i.Width)
}

// gcd calculates the greatest common divisor
func gcd(a, b int) int {
	for b != 0 {
		a, b = b, a%b
	}
	return a
}

// isReliableIndentLine checks if a line is a reliable indicator of indentation
func isReliableIndentLine(line string) bool {
	trimmed := strings.TrimSpace(line)
	if len(trimmed) == 0 {
		return false
	}
	
	// Check for lines with only braces
	if trimmed == "{" || trimmed == "}" || trimmed == "{}" ||
	   trimmed == "(" || trimmed == ")" || 
	   trimmed == "[" || trimmed == "]" {
		return true
	}
	
	// Check for lines starting with common keywords
	// These keywords typically start a new statement/block
	keywords := []string{
		// Control flow
		"if", "for", "while", "do", "switch", "case", "default",
		"try", "catch", "finally", "except",
		
		// Function/method definitions  
		"func", "function", "def", "sub", "proc",
		"void", "int", "char", "float", "double", "bool", "auto",
		"public", "private", "protected", "static",
		
		// Type definitions
		"class", "struct", "enum", "interface", "trait",
		"type", "typedef", "namespace", "package", "module",
		
		// Other structural elements
		"return", "break", "continue", "goto",
		"import", "include", "require", "use", "using",
		"#include", "#define", "#ifdef", "#ifndef", "#endif", "#else", "#elif",
		
		// Language specific
		"var", "let", "const", // JavaScript/TypeScript
		"async", "await",
		"template", "typename", // C++
	}
	
	// Split the line into words
	words := strings.Fields(trimmed)
	if len(words) == 0 {
		return false
	}
	
	// Check if the first word is a keyword
	firstWord := strings.ToLower(words[0])
	for _, keyword := range keywords {
		if firstWord == keyword {
			return true
		}
	}
	
	// Check for special patterns
	// Access specifiers (C++)
	if trimmed == "public:" || trimmed == "private:" || trimmed == "protected:" {
		return true
	}
	
	// Labels (but not case labels which we already handle)
	if strings.HasSuffix(trimmed, ":") && !strings.Contains(trimmed, " ") && 
	   !strings.HasPrefix(trimmed, "case") && trimmed != "default:" {
		return true
	}
	
	// Line ending with semicolon (likely a complete statement)
	if strings.HasSuffix(trimmed, ";") && len(trimmed) > 1 {
		// But exclude lines that are likely continuations
		if !strings.Contains(trimmed, "=") && !strings.Contains(trimmed, ",") {
			return true
		}
	}
	
	return false
}

// detectIndentation analyzes the buffer to determine indentation style
func (e *Editor) detectIndentation() IndentationInfo {
	lines := e.buffer.Lines()

	// Count occurrences of different indentation patterns
	tabCount := 0
	spaceCounts := []int{} // Indentation levels from reliable lines

	for _, line := range lines {
		if len(line) == 0 {
			continue
		}

		// Skip lines that aren't reliable indicators
		if !isReliableIndentLine(line) {
			continue
		}

		// Count leading whitespace
		leadingSpaces := 0
		leadingTabs := 0

		for _, ch := range line {
			if ch == '\t' {
				leadingTabs++
			} else if ch == ' ' {
				leadingSpaces++
			} else {
				break
			}
		}

		// If line starts with tabs, count it
		if leadingTabs > 0 {
			tabCount++
		} else if leadingSpaces > 0 {
			spaceCounts = append(spaceCounts, leadingSpaces)
		}
	}

	// Determine the most common indentation
	if tabCount > 0 {
		// If any tabs are found, assume tab indentation
		return IndentationInfo{Type: IndentTabs, Width: 1}
	}

	// If no reliable indented lines, use default
	if len(spaceCounts) == 0 {
		return IndentationInfo{Type: IndentSpaces, Width: 4}
	}

	// Count frequency of each indentation level
	freqMap := make(map[int]int)
	for _, spaces := range spaceCounts {
		freqMap[spaces]++
	}

	// Find the smallest common indentation
	minIndent := -1
	for spaces := range freqMap {
		if spaces > 0 && (minIndent == -1 || spaces < minIndent) {
			minIndent = spaces
		}
	}

	// If the minimum indentation is a standard width, use it
	if minIndent == 2 || minIndent == 4 || minIndent == 8 {
		return IndentationInfo{Type: IndentSpaces, Width: minIndent}
	}

	// Otherwise, find GCD of all indentation levels
	if len(spaceCounts) > 0 {
		indentWidth := spaceCounts[0]
		for i := 1; i < len(spaceCounts); i++ {
			indentWidth = gcd(indentWidth, spaceCounts[i])
		}
		
		// Ensure it's a reasonable width
		if indentWidth == 2 || indentWidth == 4 || indentWidth == 8 {
			return IndentationInfo{Type: IndentSpaces, Width: indentWidth}
		}
		
		// If GCD gives an odd result, check which standard width fits best
		bestWidth := 4
		bestScore := 0
		
		for _, width := range []int{2, 4, 8} {
			score := 0
			for spaces, count := range freqMap {
				if spaces%width == 0 {
					score += count
				}
			}
			if score > bestScore {
				bestScore = score
				bestWidth = width
			}
		}
		
		return IndentationInfo{Type: IndentSpaces, Width: bestWidth}
	}

	// Default fallback
	return IndentationInfo{Type: IndentSpaces, Width: 4}
}

// setIndentationType sets the indentation type for the current buffer
func (e *Editor) setIndentationType(indentType IndentationType) {
	e.indentation.Type = indentType
	if indentType == IndentTabs {
		e.indentation.Width = 1
	}
}

// setIndentationWidth sets the indentation width (for spaces)
func (e *Editor) setIndentationWidth(width int) {
	if width > 0 && width <= 8 {
		e.indentation.Width = width
		if e.indentation.Type == IndentTabs {
			e.indentation.Type = IndentSpaces
		}
	}
}

// getIndentationString returns the string to use for one level of indentation
func (e *Editor) getIndentationString() string {
	if e.indentation.Type == IndentTabs {
		return "\t"
	}
	return strings.Repeat(" ", e.indentation.Width)
}

// getLineIndentation returns the indentation string of a given line
func getLineIndentation(line string) string {
	indentEnd := 0
	for i, ch := range line {
		if ch != ' ' && ch != '\t' {
			indentEnd = i
			break
		}
	}

	if indentEnd == 0 && len(line) > 0 && (line[0] == ' ' || line[0] == '\t') {
		// Entire line is whitespace
		return line
	}

	return line[:indentEnd]
}

// getLineIndentLevel calculates the indent level of a line
func (e *Editor) getLineIndentLevel(line string) int {
	level := 0
	for _, ch := range line {
		if ch == '\t' {
			level++
		} else if ch == ' ' {
			if e.indentation.Type == IndentSpaces && e.indentation.Width > 0 {
				// For spaces, we'll count later
			} else {
				break
			}
		} else {
			break
		}
	}

	// If using spaces, count space groups
	if e.indentation.Type == IndentSpaces && e.indentation.Width > 0 {
		spaceCount := 0
		for _, ch := range line {
			if ch == ' ' {
				spaceCount++
			} else if ch != '\t' {
				break
			}
		}
		level += spaceCount / e.indentation.Width
	}

	return level
}