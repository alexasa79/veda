package utils

import (
	"testing"
)

func TestGetSymbolIcon(t *testing.T) {
	tests := []struct {
		kind     int
		expected string
	}{
		{SymbolKindFunction, "󰊕"},
		{SymbolKindMethod, ""},
		{SymbolKindClass, ""},
		{SymbolKindVariable, ""},
		{SymbolKindConstant, ""},
		{SymbolKindInterface, ""},
		{SymbolKindStruct, ""},
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
		{"app.js", "file", IconJavaScript},
		{"types.ts", "file", IconTypeScript},
		{"lib.rs", "file", IconRust},
		{"main.c", "file", IconC},
		{"class.cpp", "file", IconCpp},
		{"README.md", "file", IconReadme},
		{"Makefile", "file", IconMakefile},
		{"Dockerfile", "file", IconDocker},
		{".gitignore", "file", IconGit},
		{"data.json", "file", IconJson},
		{"config.yaml", "file", IconYaml},
		{"test_file.go", "file", IconTest},
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