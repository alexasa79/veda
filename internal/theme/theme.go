package theme

import "github.com/gdamore/tcell/v2"

// Theme represents a color theme for the editor
type Theme struct {
	Name string

	// Editor colors
	StatusBar   StyleDef
	CommandLine StyleDef

	// Dialog colors
	DialogBorder  StyleDef
	DialogContent StyleDef

	// Widget colors
	FuzzyListBorder    StyleDef
	FuzzyListContent   StyleDef
	FuzzyListSelection StyleDef

	// Text editor colors
	Normal      StyleDef
	Selection   StyleDef
	LineNumber  StyleDef
	CurrentLine StyleDef
}

// GetStyle returns a tcell.Style for the given style definition
func (t *Theme) GetStyle(def StyleDef) tcell.Style {
	style, err := ParseStyleDef(def)
	if err != nil {
		// Fallback to default style on error
		return tcell.StyleDefault
	}
	return style
}

// NewTheme returns the theme
func NewTheme() *Theme {
	return &Theme{
		Name: "main",

		StatusBar: StyleDef{
			Foreground: "#000",
			Background: "#fff",
		},

		CommandLine: StyleDef{
			Foreground: "#000",
			Background: "#fff",
		},

		DialogBorder: StyleDef{
			Foreground: "#333",
			Background: "#ddd",
		},

		DialogContent: StyleDef{
			Foreground: "#000",
			Background: "#fff",
		},

		FuzzyListBorder: StyleDef{
			Foreground: "#b8b9b4",
			Background: "#26292c",
		},

		FuzzyListContent: StyleDef{
			Foreground: "#fff",
			Background: "#000",
		},

		FuzzyListSelection: StyleDef{
			Foreground: "#000",
			Background: "#0af",
		},

		Normal: StyleDef{
			Foreground: "#f8f8f1",
			Background: "#26292c",
		},

		Selection: StyleDef{
			Foreground: "#000",
			Background: "#ff0",
		},

		LineNumber: StyleDef{
			Foreground: "#75715e",
			Background: "#26292c",
		},

		CurrentLine: StyleDef{
			Foreground: "#000",
			Background: "#eee",
		},
	}
}