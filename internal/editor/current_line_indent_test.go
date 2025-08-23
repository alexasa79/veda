package editor

import (
	"fmt"
	"strings"
	"testing"
	"veda/internal/buffer"
	"veda/internal/config"
)

func TestGetCurrentLineIndentInfo(t *testing.T) {
	tests := []struct {
		name           string
		content        string
		cursorLine     int
		indentType     config.IndentationType
		indentWidth    int
		expectedLevel  int
		expectedSpaces int
		expectedTabs   int
		isContinuation bool
	}{
		// Basic space indentation tests
		{
			name:           "no indentation",
			content:        "func main() {",
			cursorLine:     0,
			indentType:     config.IndentSpaces,
			indentWidth:    4,
			expectedLevel:  0,
			expectedSpaces: 0,
			expectedTabs:   0,
			isContinuation: false,
		},
		{
			name:           "one level with 4 spaces",
			content:        "    if true {",
			cursorLine:     0,
			indentType:     config.IndentSpaces,
			indentWidth:    4,
			expectedLevel:  1,
			expectedSpaces: 4,
			expectedTabs:   0,
			isContinuation: false,
		},
		{
			name:           "two levels with 4 spaces",
			content:        "        fmt.Println()",
			cursorLine:     0,
			indentType:     config.IndentSpaces,
			indentWidth:    4,
			expectedLevel:  2,
			expectedSpaces: 8,
			expectedTabs:   0,
			isContinuation: false,
		},
		{
			name:           "continuation line - 6 spaces with 4-space indent",
			content:        "      continued line",
			cursorLine:     0,
			indentType:     config.IndentSpaces,
			indentWidth:    4,
			expectedLevel:  1, // 6 / 4 = 1 (remainder 2)
			expectedSpaces: 6,
			expectedTabs:   0,
			isContinuation: true, // Not a multiple of 4
		},
		{
			name:           "one level with 2 spaces",
			content:        "  if true {",
			cursorLine:     0,
			indentType:     config.IndentSpaces,
			indentWidth:    2,
			expectedLevel:  1,
			expectedSpaces: 2,
			expectedTabs:   0,
			isContinuation: false,
		},
		{
			name:           "continuation line - 3 spaces with 2-space indent",
			content:        "   continued",
			cursorLine:     0,
			indentType:     config.IndentSpaces,
			indentWidth:    2,
			expectedLevel:  1, // 3 / 2 = 1 (remainder 1)
			expectedSpaces: 3,
			expectedTabs:   0,
			isContinuation: true,
		},
		// Tab indentation tests
		{
			name:           "one tab",
			content:        "\tif true {",
			cursorLine:     0,
			indentType:     config.IndentTabs,
			indentWidth:    1,
			expectedLevel:  1,
			expectedSpaces: 0,
			expectedTabs:   1,
			isContinuation: false,
		},
		{
			name:           "two tabs",
			content:        "\t\tfmt.Println()",
			cursorLine:     0,
			indentType:     config.IndentTabs,
			indentWidth:    1,
			expectedLevel:  2,
			expectedSpaces: 0,
			expectedTabs:   2,
			isContinuation: false,
		},
		{
			name:           "tab with spaces - continuation",
			content:        "\t  continued",
			cursorLine:     0,
			indentType:     config.IndentTabs,
			indentWidth:    1,
			expectedLevel:  1,
			expectedSpaces: 2,
			expectedTabs:   1,
			isContinuation: true, // Mixed tabs and spaces
		},
		// Mixed indentation (always continuation)
		{
			name:           "spaces in tab file",
			content:        "    code",
			cursorLine:     0,
			indentType:     config.IndentTabs,
			indentWidth:    1,
			expectedLevel:  0, // No tabs
			expectedSpaces: 4,
			expectedTabs:   0,
			isContinuation: true, // Spaces in a tab file
		},
		{
			name:           "tabs in space file",
			content:        "\t\tcode",
			cursorLine:     0,
			indentType:     config.IndentSpaces,
			indentWidth:    4,
			expectedLevel:  0, // No spaces to count
			expectedSpaces: 0,
			expectedTabs:   2,
			isContinuation: true, // Tabs in a space file
		},
		// Multi-line tests
		{
			name: "check specific line in multi-line content",
			content: strings.Join([]string{
				"func main() {",
				"    if true {",
				"        fmt.Println(\"Hello\")",
				"          fmt.Println(\"Continued\")", // 10 spaces - continuation
				"    }",
				"}",
			}, "\n"),
			cursorLine:     3, // The continuation line
			indentType:     config.IndentSpaces,
			indentWidth:    4,
			expectedLevel:  2, // 10 / 4 = 2 (remainder 2)
			expectedSpaces: 10,
			expectedTabs:   0,
			isContinuation: true,
		},
		// Empty and whitespace-only lines
		{
			name:           "empty line",
			content:        "",
			cursorLine:     0,
			indentType:     config.IndentSpaces,
			indentWidth:    4,
			expectedLevel:  0,
			expectedSpaces: 0,
			expectedTabs:   0,
			isContinuation: false,
		},
		{
			name:           "whitespace-only line with spaces",
			content:        "    ",
			cursorLine:     0,
			indentType:     config.IndentSpaces,
			indentWidth:    4,
			expectedLevel:  1,
			expectedSpaces: 4,
			expectedTabs:   0,
			isContinuation: false,
		},
		{
			name:           "whitespace-only line with tabs",
			content:        "\t\t",
			cursorLine:     0,
			indentType:     config.IndentTabs,
			indentWidth:    1,
			expectedLevel:  2,
			expectedSpaces: 0,
			expectedTabs:   2,
			isContinuation: false,
		},
		// Real-world continuation examples
		{
			name: "C++ function parameters",
			content: strings.Join([]string{
				"void processData(const std::string& input,",
				"                 const std::string& output,",
				"                 int flags) {",
				"    doSomething();",
			}, "\n"),
			cursorLine:     1, // The continuation line
			indentType:     config.IndentSpaces,
			indentWidth:    4,
			expectedLevel:  4, // 17 / 4 = 4 (remainder 1)
			expectedSpaces: 17,
			expectedTabs:   0,
			isContinuation: true,
		},
		{
			name: "Python multi-line statement",
			content: strings.Join([]string{
				"def calculate(a, b, c,",
				"              d, e, f):",
				"    return a + b",
			}, "\n"),
			cursorLine:     1,
			indentType:     config.IndentSpaces,
			indentWidth:    4,
			expectedLevel:  3, // 14 / 4 = 3 (remainder 2)
			expectedSpaces: 14,
			expectedTabs:   0,
			isContinuation: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			te := NewTestEditorWithContent(80, 24, tt.content)
			te.CurrentBuffer().Indentation.Type = tt.indentType
			te.CurrentBuffer().Indentation.Width = tt.indentWidth

			// Position cursor on the line to test
			lines := strings.Split(tt.content, "\n")
			if tt.cursorLine < len(lines) {
				te.SetCursorPosition(0, tt.cursorLine)
			}

			// Get the line being tested
			line := ""
			if tt.cursorLine < len(lines) {
				line = lines[tt.cursorLine]
			}

			info := te.CurrentBuffer().GetLineIndentInfo(line)

			if info.Level != tt.expectedLevel {
				t.Errorf("Expected level %d, got %d", tt.expectedLevel, info.Level)
			}

			if info.Spaces != tt.expectedSpaces {
				t.Errorf("Expected %d spaces, got %d", tt.expectedSpaces, info.Spaces)
			}

			if info.Tabs != tt.expectedTabs {
				t.Errorf("Expected %d tabs, got %d", tt.expectedTabs, info.Tabs)
			}

			if info.IsContinuation != tt.isContinuation {
				t.Errorf("Expected IsContinuation=%v, got %v", tt.isContinuation, info.IsContinuation)
			}

			// Verify the actual prefix is correct
			expectedPrefix := ""
			for i := 0; i < info.Tabs; i++ {
				expectedPrefix += "\t"
			}
			for i := 0; i < info.Spaces; i++ {
				expectedPrefix += " "
			}

			if info.ActualPrefix != expectedPrefix {
				t.Errorf("Expected prefix %q, got %q", expectedPrefix, info.ActualPrefix)
			}
		})
	}
}

