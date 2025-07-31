package editor

import (
	"fmt"
	"path/filepath"
	"strings"
	"veda/internal/buffer"
	"veda/internal/proxy"
)

// findReferences finds all references to the symbol under cursor
func (e *Editor) findReferences(args []string) error {
	// Only works if we have a file open
	if !e.buffer.HasFile() {
		return fmt.Errorf("no file open")
	}

	// Get cursor position (LSP uses 0-based line numbers)
	line := e.buffer.CursorIterator.LineNumber()
	character := e.buffer.CursorX

	LL.Debug("findReferences: file=%s, line=%d, character=%d", e.buffer.FilePath, line, character)

	// Get references from LSP
	locations, err := e.proxyManager.FindReferences(e.buffer.FilePath, line, character, true)
	if err != nil {
		LL.Debug("findReferences failed: %v", err)
		return err
	}

	LL.Debug("findReferences: got %d references", len(locations))

	// If no references found, nothing to do
	if len(locations) == 0 {
		// TODO: Show message to user
		return nil
	}

	// Get the symbol name under cursor
	symbolName := e.getSymbolAtCursor()
	
	// Save source file path before switching buffers
	sourceFilePath := e.buffer.FilePath
	
	// Create references buffer content in markdown format
	content := e.formatReferencesAsMarkdown(locations, sourceFilePath, line, character)

	// Create a unique buffer name
	bufferName := fmt.Sprintf("*References: %s:%d:%d*", sourceFilePath, line+1, character+1)

	// Check if references buffer already exists
	if existingBuffer, exists := e.buffers[bufferName]; exists {
		// Update existing buffer's content
		existingBuffer.Rope = buffer.NewRopeFromString(content)
		existingBuffer.CursorX = 0
		existingBuffer.ViewIterator = buffer.NewBufferIterator(existingBuffer)
		existingBuffer.CursorIterator = buffer.NewBufferIterator(existingBuffer)
		e.switchToBuffer(existingBuffer)
	} else {
		// Create new buffer
		newBuffer := buffer.NewBuffer(content, bufferName, e.config.DefaultIndentType, e.config.DefaultIndentWidth)
		e.buffers[bufferName] = newBuffer
		e.bufferOrder = append(e.bufferOrder, bufferName)
		e.switchToBuffer(newBuffer)
	}

	// Create and store the references buffer tracking info
	refsBuffer := &ReferencesBuffer{
		References:   locations,
		CurrentIndex: -1,
		SourceFile:   sourceFilePath,
		SourceLine:   line,
		SourceChar:   character,
		BufferName:   bufferName,
		SymbolName:   symbolName,
	}
	e.referencesBuffers[bufferName] = refsBuffer
	e.currentReferencesBuffer = refsBuffer

	// Position cursor on the current reference (where find-references was invoked)
	// The markdown format has a header, source line, empty line, count line, empty line, then references start
	// References start at line 5 (0-based)
	if len(locations) > 0 {
		// Find the reference that matches our source location
		currentRefIndex := -1
		sourceAbsPath := sourceFilePath
		LL.Debug("Original source path: %s", sourceAbsPath)

		// Convert to absolute path if needed
		if !strings.HasPrefix(sourceAbsPath, "/") && !strings.Contains(sourceAbsPath, ":") {
			// Get absolute path
			if absPath, err := filepath.Abs(sourceAbsPath); err == nil {
				sourceAbsPath = absPath
				LL.Debug("Converted to absolute path: %s", sourceAbsPath)
			}
		}

		for i, loc := range locations {
			// Parse the file URI to compare with source file
			locFilePath := strings.TrimPrefix(loc.URI, "file://")

			// Handle remote files
			if strings.Contains(sourceAbsPath, ":") {
				parts := strings.SplitN(sourceAbsPath, ":", 2)
				if len(parts) == 2 {
					host := parts[0]
					locFilePath = host + ":" + locFilePath
				}
			}

			LL.Debug("Comparing reference %d: locFilePath=%s sourceAbsPath=%s, loc.Line=%d sourceLine=%d, loc.Start.Char=%d loc.End.Char=%d sourceChar=%d",
				i, locFilePath, sourceAbsPath, loc.Range.Start.Line, line, loc.Range.Start.Character, loc.Range.End.Character, character)

			// Check if this reference matches our source location
			// First try exact path match
			pathMatch := locFilePath == sourceAbsPath
			// If that fails, try matching by ending with the source path (for relative vs absolute paths)
			if !pathMatch && strings.HasSuffix(locFilePath, sourceAbsPath) {
				pathMatch = true
				LL.Debug("Matched by suffix: %s ends with %s", locFilePath, sourceAbsPath)
			}
			
			if pathMatch &&
			   loc.Range.Start.Line == line &&
			   loc.Range.Start.Character <= character &&
			   loc.Range.End.Character > character {
				currentRefIndex = i
				LL.Debug("Found current reference at index %d", i)
				break
			}
		}

		// Position cursor on the appropriate reference line
		targetLine := 5 // Default to first reference (line 5, 0-based)
		if currentRefIndex >= 0 {
			targetLine = 5 + currentRefIndex
			// Update the current index in the references buffer
			refsBuffer.CurrentIndex = currentRefIndex
			LL.Debug("Setting cursor to line %d for reference index %d", targetLine, currentRefIndex)
		} else {
			LL.Debug("No matching reference found, defaulting to first reference at line %d", targetLine)
		}

		e.buffer.CursorIterator = buffer.NewBufferIteratorAtLine(e.buffer, targetLine)
		e.buffer.CursorX = 0
		// Ensure the cursor is visible
		e.scroll()
	}

	return nil
}

