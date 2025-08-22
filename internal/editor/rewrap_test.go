package editor

import (
	"strings"
	"testing"
)

func TestDetectWrapContext(t *testing.T) {
	tests := []struct {
		name        string
		lines       []string
		startLine   int
		wantContext wrapContext
		wantIndent  string
		wantPrefix  string
	}{
		{
			name:        "plain text with indentation",
			lines:       []string{"    This is plain text"},
			startLine:   0,
			wantContext: wrapContextPlain,
			wantIndent:  "    ",
			wantPrefix:  "",
		},
		{
			name:        "markdown numbered list",
			lines:       []string{"1. This is a list item"},
			startLine:   0,
			wantContext: wrapContextMarkdownList,
			wantIndent:  "",
			wantPrefix:  "   ",
		},
		{
			name:        "markdown parenthesized numbered list",
			lines:       []string{"1) This is a list item"},
			startLine:   0,
			wantContext: wrapContextMarkdownList,
			wantIndent:  "",
			wantPrefix:  "   ",
		},
		{
			name:        "markdown bullet list",
			lines:       []string{"- This is a list item"},
			startLine:   0,
			wantContext: wrapContextMarkdownList,
			wantIndent:  "",
			wantPrefix:  "  ",
		},
		{
			name:        "C++ line comment",
			lines:       []string{"    // This is a comment"},
			startLine:   0,
			wantContext: wrapContextCLineComment,
			wantIndent:  "    ",
			wantPrefix:  "// ",
		},
		{
			name:        "Python comment",
			lines:       []string{"    # This is a Python comment"},
			startLine:   0,
			wantContext: wrapContextPythonComment,
			wantIndent:  "    ",
			wantPrefix:  "# ",
		},
		{
			name:        "C block comment",
			lines:       []string{"    /* This is a block comment"},
			startLine:   0,
			wantContext: wrapContextCBlockComment,
			wantIndent:  "       ",
			wantPrefix:  "",
		},
		{
			name:        "markdown heading",
			lines:       []string{"# This is a heading"},
			startLine:   0,
			wantContext: wrapContextMarkdownHeading,
			wantIndent:  "",
			wantPrefix:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotContext, gotIndent, gotPrefix := detectWrapContext(tt.lines, tt.startLine)
			if gotContext != tt.wantContext {
				t.Errorf("detectWrapContext() context = %v, want %v", gotContext, tt.wantContext)
			}
			if gotIndent != tt.wantIndent {
				t.Errorf("detectWrapContext() indent = %q, want %q", gotIndent, tt.wantIndent)
			}
			if gotPrefix != tt.wantPrefix {
				t.Errorf("detectWrapContext() prefix = %q, want %q", gotPrefix, tt.wantPrefix)
			}
		})
	}
}

func TestIsMarkdownListItem(t *testing.T) {
	tests := []struct {
		line string
		want bool
	}{
		{"1. Item", true},
		{"123. Item", true},
		{"1) Item", true},
		{"123) Item", true},
		{"- Item", true},
		{"* Item", true},
		{"+ Item", true},
		{"-Item", false},
		{"1.Item", false},
		{"1)Item", false},
		{"Regular text", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			if got := isMarkdownListItem(tt.line); got != tt.want {
				t.Errorf("isMarkdownListItem(%q) = %v, want %v", tt.line, got, tt.want)
			}
		})
	}
}

