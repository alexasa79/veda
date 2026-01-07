package editor

import (
	"fmt"
	"strings"
	"veda/internal/buffer"
	"veda/internal/commands"
	"veda/internal/events"
	"veda/internal/ollama"
)

// spellCheck performs spell and punctuation checking using Ollama
// If in visual mode, checks only the selected text
// Otherwise, checks the current paragraph
func (e *Editor) spellCheck(args []string) error {
	if e.CurrentBuffer() == nil {
		return fmt.Errorf("no buffer open")
	}

	var textToCheck string
	var startLine, endLine int

	if e.mode == commands.VisualMode || e.mode == commands.VisualLineMode {
		// For visual mode, use the stored line numbers directly
		anchorLine := e.CurrentBuffer().VisualAnchorLine
		currentLine := e.CurrentBuffer().CursorIterator.LineNumber()

		// Determine start and end lines
		if currentLine < anchorLine {
			startLine = currentLine
			endLine = anchorLine
		} else {
			startLine = anchorLine
			endLine = currentLine
		}

		// Extract selected text
		start, end, hasSelection := e.getSelectionRange()
		if !hasSelection {
			return fmt.Errorf("no text selected")
		}
		textToCheck = e.CurrentBuffer().Rope.Extract(start, end)

		// Exit visual mode
		e.exitVisualMode(nil)
	} else {
		// Get current paragraph - follow the same pattern as rewrap.go
		cursorLine := e.CurrentBuffer().CursorIterator.LineNumber()

		// Get all lines from buffer
		lines := make([]string, e.CurrentBuffer().LineCount)
		for i := 0; i < e.CurrentBuffer().LineCount; i++ {
			iter := buffer.NewBufferIteratorAtLine(e.CurrentBuffer(), i)
			lines[i] = iter.Line()
		}

		// Find paragraph bounds (function from rewrap.go)
		startLine, endLine = findParagraphBounds(lines, cursorLine)

		// Extract paragraph text
		var paragraphBuilder strings.Builder
		for i := startLine; i <= endLine; i++ {
			if i > startLine {
				paragraphBuilder.WriteString("\n")
			}
			paragraphBuilder.WriteString(lines[i])
		}
		textToCheck = paragraphBuilder.String()
	}

	if strings.TrimSpace(textToCheck) == "" {
		e.errorManager.ShowError("No text to check")
		return nil
	}

	// Calculate max line length and find appropriate ruler width for rewrapping
	maxLineLen := 0
	for _, line := range strings.Split(textToCheck, "\n") {
		if len(line) > maxLineLen {
			maxLineLen = len(line)
		}
	}

	// Find the smallest ruler that the original text fits within
	targetWidth := 0
	rulers := e.config.VerticalRulers
	for _, ruler := range rulers {
		if maxLineLen <= ruler {
			targetWidth = ruler
			break
		}
	}

	// Set persistent highlight for the spell check region
	e.spellCheckStartLine = startLine
	e.spellCheckEndLine = endLine
	e.spellCheckActive = true
	e.spellCheckBuffer = e.CurrentBuffer()

	// Show async operation indicator
	e.startAsyncOperation("spellcheck", "Checking spelling...")
	e.ShowInfo("Checking spelling and punctuation...")

	// Save original text and cursor position for the history entry
	originalText := textToCheck
	cursorX := e.CurrentBuffer().CursorX
	cursorY := e.CurrentBuffer().CursorIterator.LineNumber()
	buf := e.CurrentBuffer()

	// Run the spell check asynchronously
	go func() {
		client := ollama.NewClient()

		correctedText, err := client.SpellCheckAndFix(originalText)

		// Send result back to main thread via async events
		e.asyncEvents <- events.NewCustomEvent("apply", func(ed *Editor) {
			ed.handleSpellCheckResult(correctedText, err, startLine, endLine, cursorX, cursorY, buf, originalText, targetWidth)
		})
	}()

	return nil
}

// handleSpellCheckResult processes the result of a spell check operation
func (e *Editor) handleSpellCheckResult(correctedText string, err error, startLine, endLine, oldCursorX, oldCursorY int, buf *buffer.Buffer, originalText string, targetWidth int) {
	e.endAsyncOperation("spellcheck")

	// Clear the spell check highlight
	e.spellCheckActive = false
	e.spellCheckBuffer = nil

	if err != nil {
		e.errorManager.ShowError("Spell check failed: %v", err)
		return
	}

	// Make sure we're still on the same buffer
	if e.CurrentBuffer() != buf {
		e.errorManager.ShowError("Buffer changed during spell check")
		return
	}

	// Trim any extra whitespace the LLM might have added
	correctedText = strings.TrimSpace(correctedText)
	originalText = strings.TrimSpace(originalText)

	// If the text wasn't changed, just show a message
	if correctedText == originalText {
		e.ShowInfo("No spelling or punctuation errors found")
		return
	}

	// If we have a target width, check if rewrapping is needed
	if targetWidth > 0 {
		maxResultLineLen := 0
		for _, line := range strings.Split(correctedText, "\n") {
			if len(line) > maxResultLineLen {
				maxResultLineLen = len(line)
			}
		}

		// If the result has lines exceeding the target width, rewrap
		if maxResultLineLen > targetWidth {
			// Detect indentation from the first line
			indent := ""
			lines := strings.Split(correctedText, "\n")
			if len(lines) > 0 {
				for _, ch := range lines[0] {
					if ch == ' ' || ch == '\t' {
						indent += string(ch)
					} else {
						break
					}
				}
			}

			// Join all text and rewrap
			allText := strings.Join(strings.Fields(correctedText), " ")
			wrappedLines := wrapText(allText, targetWidth, indent, "")
			correctedText = strings.Join(wrappedLines, "\n")
		}
	}

	// Save view line number BEFORE modifying the buffer (iterators become invalid after modification)
	viewLine := buf.ViewIterator.LineNumber()

	// Create a composite entry for single undo
	composite := buffer.NewCompositeEntry("Spell check")

	// First, delete the old text
	composite.Add(buffer.NewDeleteTextEntry(startLine, 0, originalText))

	// Then, insert the corrected text
	composite.Add(buffer.NewInsertTextEntry(startLine, 0, correctedText))

	// Execute through history
	if buf.History != nil {
		if executed, histErr := buf.History.Execute(buf, composite); histErr != nil {
			e.errorManager.ShowError("Failed to apply correction: %v", histErr)
			return
		} else if executed != nil {
			e.lastChange = executed
		}
	}

	// Mark buffer as modified
	buf.IsModified = true

	// Recreate iterators after buffer modification
	newLineCount := strings.Count(correctedText, "\n") + 1
	newEndLine := startLine + newLineCount - 1
	if newEndLine >= buf.LineCount {
		newEndLine = buf.LineCount - 1
	}

	// Position cursor at start of changed region
	buf.CursorIterator = buffer.NewBufferIteratorAtLine(buf, startLine)
	buf.CursorX = 0

	// Update view iterator if needed
	if viewLine >= buf.LineCount {
		viewLine = buf.LineCount - 1
	}
	buf.ViewIterator = buffer.NewBufferIteratorAtLine(buf, viewLine)

	// Update syntax highlighting
	e.updateSyntaxHighlighting()

	e.ShowInfo("Spell check complete - corrections applied")
}
