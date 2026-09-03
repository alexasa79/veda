package editor

import (
	"fmt"
	"strings"
	"testing"
	"unicode"
	"veda/internal/events"
	"veda/internal/ui"
)

// TestLineWrapping_Basic tests basic line wrapping at screen width
func TestLineWrapping_Basic(t *testing.T) {
	// Create editor with 20 columns width
	te := NewTestEditorWithContent(20, 10, "")

	// Enter text that should wrap
	te.ExecuteCommand("insert")
	text := "This is a very long line that should wrap"
	for _, ch := range text {
		te.InsertChar(ch)
	}
	te.PressKey(events.KeyEscape, 0)

	// Render and check output
	te.renderPane()
	screen := te.mockScreen.GetContent()

	// Line 1 should show: "1 This is a very"
	// Line 2 should show: "  long line that"
	// Line 3 should show: "  should wrap"

	// Check line numbers (only on first visual line)
	AssertScreenContent(t, screen, 0, 0, "1")
	AssertScreenContent(t, screen, 0, 1, " ") // No line number on wrapped lines
	AssertScreenContent(t, screen, 0, 2, " ")

	// Build expected lines with new gutter width (line number + 2-indicator column + 1 space)
	digitsWidth := te.CurrentBuffer().LineNumberWidth - 3
	if digitsWidth < 1 {
		digitsWidth = 1
	}
	gutter1 := strings.Repeat(" ", te.CurrentBuffer().LineNumberWidth)
	// Compose numeric gutter directly
	numGutter := func(n int) string {
		digits := fmt.Sprintf("%*d", digitsWidth, n)
		padding := te.CurrentBuffer().LineNumberWidth - digitsWidth
		if padding < 0 {
			padding = 0
		}
		return digits + strings.Repeat(" ", padding)
	}
	// For single-digit, numGutter produces digit + 3 spaces
	AssertScreenLine(t, screen, 0, numGutter(1)+"This is a very")
	AssertScreenLine(t, screen, 1, gutter1+"long line that")
	AssertScreenLine(t, screen, 2, gutter1+"should wrap")
}

// TestLineWrapping_WordBoundary tests wrapping happens at word boundaries
func TestLineWrapping_WordBoundary(t *testing.T) {
	// Create editor with 15 columns
	te := NewTestEditorWithContent(15, 10, "")

	// Enter text where wrapping should happen at word boundary
	te.ExecuteCommand("insert")
	text := "Hello world programming"
	for _, ch := range text {
		te.InsertChar(ch)
	}
	te.PressKey(events.KeyEscape, 0)

	te.renderPane()
	screen := te.mockScreen.GetContent()

	// Should wrap at word boundaries, not in the middle of words
	// Compose expected with dynamic gutter
	digitsWidth := te.CurrentBuffer().LineNumberWidth - 3
	if digitsWidth < 1 {
		digitsWidth = 1
	}
	numGutter := func(n int) string {
		digits := fmt.Sprintf("%*d", digitsWidth, n)
		padding := te.CurrentBuffer().LineNumberWidth - digitsWidth
		if padding < 0 {
			padding = 0
		}
		return digits + strings.Repeat(" ", padding)
	}
	wrappedGutter := strings.Repeat(" ", te.CurrentBuffer().LineNumberWidth)
	AssertScreenLine(t, screen, 0, numGutter(1)+"Hello world")
	AssertScreenLine(t, screen, 1, wrappedGutter+"programming")
}

