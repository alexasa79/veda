package buffer

import (
	"testing"
	"veda/internal/config"
)

func TestCalculateAutoIndentWithBraces(t *testing.T) {
	tests := []struct {
		name           string
		line           string
		cursorCol      int
		indentType     config.IndentationType
		indentWidth    int
		expectedColumn int
		expectedIndent string
		description    string
	}{
		{
			name:           "cursor between braces",
			line:           "if (true) {}",
			cursorCol:      11, // Position between { and }
			indentType:     config.IndentSpaces,
			indentWidth:    4,
			expectedColumn: 4,
			expectedIndent: "    ",
			description:    "Should indent one level when cursor is between braces",
		},
		{
			name:           "cursor between braces with indentation",
			line:           "    if (true) {}",
			cursorCol:      15, // Position between { and }
			indentType:     config.IndentSpaces,
			indentWidth:    4,
			expectedColumn: 8,
			expectedIndent: "        ",
			description:    "Should add one indent level to existing indentation",
		},
		{
			name:           "cursor after open brace",
			line:           "if (true) {",
			cursorCol:      11, // Position after {
			indentType:     config.IndentSpaces,
			indentWidth:    4,
			expectedColumn: 4,
			expectedIndent: "    ",
			description:    "Should indent one level after open brace",
		},
		{
			name:           "cursor between braces with tabs",
			line:           "\tif (true) {}",
			cursorCol:      12, // Position between { and }
			indentType:     config.IndentTabs,
			indentWidth:    4,
			expectedColumn: 16, // 2 tabs = 16 visual columns
			expectedIndent: "\t\t",
			description:    "Should use tabs when configured for tabs",
		},
		{
			name:           "nested braces",
			line:           "\tif (true) { if (false) {} }",
			cursorCol:      25, // Position between inner { and }
			indentType:     config.IndentTabs,
			indentWidth:    4,
			expectedColumn: 16, // 2 tabs = 16 visual columns (base + 1)
			expectedIndent: "\t\t",
			description:    "Should add one indent level when between braces",
		},
		{
			name:           "cursor not between braces",
			line:           "if (true) { }",
			cursorCol:      10, // Position before {
			indentType:     config.IndentSpaces,
			indentWidth:    4,
			expectedColumn: 0,
			expectedIndent: "",
			description:    "Should not add extra indent when not between braces",
		},
		{
			name:           "cursor at end of line with open brace",
			line:           "void foo() {",
			cursorCol:      12, // At end of line
			indentType:     config.IndentSpaces,
			indentWidth:    4,
			expectedColumn: 4,
			expectedIndent: "    ",
			description:    "Should indent after open brace at end of line",
		},
		{
			name:           "struct with brace",
			line:           "struct Point {",
			cursorCol:      14, // At end of line
			indentType:     config.IndentSpaces,
			indentWidth:    2,
			expectedColumn: 2,
			expectedIndent: "  ",
			description:    "Should respect indent width setting",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a buffer with test content
			buffer := NewBuffer("", "test.go", tt.indentType, tt.indentWidth)
			
			// Calculate auto-indent
			result := buffer.CalculateAutoIndent(tt.line, tt.cursorCol)
			
			// Check column
			if result.Column != tt.expectedColumn {
				t.Errorf("%s: Column mismatch\nExpected: %d\nGot: %d\nDescription: %s",
					tt.name, tt.expectedColumn, result.Column, tt.description)
			}
			
			// Check indent string
			if result.IndentStr != tt.expectedIndent {
				t.Errorf("%s: IndentStr mismatch\nExpected: %q\nGot: %q\nDescription: %s",
					tt.name, tt.expectedIndent, result.IndentStr, tt.description)
			}
		})
	}
}