func TestWrapText(t *testing.T) {
	tests := []struct {
		name   string
		text   string
		width  int
		indent string
		prefix string
		want   []string
	}{
		{
			name:   "simple wrap",
			text:   "This is a long line that needs to be wrapped because it exceeds the maximum width",
			width:  40,
			indent: "",
			prefix: "",
			want: []string{
				"This is a long line that needs to be",
				"wrapped because it exceeds the maximum",
				"width",
			},
		},
		{
			name:   "wrap with indentation",
			text:   "This is a long line that needs to be wrapped",
			width:  30,
			indent: "    ",
			prefix: "",
			want: []string{
				"    This is a long line that",
				"    needs to be wrapped",
			},
		},
		{
			name:   "wrap with prefix",
			text:   "This is a long comment that needs wrapping",
			width:  30,
			indent: "    ",
			prefix: "// ",
			want: []string{
				"    // This is a long comment",
				"    // that needs wrapping",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := wrapText(tt.text, tt.width, tt.indent, tt.prefix)
			if len(got) != len(tt.want) {
				t.Errorf("wrapText() returned %d lines, want %d", len(got), len(tt.want))
				t.Errorf("got: %v", got)
				t.Errorf("want: %v", tt.want)
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("wrapText() line %d = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestFindParagraphBounds(t *testing.T) {
	tests := []struct {
		name       string
		lines      []string
		cursorLine int
		wantStart  int
		wantEnd    int
	}{
		{
			name: "single paragraph",
			lines: []string{
				"First line",
				"Second line",
				"Third line",
			},
			cursorLine: 1,
			wantStart:  0,
			wantEnd:    2,
		},
		{
			name: "paragraph with empty lines",
			lines: []string{
				"First paragraph",
				"",
				"Second paragraph line 1",
				"Second paragraph line 2",
				"",
				"Third paragraph",
			},
			cursorLine: 3,
			wantStart:  2,
			wantEnd:    3,
		},
		{
			name: "cursor at start",
			lines: []string{
				"First line",
				"Second line",
			},
			cursorLine: 0,
			wantStart:  0,
			wantEnd:    1,
		},
		{
			name: "markdown list items as separate paragraphs",
			lines: []string{
				"- first item",
				"- second item",
				"- third item",
			},
			cursorLine: 1,
			wantStart:  1,
			wantEnd:    1,
		},
		{
			name: "numbered list items as separate paragraphs",
			lines: []string{
				"1. first item",
				"2. second item",
				"3. third item",
			},
			cursorLine: 0,
			wantStart:  0,
			wantEnd:    0,
		},
		{
			name: "parenthesized numbered list items as separate paragraphs",
			lines: []string{
				"1) first item",
				"2) second item",
				"3) third item",
			},
			cursorLine: 1,
			wantStart:  1,
			wantEnd:    1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotStart, gotEnd := findParagraphBounds(tt.lines, tt.cursorLine)
			if gotStart != tt.wantStart {
				t.Errorf("findParagraphBounds() start = %v, want %v", gotStart, tt.wantStart)
			}
			if gotEnd != tt.wantEnd {
				t.Errorf("findParagraphBounds() end = %v, want %v", gotEnd, tt.wantEnd)
			}
		})
	}
}

func TestRewrapParagraph(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		cursorY   int
		wrapWidth int
		want      string
	}{
		{
			name:      "plain text paragraph",
			input:     `This is a very long line that definitely needs to be wrapped because it exceeds the normal width limit that we have set for our editor.`,
			cursorY:   0,
			wrapWidth: 40,
			want: `This is a very long line that definitely
needs to be wrapped because it exceeds
the normal width limit that we have set
for our editor.`,
		},
		{
			name:      "indented paragraph",
			input:     `    This is an indented paragraph that is quite long and needs to be wrapped while preserving the indentation.`,
			cursorY:   0,
			wrapWidth: 40,
			want: `    This is an indented paragraph that
    is quite long and needs to be
    wrapped while preserving the
    indentation.`,
		},
		{
			name:      "markdown list",
			input:     `1. This is a numbered list item that is very long and needs to be wrapped properly with the correct indentation.`,
			cursorY:   0,
			wrapWidth: 40,
			want: `1. This is a numbered list item that is
   very long and needs to be wrapped
   properly with the correct
   indentation.`,
		},
		{
			name:      "C++ comment",
			input:     `// This is a C++ style comment that is very long and needs to be wrapped with comment prefixes.`,
			cursorY:   0,
			wrapWidth: 40,
			want: `// This is a C++ style comment that is
// very long and needs to be wrapped
// with comment prefixes.`,
		},
		{
			name:      "Python comment",
			input:     `# this is a python comment that is very long and needs to be wrapped with proper prefixes.`,
			cursorY:   0,
			wrapWidth: 40,
			want: `# this is a python comment that is very
# long and needs to be wrapped with
# proper prefixes.`,
		},
		{
			name:      "markdown heading should not wrap",
			input:     `# This is a very long markdown heading that should not be wrapped no matter how long it is`,
			cursorY:   0,
			wrapWidth: 40,
			want:      `# This is a very long markdown heading that should not be wrapped no matter how long it is`,
		},
		{
			name: "markdown heading without empty line - text should wrap",
			input: `# 2025-08-05T10-52-45 Discuss performance with Satbir
He went the load balancer route. He implemented something that intercepts packets as they come out of, something, not sure where. He does not seem to understand what is NAT router and router overall.`,
			cursorY:   1,
			wrapWidth: 80,
			want: `# 2025-08-05T10-52-45 Discuss performance with Satbir
He went the load balancer route. He implemented something that intercepts
packets as they come out of, something, not sure where. He does not seem to
understand what is NAT router and router overall.`,
		},
		{
			name: "markdown list items should not be merged",
			input: `- one
- two`,
			cursorY:   0,
			wrapWidth: 80,
			want: `- one
- two`,
		},
		{
			name: "numbered markdown list items should not be merged",
			input: `1. first item
2. second item`,
			cursorY:   0,
			wrapWidth: 80,
			want: `1. first item
2. second item`,
		},
		{
			name: "mixed markdown list types should not be merged",
			input: `- bullet item
1. numbered item`,
			cursorY:   0,
			wrapWidth: 80,
			want: `- bullet item
1. numbered item`,
		},
		{
			name: "long markdown list items should wrap independently",
			input: `- This is a very long bullet point that should wrap within itself but should not be merged with the next list item
- This is another bullet point`,
			cursorY:   0,
			wrapWidth: 40,
			want: `- This is a very long bullet point that
  should wrap within itself but should
  not be merged with the next list item
- This is another bullet point`,
		},
		{
			name: "paragraph stops before numbered markdown list",
			input: `This is an introductory paragraph that should be wrapped but must not consume the numbered list below when rewrapping.
1. First thing to do
2. Second thing to do`,
			cursorY:   0,
			wrapWidth: 60,
			want: `This is an introductory paragraph that should be wrapped but
must not consume the numbered list below when rewrapping.
1. First thing to do
2. Second thing to do`,
		},
		{
			name: "numbered list with parenthesis should not be merged",
			input: `1) first item
2) second item`,
			cursorY:   0,
			wrapWidth: 80,
			want: `1) first item
2) second item`,
		},
		{
			name:      "long numbered list with parenthesis wraps correctly",
			input:     `1) This is a very long numbered list item that should wrap with subsequent lines aligned under the text`,
			cursorY:   0,
			wrapWidth: 40,
			want: `1) This is a very long numbered list
   item that should wrap with subsequent
   lines aligned under the text`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test editor with content
			te := NewTestEditorWithContent(100, 24, tt.input)

			// Move cursor to the desired line
			for i := 0; i < tt.cursorY; i++ {
				te.ExecuteCommand("down")
			}

			// Execute rewrap with specified width
			err := te.Editor.rewrapParagraphWithWidth(tt.wrapWidth)
			if err != nil {
				t.Fatalf("rewrapParagraph() failed: %v", err)
			}

			// Get result
			got := strings.TrimSpace(te.GetBufferContent())
			want := strings.TrimSpace(tt.want)

			if got != want {
				t.Errorf("rewrapParagraph() result mismatch\ngot:\n%s\nwant:\n%s", got, want)
				// Debug output
				t.Logf("Input length: %d characters", len(tt.input))
				t.Logf("Wrap width: %d", tt.wrapWidth)
			}
		})
	}
}

func TestRewrapCommandUndoRedo(t *testing.T) {
	// Create test editor with a long line
	input := "This is a very long line that needs to be wrapped because it exceeds the width limit."
	te := NewTestEditorWithContent(100, 24, input)

	// Save initial state
	initialContent := te.GetBufferContent()
	initialCursorX := te.Editor.buffer.CursorX
	initialCursorY := te.Editor.buffer.CursorIterator.LineNumber() // Save as int, not iterator

	// Execute rewrap
	err := te.Editor.rewrapParagraph()
	if err != nil {
		t.Fatalf("rewrapParagraph() failed: %v", err)
	}

	// Verify content changed
	wrappedContent := te.GetBufferContent()
	if wrappedContent == initialContent {
		t.Error("Content should have changed after rewrap")
	}

	// Undo
	te.ExecuteCommand("undo")

	// Verify content restored
	restoredContent := te.GetBufferContent()
	if restoredContent != initialContent {
		t.Errorf("Content not restored after undo\ngot: %q\nwant: %q", restoredContent, initialContent)
	}

	// Verify cursor restored
	cursorX := te.Editor.buffer.CursorX
	cursorY := te.Editor.buffer.CursorIterator.LineNumber()
	if cursorX != initialCursorX || cursorY != initialCursorY {
		t.Errorf("Cursor not restored after undo: got (%d,%d), want (%d,%d)",
			cursorX, cursorY, initialCursorX, initialCursorY)
	}

	// Redo
	te.ExecuteCommand("redo")

	// Verify content re-wrapped
	redoneContent := te.GetBufferContent()
	if redoneContent != wrappedContent {
		t.Errorf("Content not re-wrapped after redo\ngot: %q\nwant: %q", redoneContent, wrappedContent)
	}
}

func TestRewrapCreatesUndoBoundaryInInsertMode(t *testing.T) {
	te := NewTestEditorWithContent(100, 24, "")

	// Enter insert mode and type a paragraph
	te.ExecuteCommand("enter-insert-mode")
	text := "This is a paragraph that will be wrapped to test undo boundaries. " +
		"Typing should be one undo unit separate from the rewrap."
	for _, ch := range text {
		te.InsertChar(ch)
	}

	// Trigger rewrap while still in insert mode
	if err := te.ExecuteCommand("rewrap-paragraph"); err != nil {
		t.Fatalf("rewrap-paragraph failed: %v", err)
	}

	// Exit insert mode to perform undo in normal mode
	te.ExecuteCommand("exit-insert-mode")

	wrappedAfter := te.GetBufferContent()
	if wrappedAfter == text {
		t.Fatalf("expected rewrap to change content")
	}

	// First undo should undo only the rewrap, keeping the typed text
	te.ExecuteCommand("undo")
	afterFirstUndo := te.GetBufferContent()
	if afterFirstUndo != text {
		t.Fatalf("first undo should revert rewrap only.\nGot:\n%q\nWant:\n%q", afterFirstUndo, text)
	}

	// Second undo should undo the insert session (typed text)
	te.ExecuteCommand("undo")
	AssertBufferEquals(t, te, "")
}

func TestRewrapPositionBasedRulerCycling(t *testing.T) {
	// Create test editor with vertical rulers at 40, 60, 80
	content := "This is a test line that is long enough to be wrapped at different widths to test the cycling behavior"
	te := NewTestEditorWithContent(100, 24, content)

	// Set vertical rulers
	te.Editor.config.VerticalRulers = []int{40, 60, 80}

	// First rewrap at cursor position (0,0) - should use first ruler (40)
	err := te.Editor.rewrapParagraph()
	if err != nil {
		t.Fatalf("First rewrapParagraph() failed: %v", err)
	}

	// Verify the line is wrapped at ~40 characters
	bufferContent := te.GetBufferContent()
	lines := strings.Split(bufferContent, "\n")
	if len(lines) < 2 {
		t.Error("Expected text to be wrapped into multiple lines")
	}
	firstLineLen := len(lines[0])
	if firstLineLen > 45 || firstLineLen < 35 {
		t.Errorf("First line length %d not near 40 characters", firstLineLen)
	}

	// Save cursor position after first rewrap; subsequent rewraps at "same place"
	// should cycle to the next ruler even if CursorX shifts. We'll reuse these to return later.
	savedLine := te.Editor.buffer.CursorIterator.LineNumber()
	savedChar := te.Editor.buffer.CursorX

	// Second rewrap at same position - should cycle to second ruler (60)
	err = te.Editor.rewrapParagraph()
	if err != nil {
		t.Fatalf("Second rewrapParagraph() failed: %v", err)
	}

	// Verify the line is wrapped at ~60 characters
	bufferContent = te.GetBufferContent()
	lines = strings.Split(bufferContent, "\n")
	firstLineLen = len(lines[0])
	if firstLineLen > 65 || firstLineLen < 55 {
		t.Errorf("First line length %d not near 60 characters after second rewrap", firstLineLen)
	}

	// Move cursor to a different position
	te.ExecuteCommand("down")

	// Third rewrap at different position - should reset to first ruler (40)
	err = te.Editor.rewrapParagraph()
	if err != nil {
		t.Fatalf("Third rewrapParagraph() failed: %v", err)
	}

	// Verify the line is wrapped at ~40 characters again
	bufferContent = te.GetBufferContent()
	lines = strings.Split(bufferContent, "\n")
	firstLineLen = len(lines[0])
	if firstLineLen > 45 || firstLineLen < 35 {
		t.Errorf("First line length %d not near 40 characters after moving cursor", firstLineLen)
	}

	// Move cursor back to original position
	te.SetCursorPosition(savedChar, savedLine)

	// Fourth rewrap at original position - should start over at first ruler (40)
	// because we've been to a different position
	err = te.Editor.rewrapParagraph()
	if err != nil {
		t.Fatalf("Fourth rewrapParagraph() failed: %v", err)
	}

	// Verify it's back to ~40 characters
	bufferContent = te.GetBufferContent()
	lines = strings.Split(bufferContent, "\n")
	firstLineLen = len(lines[0])
	if firstLineLen > 45 || firstLineLen < 35 {
		t.Errorf("First line length %d not near 40 characters after returning to original position", firstLineLen)
	}
}

func TestRewrapThirdNumberedItemPreservesAlignment(t *testing.T) {
	input := `1. Open the original version of the file (the HEAD version) in the current
   window. We will call this buffer <file>@HEAD
2. We will make sure there is at least one window to the right of the current
   window. If there is no window, we will create a new one.
3. We will then open a copy of the current buffer. Note that we will open
   another buffer for the current file, not the original file. We will call this
   buffer <file>@current.`

	te := NewTestEditorWithContent(100, 24, input)
	// Move cursor to the 3rd list item first line (zero-based index 4)
	for i := 0; i < 4; i++ {
		te.ExecuteCommand("down")
	}

	if err := te.ExecuteCommand("rewrap-paragraph"); err != nil {
		t.Fatalf("rewrap-paragraph failed: %v", err)
	}

	lines := strings.Split(te.GetBufferContent(), "\n")
	if len(lines) < 7 {
		t.Fatalf("unexpected number of lines: %d", len(lines))
	}

	// The 3rd item header must not be indented; it must start with "3. " at column 0
	if !strings.HasPrefix(lines[4], "3. ") {
		t.Fatalf("third item header got indented or altered: %q", lines[4])
	}
	// Continuation lines should be indented by three spaces
	if !strings.HasPrefix(lines[5], "   ") || !strings.HasPrefix(lines[6], "   ") {
		t.Fatalf("third item continuation lines not properly indented: %q | %q", lines[5], lines[6])
	}
}

// TestRewrapRulerCycling tests that repeated rewrapping cycles through different ruler widths
func TestRewrapRulerCycling(t *testing.T) {
	// Use a very long comment that will definitely need wrapping at all ruler widths
	content := `    /* This is a very long comment that contains many words and should definitely be wrapped when we run the rewrap command because it exceeds even the largest default wrap width of one hundred twenty characters and continues on with more text to make it really quite long indeed for testing purposes and even more text beyond that */`

	te := NewTestEditorWithContent(200, 24, content)

	// Set the default vertical rulers (80, 100, 120)
	te.Editor.config.VerticalRulers = []int{80, 100, 120}

	// Position cursor in the comment text
	te.SetCursorPosition(10, 0)

	// Expected behavior specification:
	expectedBehavior := []struct {
		rewrapNumber int
		rulerWidth   int
		description  string
	}{
		{1, 80, "First rewrap should use 80-char ruler"},
		{2, 100, "Second rewrap should use 100-char ruler"},
		{3, 120, "Third rewrap should use 120-char ruler"},
		{4, 80, "Fourth rewrap should cycle back to 80-char ruler"},
	}

	results := make([]string, len(expectedBehavior))

	for i, expected := range expectedBehavior {
		t.Logf("=== Rewrap %d (expecting %d-char ruler) ===", expected.rewrapNumber, expected.rulerWidth)

		err := te.ExecuteCommand("rewrap-paragraph")
		if err != nil {
			t.Fatalf("Rewrap %d failed: %v", expected.rewrapNumber, err)
		}

		result := te.GetBufferContent()
		results[i] = result
		lines := strings.Split(result, "\n")

		t.Logf("%s", expected.description)

		// Check that lines respect the expected width (with some tolerance for comment syntax)
		maxExpectedLength := expected.rulerWidth + 5 // Allow some buffer for comment formatting
		for j, line := range lines {
			if len(line) > maxExpectedLength {
				t.Errorf("Rewrap %d, Line %d exceeds expected max length %d: %d chars",
					expected.rewrapNumber, j+1, maxExpectedLength, len(line))
			}
		}

		// For wider rulers, we should generally get fewer lines (more text per line)
		if i > 0 && expected.rulerWidth > expectedBehavior[i-1].rulerWidth {
			prevLines := strings.Split(results[i-1], "\n")
			if len(lines) >= len(prevLines) {
				t.Logf("Note: Expected fewer lines with %d-char ruler (%d lines) than with %d-char ruler (%d lines)",
					expected.rulerWidth, len(lines), expectedBehavior[i-1].rulerWidth, len(prevLines))
			}
		}
	}

	// Check that results are different (ruler cycling is working)
	for i := 1; i < len(results); i++ {
		if results[i] == results[i-1] {
			t.Errorf("Rewrap %d produced identical results to rewrap %d - ruler cycling may not be working correctly", i+1, i)
		}
	}
}
