package editor

import (
    "fmt"
    "strings"
    "veda/internal/buffer"
)

// showDiagnostics displays all project diagnostics in a buffer
func (e *Editor) showDiagnostics(args []string) error {
	// Get diagnostics from the proxy
	diagnostics, err := e.proxyManager.GetDiagnostics(e.workingDirectory)
	if err != nil {
		LL.Error("Failed to get diagnostics: %v", err)
		return fmt.Errorf("failed to get diagnostics: %w", err)
	}

	// Create content for diagnostics buffer
	var content strings.Builder
	content.WriteString("# Project Diagnostics\n\n")

	if len(diagnostics) == 0 {
		content.WriteString("No diagnostics found.\n")
	} else {
		totalCount := 0
		for _, diags := range diagnostics {
			totalCount += len(diags)
		}
		content.WriteString(fmt.Sprintf("Found %d diagnostics in %d files:\n\n", totalCount, len(diagnostics)))

		// Process each file's diagnostics
		for uri, diags := range diagnostics {
			if len(diags) == 0 {
				continue
			}

			// Convert URI to file path
			filePath := uri
			if strings.HasPrefix(uri, "file://") {
				filePath = uri[7:] // Remove "file://" prefix
			}

			// Try to make path relative (but we can't use GetFile on "." anymore)
			// Just use the absolute path for now

			content.WriteString(fmt.Sprintf("## %s\n\n", filePath))

            // Process each diagnostic
            for _, diag := range diags {
                if diagMap, ok := diag.(map[string]interface{}); ok {
                    // Extract diagnostic details
                    severity := "Error"
                    if sev, ok := diagMap["severity"].(float64); ok {
                        switch int(sev) {
                        case 1:
                            severity = "Error"
                        case 2:
                            severity = "Warning"
                        case 3:
                            severity = "Info"
                        case 4:
                            severity = "Hint"
                        }
                    }

                    message := ""
                    if msg, ok := diagMap["message"].(string); ok {
                        message = msg
                    }

                    line := 0
                    character := 0
                    if rangeData, ok := diagMap["range"].(map[string]interface{}); ok {
                        if start, ok := rangeData["start"].(map[string]interface{}); ok {
                            if l, ok := start["line"].(float64); ok {
                                line = int(l)
                            }
                            if c, ok := start["character"].(float64); ok {
                                character = int(c)
                            }
                        }
                    }

                    // Create a proper markdown link to the file and line
                    // Display text uses just the filename for brevity, with line:col
                    displayName := filePath
                    if idx := strings.LastIndex(displayName, "/"); idx != -1 && idx+1 < len(displayName) {
                        displayName = displayName[idx+1:]
                    }
                    display := fmt.Sprintf("%s:%d:%d", displayName, line+1, character+1)
                    link := fmt.Sprintf("%s#L%d", filePath, line+1)
                    // Example: [main.go:12:5](/abs/path/main.go#L12) Warning: message
                    content.WriteString(fmt.Sprintf("[%s](%s) %s: %s\n", display, link, severity, message))
                }
            }
			content.WriteString("\n")
		}
	}

	// Save current buffer state
	e.saveCurrentBufferState()

	// Create or update diagnostics buffer
	bufferName := "*Diagnostics*"

	if buf, exists := e.buffers[bufferName]; exists {
		// Update existing buffer's content
		buf.Rope = buffer.NewRopeFromString(content.String())
		buf.CursorX = 0
		buf.ViewIterator = buffer.NewBufferIterator(buf)
		buf.CursorIterator = buffer.NewBufferIterator(buf)
		buf.IsInternal = true
		buf.RealPath = ""
		e.switchToBuffer(buf)
	} else {
		// Create new buffer
		newBuffer := buffer.NewBuffer(content.String(), bufferName, e.config.DefaultIndentType, e.config.DefaultIndentWidth)
		newBuffer.IsInternal = true
		newBuffer.RealPath = ""
		e.buffers[bufferName] = newBuffer
		e.bufferOrder = append(e.bufferOrder, bufferName)
		e.switchToBuffer(newBuffer)
	}

	// Clear any existing search
	e.clearSearch()

	return nil
}
