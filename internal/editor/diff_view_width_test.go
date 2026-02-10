package editor

import (
	"testing"
	"veda/internal/buffer"
	"veda/internal/config"
)

func TestDiffViewWindowWidth(t *testing.T) {
	// Create a narrow terminal (80 columns)
	width := 80
	height := 24
	te := NewTestEditor(width, height)

	// Set a reasonable minimum active window width for our test terminal
	te.Editor.minActiveWindowWidth = 30

	// Create some initial content for the diff
	leftContent := `Line 1
Line 2
Line 3
Line 4
Line 5`

	rightContent := `Line 1 modified
Line 2
New Line 3
Line 4
Line 5 modified`

	// Create buffers for diff view - these need special names to be recognized as diff buffers
	leftBuffer := buffer.NewBuffer(leftContent, "test.txt@HEAD", "test.txt@HEAD", config.IndentTabs, 4)
	rightBuffer := buffer.NewBuffer(rightContent, "test.txt@current", "test.txt@current", config.IndentTabs, 4)

	// Add a third buffer to have another window
	thirdBuffer := buffer.NewBuffer("Third buffer content", "third.txt", "third.txt", config.IndentTabs, 4)

	// Setup initial windows - start with three windows
	te.Editor.windows = []*Window{
		{
			Buffer:  thirdBuffer,
			Buffers: []*buffer.Buffer{thirdBuffer},
			X:       0,
			Width:   26, // Initial width for left window
		},
		{
			Buffer:  leftBuffer,
			Buffers: []*buffer.Buffer{leftBuffer},
			X:       27,
			Width:   26, // Initial width for middle window
		},
		{
			Buffer:  rightBuffer,
			Buffers: []*buffer.Buffer{rightBuffer},
			X:       54,
			Width:   26, // Initial width for right window
		},
	}
	te.Editor.activeWindow = 1 // Middle window is active

	// Calculate layout to apply window width changes
	te.Editor.recomputeWindowsLayout()

	// Test 1: Both diff view windows should have the same width
	diffWindow1 := te.Editor.windows[1]
	diffWindow2 := te.Editor.windows[2]

	if diffWindow1.Width != diffWindow2.Width {
		t.Errorf("Diff view windows should have equal width. Left diff: %d, Right diff: %d",
			diffWindow1.Width, diffWindow2.Width)
	}

	// Test 2: Both diff view windows should be wider than non-diff windows
	nonDiffWindow := te.Editor.windows[0]

	if diffWindow1.Width <= nonDiffWindow.Width {
		t.Errorf("Diff view windows should be wider than non-diff windows. Diff width: %d, Non-diff width: %d",
			diffWindow1.Width, nonDiffWindow.Width)
	}

	if diffWindow2.Width <= nonDiffWindow.Width {
		t.Errorf("Diff view windows should be wider than non-diff windows. Diff width: %d, Non-diff width: %d",
			diffWindow2.Width, nonDiffWindow.Width)
	}

	// Test 3: When switching to another buffer in one of the diff windows,
	// the diff view should end and widths should adjust

	// Switch to the third buffer in the left diff window
	te.Editor.windows[1].setActiveBuffer(thirdBuffer)
	te.Editor.windows[1].Buffers = []*buffer.Buffer{thirdBuffer}

	// Recalculate layout
	te.Editor.recomputeWindowsLayout()

	// Now the active window (index 1) should be wider, but not both former diff windows
	activeWindow := te.Editor.windows[te.Editor.activeWindow]
	for i, win := range te.Editor.windows {
		if i == te.Editor.activeWindow {
			// Active window should be wider
			if win.Width < 30 { // Expecting active window to be wider
				t.Errorf("Active window should be wider. Width: %d", win.Width)
			}
		} else {
			// Non-active windows should be narrower
			if win.Width >= activeWindow.Width {
				t.Errorf("Non-active window %d should be narrower than active window. Non-active: %d, Active: %d",
					i, win.Width, activeWindow.Width)
			}
		}
	}

	// Test 4: Verify minimum width constraints are respected
	minWidth := 20 // Assuming minimum width is 20
	for i, win := range te.Editor.windows {
		if win.Width < minWidth {
			t.Errorf("Window %d width (%d) is below minimum width (%d)", i, win.Width, minWidth)
		}
	}
}