func TestGetCurrentLineIndentInfoWithCursor(t *testing.T) {
	// Test that getCurrentLineIndentInfo uses the cursor position correctly
	content := strings.Join([]string{
		"no indent",
		"    one level",
		"        two levels",
		"      continuation",
	}, "\n")

	te := NewTestEditorWithContent(80, 24, content)
	te.CurrentBuffer().Indentation.Type = config.IndentSpaces
	te.CurrentBuffer().Indentation.Width = 4

	// Test each line
	testCases := []struct {
		cursorY        int
		expectedLevel  int
		isContinuation bool
	}{
		{0, 0, false}, // "no indent"
		{1, 1, false}, // "    one level"
		{2, 2, false}, // "        two levels"
		{3, 1, true},  // "      continuation" (6 spaces)
	}

	for _, tc := range testCases {
		te.SetCursorPosition(0, tc.cursorY)
		info := te.getCurrentLineIndentInfo(te.CurrentBuffer().CursorIterator)

		if info.Level != tc.expectedLevel {
			t.Errorf("Line %d: expected level %d, got %d",
				tc.cursorY, tc.expectedLevel, info.Level)
		}

		if info.IsContinuation != tc.isContinuation {
			t.Errorf("Line %d: expected IsContinuation=%v, got %v",
				tc.cursorY, tc.isContinuation, info.IsContinuation)
		}
	}
}

