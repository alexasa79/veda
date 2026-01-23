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
var themeNames = []string{"monokai", "dracula", "nord", "onedark", "tender", "koda", "tokyonight", "vscode"}

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
	case "tender":
		return NewTenderTheme()
	case "koda":
		return NewKodaTheme()
	case "tokyonight":
		return NewTokyoNightTheme()
	case "vscode":
		return NewVSCodeTheme()
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

// NewTenderTheme returns the Tender theme
func NewTenderTheme() *Theme {
	return &Theme{
		Name: "tender",

		StatusBar: StyleDef{
			Foreground: "#1d1d1d",
			Background: "#666666",
		},

		StatusBarSubdued: StyleDef{
			Foreground: "#999999",
			Background: "#444444",
		},

		CommandLine: StyleDef{
			Foreground: "#eeeeee",
			Background: "#282828",
		},

		DialogBorder: StyleDef{
			Foreground: "#444444",
			Background: "#282828",
		},

		DialogContent: StyleDef{
			Foreground: "#eeeeee",
			Background: "#282828",
		},

		FuzzyListBorder: StyleDef{
			Foreground: "#444444",
			Background: "#282828",
		},

		FuzzyListContent: StyleDef{
			Foreground: "#eeeeee",
			Background: "#282828",
		},

		FuzzyListSelection: StyleDef{
			Foreground: "#1d1d1d",
			Background: "#73cef4",
		},

		Normal: StyleDef{
			Foreground: "#eeeeee",
			Background: "#282828",
		},

		Selection: StyleDef{
			Foreground: "#eeeeee",
			Background: "#293b44",
		},

		LineNumber: StyleDef{
			Foreground: "#444444",
			Background: "#282828",
		},

		CurrentLineNumber: StyleDef{
			Foreground: "#73cef4[B]",
			Background: "#323232",
		},

		CurrentLine: StyleDef{
			Foreground: "#eeeeee",
			Background: "#323232",
		},

		CurrentWord: StyleDef{
			Foreground: "",
			Background: "#335261",
		},

		Context: StyleDef{
			Foreground: "#b3deef",
			Background: "#202020",
		},

		SignatureActive: StyleDef{
			Foreground: "#335261",
			Background: "#c9d05c",
		},

		SearchHighlight: StyleDef{
			Foreground: "#ffffff",
			Background: "#335261",
		},

		FlashHighlight: StyleDef{
			Foreground: "#eeeeee",
			Background: "#464632",
		},

		MatchHighlight: StyleDef{
			Foreground: "",
			Background: "#293b44",
		},

		Whitespace: StyleDef{
			Foreground: "#444444",
			Background: "",
		},

		VerticalRuler: StyleDef{
			Foreground: "",
			Background: "#323232",
		},

		VirtualCursor: StyleDef{
			Foreground: "#282828",
			Background: "#c9d05c",
		},

		// Diff backgrounds
		DiffAdded: StyleDef{
			Background: "#464632",
		},
		DiffAddedCurrentLine: StyleDef{
			Background: "#5a5a46",
		},
		DiffDeleted: StyleDef{
			Background: "#79313c",
		},
		DiffDeletedCurrentLine: StyleDef{
			Background: "#94404c",
		},

		// Git change indicators for line numbers
		GitAdded: StyleDef{
			Foreground: "#c9d05c",
			Background: "#282828",
		},

		GitModified: StyleDef{
			Foreground: "#b3deef",
			Background: "#282828",
		},

		GitDeleted: StyleDef{
			Foreground: "#f43753",
			Background: "#282828",
		},

		// Diagnostics
		DiagnosticError: StyleDef{
			Foreground: "#f43753",
			Background: "#282828",
		},
		DiagnosticWarning: StyleDef{
			Foreground: "#ffc24b",
			Background: "#282828",
		},

		// Notifications / banners
		NotificationInfo: StyleDef{
			Foreground: "#eeeeee",
			Background: "#335261",
		},
		NotificationError: StyleDef{
			Foreground: "#eeeeee",
			Background: "#f43753",
		},

		CompletionMenu: StyleDef{
			Foreground: "#dadada",
			Background: "#335261",
		},

		CompletionMenuSelected: StyleDef{
			Foreground: "#335261",
			Background: "#c9d05c",
		},

		SyntaxText: StyleDef{
			Foreground: "#eeeeee",
			Background: "#282828",
		},

		SyntaxKeyword: StyleDef{
			Foreground: "#73cef4",
			Background: "#282828",
		},

		SyntaxFunction: StyleDef{
			Foreground: "#b3deef",
			Background: "#282828",
		},

		SyntaxVariable: StyleDef{
			Foreground: "#dadada",
			Background: "#282828",
		},

		SyntaxConstant: StyleDef{
			Foreground: "#ffc24b",
			Background: "#282828",
		},

		SyntaxString: StyleDef{
			Foreground: "#d3b987",
			Background: "#282828",
		},

		SyntaxComment: StyleDef{
			Foreground: "#666666",
			Background: "#282828",
		},

		SyntaxNumber: StyleDef{
			Foreground: "#ffc24b",
			Background: "#282828",
		},

		SyntaxOperator: StyleDef{
			Foreground: "#f43753",
			Background: "#282828",
		},

		SyntaxPunctuation: StyleDef{
			Foreground: "#eeeeee",
			Background: "#282828",
		},

		SyntaxType: StyleDef{
			Foreground: "#73cef4",
			Background: "#282828",
		},

		SyntaxNamespace: StyleDef{
			Foreground: "#73cef4",
			Background: "#282828",
		},

		SyntaxAttribute: StyleDef{
			Foreground: "#c9d05c",
			Background: "#282828",
		},

		SyntaxTag: StyleDef{
			Foreground: "#f43753",
			Background: "#282828",
		},

		// Markdown styles
		MarkdownH1: StyleDef{
			Foreground: "#b3deef[B]",
			Background: "#282828",
		},

		MarkdownH2: StyleDef{
			Foreground: "#73cef4[B]",
			Background: "#282828",
		},

		MarkdownH3: StyleDef{
			Foreground: "#c9d05c[B]",
			Background: "#282828",
		},

		MarkdownH4: StyleDef{
			Foreground: "#ffc24b[B]",
			Background: "#282828",
		},

		MarkdownH5: StyleDef{
			Foreground: "#d3b987[B]",
			Background: "#282828",
		},

		MarkdownH6: StyleDef{
			Foreground: "#f43753[B]",
			Background: "#282828",
		},

		MarkdownBold: StyleDef{
			Foreground: "#eeeeee[B]",
			Background: "#282828",
		},

		MarkdownItalic: StyleDef{
			Foreground: "#eeeeee[I]",
			Background: "#282828",
		},

		MarkdownBoldItalic: StyleDef{
			Foreground: "#eeeeee[BI]",
			Background: "#282828",
		},

		MarkdownCode: StyleDef{
			Foreground: "#d3b987",
			Background: "#202020",
		},

		MarkdownCodeBlock: StyleDef{
			Foreground: "#d3b987",
			Background: "#202020",
		},

		MarkdownBlockquote: StyleDef{
			Foreground: "#ffc24b",
			Background: "#282828",
		},

		MarkdownLink: StyleDef{
			Foreground: "#73cef4[U]",
			Background: "#282828",
		},

		MarkdownLinkText: StyleDef{
			Foreground: "#c9d05c[U]",
			Background: "#282828",
		},

		MarkdownLinkUrl: StyleDef{
			Foreground: "#73cef4",
			Background: "#282828",
		},

		MarkdownImage: StyleDef{
			Foreground: "#c9d05c",
			Background: "#282828",
		},

		MarkdownList: StyleDef{
			Foreground: "#eeeeee",
			Background: "#282828",
		},

		MarkdownListMarker: StyleDef{
			Foreground: "#ffc24b",
			Background: "#282828",
		},

		MarkdownHorizontalRule: StyleDef{
			Foreground: "#c9d05c",
			Background: "#282828",
		},

		MarkdownStrikethrough: StyleDef{
			Foreground: "#666666",
			Background: "#282828",
		},

		MarkdownTable: StyleDef{
			Foreground: "#eeeeee",
			Background: "#282828",
		},

		MarkdownTableBorder: StyleDef{
			Foreground: "#444444",
			Background: "#282828",
		},
	}
}

