package editor

import (
	"strings"
	"testing"
	"veda/internal/commands"
)

func TestMarkdownBoldTextBreaksLinkHighlighting(t *testing.T) {
	// Test the markdown processing function directly
	tests := []struct {
		name         string
		line         string
		expectedText string
		description  string
	}{
		{
			name:         "link_without_bold",
			line:         "M [file.go](file.go) (+4 -13)",
			expectedText: "M file.go (+4 -13)",
			description:  "Link without bold should render correctly",
		},
		{
			name:         "link_with_bold_before",
			line:         "**M** [file.go](file.go) (+4 -13)",
			expectedText: "**M** file.go (+4 -13)",
			description:  "Link with bold text before should process links but keep bold markers",
		},
		{
			name:         "bold_and_multiple_links",
			line:         "**M** [harmony/cgw.hny](harmony/cgw.hny) (+4 -13)",
			expectedText: "**M** harmony/cgw.hny (+4 -13)",
			description:  "Bold text markers are kept, links are processed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Process the markdown line
			info := ProcessMarkdownLine(tt.line, commands.NormalMode)
			
			// Check if the rendered text matches expected
			if info.DisplayText != tt.expectedText {
				t.Errorf("ProcessMarkdownLine() = %q, want %q", info.DisplayText, tt.expectedText)
			}
			
			// Log token adjustments for debugging
			t.Logf("Token adjustments: %+v", info.TokenAdjustments)
			t.Logf("Display text: %q", info.DisplayText)
			
			// Verify that the line was processed correctly
			// Links should be simplified (URL hidden) but bold markers remain
			if strings.Contains(info.DisplayText, "**") && strings.Contains(tt.line, "**") {
				t.Logf("Bold markers preserved in display text")
			}
			
			// Check that links are processed (URL part should be removed)
			if strings.Contains(tt.line, "](") && !strings.Contains(info.DisplayText, "](") {
				t.Logf("Link URLs correctly hidden")
			}
		})
	}
}

// TestMarkdownSyntaxHighlightingWithBold tests that syntax highlighting handles bold + links correctly
func TestMarkdownSyntaxHighlightingWithBold(t *testing.T) {
	// This test would require more complex setup with the syntax highlighter
	// For now, we're focusing on the markdown processing logic
	t.Skip("Syntax highlighting test requires full editor setup")
}

