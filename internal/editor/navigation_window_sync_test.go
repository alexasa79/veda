package editor

import (
	"testing"
	"veda/internal/proxy"
)

func TestNavigateToLocationUpdatesWindowState(t *testing.T) {
	// Create test editor with multi-line content
	content := `Line 1
Line 2
Line 3
Line 4
Line 5
Line 6
Line 7
Line 8
Line 9
Line 10`

	te := NewTestEditorWithContent(80, 10, content)
	
	// Verify initial window state
	win := te.windows[te.activeWindow]
	if win.ViewLine != 0 || win.CursorLine != 0 || win.CursorX != 0 {
		t.Errorf("Initial window state incorrect: ViewLine=%d, CursorLine=%d, CursorX=%d",
			win.ViewLine, win.CursorLine, win.CursorX)
	}
	
	// Navigate to line 5, character 3
	te.navigateToLocation(5, 3)
	
	// Check that buffer state was updated
	bufferLine := te.CurrentBuffer().CursorIterator.LineNumber()
	if bufferLine != 5 {
		t.Errorf("Buffer cursor line not updated: expected 5, got %d", bufferLine)
	}
	if te.CurrentBuffer().CursorX != 3 {
		t.Errorf("Buffer cursor X not updated: expected 3, got %d", te.CurrentBuffer().CursorX)
	}
	
	// Check that window state was synchronized
	if win.CursorLine != 5 {
		t.Errorf("Window cursor line not synchronized: expected 5, got %d", win.CursorLine)
	}
	if win.CursorX != 3 {
		t.Errorf("Window cursor X not synchronized: expected 3, got %d", win.CursorX)
	}
	
	// ViewLine should be adjusted by scrollToCenter (line 5 centered in 10-line window)
	// scrollToCenter positions the cursor at the middle, so for line 5 in a 10-line window:
	// targetViewLine = 5 - (10/2) = 5 - 5 = 0, but we also need to account for getVisibleHeight
	// which likely subtracts space for status line, so visible height might be 9
	// Let's just check that ViewLine was updated from 0
	expectedViewLine := 1 // 5 - 9/2 = 5 - 4 = 1
	if win.ViewLine != expectedViewLine {
		t.Errorf("Window view line not updated correctly: expected %d, got %d", 
			expectedViewLine, win.ViewLine)
	}
}

func TestJumpToLocationUpdatesWindowState(t *testing.T) {
	// Create test editor with multi-line content
	content := `func main() {
    fmt.Println("Hello")
    doSomething()
}

func doSomething() {
    // implementation
}`

	te := NewTestEditorWithContent(80, 10, content)
	
	// Create a mock location (simulating LSP reference)
	location := proxy.Location{
		URI: "file://" + te.CurrentBuffer().FilePath,
		Range: proxy.Range{
			Start: proxy.Position{Line: 5, Character: 5}, // doSomething function
		},
	}
	
	// Jump to the location
	err := te.jumpToLocation(location)
	if err != nil {
		t.Fatalf("jumpToLocation failed: %v", err)
	}
	
	// Check window state synchronization
	win := te.windows[te.activeWindow]
	if win.CursorLine != 5 {
		t.Errorf("Window cursor line not synchronized after jump: expected 5, got %d", win.CursorLine)
	}
	if win.CursorX != 5 {
		t.Errorf("Window cursor X not synchronized after jump: expected 5, got %d", win.CursorX)
	}
}