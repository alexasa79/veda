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
}

func New() *Config {
	return &Config{
		ScrollOffset:       10,
		Theme:             theme.NewTheme(),
		DefaultIndentType: IndentSpaces,
		DefaultIndentWidth: 4,
	}
}
