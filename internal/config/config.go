package config

import "editor/internal/theme"

type Config struct {
	ScrollOffset int
	Theme        *theme.Theme
}

func New() *Config {
	return &Config{
		ScrollOffset: 10,
		Theme:        theme.NewTheme(),
	}
}