// NewKodaTheme returns the Koda theme
func NewKodaTheme() *Theme {
	return &Theme{
		Name: "koda",

		StatusBar: StyleDef{
			Foreground: "#b0b0b0",
			Background: "#272727",
		},

		StatusBarSubdued: StyleDef{
			Foreground: "#50585d",
			Background: "#101010",
		},

		CommandLine: StyleDef{
			Foreground: "#b0b0b0",
			Background: "#101010",
		},

		DialogBorder: StyleDef{
			Foreground: "#ffffff",
			Background: "#101010",
		},

		DialogContent: StyleDef{
			Foreground: "#b0b0b0",
			Background: "#101010",
		},

		FuzzyListBorder: StyleDef{
			Foreground: "#ffffff",
			Background: "#101010",
		},

		FuzzyListContent: StyleDef{
			Foreground: "#b0b0b0",
			Background: "#101010",
		},

		FuzzyListSelection: StyleDef{
			Foreground: "#101010",
			Background: "#ffffff",
		},

		Normal: StyleDef{
			Foreground: "#b0b0b0",
			Background: "#101010",
		},

		Selection: StyleDef{
			Foreground: "#ffffff",
			Background: "#254266",
		},

		LineNumber: StyleDef{
			Foreground: "#50585d",
			Background: "#101010",
		},

		CurrentLineNumber: StyleDef{
			Foreground: "#ffffff[B]",
			Background: "#272727",
		},

		CurrentLine: StyleDef{
			Foreground: "#b0b0b0",
			Background: "#272727",
		},

		CurrentWord: StyleDef{
			Foreground: "",
			Background: "#1b293b",
		},

		Context: StyleDef{
			Foreground: "#8ebeec",
			Background: "#272727",
		},

		SignatureActive: StyleDef{
			Foreground: "#101010",
			Background: "#d9ba73",
		},

		SearchHighlight: StyleDef{
			Foreground: "#101010",
			Background: "#d9ba73",
		},

		FlashHighlight: StyleDef{
			Foreground: "#ffffff",
			Background: "#254266",
		},

		MatchHighlight: StyleDef{
			Foreground: "",
			Background: "#1b293b",
		},

		Whitespace: StyleDef{
			Foreground: "#272727",
			Background: "",
		},

		VerticalRuler: StyleDef{
			Foreground: "",
			Background: "#272727",
		},

		VirtualCursor: StyleDef{
			Foreground: "#101010",
			Background: "#458ee6",
		},

		// Diff backgrounds
		DiffAdded: StyleDef{
			Background: "#283627",
		},
		DiffAddedCurrentLine: StyleDef{
			Background: "#3f5c3e",
		},
		DiffDeleted: StyleDef{
			Background: "#402424",
		},
		DiffDeletedCurrentLine: StyleDef{
			Background: "#703939",
		},

		// Git change indicators for line numbers
		GitAdded: StyleDef{
			Foreground: "#86cd82",
			Background: "#101010",
		},

		GitModified: StyleDef{
			Foreground: "#d9ba73",
			Background: "#101010",
		},

		GitDeleted: StyleDef{
			Foreground: "#ff7676",
			Background: "#101010",
		},

		// Diagnostics
		DiagnosticError: StyleDef{
			Foreground: "#ff7676",
			Background: "#101010",
		},
		DiagnosticWarning: StyleDef{
			Foreground: "#d9ba73",
			Background: "#101010",
		},

		// Notifications / banners
		NotificationInfo: StyleDef{
			Foreground: "#101010",
			Background: "#8ebeec",
		},
		NotificationError: StyleDef{
			Foreground: "#101010",
			Background: "#ff7676",
		},

		CompletionMenu: StyleDef{
			Foreground: "#b0b0b0",
			Background: "#101010",
		},

		CompletionMenuSelected: StyleDef{
			Foreground: "#101010",
			Background: "#ffffff",
		},

		SyntaxText: StyleDef{
			Foreground: "#b0b0b0",
			Background: "#101010",
		},

		SyntaxKeyword: StyleDef{
			Foreground: "#777777",
			Background: "#101010",
		},

		SyntaxFunction: StyleDef{
			Foreground: "#ffffff",
			Background: "#101010",
		},

		SyntaxVariable: StyleDef{
			Foreground: "#b0b0b0",
			Background: "#101010",
		},

		SyntaxConstant: StyleDef{
			Foreground: "#d9ba73",
			Background: "#101010",
		},

		SyntaxString: StyleDef{
			Foreground: "#ffffff",
			Background: "#101010",
		},

		SyntaxComment: StyleDef{
			Foreground: "#50585d",
			Background: "#101010",
		},

		SyntaxNumber: StyleDef{
			Foreground: "#d9ba73",
			Background: "#101010",
		},

		SyntaxOperator: StyleDef{
			Foreground: "#777777",
			Background: "#101010",
		},

		SyntaxPunctuation: StyleDef{
			Foreground: "#ffffff",
			Background: "#101010",
		},

		SyntaxType: StyleDef{
			Foreground: "#777777",
			Background: "#101010",
		},

		SyntaxNamespace: StyleDef{
			Foreground: "#777777",
			Background: "#101010",
		},

		SyntaxAttribute: StyleDef{
			Foreground: "#777777",
			Background: "#101010",
		},

		SyntaxTag: StyleDef{
			Foreground: "#777777",
			Background: "#101010",
		},

		// Markdown styles
		MarkdownH1: StyleDef{
			Foreground: "#ffffff[B]",
			Background: "#101010",
		},

		MarkdownH2: StyleDef{
			Foreground: "#ffffff[B]",
			Background: "#101010",
		},

		MarkdownH3: StyleDef{
			Foreground: "#ffffff[B]",
			Background: "#101010",
		},

		MarkdownH4: StyleDef{
			Foreground: "#ffffff[B]",
			Background: "#101010",
		},

		MarkdownH5: StyleDef{
			Foreground: "#ffffff[B]",
			Background: "#101010",
		},

		MarkdownH6: StyleDef{
			Foreground: "#ffffff[B]",
			Background: "#101010",
		},

		MarkdownBold: StyleDef{
			Foreground: "#ffffff[B]",
			Background: "#101010",
		},

		MarkdownItalic: StyleDef{
			Foreground: "#b0b0b0[I]",
			Background: "#101010",
		},

		MarkdownBoldItalic: StyleDef{
			Foreground: "#ffffff[BI]",
			Background: "#101010",
		},

		MarkdownCode: StyleDef{
			Foreground: "#d9ba73",
			Background: "#272727",
		},

		MarkdownCodeBlock: StyleDef{
			Foreground: "#d9ba73",
			Background: "#272727",
		},

		MarkdownBlockquote: StyleDef{
			Foreground: "#50585d",
			Background: "#101010",
		},

		MarkdownLink: StyleDef{
			Foreground: "#ffffff[U]",
			Background: "#101010",
		},

		MarkdownLinkText: StyleDef{
			Foreground: "#ffffff[U]",
			Background: "#101010",
		},

		MarkdownLinkUrl: StyleDef{
			Foreground: "#8ebeec[U]",
			Background: "#101010",
		},

		MarkdownImage: StyleDef{
			Foreground: "#d9ba73",
			Background: "#101010",
		},

		MarkdownList: StyleDef{
			Foreground: "#b0b0b0",
			Background: "#101010",
		},

		MarkdownListMarker: StyleDef{
			Foreground: "#ffffff",
			Background: "#101010",
		},

		MarkdownHorizontalRule: StyleDef{
			Foreground: "#50585d",
			Background: "#101010",
		},

		MarkdownStrikethrough: StyleDef{
			Foreground: "#50585d",
			Background: "#101010",
		},

		MarkdownTable: StyleDef{
			Foreground: "#b0b0b0",
			Background: "#101010",
		},

		MarkdownTableBorder: StyleDef{
			Foreground: "#272727",
			Background: "#101010",
		},
	}
}

