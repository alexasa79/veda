package editor

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"
)

func TestDiagnosticsLinksUseResolvedLocalPaths(t *testing.T) {
	te := NewTestEditor(80, 24)
	tempDir := t.TempDir()
	te.Editor.workingDirectory = tempDir

	diagnostics := map[string][]interface{}{
		"file://foo.go": {
			map[string]interface{}{
				"message":  "local issue",
				"severity": float64(1),
				"range": map[string]interface{}{
					"start": map[string]interface{}{
						"line":      float64(2),
						"character": float64(4),
					},
				},
			},
		},
	}

	content := te.Editor.buildDiagnosticsContent(diagnostics)
	expectedPath := filepath.Join(tempDir, "foo.go")
	expectedLink := fmt.Sprintf("[foo.go:3:5](%s#L3) Error: local issue", expectedPath)
	if !strings.Contains(content, expectedLink) {
		t.Fatalf("expected diagnostics content to include %q, got:\n%s", expectedLink, content)
	}
}

func TestDiagnosticsLinksIncludeRemoteHostPrefix(t *testing.T) {
	te := NewTestEditor(80, 24)
	te.Editor.workingDirectory = "remote:/home/user/project"

	diagnostics := map[string][]interface{}{
		"file:///home/user/project/main.go": {
			map[string]interface{}{
				"message":  "remote issue",
				"severity": float64(2),
				"range": map[string]interface{}{
					"start": map[string]interface{}{
						"line":      float64(9),
						"character": float64(0),
					},
				},
			},
		},
	}

	content := te.Editor.buildDiagnosticsContent(diagnostics)
	expectedLink := "[main.go:10:1](remote:/home/user/project/main.go#L10) Warning: remote issue"
	if !strings.Contains(content, expectedLink) {
		t.Fatalf("expected diagnostics content to include %q, got:\n%s", expectedLink, content)
	}
}
