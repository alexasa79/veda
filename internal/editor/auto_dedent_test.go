package editor

import (
	"testing"
)

func TestAutoDedentClosingBrace(t *testing.T) {
	tests := []struct {
		name            string
		initialContent  string
		typedChars      string
		expectedContent string
		description     string
	}{
		{
			name:            "simple if statement",
			initialContent:  "",
			typedChars:      "if (true) {\nreturn;\n}",
			expectedContent: "if (true) {\n    return;\n}",
			description:     "Closing brace should auto-dedent to match if statement indentation",
		},
		{
			name:            "nested blocks",
			initialContent:  "",
			typedChars:      "void foo() {\nif (x) {\ny++;\n}\n}",
			expectedContent: "void foo() {\n    if (x) {\n        y++;\n    }\n}",
			description:     "Nested closing braces should dedent to their matching open brace level",
		},
		{
			name:            "indented function",
			initialContent:  "",
			typedChars:      "    void bar() {\nx = 1;\n}",
			expectedContent: "    void bar() {\n        x = 1;\n    }",
			description:     "Closing brace should dedent even when function is indented",
		},
		{
			name:            "multiple closing braces",
			initialContent:  "",
			typedChars:      "if (a) {\nif (b) {\nc();\n}\n}",
			expectedContent: "if (a) {\n    if (b) {\n        c();\n    }\n}",
			description:     "Each closing brace should dedent to its matching level",
		},
		{
			name:            "brace with text before it",
			initialContent:  "",
			typedChars:      "if (true) {\nreturn 0;}",
			expectedContent: "if (true) {\n    return 0;}",
			description:     "Should not dedent if there's text before the closing brace",
		},
		{
			name:            "unmatched closing brace",
			initialContent:  "",
			typedChars:      "    }",
			expectedContent: "    }",
			description:     "Should not dedent if no matching open brace is found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test editor
			te := NewTestEditorWithContent(80, 24, tt.initialContent)

			// Enter insert mode
			te.ExecuteCommand("enter-insert-mode")

			// Type the characters
			for _, ch := range tt.typedChars {
				if ch == '\n' {
					te.ExecuteCommand("insert-newline")
				} else {
					te.InsertChar(ch)
				}
			}

			// Exit insert mode
			te.ExecuteCommand("exit-insert-mode")

			// Check the result
			actual := te.buffer.Rope.String()
			if actual != tt.expectedContent {
				t.Errorf("%s\nExpected:\n%q\nGot:\n%q\nDescription: %s",
					tt.name, tt.expectedContent, actual, tt.description)
			}
		})
	}
}

func TestAutoDedentWithTabs(t *testing.T) {
	// Create test editor with tab indentation
	te := NewTestEditorWithContent(80, 24, "")

	// Set indentation to tabs
	te.buffer.Indentation.Type = 1 // config.IndentTabs
	te.buffer.Indentation.Width = 4

	// Enter insert mode
	te.ExecuteCommand("enter-insert-mode")

	// Type code (auto-indent will add tabs)
	text := "if (true) {\nreturn;\n}"
	for _, ch := range text {
		if ch == '\n' {
			te.ExecuteCommand("insert-newline")
		} else {
			te.InsertChar(ch)
		}
	}

	// Exit insert mode
	te.ExecuteCommand("exit-insert-mode")

	// Check the result - closing brace should be dedented
	expected := "if (true) {\n\treturn;\n}"
	actual := te.buffer.Rope.String()

	if actual != expected {
		t.Errorf("Tab indentation test failed\nExpected:\n%q\nGot:\n%q", expected, actual)
	}
}

func TestAutoDedentComplexNesting(t *testing.T) {
	// Create test editor
	te := NewTestEditorWithContent(80, 24, "")

	// Enter insert mode
	te.ExecuteCommand("enter-insert-mode")

	// Type complex nested structure (let auto-indent handle indentation)
	text := "class Foo {\nvoid bar() {\nif (x) {\nwhile (y) {\nz++;\n}\n}\n}\n}"
	for _, ch := range text {
		if ch == '\n' {
			te.ExecuteCommand("insert-newline")
		} else {
			te.InsertChar(ch)
		}
	}

	// Exit insert mode
	te.ExecuteCommand("exit-insert-mode")

	// Expected result with proper dedenting
	expected := "class Foo {\n    void bar() {\n        if (x) {\n            while (y) {\n                z++;\n            }\n        }\n    }\n}"
	actual := te.buffer.Rope.String()

	if actual != expected {
		t.Errorf("Complex nesting test failed\nExpected:\n%s\nGot:\n%s", expected, actual)
	}
}
