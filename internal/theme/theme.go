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
	Normal            StyleDef
	Selection         StyleDef
	LineNumber        StyleDef
	CurrentLineNumber StyleDef
	CurrentLine       StyleDef
	CurrentWord       StyleDef // Style for word under primary cursor(s)
	Context           StyleDef // Style for context lines and signature help
	SignatureActive   StyleDef // Style for active parameter in signature help
	SearchHighlight   StyleDef // Style for search matches
	FlashHighlight    StyleDef // Style for flashing yanked/changed text
	MatchHighlight    StyleDef // Style for matching bracket/quote highlight
	Whitespace        StyleDef // Style for whitespace indicators (tabs, trailing spaces)
	VerticalRuler     StyleDef // Style for vertical rulers
	VirtualCursor     StyleDef // Style for virtual cursors in cursor mode
	// Diff backgrounds
	DiffAdded              StyleDef // Background for added lines
	DiffAddedCurrentLine   StyleDef // Background for added lines when current line
	DiffDeleted            StyleDef // Background for deleted lines (filler on opposite side)
	DiffDeletedCurrentLine StyleDef // Background for deleted lines when current line

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

// Available theme names
var themeNames = []string{"monokai", "dracula", "nord", "onedark"}

// GetThemeNames returns a list of all available theme names
func GetThemeNames() []string {
	return themeNames
}

// GetThemeByName returns a theme by its name, or nil if not found
func GetThemeByName(name string) *Theme {
	switch name {
	case "monokai":
		return NewMonokaiTheme()
	case "dracula":
		return NewDraculaTheme()
	case "nord":
		return NewNordTheme()
	case "onedark":
		return NewOneDarkTheme()
	default:
		return nil
	}
}

// NewTheme returns the default theme (Monokai)
func NewTheme() *Theme {
	return NewMonokaiTheme()
}

