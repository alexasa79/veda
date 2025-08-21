package utils

import (
	"testing"
)

func TestGetSymbolIcon(t *testing.T) {
	tests := []struct {
		kind     int
		expected string
	}{
		{SymbolKindFunction, "ƒ"},
		{SymbolKindMethod, "▸"},
		{SymbolKindClass, "◆"},
		{SymbolKindVariable, "𝑥"},
		{SymbolKindConstant, "𝐂"},
		{SymbolKindInterface, "◐"},
		{SymbolKindStruct, "◧"},
		{999, "○"}, // Unknown kind
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			icon := GetSymbolIcon(tt.kind)
			if icon != tt.expected {
				t.Errorf("GetSymbolIcon(%d) = %q, want %q", tt.kind, icon, tt.expected)
			}
		})
	}
}

func TestGetFileIconInfo(t *testing.T) {
	tests := []struct {
		name     string
		fileType string
		wantIcon string
	}{
		{"main.go", "file", IconGo},
		{"test.py", "file", IconPython},
		{"app.js", "file", IconDefault},   // JavaScript now uses default
		{"types.ts", "file", IconDefault}, // TypeScript now uses default
		{"lib.rs", "file", IconDefault},   // Rust now uses default
		{"main.c", "file", IconCpp},       // C uses C++ icon
		{"class.cpp", "file", IconCpp},
		{"README.md", "file", IconMarkdown},  // README uses markdown icon
		{"Makefile", "file", IconDefault},    // Makefile now uses default
		{"Dockerfile", "file", IconDefault},  // Docker now uses default
		{".gitignore", "file", IconDefault},  // Git now uses default
		{"data.json", "file", IconDefault},   // JSON now uses default
		{"config.yaml", "file", IconDefault}, // YAML now uses default
		{"test_file.go", "file", IconGo},     // Test files use language icon
		{"unknown.xyz", "file", IconDefault},
		{"src", "directory", IconDirectory},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := GetFileIconInfo(tt.name, tt.fileType)
			if info.Icon != tt.wantIcon {
				t.Errorf("GetFileIconInfo(%q, %q).Icon = %q, want %q",
					tt.name, tt.fileType, info.Icon, tt.wantIcon)
			}
			// Also verify that color is set for known types
			if tt.wantIcon != IconDefault && info.Color == "" {
				t.Errorf("GetFileIconInfo(%q, %q).Color is empty, expected a color",
					tt.name, tt.fileType)
			}
		})
	}
}
