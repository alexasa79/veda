package editor

import (
	"strings"
	"testing"

	"veda/internal/buffer"
	"veda/internal/commands"
)

func TestDiffViewMoveUpFromWrappedDeletionFiller(t *testing.T) {
	// The right pane has an empty filler row opposite a deleted left line.
	// Narrowing the pane wraps that left line and adds padding on the right.
	const deletedLine = "separate adjacent layout descriptions."
	for _, tc := range []struct {
		name      string
		paneWidth int
		wantRows  int
	}{
		{name: "wider", paneWidth: 44, wantRows: 1},
		{name: "narrow", paneWidth: 40, wantRows: 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			te := NewTestEditorWithContent(tc.paneWidth*2, 12, "")
			diffView, _, currentBuffer := setupWrapTestDiffView(te,
				"above\n"+deletedLine+"\nbelow", "above\nbelow")
			const fillerLine = 1
			if !currentBuffer.DiffHints[fillerLine].IsFillerLine {
				t.Fatal("test requires a right-pane deletion filler")
			}
			rows := 0
			for _, line := range te.CalculateWrappedLines() {
				if line.LogicalLineNum == fillerLine {
					rows++
				}
			}
			if rows != tc.wantRows {
				t.Fatalf("filler occupies %d visual rows, want %d", rows, tc.wantRows)
			}

			currentBuffer.CursorIterator = buffer.NewBufferIteratorAtLine(currentBuffer, fillerLine)
			currentBuffer.CursorX = 0
			diffView.SyncCursors()
			beforeRow, _ := te.GetCursorVisualPosition()
			for press := 1; press <= 3; press++ {
				if err := te.ExecuteCommand("up"); err != nil {
					t.Fatalf("up failed: %v", err)
				}
				afterRow, _ := te.GetCursorVisualPosition()
				if currentBuffer.CursorIterator.LineNumber() < fillerLine {
					return
				}
				t.Errorf("Up press %d left cursor on filler line %d, column %d; visual row %d -> %d",
					press, currentBuffer.CursorIterator.LineNumber(), currentBuffer.CursorX, beforeRow, afterRow)
			}
		})
	}
}

func TestDiffViewCursorSkipsWrapPadding(t *testing.T) {
	const paneWidth = 40
	for _, tc := range []struct {
		name string
		text string
	}{
		{name: "empty", text: ""},
		{name: "short", text: "short"},
		{name: "full row", text: strings.Repeat("b", paneWidth-4)},
		{name: "wrapped", text: strings.Repeat("b", 50)},
	} {
		for _, pane := range []string{"left", "right"} {
			t.Run(tc.name+"/"+pane, func(t *testing.T) {
				te := NewTestEditorWithContent(paneWidth*2, 12, "")
				shortContent := "above\n" + tc.text + "\nbelow"
				longContent := "above\n" + strings.Repeat("a", 90) + "\nbelow"
				leftContent, rightContent := longContent, shortContent
				if pane == "left" {
					leftContent, rightContent = shortContent, longContent
				}
				diffView, leftBuffer, rightBuffer := setupWrapTestDiffView(te, leftContent, rightContent)
				activeBuffer := rightBuffer
				if pane == "left" {
					diffView.ActivePane = 0
					activeBuffer = leftBuffer
					te.switchToBuffer(activeBuffer)
				}
				activeBuffer.CursorIterator = buffer.NewBufferIteratorAtLine(activeBuffer, 1)
				// Insert mode allows the cursor exactly at the end of the text.
				te.mode = commands.InsertMode
				activeBuffer.CursorX = len(tc.text)
				diffView.SyncCursors()

				textRows := NewWrapCalculator(paneWidth, activeBuffer.LineNumberWidth).WrapLine(tc.text, 1)
				lastTextRow := textRows[len(textRows)-1]
				wantRow := len(textRows) // "above" occupies row 0.
				wantCol := min(paneWidth-1, activeBuffer.LineNumberWidth+wrapTextWidth(lastTextRow.Text))
				row, col := te.GetCursorVisualPosition()
				if row != wantRow || col != wantCol {
					t.Fatalf("end-of-line cursor at (%d, %d), want (%d, %d) before padding", row, col, wantRow, wantCol)
				}

				if err := te.ExecuteCommand("down"); err != nil {
					t.Fatalf("down failed: %v", err)
				}
				if got := activeBuffer.CursorIterator.LineNumber(); got != 2 {
					t.Fatalf("down landed on line %d, want 2 after padding", got)
				}

				activeBuffer.CursorIterator = buffer.NewBufferIteratorAtLine(activeBuffer, 1)
				activeBuffer.CursorX = 0
				te.desiredColumn = -1
				diffView.SyncCursors()
				if err := te.ExecuteCommand("up"); err != nil {
					t.Fatalf("up failed: %v", err)
				}
				if got := activeBuffer.CursorIterator.LineNumber(); got != 0 {
					t.Fatalf("up landed on line %d, want 0", got)
				}
			})
		}
	}
}
