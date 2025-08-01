package editor

import (
	"regexp"
	"strings"
	"veda/internal/commands"
	"veda/internal/syntax"
)

// markdownLinkRegex matches markdown links [text](url)
var markdownLinkRegex = regexp.MustCompile(`\[([^\]]*)\]\(([^)]+)\)`)

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

	// In all modes except insert, hide link URLs
	info := &MarkdownRenderInfo{
		OriginalToDisplay: make(map[int]int),
		DisplayToOriginal: make(map[int]int),
		TokenAdjustments:  make([]TokenAdjustment, 0),
	}

	var displayText strings.Builder
	displayPos := 0
	lastEnd := 0

	matches := markdownLinkRegex.FindAllStringSubmatchIndex(line, -1)
	
	for _, match := range matches {
		// match[0], match[1] = full match start/end
		// match[2], match[3] = link text start/end
		// match[4], match[5] = url start/end
		
		linkStart := match[0]
		linkEnd := match[1]
		textStart := match[2]
		textEnd := match[3]
		
		// Copy text before the link
		for i := lastEnd; i < linkStart; i++ {
			displayText.WriteByte(line[i])
			info.OriginalToDisplay[i] = displayPos
			info.DisplayToOriginal[displayPos] = i
			displayPos++
		}
		
		// Copy only the link text (without brackets)
		linkText := line[textStart:textEnd]
		displayTextStart := displayPos
		
		for i, ch := range linkText {
			displayText.WriteRune(ch)
			origPos := textStart + i
			info.OriginalToDisplay[origPos] = displayPos
			info.DisplayToOriginal[displayPos] = origPos
			displayPos++
		}
		
		// Record adjustments for tokens
		// Hide the opening bracket
		info.TokenAdjustments = append(info.TokenAdjustments, TokenAdjustment{
			OriginalStart: linkStart,
			OriginalEnd:   textStart,
			Hidden:        true,
		})
		
		// Adjust the link text position
		info.TokenAdjustments = append(info.TokenAdjustments, TokenAdjustment{
			OriginalStart: textStart,
			OriginalEnd:   textEnd,
			DisplayStart:  displayTextStart,
			DisplayEnd:    displayPos,
		})
		
		// Hide everything from closing bracket to end of link
		info.TokenAdjustments = append(info.TokenAdjustments, TokenAdjustment{
			OriginalStart: textEnd,
			OriginalEnd:   linkEnd,
			Hidden:        true,
		})
		
		// Map hidden characters to the position after the link text
		for i := linkStart; i < textStart; i++ {
			info.OriginalToDisplay[i] = displayTextStart
		}
		for i := textEnd; i < linkEnd; i++ {
			info.OriginalToDisplay[i] = displayPos
		}
		
		lastEnd = linkEnd
	}
	
	// Copy remaining text
	for i := lastEnd; i < len(line); i++ {
		displayText.WriteByte(line[i])
		info.OriginalToDisplay[i] = displayPos
		info.DisplayToOriginal[displayPos] = i
		displayPos++
	}
	
	info.DisplayText = displayText.String()
	return info
}

// AdjustTokenForMarkdown adjusts a syntax token based on markdown rendering info
func AdjustTokenForMarkdown(token syntax.Token, renderInfo *MarkdownRenderInfo) (syntax.Token, bool) {
	if renderInfo == nil || len(renderInfo.TokenAdjustments) == 0 {
		return token, true
	}
	
	// Check if this token should be hidden
	for _, adj := range renderInfo.TokenAdjustments {
		if adj.Hidden && token.Start >= adj.OriginalStart && token.End <= adj.OriginalEnd {
			return token, false // Hide this token
		}
	}
	
	// Adjust token positions
	adjustedToken := token
	
	// Map start position
	if newStart, ok := renderInfo.OriginalToDisplay[token.Start]; ok {
		adjustedToken.Start = newStart
	}
	
	// Map end position
	if newEnd, ok := renderInfo.OriginalToDisplay[token.End-1]; ok {
		adjustedToken.End = newEnd + 1
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