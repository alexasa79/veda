package editor

import (
	"testing"
)

func TestArgumentListDetection(t *testing.T) {
	tests := []struct {
		name              string
		content           string
		cursorMovements   []string
		expectedInArgList bool
	}{
		{
			name:              "Empty function call",
			content:           "fmt.Printf()",
			cursorMovements:   []string{"end-of-line", "left"}, // First go to end, then move cursor between parentheses
			expectedInArgList: true,
		},
		{
			name:              "Function call with string",
			content:           `fmt.Printf("")`,
			cursorMovements:   []string{"end-of-line", "left", "left"}, // First go to end, then move cursor between quotes
			expectedInArgList: true,
		},
		{
			name:              "Outside function call",
			content:           "fmt.Printf()",
			cursorMovements:   []string{"end-of-line"}, // Explicitly go to end
			expectedInArgList: false,
		},
		{
			name:              "Before function call",
			content:           "fmt.Printf()",
			cursorMovements:   []string{"beginning-of-line"}, // Cursor at beginning
			expectedInArgList: false,
		},
		{
			name:              "Multiple arguments",
			content:           `fmt.Printf("hello", world)`,
			cursorMovements:   []string{"beginning-of-line", "right", "right", "right", "right", "right", "right", "right", "right", "right", "right", "right", "right"}, // Move to after "hello"
			expectedInArgList: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			te := NewTestEditorWithContent(80, 24, tt.content)
			
			// Set up the file so syntax highlighting works
			te.buffer.FilePath = "test.go"
			te.highlighter.SetLanguageFromFile("test.go")
			te.updateSyntaxHighlighting()
			
			// Execute cursor movements
			for _, cmd := range tt.cursorMovements {
				te.ExecuteCommand(cmd)
			}
			
			// Force update of cursor state
			te.updateCursorState()
			
			// Check if InArgumentList matches expected
			if te.buffer.InArgumentList != tt.expectedInArgList {
				// Debug: print the syntax tree
				if te.highlighter.GetTree() != nil {
					root := te.highlighter.GetTree().RootNode()
					t.Logf("Tree exists, root type: %s", root.Type())
				} else {
					t.Logf("No syntax tree available")
				}
				t.Errorf("Expected InArgumentList=%v, but got %v\nContent: %s\nCursor at: (%d, %d)", 
					tt.expectedInArgList, te.buffer.InArgumentList, 
					tt.content, te.buffer.CursorX, te.buffer.CursorIterator.LineNumber())
			}
		})
	}
}

func TestArgumentListStatusBar(t *testing.T) {
	te := NewTestEditorWithContent(80, 24, "fmt.Printf()")
	
	// Set up the file so syntax highlighting works
	te.buffer.FilePath = "test.go"
	te.highlighter.SetLanguageFromFile("test.go")
	te.updateSyntaxHighlighting()
	
	// Move cursor inside parentheses
	te.ExecuteCommand("left")
	te.updateCursorState()
	
	// Draw the status bar
	te.drawStatusBar()
	
	// Check if [args] appears in the status bar
	screenContent := te.GetScreenContent()
	statusBarY := te.getStatusBarY()
	if statusBarY >= 0 && statusBarY < len(screenContent) {
		statusLine := screenContent[statusBarY]
		if !contains(statusLine, "[args]") {
			t.Errorf("Expected status bar to contain '[args]', but got: %s", statusLine)
		}
	} else {
		t.Errorf("Invalid status bar Y position: %d", statusBarY)
	}
	
	// Move cursor outside parentheses
	te.ExecuteCommand("right")
	te.updateCursorState()
	
	// Draw the status bar again
	te.drawStatusBar()
	
	// Check that [args] no longer appears
	screenContent = te.GetScreenContent()
	if statusBarY >= 0 && statusBarY < len(screenContent) {
		statusLine := screenContent[statusBarY]
		if contains(statusLine, "[args]") {
			t.Errorf("Expected status bar NOT to contain '[args]', but got: %s", statusLine)
		}
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && containsAt(s, substr, 0)
}

func containsAt(s, substr string, start int) bool {
	for i := start; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}