// NewTokyoNightTheme returns the Tokyo Night theme (storm)
func NewTokyoNightTheme() *Theme {
	return &Theme{
		Name: "tokyonight",

		StatusBar: StyleDef{
			Foreground: "#a9b1d6",
			Background: "#1f2335",
		},

		StatusBarSubdued: StyleDef{
			Foreground: "#3b4261",
			Background: "#1f2335",
		},

		CommandLine: StyleDef{
			Foreground: "#c0caf5",
			Background: "#1f2335",
		},

		DialogBorder: StyleDef{
			Foreground: "#29a4bd",
			Background: "#1f2335",
		},

		DialogContent: StyleDef{
			Foreground: "#c0caf5",
			Background: "#1f2335",
		},

		FuzzyListBorder: StyleDef{
			Foreground: "#29a4bd",
			Background: "#1f2335",
		},

		FuzzyListContent: StyleDef{
			Foreground: "#c0caf5",
			Background: "#1f2335",
		},

		FuzzyListSelection: StyleDef{
			Foreground: "#c0caf5",
			Background: "#363d59",
		},

		Normal: StyleDef{
			Foreground: "#c0caf5",
			Background: "#24283b",
		},

		Selection: StyleDef{
			Foreground: "#c0caf5",
			Background: "#2e3c64",
		},

		LineNumber: StyleDef{
			Foreground: "#3b4261",
			Background: "#24283b",
		},

		CurrentLineNumber: StyleDef{
			Foreground: "#ff9e64[B]",
			Background: "#292e42",
		},

		CurrentLine: StyleDef{
			Foreground: "#c0caf5",
			Background: "#292e42",
		},

		CurrentWord: StyleDef{
			Foreground: "",
			Background: "#3b4261",
		},

		Context: StyleDef{
			Foreground: "#a9b1d6",
			Background: "#1b1e2d",
		},

		SignatureActive: StyleDef{
			Foreground: "#24283b",
			Background: "#e0af68",
		},

		SearchHighlight: StyleDef{
			Foreground: "#c0caf5",
			Background: "#3d59a1",
		},

		FlashHighlight: StyleDef{
			Foreground: "#c0caf5",
			Background: "#394b70",
		},

		MatchHighlight: StyleDef{
			Foreground: "",
			Background: "#2e3c64",
		},

		Whitespace: StyleDef{
			Foreground: "#3b4261",
			Background: "",
		},

		VerticalRuler: StyleDef{
			Foreground: "",
			Background: "#1b1e2d",
		},

		VirtualCursor: StyleDef{
			Foreground: "#24283b",
			Background: "#7aa2f7",
		},

		// Diff backgrounds
		DiffAdded: StyleDef{
			Background: "#2b485a",
		},
		DiffAddedCurrentLine: StyleDef{
			Background: "#305a6c",
		},
		DiffDeleted: StyleDef{
			Background: "#52313f",
		},
		DiffDeletedCurrentLine: StyleDef{
			Background: "#6d3641",
		},

		// Git change indicators for line numbers
		GitAdded: StyleDef{
			Foreground: "#449dab",
			Background: "#24283b",
		},

		GitModified: StyleDef{
			Foreground: "#6183bb",
			Background: "#24283b",
		},

		GitDeleted: StyleDef{
			Foreground: "#914c54",
			Background: "#24283b",
		},

		// Diagnostics
		DiagnosticError: StyleDef{
			Foreground: "#db4b4b",
			Background: "#24283b",
		},
		DiagnosticWarning: StyleDef{
			Foreground: "#e0af68",
			Background: "#24283b",
		},

		// Notifications / banners
		NotificationInfo: StyleDef{
			Foreground: "#c0caf5",
			Background: "#3d59a1",
		},
		NotificationError: StyleDef{
			Foreground: "#c0caf5",
			Background: "#db4b4b",
		},

		CompletionMenu: StyleDef{
			Foreground: "#c0caf5",
			Background: "#1f2335",
		},

		CompletionMenuSelected: StyleDef{
			Foreground: "#c0caf5",
			Background: "#363d59",
		},

		SyntaxText: StyleDef{
			Foreground: "#c0caf5",
			Background: "#24283b",
		},

		SyntaxKeyword: StyleDef{
			Foreground: "#9d7cd8",
			Background: "#24283b",
		},

		SyntaxFunction: StyleDef{
			Foreground: "#7aa2f7",
			Background: "#24283b",
		},

		SyntaxVariable: StyleDef{
			Foreground: "#bb9af7",
			Background: "#24283b",
		},

		SyntaxConstant: StyleDef{
			Foreground: "#ff9e64",
			Background: "#24283b",
		},

		SyntaxString: StyleDef{
			Foreground: "#9ece6a",
			Background: "#24283b",
		},

		SyntaxComment: StyleDef{
			Foreground: "#565f89",
			Background: "#24283b",
		},

		SyntaxNumber: StyleDef{
			Foreground: "#ff9e64",
			Background: "#24283b",
		},

		SyntaxOperator: StyleDef{
			Foreground: "#89ddff",
			Background: "#24283b",
		},

		SyntaxPunctuation: StyleDef{
			Foreground: "#a9b1d6",
			Background: "#24283b",
		},

		SyntaxType: StyleDef{
			Foreground: "#2ac3de",
			Background: "#24283b",
		},

		SyntaxNamespace: StyleDef{
			Foreground: "#2ac3de",
			Background: "#24283b",
		},

		SyntaxAttribute: StyleDef{
			Foreground: "#7dcfff",
			Background: "#24283b",
		},

		SyntaxTag: StyleDef{
			Foreground: "#f7768e",
			Background: "#24283b",
		},

		// Markdown styles
		MarkdownH1: StyleDef{
			Foreground: "#7aa2f7[B]",
			Background: "#24283b",
		},

		MarkdownH2: StyleDef{
			Foreground: "#e0af68[B]",
			Background: "#24283b",
		},

		MarkdownH3: StyleDef{
			Foreground: "#9ece6a[B]",
			Background: "#24283b",
		},

		MarkdownH4: StyleDef{
			Foreground: "#1abc9c[B]",
			Background: "#24283b",
		},

		MarkdownH5: StyleDef{
			Foreground: "#bb9af7[B]",
			Background: "#24283b",
		},

		MarkdownH6: StyleDef{
			Foreground: "#9d7cd8[B]",
			Background: "#24283b",
		},

		MarkdownBold: StyleDef{
			Foreground: "#c0caf5[B]",
			Background: "#24283b",
		},

		MarkdownItalic: StyleDef{
			Foreground: "#c0caf5[I]",
			Background: "#24283b",
		},

		MarkdownBoldItalic: StyleDef{
			Foreground: "#c0caf5[BI]",
			Background: "#24283b",
		},

		MarkdownCode: StyleDef{
			Foreground: "#7aa2f7",
			Background: "#414868",
		},

		MarkdownCodeBlock: StyleDef{
			Foreground: "#7aa2f7",
			Background: "#414868",
		},

		MarkdownBlockquote: StyleDef{
			Foreground: "#565f89",
			Background: "#24283b",
		},

		MarkdownLink: StyleDef{
			Foreground: "#1abc9c[U]",
			Background: "#24283b",
		},

		MarkdownLinkText: StyleDef{
			Foreground: "#1abc9c[U]",
			Background: "#24283b",
		},

		MarkdownLinkUrl: StyleDef{
			Foreground: "#0db9d7[U]",
			Background: "#24283b",
		},

		MarkdownImage: StyleDef{
			Foreground: "#9ece6a",
			Background: "#24283b",
		},

		MarkdownList: StyleDef{
			Foreground: "#c0caf5",
			Background: "#24283b",
		},

		MarkdownListMarker: StyleDef{
			Foreground: "#ff9e64",
			Background: "#24283b",
		},

		MarkdownHorizontalRule: StyleDef{
			Foreground: "#565f89",
			Background: "#24283b",
		},

		MarkdownStrikethrough: StyleDef{
			Foreground: "#565f89",
			Background: "#24283b",
		},

		MarkdownTable: StyleDef{
			Foreground: "#c0caf5",
			Background: "#24283b",
		},

		MarkdownTableBorder: StyleDef{
			Foreground: "#3b4261",
			Background: "#24283b",
		},
	}
}

