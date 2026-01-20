package theme

import (
	"testing"
)

func TestGetThemeNames(t *testing.T) {
	names := GetThemeNames()
	if len(names) != 8 {
		t.Errorf("Expected 8 themes, got %d", len(names))
	}

	expected := []string{"monokai", "dracula", "nord", "onedark", "tender", "koda", "tokyonight", "vscode"}
	for i, name := range expected {
		if names[i] != name {
			t.Errorf("Expected theme %s at index %d, got %s", name, i, names[i])
		}
	}
}

func TestGetThemeByName(t *testing.T) {
	tests := []struct {
		name     string
		expected string
		isNil    bool
	}{
		{"monokai", "monokai", false},
		{"dracula", "dracula", false},
		{"nord", "nord", false},
		{"onedark", "onedark", false},
		{"tender", "tender", false},
		{"koda", "koda", false},
		{"tokyonight", "tokyonight", false},
		{"vscode", "vscode", false},
		{"invalid", "", true},
		{"", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			theme := GetThemeByName(tt.name)
			if tt.isNil {
				if theme != nil {
					t.Errorf("Expected nil for theme '%s', got non-nil", tt.name)
				}
			} else {
				if theme == nil {
					t.Errorf("Expected non-nil theme for '%s', got nil", tt.name)
				} else if theme.Name != tt.expected {
					t.Errorf("Expected theme name '%s', got '%s'", tt.expected, theme.Name)
				}
			}
		})
	}
}

func TestNewThemeReturnsMonokai(t *testing.T) {
	theme := NewTheme()
	if theme.Name != "monokai" {
		t.Errorf("Expected default theme to be 'monokai', got '%s'", theme.Name)
	}
}

func TestAllThemesHaveRequiredStyles(t *testing.T) {
	themeNames := GetThemeNames()

	for _, name := range themeNames {
		t.Run(name, func(t *testing.T) {
			theme := GetThemeByName(name)
			if theme == nil {
				t.Fatalf("Theme '%s' returned nil", name)
			}

			// Check that key styles are defined (not empty)
			if theme.Normal.Foreground == "" && theme.Normal.Background == "" {
				t.Errorf("Theme '%s' has empty Normal style", name)
			}
			if theme.StatusBar.Foreground == "" && theme.StatusBar.Background == "" {
				t.Errorf("Theme '%s' has empty StatusBar style", name)
			}
			if theme.Selection.Foreground == "" && theme.Selection.Background == "" {
				t.Errorf("Theme '%s' has empty Selection style", name)
			}
			if theme.SearchHighlight.Foreground == "" && theme.SearchHighlight.Background == "" {
				t.Errorf("Theme '%s' has empty SearchHighlight style", name)
			}
			if theme.SyntaxKeyword.Foreground == "" && theme.SyntaxKeyword.Background == "" {
				t.Errorf("Theme '%s' has empty SyntaxKeyword style", name)
			}
		})
	}
}
