package editor

import (
	"strings"
	"veda/internal/buffer"
	"veda/internal/config"
)

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
func (e *Editor) detectIndentation() buffer.IndentationInfo {
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
		return buffer.IndentationInfo{Type: config.IndentTabs, Width: 1}
	}

	// If no reliable indented lines, use default from config
	if len(spaceCounts) == 0 {
		return buffer.IndentationInfo{Type: e.config.DefaultIndentType, Width: e.config.DefaultIndentWidth}
	}

	// Count frequency of each indentation level
	freqMap := make(map[int]int)
	for _, spaces := range spaceCounts {
		freqMap[spaces]++
	}

	// Weighted vote among standard widths {2,4,8}
	// score(width) = 2*exactCount + multiplesCount
	candidates := []int{2, 4, 8}
	bestWidth := 4
	bestScore := -1

	for _, width := range candidates {
		exact := 0
		multiples := 0
		for spaces, count := range freqMap {
			if spaces == width {
				exact += count
			}
			if spaces%width == 0 {
				multiples += count
			}
		}
		score := 2*exact + multiples
		if score > bestScore || (score == bestScore && width > bestWidth) {
			bestScore = score
			bestWidth = width
		}
	}

	// If no candidate gained any support, fall back to defaults
	if bestScore <= 0 {
		return buffer.IndentationInfo{Type: e.config.DefaultIndentType, Width: e.config.DefaultIndentWidth}
	}

	return buffer.IndentationInfo{Type: config.IndentSpaces, Width: bestWidth}
}

// getCurrentLineIndentInfo returns detailed indentation info for the line at the iterator
func (e *Editor) getCurrentLineIndentInfo(iter *buffer.BufferIterator) buffer.LineIndentInfo {
	return e.buffer.GetCurrentLineIndentInfo(iter)
}