// NewVSCodeTheme returns the VS Code Dark theme
func NewVSCodeTheme() *Theme {
	return &Theme{
		Name: "vscode",

		StatusBar: StyleDef{
			Foreground: "#d4d4d4",
			Background: "#373737",
		},

		StatusBarSubdued: StyleDef{
			Foreground: "#d4d4d4",
			Background: "#252526",
		},

		CommandLine: StyleDef{
			Foreground: "#d4d4d4",
			Background: "#1f1f1f",
		},

		DialogBorder: StyleDef{
			Foreground: "#444444",
			Background: "#202020",
		},

		DialogContent: StyleDef{
			Foreground: "#bbbbbb",
			Background: "#202020",
		},

		FuzzyListBorder: StyleDef{
			Foreground: "#444444",
			Background: "#202020",
		},

		FuzzyListContent: StyleDef{
			Foreground: "#bbbbbb",
			Background: "#202020",
		},

		FuzzyListSelection: StyleDef{
			Foreground: "#bbbbbb",
			Background: "#04395e",
		},

		Normal: StyleDef{
			Foreground: "#d4d4d4",
			Background: "#1f1f1f",
		},

		Selection: StyleDef{
			Foreground: "#d4d4d4",
			Background: "#264f78",
		},

		LineNumber: StyleDef{
			Foreground: "#5a5a5a",
			Background: "#1f1f1f",
		},

		CurrentLineNumber: StyleDef{
			Foreground: "#bbbbbb[B]",
			Background: "#222222",
		},

		CurrentLine: StyleDef{
			Foreground: "#d4d4d4",
			Background: "#222222",
		},

		CurrentWord: StyleDef{
			Foreground: "",
			Background: "#51504f",
		},

		Context: StyleDef{
			Foreground: "#bbbbbb",
			Background: "#404040",
		},

		SignatureActive: StyleDef{
			Foreground: "#d4d4d4",
			Background: "#515c6a",
		},

		SearchHighlight: StyleDef{
			Foreground: "#d4d4d4",
			Background: "#613315",
		},

		FlashHighlight: StyleDef{
			Foreground: "#d4d4d4",
			Background: "#707070",
		},

		MatchHighlight: StyleDef{
			Foreground: "",
			Background: "#51504f",
		},

		Whitespace: StyleDef{
			Foreground: "#5a5a5a",
			Background: "",
		},

		VerticalRuler: StyleDef{
			Foreground: "",
			Background: "#222222",
		},

		VirtualCursor: StyleDef{
			Foreground: "#d4d4d4",
			Background: "#04395e",
		},

		// Diff backgrounds
		DiffAdded: StyleDef{
			Background: "#4b5632",
		},
		DiffAddedCurrentLine: StyleDef{
			Background: "#373d29",
		},
		DiffDeleted: StyleDef{
			Background: "#6f1313",
		},
		DiffDeletedCurrentLine: StyleDef{
			Background: "#4b1818",
		},

		// Git change indicators for line numbers
		GitAdded: StyleDef{
			Foreground: "#81b88b",
			Background: "#1f1f1f",
		},

		GitModified: StyleDef{
			Foreground: "#e2c08d",
			Background: "#1f1f1f",
		},

		GitDeleted: StyleDef{
			Foreground: "#c74e39",
			Background: "#1f1f1f",
		},

		// Diagnostics
		DiagnosticError: StyleDef{
			Foreground: "#f44747",
			Background: "#1f1f1f",
		},
		DiagnosticWarning: StyleDef{
			Foreground: "#dcdcaa",
			Background: "#1f1f1f",
		},

		// Notifications / banners
		NotificationInfo: StyleDef{
			Foreground: "#d4d4d4",
			Background: "#084671",
		},
		NotificationError: StyleDef{
			Foreground: "#d4d4d4",
			Background: "#f44747",
		},

		CompletionMenu: StyleDef{
			Foreground: "#bbbbbb",
			Background: "#202020",
		},

		CompletionMenuSelected: StyleDef{
			Foreground: "#bbbbbb",
			Background: "#04395e",
		},

		SyntaxText: StyleDef{
			Foreground: "#d4d4d4",
			Background: "#1f1f1f",
		},

		SyntaxKeyword: StyleDef{
			Foreground: "#c586c0",
			Background: "#1f1f1f",
		},

		SyntaxFunction: StyleDef{
			Foreground: "#dcdcaa",
			Background: "#1f1f1f",
		},

		SyntaxVariable: StyleDef{
			Foreground: "#9cdcfe",
			Background: "#1f1f1f",
		},

		SyntaxConstant: StyleDef{
			Foreground: "#569cd6",
			Background: "#1f1f1f",
		},

		SyntaxString: StyleDef{
			Foreground: "#ce9178",
			Background: "#1f1f1f",
		},

		SyntaxComment: StyleDef{
			Foreground: "#6a9955",
			Background: "#1f1f1f",
		},

		SyntaxNumber: StyleDef{
			Foreground: "#b5cea8",
			Background: "#1f1f1f",
		},

		SyntaxOperator: StyleDef{
			Foreground: "#d4d4d4",
			Background: "#1f1f1f",
		},

		SyntaxPunctuation: StyleDef{
			Foreground: "#d4d4d4",
			Background: "#1f1f1f",
		},

		SyntaxType: StyleDef{
			Foreground: "#569cd6",
			Background: "#1f1f1f",
		},

		SyntaxNamespace: StyleDef{
			Foreground: "#4ec9b0",
			Background: "#1f1f1f",
		},

		SyntaxAttribute: StyleDef{
			Foreground: "#dcdcaa",
			Background: "#1f1f1f",
		},

		SyntaxTag: StyleDef{
			Foreground: "#569cd6",
			Background: "#1f1f1f",
		},

		// Markdown styles
		MarkdownH1: StyleDef{
			Foreground: "#569cd6[B]",
			Background: "#1f1f1f",
		},

		MarkdownH2: StyleDef{
			Foreground: "#ce9178[B]",
			Background: "#1f1f1f",
		},

		MarkdownH3: StyleDef{
			Foreground: "#dcdcaa[B]",
			Background: "#1f1f1f",
		},

		MarkdownH4: StyleDef{
			Foreground: "#6a9955[B]",
			Background: "#1f1f1f",
		},

		MarkdownH5: StyleDef{
			Foreground: "#569cd6[B]",
			Background: "#1f1f1f",
		},

		MarkdownH6: StyleDef{
			Foreground: "#c586c0[B]",
			Background: "#1f1f1f",
		},

		MarkdownBold: StyleDef{
			Foreground: "#569cd6[B]",
			Background: "#1f1f1f",
		},

		MarkdownItalic: StyleDef{
			Foreground: "#d4d4d4[I]",
			Background: "#1f1f1f",
		},

		MarkdownBoldItalic: StyleDef{
			Foreground: "#569cd6[BI]",
			Background: "#1f1f1f",
		},

		MarkdownCode: StyleDef{
			Foreground: "#ce9178",
			Background: "#222222",
		},

		MarkdownCodeBlock: StyleDef{
			Foreground: "#ce9178",
			Background: "#222222",
		},

		MarkdownBlockquote: StyleDef{
			Foreground: "#6a9955",
			Background: "#1f1f1f",
		},

		MarkdownLink: StyleDef{
			Foreground: "#9cdcfe[U]",
			Background: "#1f1f1f",
		},

		MarkdownLinkText: StyleDef{
			Foreground: "#9cdcfe[U]",
			Background: "#1f1f1f",
		},

		MarkdownLinkUrl: StyleDef{
			Foreground: "#d4d4d4[U]",
			Background: "#1f1f1f",
		},

		MarkdownImage: StyleDef{
			Foreground: "#4ec9b0",
			Background: "#1f1f1f",
		},

		MarkdownList: StyleDef{
			Foreground: "#d4d4d4",
			Background: "#1f1f1f",
		},

		MarkdownListMarker: StyleDef{
			Foreground: "#dcdcaa",
			Background: "#1f1f1f",
		},

		MarkdownHorizontalRule: StyleDef{
			Foreground: "#569cd6",
			Background: "#1f1f1f",
		},

		MarkdownStrikethrough: StyleDef{
			Foreground: "#808080",
			Background: "#1f1f1f",
		},

		MarkdownTable: StyleDef{
			Foreground: "#d4d4d4",
			Background: "#1f1f1f",
		},

		MarkdownTableBorder: StyleDef{
			Foreground: "#444444",
			Background: "#1f1f1f",
		},
	}
}
