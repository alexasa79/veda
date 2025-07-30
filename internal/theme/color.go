package theme

import (
	"fmt"
	"regexp"
	"strconv"

	"veda/internal/ui"
)

// StyleDef represents a complete style definition including color and attributes
type StyleDef struct {
	Foreground string
	Background string
}

// ParsedStyle represents a parsed style with color and attributes
type ParsedStyle struct {
	FgColor    ui.Color
	BgColor    ui.Color
	Bold       bool
	Italic     bool
	Underline  bool
}

// ToStyle converts ParsedStyle to ui.Style
func (ps ParsedStyle) ToStyle() ui.Style {
	style := ui.StyleDefault
	style.Fg = ps.FgColor
	style.Bg = ps.BgColor
	
	var attrs ui.AttrMask
	if ps.Bold {
		attrs |= ui.AttrBold
	}
	if ps.Italic {
		attrs |= ui.AttrItalic
	}
	if ps.Underline {
		attrs |= ui.AttrUnderline
	}
	style.Attrs = attrs
	
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
		FgColor: ui.ColorDefault,
		BgColor: ui.ColorDefault,
	}

	// Handle empty string as default color
	if notation == "" {
		return result, nil
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

	// Convert RGB values to ui.Color
	result.FgColor = ui.NewRGBColor(int32(r), int32(g), int32(b))

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

// CombineStyles creates a ui.Style from foreground and background ParsedStyles
func CombineStyles(fg, bg ParsedStyle) ui.Style {
	style := ui.StyleDefault
	style.Fg = fg.FgColor
	style.Bg = bg.BgColor
	
	// Apply foreground attributes (typically text attributes apply to foreground)
	var attrs ui.AttrMask
	if fg.Bold {
		attrs |= ui.AttrBold
	}
	if fg.Italic {
		attrs |= ui.AttrItalic
	}
	if fg.Underline {
		attrs |= ui.AttrUnderline
	}
	style.Attrs = attrs
	
	return style
}

// ParseStyleDef parses a StyleDef and returns a combined ui.Style
func ParseStyleDef(def StyleDef) (ui.Style, error) {
	fg, err := ParseColorNotation(def.Foreground)
	if err != nil {
		return ui.StyleDefault, fmt.Errorf("parsing foreground: %w", err)
	}

	bg, err := ParseColorNotation(def.Background)
	if err != nil {
		return ui.StyleDefault, fmt.Errorf("parsing background: %w", err)
	}

	// Set the background color from bg notation
	bg.BgColor = bg.FgColor
	bg.FgColor = ui.ColorDefault

	return CombineStyles(fg, bg), nil
}

// ApplyBackgroundStyle takes an existing style and applies only the background from another style
// This preserves the original foreground color and attributes
func ApplyBackgroundStyle(originalStyle ui.Style, backgroundStyle ui.Style) ui.Style {
	fg, _, attrs := originalStyle.Decompose()
	_, bg, _ := backgroundStyle.Decompose()
	return originalStyle.Foreground(fg).Background(bg).Attributes(attrs)
}