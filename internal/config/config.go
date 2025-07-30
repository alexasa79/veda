package config

import "veda/internal/theme"

// IndentationType represents the type of indentation used
type IndentationType int

const (
	IndentSpaces IndentationType = iota
	IndentTabs
)

type Config struct {
	ScrollOffset int
	Theme        *theme.Theme
	
	// Default indentation settings for new buffers
	DefaultIndentType  IndentationType
	DefaultIndentWidth int
	
	// Vertical ruler positions
	VerticalRulers []int
}

func New() *Config {
	return &Config{
		ScrollOffset:       10,
		Theme:             theme.NewTheme(),
		DefaultIndentType: IndentSpaces,
		DefaultIndentWidth: 4,
		VerticalRulers:    []int{78, 80, 100, 120},
	}
}