// TestLineWrapping_LongWord tests wrapping of words longer than screen width
func TestLineWrapping_LongWord(t *testing.T) {
	// Create editor with 10 columns
	te := NewTestEditorWithContent(10, 10, "")

	// Enter a very long word
	te.ExecuteCommand("insert")
	text := "supercalifragilisticexpialidocious"
	for _, ch := range text {
		te.InsertChar(ch)
	}
	te.PressKey(events.KeyEscape, 0)

	te.renderPane()
	screen := te.mockScreen.GetContent()

	// Long word should be broken at screen width
	// With line number gutter width dynamic
	digitsWidth := te.CurrentBuffer().LineNumberWidth - 3
	if digitsWidth < 1 {
		digitsWidth = 1
	}
	numGutter := func(n int) string {
		digits := fmt.Sprintf("%*d", digitsWidth, n)
		padding := te.CurrentBuffer().LineNumberWidth - digitsWidth
		if padding < 0 {
			padding = 0
		}
		return digits + strings.Repeat(" ", padding)
	}
	wrappedGutter := strings.Repeat(" ", te.CurrentBuffer().LineNumberWidth)
	AssertScreenLine(t, screen, 0, numGutter(1)+"superc") // length depends on width; keep prefix check minimal
	// Since AssertScreenLine checks full line, compute expected exact slices
	// For width 10 and gutter 4 -> 6 chars fit on first line
	expectedFirst := numGutter(1) + "superc"
	AssertScreenLine(t, screen, 0, expectedFirst)
	expectedSecond := wrappedGutter + "alifra"
	AssertScreenLine(t, screen, 1, expectedSecond)
}

// TestLineWrapping_MultipleLines tests wrapping with multiple logical lines
func TestLineWrapping_MultipleLines(t *testing.T) {
	te := NewTestEditorWithContent(20, 10, "")

	// Enter multiple lines, some wrapping
	te.ExecuteCommand("insert")
	lines := []string{
		"Short line",
		"This is a much longer line that wraps",
		"Another short",
	}

	for i, line := range lines {
		for _, ch := range line {
			te.InsertChar(ch)
		}
		if i < len(lines)-1 {
			te.InsertChar('\n')
		}
	}
	te.PressKey(events.KeyEscape, 0)

	te.renderPane()
	screen := te.mockScreen.GetContent()

	// Expected using dynamic gutter
	digitsWidth := te.CurrentBuffer().LineNumberWidth - 3
	if digitsWidth < 1 {
		digitsWidth = 1
	}
	numGutter := func(n int) string {
		digits := fmt.Sprintf("%*d", digitsWidth, n)
		padding := te.CurrentBuffer().LineNumberWidth - digitsWidth
		if padding < 0 {
			padding = 0
		}
		return digits + strings.Repeat(" ", padding)
	}

	displayNumber := func(logicalLine int) int {
		return logicalLine + 1
	}
	wrappedGutter := strings.Repeat(" ", te.CurrentBuffer().LineNumberWidth)
	AssertScreenLine(t, screen, 0, numGutter(displayNumber(0))+"Short line")
	AssertScreenLine(t, screen, 1, numGutter(displayNumber(1))+"This is a much")
	AssertScreenLine(t, screen, 2, wrappedGutter+"longer line that")
	AssertScreenLine(t, screen, 3, wrappedGutter+"wraps")
	AssertScreenLine(t, screen, 4, numGutter(displayNumber(2))+"Another short")

	// Check rendered line numbers using helper
	if got := extractRenderedLineNumber(te, 0); got != fmt.Sprintf("%d", displayNumber(0)) {
		t.Fatalf("line 0 number mismatch: want %d, got %q", displayNumber(0), got)
	}
	if got := extractRenderedLineNumber(te, 1); got != fmt.Sprintf("%d", displayNumber(1)) {
		t.Fatalf("line 1 number mismatch: want %d, got %q", displayNumber(1), got)
	}
	if got := extractRenderedLineNumber(te, 4); got != fmt.Sprintf("%d", displayNumber(2)) {
		t.Fatalf("line 4 number mismatch: want %d, got %q", displayNumber(2), got)
	}
}