// formatReferencesAsMarkdown formats references as a markdown document
func (e *Editor) formatReferencesAsMarkdown(locations []proxy.Location, sourceFile string, sourceLine int, sourceChar int) string {
	var content strings.Builder

	// Get the symbol name if possible
	symbolName := e.getSymbolAtCursor()
	if symbolName == "" {
		symbolName = "symbol"
	}

	content.WriteString(fmt.Sprintf("# References to '%s'\n", symbolName))
	content.WriteString(fmt.Sprintf("Source: %s:%d:%d\n\n", sourceFile, sourceLine+1, sourceChar+1))
	content.WriteString(fmt.Sprintf("Found %d references:\n\n", len(locations)))

	// Process each reference
	for i, loc := range locations {
		// Parse the file URI
		filePath := strings.TrimPrefix(loc.URI, "file://")

		// If we're working with remote files, preserve the host prefix
		if strings.Contains(sourceFile, ":") {
			// Extract host from source file path
			parts := strings.SplitN(sourceFile, ":", 2)
			if len(parts) == 2 {
				host := parts[0]
				// Add host prefix to the file path from LSP
				filePath = host + ":" + filePath
			}
		}

		// Get the line content for context
		lineContent := e.getLineContent(filePath, loc.Range.Start.Line)
		if lineContent != "" {
			// Trim whitespace for display
			lineContent = strings.TrimSpace(lineContent)
			// Truncate if too long
			if len(lineContent) > 80 {
				lineContent = lineContent[:77] + "..."
			}
		}

		// Format as markdown link with line number and character position
		content.WriteString(fmt.Sprintf("%d. [%s:%d:%d](%s#L%d:%d)",
			i+1,
			filePath,
			loc.Range.Start.Line+1,    // Convert to 1-based for display
			loc.Range.Start.Character+1, // Convert to 1-based for display
			filePath,
			loc.Range.Start.Line+1,
			loc.Range.Start.Character+1))

		// Add code context if available
		if lineContent != "" {
			content.WriteString(fmt.Sprintf(" `%s`", lineContent))
		}

		content.WriteString("\n")
	}

	return content.String()
}

// getSymbolAtCursor tries to extract the symbol name at the current cursor position
func (e *Editor) getSymbolAtCursor() string {
	line := e.buffer.CursorIterator.Line()
	x := e.buffer.CursorX

	// Simple word extraction - find word boundaries
	start := x
	end := x

	// Move start backwards to beginning of word
	for start > 0 && isWordChar(rune(line[start-1])) {
		start--
	}

	// Move end forwards to end of word
	for end < len(line) && isWordChar(rune(line[end])) {
		end++
	}

	if start < end {
		return line[start:end]
	}
	return ""
}


// getLineContent reads a specific line from a file (with caching if the file is already open)
func (e *Editor) getLineContent(filePath string, lineNum int) string {
	// Check if we have this file in a buffer
	for _, buf := range e.buffers {
		if buf.FilePath == filePath {
			// Use the buffer's content
			iter := buffer.NewBufferIteratorAtLine(buf, lineNum)
			if !iter.IsEOF() {
				return iter.Line()
			}
			return ""
		}
	}

	// File not in buffer, we could read it but for now return empty
	// TODO: Consider reading the file to get context
	return ""
}

