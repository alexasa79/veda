package buffer

import (
	"testing"
	"veda/internal/config"
)

// TestEmptyLineIndentation tests that empty lines inside blocks get proper indentation
// based on the surrounding block context
func TestEmptyLineIndentation(t *testing.T) {
	tests := []struct {
		name           string
		content        string
		cursorLine     int
		expectedColumn int
		expectedIndent string
		description    string
	}{
		{
			name:           "empty line inside function block",
			content:        "int main()\n{\n\n}",
			cursorLine:     2, // Empty line inside the block
			expectedColumn: 4,
			expectedIndent: "    ",
			description:    "Empty line inside function should be indented to L1",
		},
		{
			name:           "empty line inside nested block",
			content:        "int main()\n{\n    if (1) {\n\n    }\n}",
			cursorLine:     3, // Empty line inside nested block
			expectedColumn: 8,
			expectedIndent: "        ",
			description:    "Empty line inside nested block should be indented to L2",
		},
		{
			name:           "empty line at top level",
			content:        "int main()\n\n{",
			cursorLine:     1, // Empty line at top level
			expectedColumn: 0,
			expectedIndent: "",
			description:    "Empty line at top level should have no indentation",
		},
		{
			name:           "empty line with tab indentation",
			content:        "func main() {\n\n}",
			cursorLine:     1, // Empty line inside block
			expectedColumn: 8, // One tab = 8 visual columns
			expectedIndent: "\t",
			description:    "Empty line inside block with tab indentation should use tabs",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create buffer with appropriate indentation settings
			indentType := config.IndentSpaces
			if tt.name == "empty line with tab indentation" {
				indentType = config.IndentTabs
			}

			buffer := NewBuffer(tt.content, "test.c", indentType, 4)

			// Set cursor to the specified line for context analysis
			buffer.CursorIterator = NewBufferIteratorAtLine(buffer, tt.cursorLine)

			// Get the empty line content (should be empty)
			emptyLine := ""
			if !buffer.CursorIterator.IsEOF() {
				emptyLine = buffer.CursorIterator.Line()
			}

			// Calculate auto-indent for the empty line
			result := buffer.CalculateAutoIndent(emptyLine, 0)

			// Verify the result
			if result.Column != tt.expectedColumn {
				t.Errorf("%s: Expected column %d, got %d",
					tt.description, tt.expectedColumn, result.Column)
			}

			if result.IndentStr != tt.expectedIndent {
				t.Errorf("%s: Expected indent string %q, got %q",
					tt.description, tt.expectedIndent, result.IndentStr)
			}
		})
	}
}

// TestEmptyLineIndentationEdgeCases tests edge cases for empty line indentation
func TestEmptyLineIndentationEdgeCases(t *testing.T) {
	tests := []struct {
		name           string
		content        string
		cursorLine     int
		expectedColumn int
		description    string
	}{
		{
			name:           "empty buffer",
			content:        "",
			cursorLine:     0,
			expectedColumn: 0,
			description:    "Empty buffer should have no indentation",
		},
		{
			name:           "multiple consecutive empty lines",
			content:        "int main()\n{\n\n\n\n}",
			cursorLine:     2, // First empty line
			expectedColumn: 4,
			description:    "First of multiple empty lines should be indented",
		},
		{
			name:           "empty line after closing brace",
			content:        "int main()\n{\n    printf(\"test\");\n}\n\nint other()",
			cursorLine:     4, // Empty line after closing brace
			expectedColumn: 0,
			description:    "Empty line after closing brace should have no indentation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buffer := NewBuffer(tt.content, "test.c", config.IndentSpaces, 4)

			if tt.content != "" {
				buffer.CursorIterator = NewBufferIteratorAtLine(buffer, tt.cursorLine)
			}

			// Calculate auto-indent for empty line
			result := buffer.CalculateAutoIndent("", 0)

			if result.Column != tt.expectedColumn {
				t.Errorf("%s: Expected column %d, got %d",
					tt.description, tt.expectedColumn, result.Column)
			}
		})
	}
}