// TestLineWrapping_CursorNavigation tests cursor movement with wrapped lines
func TestLineWrapping_CursorNavigation(t *testing.T) {
	te := NewTestEditorWithContent(20, 10, "This is a very long line that should wrap")

	// Verify initial setup
	content := te.GetBufferContent()
	t.Logf("Buffer content: %q", content)
	t.Logf("Line count: %d", te.CurrentBuffer().LineCount)
	logX, logY := te.GetCursorPosition()
	t.Logf("Initial logical position: (%d,%d)", logX, logY)
	line, col := te.GetVisualCursorPosition()
	t.Logf("Initial visual position: (%d,%d)", line, col)

	// Test moving down through wrapped lines
	te.ExecuteCommand("beginning_of_line")

	// Move down should go to next visual line (wrapped portion)
	te.ExecuteCommand("down")

	// Debug: check wrapped lines
	wrappedLines := te.CalculateWrappedLines()
	t.Logf("Total wrapped lines: %d", len(wrappedLines))
	width, _ := te.screen.Size()
	t.Logf("Screen width: %d, line number width: %d", width, te.CurrentBuffer().LineNumberWidth)
	t.Logf("ViewIterator line: %d", te.CurrentBuffer().ViewIterator.LineNumber())
	for i, wl := range wrappedLines {
		t.Logf("  [%d] logical=%d, start=%d, end=%d, text=%q",
			i, wl.LogicalLineNum, wl.StartChar, wl.EndChar, wl.Text)
	}

	line, col = te.GetVisualCursorPosition()
	if line != 1 {
		logX, logY := te.GetCursorPosition()
		t.Errorf("Expected cursor on visual line 1, got %d (visual col %d) (logical: %d,%d)", line, col, logX, logY)

		// Debug GetCursorVisualPosition
		for i, wl := range wrappedLines {
			t.Logf("  Checking wrapped line %d: logical=%d, start=%d, end=%d", i, wl.LogicalLineNum, wl.StartChar, wl.EndChar)
			if wl.LogicalLineNum == logY {
				if logX >= wl.StartChar && logX < wl.EndChar {
					t.Logf("    -> Cursor IS in this wrapped line (< check)")
				} else if logX == wl.EndChar {
					t.Logf("    -> Cursor is AT END of this wrapped line")
				}
			}
		}

		// Call GetVisualCursorPosition directly to understand the issue
		directWrappedLines := te.CalculateWrappedLines()
		t.Logf("  Direct calculation: %d wrapped lines", len(directWrappedLines))
		if len(directWrappedLines) != len(wrappedLines) {
			t.Logf("  WARNING: wrapped lines count mismatch!")
		}
	}

	// Move down again
	te.ExecuteCommand("down")
	line, _ = te.GetVisualCursorPosition()
	if line != 2 {
		t.Errorf("Expected cursor on visual line 2, got %d", line)
	}

	// Move up should go back
	te.ExecuteCommand("up")
	line, _ = te.GetVisualCursorPosition()
	if line != 1 {
		t.Errorf("Expected cursor on visual line 1 after up, got %d", line)
	}

	// Move to beginning, ensure we're on first visual line, then go down to next wrapped segment
	te.ExecuteCommand("beginning_of_line")
	te.ExecuteCommand("up")
	te.ExecuteCommand("down")
	// Note: GetVisualCursorPosition returns (visualLine, visualCol)
	y, x := te.GetVisualCursorPosition()
	expectedStartX := te.CurrentBuffer().LineNumberWidth
	if y != 1 || x != expectedStartX {
		t.Errorf("Expected cursor at start of next wrapped line (%d,1), got (%d,%d)", expectedStartX, x, y)
	}

	// Moving up should return to first visual line
	te.ExecuteCommand("up")
	y, _ = te.GetVisualCursorPosition()
	if y != 0 {
		t.Errorf("Expected cursor back on first visual line, got visual line %d", y)
	}
}

// TestLineWrapping_InsertMode tests inserting text with line wrapping
func TestLineWrapping_InsertMode(t *testing.T) {
	te := NewTestEditorWithContent(20, 10, "Hello world")

	// Move to end and continue typing
	te.ExecuteCommand("end_of_line")
	te.ExecuteCommand("insert")

	// Add text that will cause wrapping
	text := " and more text here"
	for _, ch := range text {
		te.InsertChar(ch)
	}

	te.renderPane()
	_ = te.mockScreen.GetContent()

	// Should wrap appropriately. Validate via wrapped segments rather than exact render text.
	wrapped := te.CalculateWrappedLines()
	// Collect wrapped segments for the first logical line
	var segs []WrappedLine
	for _, wl := range wrapped {
		if wl.LogicalLineNum == 0 {
			segs = append(segs, wl)
		} else if len(segs) > 0 {
			break
		}
	}
	if len(segs) < 2 {
		t.Fatalf("Expected at least 2 wrapped segments, got %d", len(segs))
	}
	if !segs[0].IsFirstLine {
		t.Errorf("First segment should be marked IsFirstLine")
	}
	if strings.HasPrefix(segs[1].Text, " ") {
		t.Errorf("Second segment should not start with a space: %q", segs[1].Text)
	}
	// Ensure both key substrings are present across segments
	joined := strings.Join([]string{segs[0].Text, segs[1].Text}, "")
	if !strings.Contains(joined, "Hello world") || !strings.Contains(joined, "more text here") {
		t.Errorf("Wrapped segments do not contain expected content: %q", joined)
	}

	// Cursor should be on a wrapped visual line
	y, _ := te.GetVisualCursorPosition()
	if y < 1 {
		t.Errorf("Expected visual cursor on wrapped line, got visual line %d", y)
	}
}