func TestIndentInfoFormatting(t *testing.T) {
	// Test how indent info should be formatted for status bar
	tests := []struct {
		name            string
		info            buffer.LineIndentInfo
		fileIndentType  config.IndentationType
		fileIndentWidth int
		expectedFormat  string
	}{
		{
			name: "normal indent level 1",
			info: buffer.LineIndentInfo{
				Level:          1,
				Spaces:         4,
				IsContinuation: false,
			},
			fileIndentType:  config.IndentSpaces,
			fileIndentWidth: 4,
			expectedFormat:  "L1",
		},
		{
			name: "normal indent level 2",
			info: buffer.LineIndentInfo{
				Level:          2,
				Spaces:         8,
				IsContinuation: false,
			},
			fileIndentType:  config.IndentSpaces,
			fileIndentWidth: 4,
			expectedFormat:  "L2",
		},
		{
			name: "continuation indent",
			info: buffer.LineIndentInfo{
				Level:          1,
				Spaces:         6,
				IsContinuation: true,
			},
			fileIndentType:  config.IndentSpaces,
			fileIndentWidth: 4,
			expectedFormat:  "L1+C",
		},
		{
			name: "no indent",
			info: buffer.LineIndentInfo{
				Level:          0,
				Spaces:         0,
				IsContinuation: false,
			},
			fileIndentType:  config.IndentSpaces,
			fileIndentWidth: 4,
			expectedFormat:  "L0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This tests the expected format - we'll implement the actual
			// formatting function when we add it to the status bar
			format := formatIndentInfo(tt.info)
			if format != tt.expectedFormat {
				t.Errorf("Expected format %q, got %q", tt.expectedFormat, format)
			}
		})
	}
}

// Helper function to format indent info for status bar
// This will be moved to the main code when we implement the feature
func formatIndentInfo(info buffer.LineIndentInfo) string {
	format := fmt.Sprintf("L%d", info.Level)
	if info.IsContinuation {
		format += "+C"
	}
	return format
}
