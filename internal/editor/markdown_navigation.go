// Package editor contains markdown link navigation functionality.
// This file handles navigation when pressing Enter on markdown links,
// supporting both regular links and links with line numbers (e.g., file.go#L42).
package editor

import (
	"regexp"
	"strconv"
	"strings"
	"veda/internal/buffer"
	"veda/internal/events"
	"veda/internal/proxy"
)

// markdownLinkRegex matches markdown links [text](url)
var markdownLinkRegex = regexp.MustCompile(`\[([^\[\]]*(?:\[[^\[\]]*\][^\[\]]*)*)\]\(([^)]+)\)`)

// jumpToReference is a command wrapper that checks if we're in a references buffer or on a markdown link
func (e *Editor) jumpToReference(args []string) error {
	// Check if current buffer is a references buffer
	_, isRefsBuffer := e.referencesBuffers[e.CurrentBuffer().DisplayName()]
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
		// Always create a new window at the far right and open the link
		oldActive := e.activeWindow
		e.createRightmostWindow(e.CurrentBuffer())
		e.lastWindow = oldActive
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
	line := e.CurrentBuffer().CursorIterator.Line()
	cursorX := e.CurrentBuffer().CursorX

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
	// Resolve the link relative to working directory and handle HOST:PATH
	target := e.resolvePath(link)

	// If a buffer already exists (unit tests often precreate), navigate without FS checks
	if _, exists := e.buffers[target]; exists {
		e.pushNavigationHistory()
		if buf, ok := e.buffers[target]; ok {
			e.switchToBuffer(buf)
		}
		if lineNum > 0 {
			tl := lineNum - 1
			e.CurrentBuffer().CursorIterator = buffer.NewBufferIteratorAtLine(e.CurrentBuffer(), tl)
			e.CurrentBuffer().CursorX = 0
			e.scrollToCenter()
		}
		return nil
	}

	// If the target might be a directory, resolve it asynchronously so the
	// status bar can show proxy setup progress (important for remote paths).
	// We optimistically handle directory detection in a goroutine to avoid
	// blocking the main loop while remote proxy starts/copies.
	host, _, isRemote := parsePathComponents(target)
	operationID := "open-link-" + target
	e.startAsyncOperation(operationID, "opening: "+target)
	go func(tgt string, ln int, remote bool, h string, opID string) {
		defer e.endAsyncOperation(opID)

		// Give immediate feedback while we resolve attributes; the proxy
		// callback will also provide granular status like Starting/Updating.
		if remote {
			e.connectionStatus = "Connecting to " + h + "..."
		} else {
			e.connectionStatus = "Loading..."
		}
		// Nudge the main loop to render the status immediately
		e.asyncEvents <- events.NewCustomEvent("spinner-tick", nil)

		attrs, err := e.proxyManager.GetFileAttributes(tgt)

		// Clear transient status once the attribute check completes
		e.connectionStatus = ""
		e.asyncEvents <- events.NewCustomEvent("render", nil)

		// Check if file doesn't exist - show notification for deleted files
		if err != nil {
			e.Post(func(ed *Editor) {
				ed.ShowInfo("File does not exist (deleted): %s", tgt)
			})
			return
		}

		if attrs != nil && attrs.IsDirectory {
			// Switch working directory and show the fuzzy list on the main loop
			e.Post(func(ed *Editor) {
				ed.workingDirectory = tgt
				ed.ShowFuzzyListWidget()
			})
			return
		}

		// Not a directory – open as a file at optional line
		targetLine := 0
		if ln > 0 {
			targetLine = ln - 1
		}
		location := proxy.Location{
			URI: "file://" + tgt,
			Range: proxy.Range{
				Start: proxy.Position{Line: targetLine, Character: 0},
				End:   proxy.Position{Line: targetLine, Character: 0},
			},
		}
		// Perform the jump on the main loop for safety
		e.Post(func(ed *Editor) {
			_ = ed.jumpToLocation(location)
		})
	}(target, lineNum, isRemote, host, operationID)

	// We return immediately; completion is handled asynchronously.
	return nil
}
