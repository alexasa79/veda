package theme

import "veda/internal/ui"

// Theme represents a color theme for the editor
type Theme struct {
	Name string

	// Editor colors
	StatusBar       StyleDef
	StatusBarSubdued StyleDef
	CommandLine     StyleDef

	// Dialog colors
	DialogBorder  StyleDef
	DialogContent StyleDef

	// Widget colors
	FuzzyListBorder    StyleDef
	FuzzyListContent   StyleDef
	FuzzyListSelection StyleDef
	
	// Completion menu colors
	CompletionMenu         StyleDef
	CompletionMenuSelected StyleDef

	// Text editor colors
	Normal           StyleDef
	Selection        StyleDef
	LineNumber       StyleDef
	CurrentLine      StyleDef
	Context StyleDef // Style for context lines and signature help
	SearchHighlight  StyleDef // Style for search matches
	FlashHighlight   StyleDef // Style for flashing yanked/changed text

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

	// Markdown-specific styles
	MarkdownH1        StyleDef
	MarkdownH2        StyleDef
	MarkdownH3        StyleDef
	MarkdownH4        StyleDef
	MarkdownH5        StyleDef
	MarkdownH6        StyleDef
	MarkdownBold      StyleDef
	MarkdownItalic    StyleDef
	MarkdownBoldItalic StyleDef
	MarkdownCode      StyleDef
	MarkdownCodeBlock StyleDef
	MarkdownBlockquote StyleDef
	MarkdownLink      StyleDef
	MarkdownLinkText  StyleDef
	MarkdownLinkUrl   StyleDef
	MarkdownImage     StyleDef
	MarkdownList      StyleDef
	MarkdownListMarker StyleDef
	MarkdownHorizontalRule StyleDef
	MarkdownStrikethrough StyleDef
	MarkdownTable     StyleDef
	MarkdownTableBorder StyleDef
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

		StatusBarSubdued: StyleDef{
			Foreground: "#666",
			Background: "#ddd",
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
			Foreground: "#f8f8f2",
			Background: "#364462", // Muted blue-grey for better visibility
		},

		LineNumber: StyleDef{
			Foreground: "#75715e",
			Background: "#272822",
		},

		CurrentLine: StyleDef{
			Foreground: "#f8f8f2",
			Background: "#3e3d32", // Slightly brighter than #272822
		},

		Context: StyleDef{
			Foreground: "#f8f8f2",
			Background: "#000000", // Darker than normal background
		},

		SearchHighlight: StyleDef{
			Foreground: "#f8f8f2",
			Background: "#5a5a52", // Brighter grey background for search matches
		},

		FlashHighlight: StyleDef{
			Foreground: "#000000",
			Background: "#9c9c9c",
		},
		
		CompletionMenu: StyleDef{
			Foreground: "#f8f8f2", // Bright foreground
			Background: "#1a1a1a", // Dark background
		},
		
		CompletionMenuSelected: StyleDef{
			Foreground: "#000000", // Black text
			Background: "#66d9ef", // Bright cyan background for selection
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

		// Markdown styles
		MarkdownH1: StyleDef{
			Foreground: "#f92672[B]",
			Background: "#272822",
		},

		MarkdownH2: StyleDef{
			Foreground: "#ae81ff[B]",
			Background: "#272822",
		},

		MarkdownH3: StyleDef{
			Foreground: "#a6e22e[B]",
			Background: "#272822",
		},

		MarkdownH4: StyleDef{
			Foreground: "#66d9ef[B]",
			Background: "#272822",
		},

		MarkdownH5: StyleDef{
			Foreground: "#e6db74[B]",
			Background: "#272822",
		},

		MarkdownH6: StyleDef{
			Foreground: "#fd971f[B]",
			Background: "#272822",
		},

		MarkdownBold: StyleDef{
			Foreground: "#f8f8f2[B]",
			Background: "#272822",
		},

		MarkdownItalic: StyleDef{
			Foreground: "#f8f8f2[I]",
			Background: "#272822",
		},

		MarkdownBoldItalic: StyleDef{
			Foreground: "#f8f8f2[BI]",
			Background: "#272822",
		},

		MarkdownCode: StyleDef{
			Foreground: "#e6db74",
			Background: "#3e3d32",
		},

		MarkdownCodeBlock: StyleDef{
			Foreground: "#e6db74",
			Background: "#3e3d32",
		},

		MarkdownBlockquote: StyleDef{
			Foreground: "#24ac15ff[I]",
			Background: "#272822",
		},

		MarkdownLink: StyleDef{
			Foreground: "#66d9ef[U]",
			Background: "#272822",
		},

		MarkdownLinkText: StyleDef{
			Foreground: "#66d9ef[U]",
			Background: "#272822",
		},

		MarkdownLinkUrl: StyleDef{
			Foreground: "#ae81ff",
			Background: "#272822",
		},

		MarkdownImage: StyleDef{
			Foreground: "#a6e22e",
			Background: "#272822",
		},

		MarkdownList: StyleDef{
			Foreground: "#f8f8f2",
			Background: "#272822",
		},

		MarkdownListMarker: StyleDef{
			Foreground: "#a2a2a2",
			Background: "#272822",
		},

		MarkdownHorizontalRule: StyleDef{
			Foreground: "#75715e",
			Background: "#272822",
		},

		MarkdownStrikethrough: StyleDef{
			Foreground: "#75715e",
			Background: "#272822",
		},

		MarkdownTable: StyleDef{
			Foreground: "#f8f8f2",
			Background: "#272822",
		},

		MarkdownTableBorder: StyleDef{
			Foreground: "#75715e",
			Background: "#272822",
		},
	}
}