// TestLineWrapping_DeleteAcrossWrap tests deleting text across wrap boundary
func TestLineWrapping_DeleteAcrossWrap(t *testing.T) {
	te := NewTestEditorWithContent(20, 10, "This is a very long line that should wrap")

	// Position cursor at end of first visual line
	te.ExecuteCommand("beginning_of_line")
	for i := 0; i < 15; i++ { // Move to near wrap point
		te.ExecuteCommand("right")
	}

	// Delete forward should delete across wrap
	te.ExecuteCommand("delete_char")

	te.renderPane()
	// We don't use screen here, just checking content

	// Check that deletion worked properly
	content := te.GetBufferContent()
	if strings.Contains(content, "This is a very ong") {
		t.Errorf("Delete didn't work properly, got: %s", content)
	}
}

// TestLineWrapping_Scrolling tests scrolling with wrapped lines
func TestLineWrapping_Scrolling(t *testing.T) {
	// Small screen height to test scrolling
	te := NewTestEditorWithContent(20, 5, "")

	// Add multiple lines that wrap
	te.ExecuteCommand("insert")
	for i := 1; i <= 10; i++ {
		text := "This is line number " + string('0'+byte(i)) + " with some long text that wraps"
		for _, ch := range text {
			te.InsertChar(ch)
		}
		if i < 10 {
			te.InsertChar('\n')
		}
	}
	te.PressKey(events.KeyEscape, 0)

	// Scroll down and verify wrapped lines scroll correctly
	te.ExecuteCommand("page_down")
	te.renderPane()

	// Verify that wrapped lines maintain their visual structure during scrolling
	screen := te.mockScreen.GetContent()
	wrapped := te.CalculateWrappedLines()
	if len(wrapped) == 0 {
		t.Fatal("No wrapped lines calculated after scrolling")
	}
	first := wrapped[0]
	// Examine the gutter region of the first visible screen line
	gutter := GetScreenLine(screen, 0)
	if len(gutter) > te.CurrentBuffer().LineNumberWidth {
		gutter = gutter[:te.CurrentBuffer().LineNumberWidth]
	}
	hasDigit := false
	for _, r := range gutter {
		if unicode.IsDigit(r) {
			hasDigit = true
			break
		}
	}
	if first.IsFirstLine {
		if !hasDigit {
			t.Error("Expected a digit in the gutter for the first visible line")
		}
	} else {
		if hasDigit {
			t.Error("Expected blank gutter (no digits) for continuation wrapped line")
		}
	}
}

// Helper function to get a screen line as string
func GetScreenLine(screen map[int]map[int]Cell, y int) string {
	row, ok := screen[y]
	if !ok {
		return ""
	}

	var line strings.Builder
	maxX := 0
	for x := range row {
		if x > maxX {
			maxX = x
		}
	}

	for x := 0; x <= maxX; x++ {
		if cell, ok := row[x]; ok {
			line.WriteRune(cell.Rune)
		} else {
			line.WriteRune(' ')
		}
	}

	return strings.TrimRight(line.String(), " ")
}

// Helper to assert screen content at specific position
func AssertScreenContent(t *testing.T, screen map[int]map[int]Cell, x, y int, expected string) {
	t.Helper()

	row, ok := screen[y]
	if !ok {
		t.Errorf("No content at row %d", y)
		return
	}

	var actual strings.Builder
	for i := 0; i < len(expected); i++ {
		if cell, ok := row[x+i]; ok {
			actual.WriteRune(cell.Rune)
		} else {
			actual.WriteRune(' ')
		}
	}

	if actual.String() != expected {
		t.Errorf("At (%d,%d): expected %q, got %q", x, y, expected, actual.String())
	}
}

