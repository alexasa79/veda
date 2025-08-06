package editor

import (
	"regexp"
	"strings"
	"veda/internal/commands"
	"veda/internal/syntax"
)

// markdownLinkRegex matches markdown links [text](url)
// This regex handles nested brackets in the link text by matching balanced brackets
var markdownLinkRegex = regexp.MustCompile(`\[([^\[\]]*(?:\[[^\[\]]*\][^\[\]]*)*)\]\(([^)]+)\)`)

// markdownBoldRegex matches bold text **text** or __text__
var markdownBoldRegex = regexp.MustCompile(`\*\*([^*]+)\*\*|__([^_]+)__`)

// MarkdownRenderInfo holds information about how to render markdown content
type MarkdownRenderInfo struct {
	// DisplayText is what should be shown on screen
	DisplayText string
	// OriginalToDisplay maps original character positions to display positions
	OriginalToDisplay map[int]int
	// DisplayToOriginal maps display character positions to original positions
	DisplayToOriginal map[int]int
	// TokenAdjustments maps original token positions to adjusted positions
	TokenAdjustments []TokenAdjustment
}

// TokenAdjustment describes how to adjust a token for display
type TokenAdjustment struct {
	OriginalStart int
	OriginalEnd   int
	DisplayStart  int
	DisplayEnd    int
	Hidden        bool // If true, this token should not be displayed
}

// ProcessMarkdownLine processes a line of markdown text for mode-aware rendering
func ProcessMarkdownLine(line string, mode commands.Mode) *MarkdownRenderInfo {
	if mode == commands.InsertMode {
		// In insert mode, show everything as-is
		return &MarkdownRenderInfo{
			DisplayText:       line,
			OriginalToDisplay: identityMap(len(line)),
			DisplayToOriginal: identityMap(len(line)),
			TokenAdjustments:  nil,
		}
	}

	// In all modes except insert, hide link URLs and remove bold markers
	info := &MarkdownRenderInfo{
		OriginalToDisplay: make(map[int]int),
		DisplayToOriginal: make(map[int]int),
		TokenAdjustments:  make([]TokenAdjustment, 0),
	}

	var displayText strings.Builder
	displayPos := 0
	
	// Process the line character by character
	i := 0
	for i < len(line) {
		// Check for markdown links
		if line[i] == '[' {
			// Try to find a complete link pattern
			linkMatch := markdownLinkRegex.FindStringSubmatchIndex(line[i:])
			if linkMatch != nil {
				// We found a link!
				// linkMatch indices are relative to line[i:]
				linkStart := i + linkMatch[0]
				linkEnd := i + linkMatch[1]
				textStart := i + linkMatch[2]
				textEnd := i + linkMatch[3]
				
				// Map the opening bracket positions to the start of link text
				for j := linkStart; j < textStart; j++ {
					info.OriginalToDisplay[j] = displayPos
				}
				
				// Copy and map the link text
				for j := textStart; j < textEnd; j++ {
					displayText.WriteByte(line[j])
					info.OriginalToDisplay[j] = displayPos
					info.DisplayToOriginal[displayPos] = j
					displayPos++
				}
				
				// Map the closing bracket and URL to the end of link text
				for j := textEnd; j < linkEnd; j++ {
					info.OriginalToDisplay[j] = displayPos
				}
				
				// Skip to the end of the link
				i = linkEnd
				continue
			}
		}
		
		// Regular character - copy and map
		displayText.WriteByte(line[i])
		info.OriginalToDisplay[i] = displayPos
		info.DisplayToOriginal[displayPos] = i
		displayPos++
		i++
	}
	
	info.DisplayText = displayText.String()
	return info
}

// AdjustTokenForMarkdown adjusts a syntax token based on markdown rendering info
func AdjustTokenForMarkdown(token syntax.Token, renderInfo *MarkdownRenderInfo) (syntax.Token, bool) {
	if renderInfo == nil {
		return token, true
	}
	
	// Find the display positions for the token boundaries
	// We need to handle tokens that span across hidden text
	
	// Find the first visible position at or after token.Start
	displayStart := -1
	for pos := token.Start; pos < token.End; pos++ {
		if mappedPos, ok := renderInfo.OriginalToDisplay[pos]; ok {
			displayStart = mappedPos
			break
		}
	}
	
	// If no visible start position found, token is completely hidden
	if displayStart == -1 {
		return token, false
	}
	
	// Find the last visible position before token.End
	displayEnd := displayStart
	for pos := token.Start; pos < token.End; pos++ {
		if mappedPos, ok := renderInfo.OriginalToDisplay[pos]; ok {
			displayEnd = mappedPos + 1
		}
	}
	
	// Create adjusted token with mapped positions
	adjustedToken := token
	adjustedToken.Start = displayStart
	adjustedToken.End = displayEnd
	
	// Don't return empty tokens
	if adjustedToken.Start >= adjustedToken.End {
		return token, false
	}
	
	return adjustedToken, true
}

// identityMap creates a map where each position maps to itself
func identityMap(length int) map[int]int {
	m := make(map[int]int, length)
	for i := 0; i < length; i++ {
		m[i] = i
	}
	return m
}

// MapCursorPosition maps a cursor position from original to display coordinates
func (info *MarkdownRenderInfo) MapCursorPosition(originalPos int) int {
	if info == nil {
		return originalPos
	}
	
	if displayPos, ok := info.OriginalToDisplay[originalPos]; ok {
		return displayPos
	}
	
	// If exact position not found, find the closest mapped position
	bestPos := originalPos
	for origPos, dispPos := range info.OriginalToDisplay {
		if origPos <= originalPos && origPos > bestPos {
			bestPos = dispPos
		}
	}
	
	return bestPos
}