// nextReference jumps to the next reference in the current references buffer
func (e *Editor) nextReference(args []string) error {
	if e.currentReferencesBuffer == nil || len(e.currentReferencesBuffer.References) == 0 {
		return fmt.Errorf("no references to navigate")
	}

	refs := e.currentReferencesBuffer
	refs.CurrentIndex++
	if refs.CurrentIndex >= len(refs.References) {
		refs.CurrentIndex = 0 // Wrap around
	}

	return e.jumpToLocation(refs.References[refs.CurrentIndex])
}

// previousReference jumps to the previous reference in the current references buffer
func (e *Editor) previousReference(args []string) error {
	if e.currentReferencesBuffer == nil || len(e.currentReferencesBuffer.References) == 0 {
		return fmt.Errorf("no references to navigate")
	}

	refs := e.currentReferencesBuffer
	refs.CurrentIndex--
	if refs.CurrentIndex < 0 {
		refs.CurrentIndex = len(refs.References) - 1 // Wrap around
	}

	return e.jumpToLocation(refs.References[refs.CurrentIndex])
}

// jumpToLocation jumps to a specific reference location
func (e *Editor) jumpToLocation(location proxy.Location) error {
	// Parse the file URI
	if !strings.HasPrefix(location.URI, "file://") {
		return fmt.Errorf("unsupported URI scheme: %s", location.URI)
	}

	filePath := strings.TrimPrefix(location.URI, "file://")

	// If we're working with a remote file, preserve the host prefix
	if e.currentReferencesBuffer != nil && strings.Contains(e.currentReferencesBuffer.SourceFile, ":") {
		// Extract host from source file path
		parts := strings.SplitN(e.currentReferencesBuffer.SourceFile, ":", 2)
		if len(parts) == 2 {
			host := parts[0]
			// Add host prefix to the file path from LSP
			filePath = host + ":" + filePath
		}
	}

	LL.Debug("jumpToReference: location URI=%s, filePath=%s", location.URI, filePath)

	// Push current position to navigation history before jumping
	e.pushNavigationHistory()

	// If it's a different file, open it
	if e.buffer.FilePath != filePath {
		LL.Debug("jumpToReference: opening different file: %s", filePath)
		e.openFile(filePath)
	}

	// Jump to the location
	targetLine := location.Range.Start.Line
	e.buffer.CursorX = location.Range.Start.Character

	// Create iterator at the target line directly
	e.buffer.CursorIterator = buffer.NewBufferIteratorAtLine(e.buffer, targetLine)

	// Ensure cursor X is within bounds
	currentLine := e.buffer.CursorIterator.Line()
	if e.buffer.CursorX > len(currentLine) {
		e.buffer.CursorX = len(currentLine)
		if e.buffer.CursorX > 0 {
			e.buffer.CursorX-- // Don't go past last character in normal mode
		}
	}

	// Position the reference at the center of the screen
	e.scrollToCenter()

	return nil
}

// jumpToReferenceUnderCursor handles pressing Enter in a references buffer
func (e *Editor) jumpToReferenceUnderCursor(args []string) error {
	// Check if current buffer is a references buffer
	refsBuffer, isRefsBuffer := e.referencesBuffers[e.buffer.DisplayName()]
	if !isRefsBuffer {
		return fmt.Errorf("not in a references buffer")
	}

	// Make this the current references buffer
	e.currentReferencesBuffer = refsBuffer

	// Parse the line to extract the reference index
	line := e.buffer.CursorIterator.Line()

	// Look for pattern like "1. [file:line:col]"
	var index int
	if _, err := fmt.Sscanf(line, "%d.", &index); err == nil && index > 0 && index <= len(refsBuffer.References) {
		// Jump to this reference
		refsBuffer.CurrentIndex = index - 1
		return e.jumpToLocation(refsBuffer.References[index-1])
	}

	return fmt.Errorf("no valid reference on this line")
}

// jumpToReference is a command wrapper that checks if we're in a references buffer
func (e *Editor) jumpToReference(args []string) error {
	// Check if current buffer is a references buffer
	_, isRefsBuffer := e.referencesBuffers[e.buffer.DisplayName()]
	if !isRefsBuffer {
		// Not in a references buffer, do nothing (or could do something else with Enter)
		return nil
	}

	return e.jumpToReferenceUnderCursor(args)
}