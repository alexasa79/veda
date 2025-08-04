package editor

import (
	"testing"
	"veda/internal/commands"
)

func TestProcessMarkdownLine(t *testing.T) {
	tests := []struct {
		name         string
		line         string
		mode         commands.Mode
		expectedText string
		description  string
	}{
		{
			name:         "simple_link_normal_mode",
			line:         "Check out [this link](https://example.com) for more info",
			mode:         commands.NormalMode,
			expectedText: "Check out this link for more info",
			description:  "In normal mode, only link text should be shown",
		},
		{
			name:         "simple_link_insert_mode",
			line:         "Check out [this link](https://example.com) for more info",
			mode:         commands.InsertMode,
			expectedText: "Check out [this link](https://example.com) for more info",
			description:  "In insert mode, full markdown syntax should be shown",
		},
		{
			name:         "multiple_links_normal_mode",
			line:         "Visit [Google](https://google.com) or [GitHub](https://github.com)",
			mode:         commands.NormalMode,
			expectedText: "Visit Google or GitHub",
			description:  "Multiple links should show only text parts in normal mode",
		},
		{
			name:         "link_at_start",
			line:         "[Home](/) page is here",
			mode:         commands.NormalMode,
			expectedText: "Home page is here",
			description:  "Link at start of line should render correctly",
		},
		{
			name:         "link_at_end",
			line:         "Click here for [details](http://example.com/details)",
			mode:         commands.NormalMode,
			expectedText: "Click here for details",
			description:  "Link at end of line should render correctly",
		},
		{
			name:         "empty_link_text",
			line:         "Empty link [](https://example.com) here",
			mode:         commands.NormalMode,
			expectedText: "Empty link  here",
			description:  "Empty link text should render as empty space",
		},
		{
			name:         "nested_brackets",
			line:         "Link with [brackets [inside]](https://example.com) text",
			mode:         commands.NormalMode,
			expectedText: "Link with brackets [inside] text",
			description:  "Nested brackets in link text should be preserved",
		},
		{
			name:         "no_links",
			line:         "This is plain text without any links",
			mode:         commands.NormalMode,
			expectedText: "This is plain text without any links",
			description:  "Plain text should remain unchanged",
		},
		{
			name:         "incomplete_link",
			line:         "This has an incomplete [link",
			mode:         commands.NormalMode,
			expectedText: "This has an incomplete [link",
			description:  "Incomplete markdown should remain unchanged",
		},
		{
			name:         "visual_mode_hides_urls",
			line:         "Check [link](https://example.com) in visual mode",
			mode:         commands.VisualMode,
			expectedText: "Check link in visual mode",
			description:  "Visual mode should also hide URLs like normal mode",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Process the markdown line
			renderInfo := ProcessMarkdownLine(tt.line, tt.mode)
			
			// Check the display text
			if renderInfo.DisplayText != tt.expectedText {
				t.Errorf("%s\nInput:    %q\nExpected: %q\nGot:      %q",
					tt.description, tt.line, tt.expectedText, renderInfo.DisplayText)
			}
			
			// Verify mapping integrity
			if tt.mode != commands.InsertMode {
				// In non-insert modes, verify that mappings are consistent
				for origPos, displayPos := range renderInfo.OriginalToDisplay {
					if origPos < 0 || origPos > len(tt.line) {
						t.Errorf("Invalid original position in mapping: %d (line length: %d)", origPos, len(tt.line))
					}
					if displayPos < 0 || displayPos > len(renderInfo.DisplayText) {
						t.Errorf("Invalid display position in mapping: %d (display length: %d)", displayPos, len(renderInfo.DisplayText))
					}
				}
			}
		})
	}
}

func TestMarkdownCursorMapping(t *testing.T) {
	// Test cursor position mapping for markdown links
	line := "Text before [link text](https://url.com) text after"
	renderInfo := ProcessMarkdownLine(line, commands.NormalMode)
	
	testCases := []struct {
		originalPos  int
		expectedPos  int
		description  string
	}{
		{0, 0, "Start of line"},
		{11, 11, "Just before opening bracket"},
		{12, 12, "Opening bracket position maps to start of link text"},
		{13, 12, "First char of link text"},
		{22, 21, "Last char of link text"},
		{23, 21, "Closing bracket position maps after link text"},
		{24, 21, "Opening paren position maps after link text"},
		{40, 21, "Closing paren position maps after link text"},
		{41, 22, "First char after link"},
		{50, 31, "Last char of line"},
	}
	
	for _, tc := range testCases {
		mapped := renderInfo.MapCursorPosition(tc.originalPos)
		if mapped != tc.expectedPos {
			t.Errorf("%s: cursor at original pos %d should map to %d, got %d",
				tc.description, tc.originalPos, tc.expectedPos, mapped)
		}
	}
}

