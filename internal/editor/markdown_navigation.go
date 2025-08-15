// Package editor contains markdown link navigation functionality.
// This file handles navigation when pressing Enter on markdown links,
// supporting both regular links and links with line numbers (e.g., file.go#L42).
package editor

import (
	"regexp"
	"strconv"
	"strings"
	"veda/internal/buffer"
	"veda/internal/proxy"
)

// markdownLinkRegex matches markdown links [text](url)
var markdownLinkRegex = regexp.MustCompile(`\[([^\[\]]*(?:\[[^\[\]]*\][^\[\]]*)*)\]\(([^)]+)\)`)

// jumpToReference is a command wrapper that checks if we're in a references buffer or on a markdown link
func (e *Editor) jumpToReference(args []string) error {
	// Check if current buffer is a references buffer
	_, isRefsBuffer := e.referencesBuffers[e.buffer.DisplayName()]
	if isRefsBuffer {
		// Try the index-based approach first for references buffers
		if err := e.jumpToReferenceUnderCursor(args); err == nil {
			return nil
		}
		// Fall through to try markdown link parsing
	}

	// Check if we're on a markdown link (works in any buffer including references)
    if link, lineNum := e.getMarkdownLinkAtCursor(); link != "" {
        return e.openMarkdownLink(link, lineNum)
	}

	// Not on a reference or link, do nothing
	return nil
}

// openMarkdownLinkInNewWindow opens the markdown link target in a new window
func (e *Editor) openMarkdownLinkInNewWindow(args []string) error {
    if link, lineNum := e.getMarkdownLinkAtCursor(); link != "" {
        // Create a new window to the right and open the link
        insertAt := e.activeWindow + 1
        oldActive := e.activeWindow
        if insertAt >= len(e.windows) {
            e.windows = append(e.windows, &Window{Buffer: e.buffer, ViewLine: 0, CursorLine: 0, CursorX: 0})
        } else {
            e.windows = append(e.windows[:insertAt+1], e.windows[insertAt:]...)
            e.windows[insertAt] = &Window{Buffer: e.buffer, ViewLine: 0, CursorLine: 0, CursorX: 0}
        }
        e.recomputeWindowsLayout()
        e.lastWindow = oldActive
        e.activeWindow = insertAt
        // Open in new window
        if err := e.openMarkdownLink(link, lineNum); err != nil {
            return err
        }
        return nil
    }
    return nil
}

// getMarkdownLinkAtCursor returns the link URL and line number (if any) at the current cursor position
func (e *Editor) getMarkdownLinkAtCursor() (string, int) {
	line := e.buffer.CursorIterator.Line()
	cursorX := e.buffer.CursorX
	
	// Find all markdown links in the line
	matches := markdownLinkRegex.FindAllStringSubmatchIndex(line, -1)
	
	for _, match := range matches {
		// match[0] and match[1] are the start and end of the entire match
		// match[4] and match[5] are the start and end of the URL group
		linkStart := match[0]
		linkEnd := match[1]
		urlStart := match[4]
		urlEnd := match[5]
		
		// Check if cursor is within the link
		if cursorX >= linkStart && cursorX < linkEnd {
			url := line[urlStart:urlEnd]
			
			// Parse line number from URL if present (e.g., file.go#L42)
			lineNum := 0
			if idx := strings.LastIndex(url, "#L"); idx != -1 {
				if num, err := strconv.Atoi(url[idx+2:]); err == nil {
					lineNum = num
					url = url[:idx] // Remove the line number part from URL
				}
			}
			
			return url, lineNum
		}
	}
	
	return "", 0
}

// openMarkdownLink opens a file from a markdown link, optionally jumping to a specific line
func (e *Editor) openMarkdownLink(link string, lineNum int) error {
	// Convert to a Location object and use the existing jumpToLocation function
	// This ensures consistent behavior with reference jumping
	// If no line number specified, use 0 (first line)
	targetLine := 0
	if lineNum > 0 {
		targetLine = lineNum - 1 // Convert 1-based to 0-based
	}
	
	location := proxy.Location{
		URI: "file://" + link,
		Range: proxy.Range{
			Start: proxy.Position{
				Line:      targetLine,
				Character: 0,
			},
			End: proxy.Position{
				Line:      targetLine,
				Character: 0,
			},
		},
	}
	
	// For testing, we need to handle pre-existing buffers
	if _, exists := e.buffers[link]; exists {
		// Push navigation history before switching
		e.pushNavigationHistory()
		
		// Switch to the buffer
		if buf, ok := e.buffers[link]; ok {
			e.switchToBuffer(buf)
		}
		
		// Jump to the line if specified
		if lineNum > 0 {
			targetLine := lineNum - 1
			e.buffer.CursorIterator = buffer.NewBufferIteratorAtLine(e.buffer, targetLine)
			e.buffer.CursorX = 0
			e.scrollToCenter()
		}
		
		return nil
	}
	
	// Use the standard jumpToLocation for real files
	return e.jumpToLocation(location)
}