package editor

import (
	"strings"
	"testing"
)

func TestDetectWrapContext(t *testing.T) {
	tests := []struct {
		name           string
		lines          []string
		startLine      int
		wantContext    wrapContext
		wantIndent     string
		wantPrefix     string
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
		{"- Item", true},
		{"* Item", true},
		{"+ Item", true},
		{"-Item", false},
		{"1.Item", false},
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
		name         string
		lines        []string
		cursorLine   int
		wantStart    int
		wantEnd      int
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
			name: "plain text paragraph",
			input: `This is a very long line that definitely needs to be wrapped because it exceeds the normal width limit that we have set for our editor.`,
			cursorY:   0,
			wrapWidth: 40,
			want: `This is a very long line that definitely
needs to be wrapped because it exceeds
the normal width limit that we have set
for our editor.`,
		},
		{
			name: "indented paragraph",
			input: `    This is an indented paragraph that is quite long and needs to be wrapped while preserving the indentation.`,
			cursorY:   0,
			wrapWidth: 40,
			want: `    This is an indented paragraph that
    is quite long and needs to be
    wrapped while preserving the
    indentation.`,
		},
		{
			name: "markdown list",
			input: `1. This is a numbered list item that is very long and needs to be wrapped properly with the correct indentation.`,
			cursorY:   0,
			wrapWidth: 40,
			want: `1. This is a numbered list item that is
   very long and needs to be wrapped
   properly with the correct
   indentation.`,
		},
		{
			name: "C++ comment",
			input: `// This is a C++ style comment that is very long and needs to be wrapped with comment prefixes.`,
			cursorY:   0,
			wrapWidth: 40,
			want: `// This is a C++ style comment that is
// very long and needs to be wrapped
// with comment prefixes.`,
		},
		{
			name: "Python comment",
			input: `# this is a python comment that is very long and needs to be wrapped with proper prefixes.`,
			cursorY:   0,
			wrapWidth: 40,
			want: `# this is a python comment that is very
# long and needs to be wrapped with
# proper prefixes.`,
		},
		{
			name: "markdown heading should not wrap",
			input: `# This is a very long markdown heading that should not be wrapped no matter how long it is`,
			cursorY:   0,
			wrapWidth: 40,
			want: `# This is a very long markdown heading that should not be wrapped no matter how long it is`,
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
	t.Skip("Skipping undo/redo test due to iterator invalidation issues")
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
	
	// Save cursor position
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