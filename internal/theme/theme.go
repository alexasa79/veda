package theme

import "veda/internal/ui"

// Theme represents a color theme for the editor
type Theme struct {
	Name string

	// Editor colors
	StatusBar        StyleDef
	StatusBarSubdued StyleDef
	CommandLine      StyleDef

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
	Normal          StyleDef
	Selection       StyleDef
	LineNumber      StyleDef
	CurrentLine     StyleDef
	Context         StyleDef // Style for context lines and signature help
	SearchHighlight StyleDef // Style for search matches
	FlashHighlight  StyleDef // Style for flashing yanked/changed text
	Whitespace      StyleDef // Style for whitespace indicators (tabs, trailing spaces)
	VerticalRuler   StyleDef // Style for vertical rulers
	VirtualCursor   StyleDef // Style for virtual cursors in cursor mode

	// Git change indicators for line numbers
	GitAdded    StyleDef // Style for added lines
	GitModified StyleDef // Style for modified lines
	GitDeleted  StyleDef // Style for deleted lines

	// Diagnostics
	DiagnosticError   StyleDef // Foreground for error indicator
	DiagnosticWarning StyleDef // Foreground for warning indicator

	// Notifications / banners
	NotificationInfo  StyleDef // Style for low-severity info banner
	NotificationError StyleDef // Style for error banner

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
	MarkdownH1             StyleDef
	MarkdownH2             StyleDef
	MarkdownH3             StyleDef
	MarkdownH4             StyleDef
	MarkdownH5             StyleDef
	MarkdownH6             StyleDef
	MarkdownBold           StyleDef
	MarkdownItalic         StyleDef
	MarkdownBoldItalic     StyleDef
	MarkdownCode           StyleDef
	MarkdownCodeBlock      StyleDef
	MarkdownBlockquote     StyleDef
	MarkdownLink           StyleDef
	MarkdownLinkText       StyleDef
	MarkdownLinkUrl        StyleDef
	MarkdownImage          StyleDef
	MarkdownList           StyleDef
	MarkdownListMarker     StyleDef
	MarkdownHorizontalRule StyleDef
	MarkdownStrikethrough  StyleDef
	MarkdownTable          StyleDef
	MarkdownTableBorder    StyleDef
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

// GetStyleWithFg returns a style based on the given base style but with a custom foreground color
func (t *Theme) GetStyleWithFg(baseStyle ui.Style, fgColor string) ui.Style {
	// Parse the color
	fg, err := ui.ParseHexColor(fgColor)
	if err != nil {
		// If color parsing fails, return the base style unchanged
		return baseStyle
	}

	// Create a new style with the custom foreground but keeping background and attributes
	return baseStyle.Foreground(fg)
}

// NewTheme returns the theme
func NewTheme() *Theme {
	return &Theme{
		Name: "main",

		StatusBar: StyleDef{
			Foreground: "#ffffff",
			Background: "#414339",
		},

		StatusBarSubdued: StyleDef{
			Foreground: "#ffffff",
			Background: "#75715e",
		},

		CommandLine: StyleDef{
			Foreground: "#f8f8f2",
			Background: "#1e1f1c",
		},

		DialogBorder: StyleDef{
			Foreground: "#8f8f8f",
			Background: "#1e1f1c",
		},

		DialogContent: StyleDef{
			Foreground: "#f8f8f2",
			Background: "#1e1f1c",
		},

		FuzzyListBorder: StyleDef{
			Foreground: "#8f8f8f",
			Background: "#1e1f1c",
		},

		FuzzyListContent: StyleDef{
			Foreground: "#f8f8f2",
			Background: "#1e1f1c",
		},

		FuzzyListSelection: StyleDef{
			Foreground: "#f8f8f2",
			Background: "#414339",
		},

		Normal: StyleDef{
			Foreground: "#f8f8f2",
			Background: "#1b1d1e",
		},

		Selection: StyleDef{
			Foreground: "#f8f8f2",
			Background: "#403d3d",
		},

		LineNumber: StyleDef{
			Foreground: "#465457",
			// Distinct gutter background to stand out from content
			Background: "#24241e",
		},

		CurrentLine: StyleDef{
			Foreground: "#f8f8f2",
			Background: "#24241e",
		},

		Context: StyleDef{
			Foreground: "#66d9ef",
			Background: "#000000",
		},

		SearchHighlight: StyleDef{
			Foreground: "#000000",
			Background: "#fd971f",
		},

		FlashHighlight: StyleDef{
			Foreground: "#f8f8f2",
			Background: "#364462",
		},

		Whitespace: StyleDef{
			Foreground: "#2b2a24",
			Background: "",
		},

		VerticalRuler: StyleDef{
			Foreground: "",
			Background: "#24241e",
		},

		VirtualCursor: StyleDef{
			Foreground: "#000000",
			Background: "#f8f8f0",
		},

		// Git change indicators for line numbers
		// Make added vs modified more distinct while keeping same glyph ▍
		GitAdded: StyleDef{
			Foreground: "#7ee031", // brighter, cooler green
			Background: "#183d18",
		},

		GitModified: StyleDef{
			Foreground: "#ffd866", // warmer yellow-orange
			Background: "#3d3916",
		},

		GitDeleted: StyleDef{
			Foreground: "#f92672",
			Background: "#4d1a1a",
		},

		// Diagnostics
		DiagnosticError: StyleDef{
			Foreground: "#f92672",
			Background: "#1b1d1e",
		},
		DiagnosticWarning: StyleDef{
			Foreground: "#ffd866",
			Background: "#1b1d1e",
		},

		// Notifications / banners
		NotificationInfo: StyleDef{
			Foreground: "#ffffff",
			Background: "#324ba0", // subtle blue
		},
		NotificationError: StyleDef{
			Foreground: "#ffffff",
			Background: "#ff0000",
		},

		CompletionMenu: StyleDef{
			Foreground: "#f8f8f2",
			Background: "#1e1f1c",
		},

		CompletionMenuSelected: StyleDef{
			Foreground: "#f8f8f2",
			Background: "#414339",
		},

		SyntaxText: StyleDef{
			Foreground: "#f8f8f2",
			Background: "#1b1d1e",
		},

		SyntaxKeyword: StyleDef{
			Foreground: "#f92672[B]",
			Background: "#1b1d1e",
		},

		SyntaxFunction: StyleDef{
			Foreground: "#a6e22e",
			Background: "#1b1d1e",
		},

		SyntaxVariable: StyleDef{
			Foreground: "#f8f8f2",
			Background: "#1b1d1e",
		},

		SyntaxConstant: StyleDef{
			Foreground: "#ae81ff[B]",
			Background: "#1b1d1e",
		},

		SyntaxString: StyleDef{
			Foreground: "#e6db74",
			Background: "#1b1d1e",
		},

		SyntaxComment: StyleDef{
			Foreground: "#7e8e91",
			Background: "#1b1d1e",
		},

		SyntaxNumber: StyleDef{
			Foreground: "#ae81ff",
			Background: "#1b1d1e",
		},

		SyntaxOperator: StyleDef{
			Foreground: "#f92672",
			Background: "#1b1d1e",
		},

		SyntaxPunctuation: StyleDef{
			Foreground: "#8f8f8f",
			Background: "#1b1d1e",
		},

		SyntaxType: StyleDef{
			Foreground: "#66d9ef",
			Background: "#1b1d1e",
		},

		SyntaxNamespace: StyleDef{
			Foreground: "#66d9ef",
			Background: "#1b1d1e",
		},

		SyntaxAttribute: StyleDef{
			Foreground: "#a6e22e",
			Background: "#1b1d1e",
		},

		SyntaxTag: StyleDef{
			Foreground: "#f92672",
			Background: "#1b1d1e",
		},

		// Markdown styles
		MarkdownH1: StyleDef{
			Foreground: "#f92672[B]",
			Background: "#1b1d1e",
		},

		MarkdownH2: StyleDef{
			Foreground: "#ae81ff[B]",
			Background: "#1b1d1e",
		},

		MarkdownH3: StyleDef{
			Foreground: "#a6e22e[B]",
			Background: "#1b1d1e",
		},

		MarkdownH4: StyleDef{
			Foreground: "#66d9ef[B]",
			Background: "#1b1d1e",
		},

		MarkdownH5: StyleDef{
			Foreground: "#e6db74[B]",
			Background: "#1b1d1e",
		},

		MarkdownH6: StyleDef{
			Foreground: "#fd971f[B]",
			Background: "#1b1d1e",
		},

		MarkdownBold: StyleDef{
			Foreground: "#f8f8f2[B]",
			Background: "#1b1d1e",
		},

		MarkdownItalic: StyleDef{
			Foreground: "#f8f8f2[I]",
			Background: "#1b1d1e",
		},

		MarkdownBoldItalic: StyleDef{
			Foreground: "#f8f8f2[BI]",
			Background: "#1b1d1e",
		},

		MarkdownCode: StyleDef{
			Foreground: "#e6db74",
			Background: "#24241e",
		},

		MarkdownCodeBlock: StyleDef{
			Foreground: "#e6db74",
			Background: "#24241e",
		},

		MarkdownBlockquote: StyleDef{
			Foreground: "#7e8e91[I]",
			Background: "#1b1d1e",
		},

		MarkdownLink: StyleDef{
			Foreground: "#66d9ef[U]",
			Background: "#1b1d1e",
		},

		MarkdownLinkText: StyleDef{
			Foreground: "#66d9ef[U]",
			Background: "#1b1d1e",
		},

		MarkdownLinkUrl: StyleDef{
			Foreground: "#ae81ff",
			Background: "#1b1d1e",
		},

		MarkdownImage: StyleDef{
			Foreground: "#a6e22e",
			Background: "#1b1d1e",
		},

		MarkdownList: StyleDef{
			Foreground: "#f8f8f2",
			Background: "#1b1d1e",
		},

		MarkdownListMarker: StyleDef{
			Foreground: "#a2a2a2",
			Background: "#1b1d1e",
		},

		MarkdownHorizontalRule: StyleDef{
			Foreground: "#7e8e91",
			Background: "#1b1d1e",
		},

		MarkdownStrikethrough: StyleDef{
			Foreground: "#7e8e91",
			Background: "#1b1d1e",
		},

		MarkdownTable: StyleDef{
			Foreground: "#f8f8f2",
			Background: "#1b1d1e",
		},

		MarkdownTableBorder: StyleDef{
			Foreground: "#7e8e91",
			Background: "#1b1d1e",
		},
	}
}
