package theme

import "veda/internal/ui"

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
	Normal           StyleDef
	Selection        StyleDef
	LineNumber       StyleDef
	CurrentLine      StyleDef
	LineColumn       StyleDef // Line and column display in status bar
	Context StyleDef // Style for context lines and signature help
	SearchHighlight  StyleDef // Style for search matches

	// Syntax highlighting colors (Monokai theme)
	SyntaxText        StyleDef
	SyntaxKeyword     StyleDef
	SyntaxFunction    StyleDef
	SyntaxVariable    StyleDef
	SyntaxConstant    StyleDef
	SyntaxString      StyleDef
	SyntaxComment     StyleDef
	SyntaxNumber      StyleDef
	SyntaxOperator    StyleDef
	SyntaxPunctuation StyleDef
	SyntaxType        StyleDef
	SyntaxNamespace   StyleDef
	SyntaxAttribute   StyleDef
	SyntaxTag         StyleDef
}

// GetStyle returns a ui.Style for the given style definition
func (t *Theme) GetStyle(def StyleDef) ui.Style {
	style, err := ParseStyleDef(def)
	if err != nil {
		// Fallback to default style on error
		return ui.StyleDefault
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
			Foreground: "#f8f8f2",
			Background: "#272822",
		},

		Selection: StyleDef{
			Foreground: "#000",
			Background: "#ff0",
		},

		LineNumber: StyleDef{
			Foreground: "#75715e",
			Background: "#272822",
		},

		CurrentLine: StyleDef{
			Foreground: "#f8f8f2",
			Background: "#3e3d32", // Slightly brighter than #272822
		},

		LineColumn: StyleDef{
			Foreground: "#000000[B]",
			Background: "#a0a0a0", // Darker gray background to make it stand out
		},

		Context: StyleDef{
			Foreground: "#f8f8f2",
			Background: "#000000", // Darker than normal background
		},

		SearchHighlight: StyleDef{
			Foreground: "#f8f8f2",
			Background: "#5a5a52", // Brighter grey background for search matches
		},

		// Monokai syntax highlighting colors
		SyntaxText: StyleDef{
			Foreground: "#f8f8f2",
			Background: "#272822",
		},

		SyntaxKeyword: StyleDef{
			Foreground: "#f92672",
			Background: "#272822",
		},

		SyntaxFunction: StyleDef{
			Foreground: "#a6e22e",
			Background: "#272822",
		},

		SyntaxVariable: StyleDef{
			Foreground: "#f8f8f2",
			Background: "#272822",
		},

		SyntaxConstant: StyleDef{
			Foreground: "#ae81ff",
			Background: "#272822",
		},

		SyntaxString: StyleDef{
			Foreground: "#e6db74",
			Background: "#272822",
		},

		SyntaxComment: StyleDef{
			Foreground: "#75715e",
			Background: "#272822",
		},

		SyntaxNumber: StyleDef{
			Foreground: "#ae81ff",
			Background: "#272822",
		},

		SyntaxOperator: StyleDef{
			Foreground: "#f92672",
			Background: "#272822",
		},

		SyntaxPunctuation: StyleDef{
			Foreground: "#f8f8f2",
			Background: "#272822",
		},

		SyntaxType: StyleDef{
			Foreground: "#66d9ef[I]",
			Background: "#272822",
		},

		SyntaxNamespace: StyleDef{
			Foreground: "#a6e22e",
			Background: "#272822",
		},

		SyntaxAttribute: StyleDef{
			Foreground: "#a6e22e",
			Background: "#272822",
		},

		SyntaxTag: StyleDef{
			Foreground: "#f92672",
			Background: "#272822",
		},
	}
}