// Helper to assert entire screen line content
func AssertScreenLine(t *testing.T, screen map[int]map[int]Cell, y int, expected string) {
	t.Helper()

	actual := GetScreenLine(screen, y)
	if actual != expected {
		t.Errorf("Line %d: expected %q, got %q", y, expected, actual)
	}
}

// TestLineWrapping_SearchHighlight tests search highlighting across wrapped lines
func TestLineWrapping_SearchHighlight(t *testing.T) {
	te := NewTestEditorWithContent(20, 10, "This is a very long line with search term that wraps")

	// Search for "search"
	te.ExecuteCommand("search")
	te.TypeKeys("/search\n")

	te.renderPane()
	screen := te.mockScreen.GetContent()

	// The word "search" should be highlighted even though it's on the wrapped portion
	// Find where "search" appears and verify it has highlight style
	searchFound := false
	for y := 0; y < 10; y++ {
		row, ok := screen[y]
		if !ok {
			continue
		}

		line := GetScreenLine(screen, y)
		if strings.Contains(line, "search") {
			// Check that the cells containing "search" have highlight style
			idx := strings.Index(line, "search")
			if idx >= 0 {
				for i := 0; i < len("search"); i++ {
					if cell, ok := row[idx+i]; ok {
						// Check if cell has search highlight style
						// This assumes search highlighting modifies the style
						if cell.Style != ui.StyleDefault {
							searchFound = true
						}
					}
				}
			}
		}
	}

	if !searchFound {
		t.Error("Search term should be highlighted on wrapped line")
	}
}

// TestLineWrapping_SyntaxHighlight tests syntax highlighting across wrapped lines
func TestLineWrapping_SyntaxHighlight(t *testing.T) {
	// Create a Go file with long lines
	te := NewTestEditorWithContent(30, 10, "")
	te.CurrentBuffer().FilePath = "test.go"

	// Insert Go code with a long string that wraps
	te.ExecuteCommand("insert")
	code := `func main() {
	fmt.Println("This is a very long string literal that should wrap to the next line")
}`
	for _, ch := range code {
		te.InsertChar(ch)
	}
	te.PressKey(events.KeyEscape, 0)

	te.renderPane()
	screen := te.mockScreen.GetContent()

	// Verify that string highlighting continues on wrapped line
	// Find the wrapped portion of the string
	stringHighlightFound := false
	for y := 0; y < 10; y++ {
		row, ok := screen[y]
		if !ok {
			continue
		}

		line := GetScreenLine(screen, y)
		// Look for the wrapped portion of the string
		if strings.Contains(line, "next line") {
			// This is part of the string literal, should have string style
			for _, cell := range row {
				if cell.Rune == 'n' || cell.Rune == 'e' || cell.Rune == 'x' || cell.Rune == 't' {
					// Check if it has syntax highlighting (not default style)
					if cell.Style != ui.StyleDefault {
						stringHighlightFound = true
						break
					}
				}
			}
			break
		}
	}

	if !stringHighlightFound {
		t.Error("String literal highlighting should continue on wrapped line")
	}
}

// TestLineWrapping_VisualMode tests visual mode selection across wrapped lines
func TestLineWrapping_VisualMode(t *testing.T) {
	te := NewTestEditorWithContent(20, 10, "This is a very long line that wraps multiple times")

	// Enter visual mode and select across wrap
	te.ExecuteCommand("visual_mode")

	// Move to select text that spans wrapped lines
	for i := 0; i < 25; i++ {
		te.ExecuteCommand("right")
	}

	te.renderPane()
	screen := te.mockScreen.GetContent()

	// Check that selection highlighting spans wrapped lines
	selectionFound := false
	for y := 0; y < 3; y++ { // Check first 3 visual lines
		row, ok := screen[y]
		if !ok {
			continue
		}

		for _, cell := range row {
			// Look for cells with selection style
			if cell.Style != ui.StyleDefault {
				selectionFound = true
				break
			}
		}
	}

	if !selectionFound {
		t.Error("Visual selection should span wrapped lines")
	}
}