// NewMonokaiTheme returns the Monokai theme
func NewMonokaiTheme() *Theme {
	return &Theme{
		Name: "monokai",

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

		CurrentLineNumber: StyleDef{
			Foreground: "#f8f8f2[B]",
			Background: "#24241e",
		},

		CurrentLine: StyleDef{
			Foreground: "#f8f8f2",
			Background: "#24241e",
		},

		CurrentWord: StyleDef{
			Foreground: "",
			Background: "#353c29",
		},

		Context: StyleDef{
			Foreground: "#66d9ef",
			Background: "#000000",
		},

		SignatureActive: StyleDef{
			Foreground: "#1b1d1e",
			Background: "#a6e22e",
		},

		SearchHighlight: StyleDef{
			Foreground: "#000000",
			Background: "#fd971f",
		},

		FlashHighlight: StyleDef{
			Foreground: "#f8f8f2",
			Background: "#364462",
		},

		MatchHighlight: StyleDef{
			Foreground: "",        // keep existing foreground
			Background: "#2f5968", // darker teal for better contrast with light text
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

		// Diff backgrounds (consolidated styles)
		// Green for additions; brighter variant for the current line
		DiffAdded: StyleDef{
			Background: "#286428", // rgb(40,100,40)
		},
		DiffAddedCurrentLine: StyleDef{
			Background: "#3C8C3C", // rgb(60,140,60)
		},
		// Red for deletions; brighter variant for the current line
		DiffDeleted: StyleDef{
			Background: "#642828", // rgb(100,40,40)
		},
		DiffDeletedCurrentLine: StyleDef{
			Background: "#8C3C3C", // rgb(140,60,60)
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
			Foreground: "#6ab35f[BI]",
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

// NewDraculaTheme returns the Dracula theme
func NewDraculaTheme() *Theme {
	return &Theme{
		Name: "dracula",

		StatusBar: StyleDef{
			Foreground: "#f8f8f2",
			Background: "#44475a",
		},

		StatusBarSubdued: StyleDef{
			Foreground: "#f8f8f2",
			Background: "#6272a4",
		},

		CommandLine: StyleDef{
			Foreground: "#f8f8f2",
			Background: "#282a36",
		},

		DialogBorder: StyleDef{
			Foreground: "#6272a4",
			Background: "#282a36",
		},

		DialogContent: StyleDef{
			Foreground: "#f8f8f2",
			Background: "#282a36",
		},

		FuzzyListBorder: StyleDef{
			Foreground: "#6272a4",
			Background: "#282a36",
		},

		FuzzyListContent: StyleDef{
			Foreground: "#f8f8f2",
			Background: "#282a36",
		},

		FuzzyListSelection: StyleDef{
			Foreground: "#f8f8f2",
			Background: "#44475a",
		},

		Normal: StyleDef{
			Foreground: "#f8f8f2",
			Background: "#282a36",
		},

		Selection: StyleDef{
			Foreground: "#f8f8f2",
			Background: "#44475a",
		},

		LineNumber: StyleDef{
			Foreground: "#6272a4",
			Background: "#21222c",
		},

		CurrentLineNumber: StyleDef{
			Foreground: "#f8f8f2[B]",
			Background: "#21222c",
		},

		CurrentLine: StyleDef{
			Foreground: "#f8f8f2",
			Background: "#21222c",
		},

		CurrentWord: StyleDef{
			Foreground: "",
			Background: "#3d4259",
		},

		Context: StyleDef{
			Foreground: "#8be9fd",
			Background: "#1e1f29",
		},

		SignatureActive: StyleDef{
			Foreground: "#282a36",
			Background: "#50fa7b",
		},

		SearchHighlight: StyleDef{
			Foreground: "#282a36",
			Background: "#ffb86c",
		},

		FlashHighlight: StyleDef{
			Foreground: "#f8f8f2",
			Background: "#4d5678",
		},

		MatchHighlight: StyleDef{
			Foreground: "",
			Background: "#3d5c6e",
		},

		Whitespace: StyleDef{
			Foreground: "#3a3c4e",
			Background: "",
		},

		VerticalRuler: StyleDef{
			Foreground: "",
			Background: "#21222c",
		},

		VirtualCursor: StyleDef{
			Foreground: "#282a36",
			Background: "#f8f8f2",
		},

		DiffAdded: StyleDef{
			Background: "#2a4a2a",
		},
		DiffAddedCurrentLine: StyleDef{
			Background: "#3a6a3a",
		},
		DiffDeleted: StyleDef{
			Background: "#4a2a2a",
		},
		DiffDeletedCurrentLine: StyleDef{
			Background: "#6a3a3a",
		},

		GitAdded: StyleDef{
			Foreground: "#50fa7b",
			Background: "#1a3a1a",
		},

		GitModified: StyleDef{
			Foreground: "#ffb86c",
			Background: "#3a3016",
		},

		GitDeleted: StyleDef{
			Foreground: "#ff5555",
			Background: "#4a1a1a",
		},

		DiagnosticError: StyleDef{
			Foreground: "#ff5555",
			Background: "#282a36",
		},
		DiagnosticWarning: StyleDef{
			Foreground: "#ffb86c",
			Background: "#282a36",
		},

		NotificationInfo: StyleDef{
			Foreground: "#f8f8f2",
			Background: "#6272a4",
		},
		NotificationError: StyleDef{
			Foreground: "#f8f8f2",
			Background: "#ff5555",
		},

		CompletionMenu: StyleDef{
			Foreground: "#f8f8f2",
			Background: "#282a36",
		},

		CompletionMenuSelected: StyleDef{
			Foreground: "#f8f8f2",
			Background: "#44475a",
		},

		SyntaxText: StyleDef{
			Foreground: "#f8f8f2",
			Background: "#282a36",
		},

		SyntaxKeyword: StyleDef{
			Foreground: "#ff79c6[B]",
			Background: "#282a36",
		},

		SyntaxFunction: StyleDef{
			Foreground: "#50fa7b",
			Background: "#282a36",
		},

		SyntaxVariable: StyleDef{
			Foreground: "#f8f8f2",
			Background: "#282a36",
		},

		SyntaxConstant: StyleDef{
			Foreground: "#bd93f9[B]",
			Background: "#282a36",
		},

		SyntaxString: StyleDef{
			Foreground: "#f1fa8c",
			Background: "#282a36",
		},

		SyntaxComment: StyleDef{
			Foreground: "#6272a4",
			Background: "#282a36",
		},

		SyntaxNumber: StyleDef{
			Foreground: "#bd93f9",
			Background: "#282a36",
		},

		SyntaxOperator: StyleDef{
			Foreground: "#ff79c6",
			Background: "#282a36",
		},

		SyntaxPunctuation: StyleDef{
			Foreground: "#f8f8f2",
			Background: "#282a36",
		},

		SyntaxType: StyleDef{
			Foreground: "#8be9fd",
			Background: "#282a36",
		},

		SyntaxNamespace: StyleDef{
			Foreground: "#8be9fd",
			Background: "#282a36",
		},

		SyntaxAttribute: StyleDef{
			Foreground: "#50fa7b",
			Background: "#282a36",
		},

		SyntaxTag: StyleDef{
			Foreground: "#ff79c6",
			Background: "#282a36",
		},

		MarkdownH1: StyleDef{
			Foreground: "#ff79c6[B]",
			Background: "#282a36",
		},

		MarkdownH2: StyleDef{
			Foreground: "#bd93f9[B]",
			Background: "#282a36",
		},

		MarkdownH3: StyleDef{
			Foreground: "#50fa7b[B]",
			Background: "#282a36",
		},

		MarkdownH4: StyleDef{
			Foreground: "#8be9fd[B]",
			Background: "#282a36",
		},

		MarkdownH5: StyleDef{
			Foreground: "#f1fa8c[B]",
			Background: "#282a36",
		},

		MarkdownH6: StyleDef{
			Foreground: "#ffb86c[B]",
			Background: "#282a36",
		},

		MarkdownBold: StyleDef{
			Foreground: "#f8f8f2[B]",
			Background: "#282a36",
		},

		MarkdownItalic: StyleDef{
			Foreground: "#f8f8f2[I]",
			Background: "#282a36",
		},

		MarkdownBoldItalic: StyleDef{
			Foreground: "#f8f8f2[BI]",
			Background: "#282a36",
		},

		MarkdownCode: StyleDef{
			Foreground: "#f1fa8c",
			Background: "#21222c",
		},

		MarkdownCodeBlock: StyleDef{
			Foreground: "#f1fa8c",
			Background: "#21222c",
		},

		MarkdownBlockquote: StyleDef{
			Foreground: "#50fa7b[BI]",
			Background: "#282a36",
		},

		MarkdownLink: StyleDef{
			Foreground: "#8be9fd[U]",
			Background: "#282a36",
		},

		MarkdownLinkText: StyleDef{
			Foreground: "#8be9fd[U]",
			Background: "#282a36",
		},

		MarkdownLinkUrl: StyleDef{
			Foreground: "#bd93f9",
			Background: "#282a36",
		},

		MarkdownImage: StyleDef{
			Foreground: "#50fa7b",
			Background: "#282a36",
		},

		MarkdownList: StyleDef{
			Foreground: "#f8f8f2",
			Background: "#282a36",
		},

		MarkdownListMarker: StyleDef{
			Foreground: "#ff79c6",
			Background: "#282a36",
		},

		MarkdownHorizontalRule: StyleDef{
			Foreground: "#6272a4",
			Background: "#282a36",
		},

		MarkdownStrikethrough: StyleDef{
			Foreground: "#6272a4",
			Background: "#282a36",
		},

		MarkdownTable: StyleDef{
			Foreground: "#f8f8f2",
			Background: "#282a36",
		},

		MarkdownTableBorder: StyleDef{
			Foreground: "#6272a4",
			Background: "#282a36",
		},
	}
}

// NewNordTheme returns the Nord theme
func NewNordTheme() *Theme {
	return &Theme{
		Name: "nord",

		StatusBar: StyleDef{
			Foreground: "#eceff4",
			Background: "#434c5e",
		},

		StatusBarSubdued: StyleDef{
			Foreground: "#eceff4",
			Background: "#4c566a",
		},

		CommandLine: StyleDef{
			Foreground: "#eceff4",
			Background: "#2e3440",
		},

		DialogBorder: StyleDef{
			Foreground: "#4c566a",
			Background: "#2e3440",
		},

		DialogContent: StyleDef{
			Foreground: "#eceff4",
			Background: "#2e3440",
		},

		FuzzyListBorder: StyleDef{
			Foreground: "#4c566a",
			Background: "#2e3440",
		},

		FuzzyListContent: StyleDef{
			Foreground: "#eceff4",
			Background: "#2e3440",
		},

		FuzzyListSelection: StyleDef{
			Foreground: "#eceff4",
			Background: "#434c5e",
		},

		Normal: StyleDef{
			Foreground: "#eceff4",
			Background: "#2e3440",
		},

		Selection: StyleDef{
			Foreground: "#eceff4",
			Background: "#434c5e",
		},

		LineNumber: StyleDef{
			Foreground: "#4c566a",
			Background: "#292e39",
		},

		CurrentLineNumber: StyleDef{
			Foreground: "#eceff4[B]",
			Background: "#292e39",
		},

		CurrentLine: StyleDef{
			Foreground: "#eceff4",
			Background: "#292e39",
		},

		CurrentWord: StyleDef{
			Foreground: "",
			Background: "#3b4252",
		},

		Context: StyleDef{
			Foreground: "#88c0d0",
			Background: "#242933",
		},

		SignatureActive: StyleDef{
			Foreground: "#2e3440",
			Background: "#a3be8c",
		},

		SearchHighlight: StyleDef{
			Foreground: "#2e3440",
			Background: "#d08770",
		},

		FlashHighlight: StyleDef{
			Foreground: "#eceff4",
			Background: "#4c566a",
		},

		MatchHighlight: StyleDef{
			Foreground: "",
			Background: "#3b4f5c",
		},

		Whitespace: StyleDef{
			Foreground: "#3b4252",
			Background: "",
		},

		VerticalRuler: StyleDef{
			Foreground: "",
			Background: "#292e39",
		},

		VirtualCursor: StyleDef{
			Foreground: "#2e3440",
			Background: "#eceff4",
		},

		DiffAdded: StyleDef{
			Background: "#2e4a3a",
		},
		DiffAddedCurrentLine: StyleDef{
			Background: "#3e6a4a",
		},
		DiffDeleted: StyleDef{
			Background: "#4a2e2e",
		},
		DiffDeletedCurrentLine: StyleDef{
			Background: "#6a3e3e",
		},

		GitAdded: StyleDef{
			Foreground: "#a3be8c",
			Background: "#1e3a2a",
		},

		GitModified: StyleDef{
			Foreground: "#ebcb8b",
			Background: "#3a3516",
		},

		GitDeleted: StyleDef{
			Foreground: "#bf616a",
			Background: "#4a1a1a",
		},

		DiagnosticError: StyleDef{
			Foreground: "#bf616a",
			Background: "#2e3440",
		},
		DiagnosticWarning: StyleDef{
			Foreground: "#ebcb8b",
			Background: "#2e3440",
		},

		NotificationInfo: StyleDef{
			Foreground: "#eceff4",
			Background: "#5e81ac",
		},
		NotificationError: StyleDef{
			Foreground: "#eceff4",
			Background: "#bf616a",
		},

		CompletionMenu: StyleDef{
			Foreground: "#eceff4",
			Background: "#2e3440",
		},

		CompletionMenuSelected: StyleDef{
			Foreground: "#eceff4",
			Background: "#434c5e",
		},

		SyntaxText: StyleDef{
			Foreground: "#eceff4",
			Background: "#2e3440",
		},

		SyntaxKeyword: StyleDef{
			Foreground: "#81a1c1[B]",
			Background: "#2e3440",
		},

		SyntaxFunction: StyleDef{
			Foreground: "#88c0d0",
			Background: "#2e3440",
		},

		SyntaxVariable: StyleDef{
			Foreground: "#eceff4",
			Background: "#2e3440",
		},

		SyntaxConstant: StyleDef{
			Foreground: "#b48ead[B]",
			Background: "#2e3440",
		},

		SyntaxString: StyleDef{
			Foreground: "#a3be8c",
			Background: "#2e3440",
		},

		SyntaxComment: StyleDef{
			Foreground: "#616e88",
			Background: "#2e3440",
		},

		SyntaxNumber: StyleDef{
			Foreground: "#b48ead",
			Background: "#2e3440",
		},

		SyntaxOperator: StyleDef{
			Foreground: "#81a1c1",
			Background: "#2e3440",
		},

		SyntaxPunctuation: StyleDef{
			Foreground: "#d8dee9",
			Background: "#2e3440",
		},

		SyntaxType: StyleDef{
			Foreground: "#8fbcbb",
			Background: "#2e3440",
		},

		SyntaxNamespace: StyleDef{
			Foreground: "#8fbcbb",
			Background: "#2e3440",
		},

		SyntaxAttribute: StyleDef{
			Foreground: "#88c0d0",
			Background: "#2e3440",
		},

		SyntaxTag: StyleDef{
			Foreground: "#81a1c1",
			Background: "#2e3440",
		},

		MarkdownH1: StyleDef{
			Foreground: "#81a1c1[B]",
			Background: "#2e3440",
		},

		MarkdownH2: StyleDef{
			Foreground: "#b48ead[B]",
			Background: "#2e3440",
		},

		MarkdownH3: StyleDef{
			Foreground: "#a3be8c[B]",
			Background: "#2e3440",
		},

		MarkdownH4: StyleDef{
			Foreground: "#88c0d0[B]",
			Background: "#2e3440",
		},

		MarkdownH5: StyleDef{
			Foreground: "#ebcb8b[B]",
			Background: "#2e3440",
		},

		MarkdownH6: StyleDef{
			Foreground: "#d08770[B]",
			Background: "#2e3440",
		},

		MarkdownBold: StyleDef{
			Foreground: "#eceff4[B]",
			Background: "#2e3440",
		},

		MarkdownItalic: StyleDef{
			Foreground: "#eceff4[I]",
			Background: "#2e3440",
		},

		MarkdownBoldItalic: StyleDef{
			Foreground: "#eceff4[BI]",
			Background: "#2e3440",
		},

		MarkdownCode: StyleDef{
			Foreground: "#a3be8c",
			Background: "#292e39",
		},

		MarkdownCodeBlock: StyleDef{
			Foreground: "#a3be8c",
			Background: "#292e39",
		},

		MarkdownBlockquote: StyleDef{
			Foreground: "#88c0d0[BI]",
			Background: "#2e3440",
		},

		MarkdownLink: StyleDef{
			Foreground: "#88c0d0[U]",
			Background: "#2e3440",
		},

		MarkdownLinkText: StyleDef{
			Foreground: "#88c0d0[U]",
			Background: "#2e3440",
		},

		MarkdownLinkUrl: StyleDef{
			Foreground: "#b48ead",
			Background: "#2e3440",
		},

		MarkdownImage: StyleDef{
			Foreground: "#a3be8c",
			Background: "#2e3440",
		},

		MarkdownList: StyleDef{
			Foreground: "#eceff4",
			Background: "#2e3440",
		},

		MarkdownListMarker: StyleDef{
			Foreground: "#81a1c1",
			Background: "#2e3440",
		},

		MarkdownHorizontalRule: StyleDef{
			Foreground: "#4c566a",
			Background: "#2e3440",
		},

		MarkdownStrikethrough: StyleDef{
			Foreground: "#4c566a",
			Background: "#2e3440",
		},

		MarkdownTable: StyleDef{
			Foreground: "#eceff4",
			Background: "#2e3440",
		},

		MarkdownTableBorder: StyleDef{
			Foreground: "#4c566a",
			Background: "#2e3440",
		},
	}
}

// NewOneDarkTheme returns the One Dark theme
func NewOneDarkTheme() *Theme {
	return &Theme{
		Name: "onedark",

		StatusBar: StyleDef{
			Foreground: "#abb2bf",
			Background: "#3e4451",
		},

		StatusBarSubdued: StyleDef{
			Foreground: "#abb2bf",
			Background: "#4b5263",
		},

		CommandLine: StyleDef{
			Foreground: "#abb2bf",
			Background: "#282c34",
		},

		DialogBorder: StyleDef{
			Foreground: "#5c6370",
			Background: "#282c34",
		},

		DialogContent: StyleDef{
			Foreground: "#abb2bf",
			Background: "#282c34",
		},

		FuzzyListBorder: StyleDef{
			Foreground: "#5c6370",
			Background: "#282c34",
		},

		FuzzyListContent: StyleDef{
			Foreground: "#abb2bf",
			Background: "#282c34",
		},

		FuzzyListSelection: StyleDef{
			Foreground: "#abb2bf",
			Background: "#3e4451",
		},

		Normal: StyleDef{
			Foreground: "#abb2bf",
			Background: "#282c34",
		},

		Selection: StyleDef{
			Foreground: "#abb2bf",
			Background: "#3e4451",
		},

		LineNumber: StyleDef{
			Foreground: "#4b5263",
			Background: "#22252c",
		},

		CurrentLineNumber: StyleDef{
			Foreground: "#abb2bf[B]",
			Background: "#22252c",
		},

		CurrentLine: StyleDef{
			Foreground: "#abb2bf",
			Background: "#22252c",
		},

		CurrentWord: StyleDef{
			Foreground: "",
			Background: "#353b45",
		},

		Context: StyleDef{
			Foreground: "#56b6c2",
			Background: "#21252b",
		},

		SignatureActive: StyleDef{
			Foreground: "#282c34",
			Background: "#98c379",
		},

		SearchHighlight: StyleDef{
			Foreground: "#282c34",
			Background: "#d19a66",
		},

		FlashHighlight: StyleDef{
			Foreground: "#abb2bf",
			Background: "#4b5263",
		},

		MatchHighlight: StyleDef{
			Foreground: "",
			Background: "#3a4a5c",
		},

		Whitespace: StyleDef{
			Foreground: "#3b4048",
			Background: "",
		},

		VerticalRuler: StyleDef{
			Foreground: "",
			Background: "#22252c",
		},

		VirtualCursor: StyleDef{
			Foreground: "#282c34",
			Background: "#abb2bf",
		},

		DiffAdded: StyleDef{
			Background: "#2a4a2a",
		},
		DiffAddedCurrentLine: StyleDef{
			Background: "#3a6a3a",
		},
		DiffDeleted: StyleDef{
			Background: "#4a2a2a",
		},
		DiffDeletedCurrentLine: StyleDef{
			Background: "#6a3a3a",
		},

		GitAdded: StyleDef{
			Foreground: "#98c379",
			Background: "#1e3a2a",
		},

		GitModified: StyleDef{
			Foreground: "#d19a66",
			Background: "#3a3016",
		},

		GitDeleted: StyleDef{
			Foreground: "#e06c75",
			Background: "#4a1a1a",
		},

		DiagnosticError: StyleDef{
			Foreground: "#e06c75",
			Background: "#282c34",
		},
		DiagnosticWarning: StyleDef{
			Foreground: "#d19a66",
			Background: "#282c34",
		},

		NotificationInfo: StyleDef{
			Foreground: "#abb2bf",
			Background: "#61afef",
		},
		NotificationError: StyleDef{
			Foreground: "#abb2bf",
			Background: "#e06c75",
		},

		CompletionMenu: StyleDef{
			Foreground: "#abb2bf",
			Background: "#282c34",
		},

		CompletionMenuSelected: StyleDef{
			Foreground: "#abb2bf",
			Background: "#3e4451",
		},

		SyntaxText: StyleDef{
			Foreground: "#abb2bf",
			Background: "#282c34",
		},

		SyntaxKeyword: StyleDef{
			Foreground: "#c678dd[B]",
			Background: "#282c34",
		},

		SyntaxFunction: StyleDef{
			Foreground: "#61afef",
			Background: "#282c34",
		},

		SyntaxVariable: StyleDef{
			Foreground: "#e06c75",
			Background: "#282c34",
		},

		SyntaxConstant: StyleDef{
			Foreground: "#d19a66[B]",
			Background: "#282c34",
		},

		SyntaxString: StyleDef{
			Foreground: "#98c379",
			Background: "#282c34",
		},

		SyntaxComment: StyleDef{
			Foreground: "#5c6370",
			Background: "#282c34",
		},

		SyntaxNumber: StyleDef{
			Foreground: "#d19a66",
			Background: "#282c34",
		},

		SyntaxOperator: StyleDef{
			Foreground: "#56b6c2",
			Background: "#282c34",
		},

		SyntaxPunctuation: StyleDef{
			Foreground: "#abb2bf",
			Background: "#282c34",
		},

		SyntaxType: StyleDef{
			Foreground: "#e5c07b",
			Background: "#282c34",
		},

		SyntaxNamespace: StyleDef{
			Foreground: "#e5c07b",
			Background: "#282c34",
		},

		SyntaxAttribute: StyleDef{
			Foreground: "#d19a66",
			Background: "#282c34",
		},

		SyntaxTag: StyleDef{
			Foreground: "#e06c75",
			Background: "#282c34",
		},

		MarkdownH1: StyleDef{
			Foreground: "#e06c75[B]",
			Background: "#282c34",
		},

		MarkdownH2: StyleDef{
			Foreground: "#c678dd[B]",
			Background: "#282c34",
		},

		MarkdownH3: StyleDef{
			Foreground: "#98c379[B]",
			Background: "#282c34",
		},

		MarkdownH4: StyleDef{
			Foreground: "#61afef[B]",
			Background: "#282c34",
		},

		MarkdownH5: StyleDef{
			Foreground: "#e5c07b[B]",
			Background: "#282c34",
		},

		MarkdownH6: StyleDef{
			Foreground: "#d19a66[B]",
			Background: "#282c34",
		},

		MarkdownBold: StyleDef{
			Foreground: "#abb2bf[B]",
			Background: "#282c34",
		},

		MarkdownItalic: StyleDef{
			Foreground: "#abb2bf[I]",
			Background: "#282c34",
		},

		MarkdownBoldItalic: StyleDef{
			Foreground: "#abb2bf[BI]",
			Background: "#282c34",
		},

		MarkdownCode: StyleDef{
			Foreground: "#98c379",
			Background: "#22252c",
		},

		MarkdownCodeBlock: StyleDef{
			Foreground: "#98c379",
			Background: "#22252c",
		},

		MarkdownBlockquote: StyleDef{
			Foreground: "#56b6c2[BI]",
			Background: "#282c34",
		},

		MarkdownLink: StyleDef{
			Foreground: "#61afef[U]",
			Background: "#282c34",
		},

		MarkdownLinkText: StyleDef{
			Foreground: "#61afef[U]",
			Background: "#282c34",
		},

		MarkdownLinkUrl: StyleDef{
			Foreground: "#c678dd",
			Background: "#282c34",
		},

		MarkdownImage: StyleDef{
			Foreground: "#98c379",
			Background: "#282c34",
		},

		MarkdownList: StyleDef{
			Foreground: "#abb2bf",
			Background: "#282c34",
		},

		MarkdownListMarker: StyleDef{
			Foreground: "#e06c75",
			Background: "#282c34",
		},

		MarkdownHorizontalRule: StyleDef{
			Foreground: "#5c6370",
			Background: "#282c34",
		},

		MarkdownStrikethrough: StyleDef{
			Foreground: "#5c6370",
			Background: "#282c34",
		},

		MarkdownTable: StyleDef{
			Foreground: "#abb2bf",
			Background: "#282c34",
		},

		MarkdownTableBorder: StyleDef{
			Foreground: "#5c6370",
			Background: "#282c34",
		},
	}
}
