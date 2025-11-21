package editor

import (
	"fmt"
	"path/filepath"
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

	content := e.buildDiagnosticsContent(diagnostics)

	// Save current buffer state
	e.saveCurrentBufferState()

	// Create or update diagnostics buffer
	bufferName := "*Diagnostics*"

	if buf, exists := e.buffers[bufferName]; exists {
		// Update existing buffer's content
		buf.Rope = buffer.NewRopeFromString(content)
		buf.CursorX = 0
		buf.ViewIterator = buffer.NewBufferIterator(buf)
		buf.CursorIterator = buffer.NewBufferIterator(buf)
		buf.IsInternal = true
		buf.RealPath = ""
		e.switchToBuffer(buf)
	} else {
		// Create new buffer
		newBuffer := buffer.NewBuffer(content, bufferName, bufferName, e.config.DefaultIndentType, e.config.DefaultIndentWidth)
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

func (e *Editor) buildDiagnosticsContent(diagnostics map[string][]interface{}) string {
	var content strings.Builder
	content.WriteString("# Project Diagnostics\n\n")

	if len(diagnostics) == 0 {
		content.WriteString("No diagnostics found.\n")
		return content.String()
	}

	totalCount := 0
	for _, diags := range diagnostics {
		totalCount += len(diags)
	}
	content.WriteString(fmt.Sprintf("Found %d diagnostics in %d files:\n\n", totalCount, len(diagnostics)))

	for uri, diags := range diagnostics {
		if len(diags) == 0 {
			continue
		}

		filePath := uri
		if strings.HasPrefix(uri, "file://") {
			filePath = uri[7:]
		}
		resolvedPath := e.normalizeDiagnosticPath(filePath)
		content.WriteString(fmt.Sprintf("## %s\n\n", resolvedPath))

		for _, diag := range diags {
			diagMap, ok := diag.(map[string]interface{})
			if !ok {
				continue
			}

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

			displayName := filepath.Base(resolvedPath)
			display := fmt.Sprintf("%s:%d:%d", displayName, line+1, character+1)
			link := fmt.Sprintf("%s#L%d", resolvedPath, line+1)
			content.WriteString(fmt.Sprintf("[%s](%s) %s: %s\n", display, link, severity, message))
		}

		content.WriteString("\n")
	}

	return content.String()
}

func (e *Editor) normalizeDiagnosticPath(path string) string {
	resolved := path
	if resolved == "" {
		return resolved
	}

	if !strings.HasPrefix(resolved, "/") && !strings.Contains(resolved, ":") {
		resolved = e.resolvePath(resolved)
	}

	if strings.Contains(e.workingDirectory, ":") && !strings.Contains(resolved, ":") {
		if idx := strings.Index(e.workingDirectory, ":"); idx != -1 {
			host := e.workingDirectory[:idx]
			if host != "" {
				resolved = host + ":" + resolved
			}
		}
	}

	return resolved
}
