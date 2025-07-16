package buffer

import (
	"testing"
	"veda/internal/config"
)

func TestCalculateAutoIndent(t *testing.T) {
	tests := []struct {
		name           string
		content        string
		cursorCol      int
		indentType     config.IndentationType
		indentWidth    int
		expectedColumn int
		expectedIndent string
	}{
		// Test case 1: Function call with parameter after open paren
		{
			name:           "function with param after paren",
			content:        "    function_call(parameter,",
			cursorCol:      28, // After the comma
			indentType:     config.IndentSpaces,
			indentWidth:    4,
			expectedColumn: 18, // Column of 'p' in parameter
			expectedIndent: "                  ", // 18 spaces
		},
		// Test case 2: Function call with just open paren
		{
			name:           "function with just open paren",
			content:        "    function_call(",
			cursorCol:      18, // After the open paren
			indentType:     config.IndentSpaces,
			indentWidth:    4,
			expectedColumn: 8, // Two indent levels (4 + 4)
			expectedIndent: "        ", // 8 spaces
		},
		// Test case 3: No unbalanced parens
		{
			name:           "no unbalanced parens",
			content:        "    let x = 5",
			cursorCol:      13, // End of line
			indentType:     config.IndentSpaces,
			indentWidth:    4,
			expectedColumn: 4, // Same indent level
			expectedIndent: "    ", // 4 spaces
		},
		// Test case 4: Balanced parens
		{
			name:           "balanced parens",
			content:        "    let x = foo(bar())",
			cursorCol:      22, // End of line
			indentType:     config.IndentSpaces,
			indentWidth:    4,
			expectedColumn: 4, // Same indent level
			expectedIndent: "    ", // 4 spaces
		},
		// Test case 5: Multiple open parens
		{
			name:           "multiple open parens",
			content:        "    foo(bar(baz,",
			cursorCol:      16, // After comma
			indentType:     config.IndentSpaces,
			indentWidth:    4,
			expectedColumn: 12, // Column of 'b' in baz
			expectedIndent: "            ", // 12 spaces
		},
		// Test case 6: Tab indentation with param after paren
		{
			name:           "tab indent with param after paren",
			content:        "\tfunc(param,",
			cursorCol:      12, // After comma
			indentType:     config.IndentTabs,
			indentWidth:    1,
			expectedColumn: 13, // Column of 'p' in param (tab=8 + 'func('=5)
			expectedIndent: "\t     ", // 1 tab + 5 spaces
		},
		// Test case 7: Tab indentation with just open paren
		{
			name:           "tab indent with just open paren",
			content:        "\tfunc(",
			cursorCol:      6, // After open paren
			indentType:     config.IndentTabs,
			indentWidth:    1,
			expectedColumn: 16, // Two tab levels
			expectedIndent: "\t\t", // 2 tabs
		},
		// Test case 8: Mixed content before paren
		{
			name:           "mixed content before paren",
			content:        "if x > 0 && func(param",
			cursorCol:      22, // After 'param'
			indentType:     config.IndentSpaces,
			indentWidth:    4,
			expectedColumn: 17, // Column of 'p' in param
			expectedIndent: "                 ", // 17 spaces
		},
		// Test case 9: Empty line
		{
			name:           "empty line",
			content:        "",
			cursorCol:      0,
			indentType:     config.IndentSpaces,
			indentWidth:    4,
			expectedColumn: 0, // Same indent level (which is 0)
			expectedIndent: "", // No spaces
		},
		// Test case 10: Cursor before open paren
		{
			name:           "cursor before open paren",
			content:        "    func(param",
			cursorCol:      8, // Before the open paren
			indentType:     config.IndentSpaces,
			indentWidth:    4,
			expectedColumn: 4, // Same indent level (no unbalanced parens considered)
			expectedIndent: "    ", // 4 spaces
		},
		// Test case 11: Whitespace after open paren
		{
			name:           "whitespace after open paren",
			content:        "    func(   ",
			cursorCol:      12, // After spaces
			indentType:     config.IndentSpaces,
			indentWidth:    4,
			expectedColumn: 8, // Two indent levels (no chars after paren)
			expectedIndent: "        ", // 8 spaces
		},
		// Test case 12: Nested parens with content
		{
			name:           "nested parens with content",
			content:        "    outer(inner(value),",
			cursorCol:      23, // After comma
			indentType:     config.IndentSpaces,
			indentWidth:    4,
			expectedColumn: 10, // Column of 'i' in inner (char after the unbalanced open paren at 9)
			expectedIndent: "          ", // 10 spaces
		},
		// Test case 13: Tab indent with partial position
		{
			name:           "tab indent needs mixed tabs and spaces",
			content:        "\t\tfunc(param,", 
			cursorCol:      20, // After comma
			indentType:     config.IndentTabs,
			indentWidth:    1,
			expectedColumn: 21, // Position of 'p': 2 tabs (16) + 'func(' (5)
			expectedIndent: "\t\t     ", // 2 tabs + 5 spaces
		},
		// Test case 14: Two space indentation
		{
			name:           "two space indentation",
			content:        "  if condition:",
			cursorCol:      15, // End of line
			indentType:     config.IndentSpaces,
			indentWidth:    2,
			expectedColumn: 2, // Same level with 2-space indent
			expectedIndent: "  ", // 2 spaces
		},
		// Test case 15: Long line with cursor in middle
		{
			name:           "cursor in middle of line",
			content:        "    func(a, b, c)",
			cursorCol:      10, // After 'a'
			indentType:     config.IndentSpaces,
			indentWidth:    4,
			expectedColumn: 9, // Column of 'a'
			expectedIndent: "         ", // 9 spaces
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create buffer with specific indentation settings
			buf := NewBuffer(tt.content, "", tt.indentType, tt.indentWidth)
			
			// Calculate auto indent
			result := buf.CalculateAutoIndent(tt.content, tt.cursorCol)
			
			// Check column
			if result.Column != tt.expectedColumn {
				t.Errorf("Expected column %d, got %d", tt.expectedColumn, result.Column)
			}
			
			// Check indent string
			if result.IndentStr != tt.expectedIndent {
				t.Errorf("Expected indent string %q, got %q", tt.expectedIndent, result.IndentStr)
			}
		})
	}
}

