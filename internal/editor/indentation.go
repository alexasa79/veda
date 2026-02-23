package editor

import (
	"path/filepath"
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

func shouldUseReliableIndentHeuristic(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".c", ".cc", ".cpp", ".cxx", ".c++", ".h", ".hh", ".hpp", ".hxx", ".m", ".go", ".js", ".jsx", ".ts", ".tsx":
		return true
	default:
		return false
	}
}

func collectIndentationStats(lines []string, include func(string) bool) (int, []int) {
	tabCount := 0
	spaceCounts := []int{}
	for _, line := range lines {
		if len(line) == 0 {
			continue
		}
		if strings.TrimSpace(line) == "" {
			continue
		}
		if !include(line) {
			continue
		}

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

		if leadingTabs > 0 {
			tabCount++
		} else if leadingSpaces > 0 {
			spaceCounts = append(spaceCounts, leadingSpaces)
		}
	}
	return tabCount, spaceCounts
}

// detectIndentation analyzes the buffer to determine indentation style
func (e *Editor) detectIndentation() buffer.IndentationInfo {
	lines := e.CurrentBuffer().Lines()
	filePath := e.CurrentBuffer().RealPath
	if filePath == "" {
		filePath = e.CurrentBuffer().FilePath
	}
	useReliable := shouldUseReliableIndentHeuristic(filePath)

	tabCount, spaceCounts := collectIndentationStats(lines, func(line string) bool {
		if !useReliable {
			return true
		}
		return isReliableIndentLine(line)
	})
	if useReliable && tabCount == 0 && len(spaceCounts) == 0 {
		tabCount, spaceCounts = collectIndentationStats(lines, func(string) bool { return true })
	}

	// If we saw neither tabs nor spaces on reliable lines, use defaults
	if tabCount == 0 && len(spaceCounts) == 0 {
		return buffer.IndentationInfo{Type: e.config.DefaultIndentType, Width: e.config.DefaultIndentWidth}
	}

	// Count frequency of each indentation level (for spaces)
	freqMap := make(map[int]int)
	for _, spaces := range spaceCounts {
		freqMap[spaces]++
	}

	// Weighted vote among standard widths {2,4,8}
	// score(width) = 2*exactCount + multiplesCount
	candidates := []int{2, 4, 8}
	bestWidth := e.config.DefaultIndentWidth
	bestScore := 0

	if len(spaceCounts) > 0 {
		bestScore = -1
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
		// If no candidate gained any support, treat as zero support
		if bestScore < 0 {
			bestScore = 0
		}
	}

	// Decide between Tabs vs. Spaces by majority-style comparison.
	// Tabs win only if they have strictly more support than the best spaces width.
	// On ties, or when spaces have equal/greater support, choose spaces.
	if tabCount > bestScore {
		return buffer.IndentationInfo{Type: config.IndentTabs, Width: 1}
	}
	if bestScore > 0 {
		return buffer.IndentationInfo{Type: config.IndentSpaces, Width: bestWidth}
	}

	// Fallback if both have zero support
	return buffer.IndentationInfo{Type: e.config.DefaultIndentType, Width: e.config.DefaultIndentWidth}
}

// getCurrentLineIndentInfo returns detailed indentation info for the line at the iterator
func (e *Editor) getCurrentLineIndentInfo(iter *buffer.BufferIterator) buffer.LineIndentInfo {
	info := e.CurrentBuffer().GetCurrentLineIndentInfo(iter)
	// For display purposes, treat lines aligned under an open paren/bracket as continuation
	// even if their indentation is a clean multiple of the width. Map to nearest base level.
	if e.CurrentBuffer().Indentation.Type == config.IndentSpaces {
		curLine := iter.LineNumber()
		// Detect cross-line paren continuation (ignore same-line balanced)
		// Instrument the delimiter scan to confirm its runtime impact.
		tScan := e.perfTracker.Start()
		baseLevel, cont := findBaseLevelFromOpenDelimiterForEditor(e.CurrentBuffer(), curLine)
		_ = e.perfTracker.ReportSince("indent.findBaseLevelFromOpenDelimiter", tScan)
		if cont {
			info.IsContinuation = true
			if baseLevel < info.Level {
				info.Level = baseLevel
			}
		}
	}
	return info
}

// findBaseLevelFromOpenDelimiterForEditor mirrors the buffer logic to discover the
// base indent level from the unmatched opening '(' or '[' scanning backward.
func findBaseLevelFromOpenDelimiterForEditor(b *buffer.Buffer, curLine int) (int, bool) {
	if curLine >= b.LineCount {
		curLine = b.LineCount - 1
	}
	needParen := 0
	needBracket := 0
	// Limit how far we scan upward to avoid O(N) work on huge files
	const maxScanLines = 50
	linesScanned := 0
	for ln := curLine; ln >= 0; ln-- {
		it := buffer.NewBufferIteratorAtLine(b, ln)
		if it.IsEOF() {
			continue
		}
		raw := it.Line()
		if strings.TrimSpace(raw) == "" {
			continue
		}

		// Ignore any closing delimiters that appear on the current line.
		// We want a line that contains a same-line closing ')' to still be
		// treated as a continuation of the open delimiter from a previous line
		// for display purposes (show base level with +C).
		if ln == curLine {
			continue
		}
		linesScanned++
		if linesScanned > maxScanLines {
			break
		}
		for i := len(raw) - 1; i >= 0; i-- {
			switch raw[i] {
			case ')':
				needParen++
			case ']':
				needBracket++
			case '(':
				if needParen > 0 {
					// closes a right-side ')'; not an anchor
					needParen--
				} else {
					if ln < curLine {
						ii := b.GetLineIndentInfo(raw)
						return ii.Level, true
					}
				}
			case '[':
				if needBracket > 0 {
					// closes a right-side ']'; not an anchor
					needBracket--
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

// findNearestNonContinuationLevelForEditor returns the nearest previous
// non-empty line's Level that is not continuation-aligned.
func findNearestNonContinuationLevelForEditor(b *buffer.Buffer, fromLine int) int {
	if fromLine >= b.LineCount {
		fromLine = b.LineCount - 1
	}
	for ln := fromLine; ln >= 0; ln-- {
		it := buffer.NewBufferIteratorAtLine(b, ln)
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
