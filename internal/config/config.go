package config

import (
	"encoding/json"
	"os"
	"path/filepath"

	"veda/internal/theme"
)

// IndentationType represents the type of indentation used
type IndentationType int

const (
	IndentSpaces IndentationType = iota
	IndentTabs
)

// ConfigFile represents the JSON structure saved to disk
type ConfigFile struct {
	Theme               string `json:"theme,omitempty"`
	ScrollOffset        int    `json:"scroll_offset,omitempty"`
	IndentType          string `json:"indent_type,omitempty"`
	IndentWidth         int    `json:"indent_width,omitempty"`
	VerticalRulers      []int  `json:"vertical_rulers,omitempty"`
	RelativeLineNumbers bool   `json:"relative_line_numbers,omitempty"`
}

type Config struct {
	ScrollOffset int
	Theme        *theme.Theme

	// Default indentation settings for new buffers
	DefaultIndentType  IndentationType
	DefaultIndentWidth int

	// Vertical ruler positions
	VerticalRulers []int

	RelativeLineNumbers bool
}

func New() *Config {
	return &Config{
		ScrollOffset:        10,
		Theme:               theme.NewTheme(),
		DefaultIndentType:   IndentSpaces,
		DefaultIndentWidth:  4,
		VerticalRulers:      []int{80, 100, 120},
		RelativeLineNumbers: false,
	}
}

// ConfigPath returns the path to the config file (~/.config/veda/config.json)
func ConfigPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(homeDir, ".config", "veda", "config.json")
}

// Load reads the config file and applies settings to the config
func (c *Config) Load() error {
	path := ConfigPath()
	if path == "" {
		return nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No config file yet, use defaults
		}
		return err
	}

	var cf ConfigFile
	if err := json.Unmarshal(data, &cf); err != nil {
		return err
	}

	// Apply theme if specified
	if cf.Theme != "" {
		if t := theme.GetThemeByName(cf.Theme); t != nil {
			c.Theme = t
		}
	}

	// Apply scroll offset if specified
	if cf.ScrollOffset > 0 {
		c.ScrollOffset = cf.ScrollOffset
	}

	// Apply indent type if specified
	if cf.IndentType == "tabs" {
		c.DefaultIndentType = IndentTabs
	} else if cf.IndentType == "spaces" {
		c.DefaultIndentType = IndentSpaces
	}

	// Apply indent width if specified
	if cf.IndentWidth > 0 {
		c.DefaultIndentWidth = cf.IndentWidth
	}

	// Apply vertical rulers if specified
	if len(cf.VerticalRulers) > 0 {
		c.VerticalRulers = cf.VerticalRulers
	}

	c.RelativeLineNumbers = cf.RelativeLineNumbers

	return nil
}

// Save writes the current config to the config file
func (c *Config) Save() error {
	path := ConfigPath()
	if path == "" {
		return nil
	}

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// Build config file struct
	indentType := "spaces"
	if c.DefaultIndentType == IndentTabs {
		indentType = "tabs"
	}

	cf := ConfigFile{
		Theme:               c.Theme.Name,
		ScrollOffset:        c.ScrollOffset,
		IndentType:          indentType,
		IndentWidth:         c.DefaultIndentWidth,
		VerticalRulers:      c.VerticalRulers,
		RelativeLineNumbers: c.RelativeLineNumbers,
	}

	data, err := json.MarshalIndent(cf, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}