func TestAnalyzeParentheses(t *testing.T) {
	tests := []struct {
		name              string
		line              string
		upToCol           int
		expectedBalance   int
		expectedLastOpen  int
		expectedHasChars  bool
	}{
		{
			name:              "simple open paren with content",
			line:              "func(param",
			upToCol:           10,
			expectedBalance:   1,
			expectedLastOpen:  4,
			expectedHasChars:  true,
		},
		{
			name:              "simple open paren no content",
			line:              "func(",
			upToCol:           5,
			expectedBalance:   1,
			expectedLastOpen:  4,
			expectedHasChars:  false,
		},
		{
			name:              "balanced parens",
			line:              "func(param)",
			upToCol:           11,
			expectedBalance:   0,
			expectedLastOpen:  -1, // No unbalanced parens
			expectedHasChars:  false, // No unbalanced parens to check
		},
		{
			name:              "nested parens",
			line:              "outer(inner(",
			upToCol:           12,
			expectedBalance:   2,
			expectedLastOpen:  11,
			expectedHasChars:  false,
		},
		{
			name:              "no parens",
			line:              "let x = 5",
			upToCol:           9,
			expectedBalance:   0,
			expectedLastOpen:  -1,
			expectedHasChars:  false,
		},
		{
			name:              "cursor before paren",
			line:              "func(param",
			upToCol:           4,
			expectedBalance:   0,
			expectedLastOpen:  -1,
			expectedHasChars:  false,
		},
		{
			name:              "whitespace after paren",
			line:              "func(   ",
			upToCol:           8,
			expectedBalance:   1,
			expectedLastOpen:  4,
			expectedHasChars:  false,
		},
		{
			name:              "multiple opens and closes",
			line:              "f(a) + g(b) + h(c",
			upToCol:           17,
			expectedBalance:   1,
			expectedLastOpen:  15, // Only h( is unbalanced
			expectedHasChars:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			balance, lastOpen, hasChars := analyzeParentheses(tt.line, tt.upToCol)
			
			if balance != tt.expectedBalance {
				t.Errorf("Expected balance %d, got %d", tt.expectedBalance, balance)
			}
			
			if lastOpen != tt.expectedLastOpen {
				t.Errorf("Expected lastOpen %d, got %d", tt.expectedLastOpen, lastOpen)
			}
			
			if hasChars != tt.expectedHasChars {
				t.Errorf("Expected hasChars %v, got %v", tt.expectedHasChars, hasChars)
			}
		})
	}
}

func TestCalculateVisualColumn(t *testing.T) {
	tests := []struct {
		name        string
		line        string
		bytePos     int
		expectedCol int
	}{
		{
			name:        "no tabs",
			line:        "hello world",
			bytePos:     5,
			expectedCol: 5,
		},
		{
			name:        "single tab at start",
			line:        "\thello",
			bytePos:     1,
			expectedCol: 8,
		},
		{
			name:        "single tab at start, pos after",
			line:        "\thello",
			bytePos:     3,
			expectedCol: 10,
		},
		{
			name:        "multiple tabs",
			line:        "\t\thello",
			bytePos:     2,
			expectedCol: 16,
		},
		{
			name:        "mixed tabs and spaces",
			line:        "  \t  hello",
			bytePos:     5,
			expectedCol: 10,
		},
		{
			name:        "tab in middle",
			line:        "abc\tdef",
			bytePos:     4,
			expectedCol: 8,
		},
		{
			name:        "beyond line length",
			line:        "hello",
			bytePos:     10,
			expectedCol: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			col := calculateVisualColumn(tt.line, tt.bytePos)
			if col != tt.expectedCol {
				t.Errorf("Expected column %d, got %d", tt.expectedCol, col)
			}
		})
	}
}

func TestCreateIndentStringForColumn(t *testing.T) {
	tests := []struct {
		name           string
		indentType     config.IndentationType
		indentWidth    int
		targetCol      int
		expectedIndent string
	}{
		{
			name:           "spaces exact",
			indentType:     config.IndentSpaces,
			indentWidth:    4,
			targetCol:      12,
			expectedIndent: "            ", // 12 spaces
		},
		{
			name:           "tabs exact",
			indentType:     config.IndentTabs,
			indentWidth:    1,
			targetCol:      16,
			expectedIndent: "\t\t", // 2 tabs
		},
		{
			name:           "tabs with remainder",
			indentType:     config.IndentTabs,
			indentWidth:    1,
			targetCol:      10,
			expectedIndent: "\t  ", // 1 tab + 2 spaces
		},
		{
			name:           "tabs with larger remainder",
			indentType:     config.IndentTabs,
			indentWidth:    1,
			targetCol:      22,
			expectedIndent: "\t\t      ", // 2 tabs + 6 spaces
		},
		{
			name:           "zero column",
			indentType:     config.IndentSpaces,
			indentWidth:    4,
			targetCol:      0,
			expectedIndent: "", // No indentation
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := NewBuffer("", "", tt.indentType, tt.indentWidth)
			result := buf.createIndentStringForColumn(tt.targetCol)
			
			if result != tt.expectedIndent {
				t.Errorf("Expected indent %q, got %q", tt.expectedIndent, result)
			}
		})
	}
}