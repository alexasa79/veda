package theme

import (
	"fmt"
	"regexp"
	"strconv"

	"github.com/gdamore/tcell/v2"
)

// StyleDef represents a complete style definition including color and attributes
type StyleDef struct {
	Foreground string
	Background string
}

// ParsedStyle represents a parsed style with color and attributes
type ParsedStyle struct {
	Color      tcell.Color
	Bold       bool
	Italic     bool
	Underline  bool
}

// ToStyle converts ParsedStyle to tcell.Style
func (ps ParsedStyle) ToStyle() tcell.Style {
	style := tcell.StyleDefault.Foreground(ps.Color)
	if ps.Bold {
		style = style.Bold(true)
	}
	if ps.Italic {
		style = style.Italic(true)
	}
	if ps.Underline {
		style = style.Underline(true)
	}
	return style
}

// colorNotationRegex matches color notations like #RGB, #RGBA, #RRGGBB, #RRGGBBAA with optional [BUI]
var colorNotationRegex = regexp.MustCompile(`^#([0-9A-Fa-f]{3}|[0-9A-Fa-f]{4}|[0-9A-Fa-f]{6}|[0-9A-Fa-f]{8})(?:\[([BUI]+)\])?$`)

// ParseColorNotation parses color notation strings
// Formats supported:
// - "#RGB" - 3 digit hex (expanded to RRGGBB)
// - "#RGBA" - 4 digit hex (expanded to RRGGBBAA, alpha ignored in terminal)
// - "#RRGGBB" - 6 digit hex
// - "#RRGGBBAA" - 8 digit hex with alpha (alpha ignored in terminal)
// - Any of above with "[BUI]" suffix for Bold, Underline, Italic
// Examples: "#FFF", "#FFFA", "#FFFFFF", "#FFFFFFAA", "#FFF[B]", "#FFFFFF[BUI]"
func ParseColorNotation(notation string) (ParsedStyle, error) {
	result := ParsedStyle{
		Color: tcell.ColorDefault,
	}

	// Parse hex notation
	matches := colorNotationRegex.FindStringSubmatch(notation)
	if matches == nil {
		return result, fmt.Errorf("invalid color notation: %s", notation)
	}

	hex := matches[1]
	attributes := matches[2]

	// Parse hex color
	var r, g, b uint8

	switch len(hex) {
	case 3: // #RGB
		rr, _ := strconv.ParseUint(hex[0:1], 16, 8)
		gg, _ := strconv.ParseUint(hex[1:2], 16, 8)
		bb, _ := strconv.ParseUint(hex[2:3], 16, 8)
		// Expand single digits (F -> FF)
		r = uint8(rr<<4 | rr)
		g = uint8(gg<<4 | gg)
		b = uint8(bb<<4 | bb)
	case 4: // #RGBA
		rr, _ := strconv.ParseUint(hex[0:1], 16, 8)
		gg, _ := strconv.ParseUint(hex[1:2], 16, 8)
		bb, _ := strconv.ParseUint(hex[2:3], 16, 8)
		// aa is parsed but ignored for terminal colors
		// Expand single digits (F -> FF)
		r = uint8(rr<<4 | rr)
		g = uint8(gg<<4 | gg)
		b = uint8(bb<<4 | bb)
	case 6: // #RRGGBB
		rr, _ := strconv.ParseUint(hex[0:2], 16, 8)
		gg, _ := strconv.ParseUint(hex[2:4], 16, 8)
		bb, _ := strconv.ParseUint(hex[4:6], 16, 8)
		r = uint8(rr)
		g = uint8(gg)
		b = uint8(bb)
	case 8: // #RRGGBBAA
		rr, _ := strconv.ParseUint(hex[0:2], 16, 8)
		gg, _ := strconv.ParseUint(hex[2:4], 16, 8)
		bb, _ := strconv.ParseUint(hex[4:6], 16, 8)
		// aa is parsed but ignored for terminal colors
		r = uint8(rr)
		g = uint8(gg)
		b = uint8(bb)
	}

	// tcell uses RGB values directly
	result.Color = tcell.NewRGBColor(int32(r), int32(g), int32(b))

	// Parse attributes
	for _, attr := range attributes {
		switch attr {
		case 'B':
			result.Bold = true
		case 'U':
			result.Underline = true
		case 'I':
			result.Italic = true
		}
	}

	return result, nil
}

// CombineStyles creates a tcell.Style from foreground and background ParsedStyles
func CombineStyles(fg, bg ParsedStyle) tcell.Style {
	style := tcell.StyleDefault.Foreground(fg.Color).Background(bg.Color)
	
	// Apply foreground attributes (typically text attributes apply to foreground)
	if fg.Bold {
		style = style.Bold(true)
	}
	if fg.Italic {
		style = style.Italic(true)
	}
	if fg.Underline {
		style = style.Underline(true)
	}
	
	return style
}

// ParseStyleDef parses a StyleDef and returns a combined tcell.Style
func ParseStyleDef(def StyleDef) (tcell.Style, error) {
	fg, err := ParseColorNotation(def.Foreground)
	if err != nil {
		return tcell.StyleDefault, fmt.Errorf("parsing foreground: %w", err)
	}

	bg, err := ParseColorNotation(def.Background)
	if err != nil {
		return tcell.StyleDefault, fmt.Errorf("parsing background: %w", err)
	}

	return CombineStyles(fg, bg), nil
}