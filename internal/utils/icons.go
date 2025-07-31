package utils

import (
	"path/filepath"
	"strings"
)

// Common unicode symbols for files and directories
const (
	IconDefault    = ""
	IconDirectory  = ""

	// Programming language symbols (Nerd Font)
	IconGo         = ""
	IconPython     = ""
	IconJavaScript = ""
	IconRust       = ""
	IconC          = ""
	IconCpp        = ""

	// File type symbols (Nerd Font)
	IconText       = ""
	IconTypeScript = ""
	IconMarkdown   = ""
	IconJson       = ""
	IconYaml       = ""
	IconHtml       = ""
	IconCss        = ""
	
	// Special files
	IconGit        = ""
	IconDocker     = ""
	IconMakefile   = ""
	IconReadme     = ""
	IconTest       = ""
)

// FileIconInfo contains icon and color information
type FileIconInfo struct {
	Icon  string
	Color string
}

// SymbolIconInfo contains icon and color information for symbols
type SymbolIconInfo struct {
	Icon  string
	Color string
}

// GetFileIcon returns the appropriate unicode symbol for a file based on its name and type
func GetFileIcon(name string, fileType string) string {
	info := GetFileIconInfo(name, fileType)
	return info.Icon
}

// GetFileIconInfo returns icon and color information for a file
func GetFileIconInfo(name string, fileType string) FileIconInfo {
	// Handle directories
	if fileType == "directory" {
		return FileIconInfo{Icon: IconDirectory, Color: "#7c9bcc"}
	}

	// Check for special files first
	baseName := strings.ToLower(filepath.Base(name))
	switch baseName {
	case "makefile":
		return FileIconInfo{Icon: IconMakefile, Color: "#6d8086"}
	case "dockerfile":
		return FileIconInfo{Icon: IconDocker, Color: "#0db7ed"}
	case ".gitignore", ".gitconfig":
		return FileIconInfo{Icon: IconGit, Color: "#f1502f"}
	case "readme.md", "readme":
		return FileIconInfo{Icon: IconReadme, Color: "#89e051"}
	}

	// Check if it's a test file
	if strings.Contains(baseName, "_test.") || strings.Contains(baseName, ".test.") || strings.Contains(baseName, "test_") {
		return FileIconInfo{Icon: IconTest, Color: "#f9c513"}
	}

	// Get file extension
	ext := strings.ToLower(filepath.Ext(name))

	// Map extensions to icons and colors
	switch ext {
	// Go files
	case ".go":
		return FileIconInfo{Icon: IconGo, Color: "#519aba"}

	// Python files
	case ".py", ".pyw", ".pyc", ".pyd", ".pyo":
		return FileIconInfo{Icon: IconPython, Color: "#ffbc03"}

	// JavaScript files
	case ".js", ".jsx", ".mjs":
		return FileIconInfo{Icon: IconJavaScript, Color: "#f1e05a"}
	
	// TypeScript files
	case ".ts", ".tsx":
		return FileIconInfo{Icon: IconTypeScript, Color: "#2b7489"}

	// Rust files
	case ".rs":
		return FileIconInfo{Icon: IconRust, Color: "#dea584"}

	// C/C++ files
	case ".c", ".h":
		return FileIconInfo{Icon: IconC, Color: "#599eff"}
	case ".cpp", ".cc", ".cxx", ".hpp", ".hxx":
		return FileIconInfo{Icon: IconCpp, Color: "#f34b7d"}

	// Markup files
	case ".md", ".markdown":
		return FileIconInfo{Icon: IconMarkdown, Color: "#89e051"}
	case ".html", ".htm":
		return FileIconInfo{Icon: IconHtml, Color: "#e34c26"}
	case ".css", ".scss", ".sass":
		return FileIconInfo{Icon: IconCss, Color: "#563d7c"}

	// Data files
	case ".json":
		return FileIconInfo{Icon: IconJson, Color: "#cbcb41"}
	case ".yaml", ".yml":
		return FileIconInfo{Icon: IconYaml, Color: "#6d8086"}

	// Text files
	case ".txt", ".text":
		return FileIconInfo{Icon: IconText, Color: "#89e051"}
	}

	return FileIconInfo{Icon: IconDefault, Color: "#ccc"}
}

// Symbol kind constants (LSP specification)
const (
	SymbolKindFile          = 1
	SymbolKindModule        = 2
	SymbolKindNamespace     = 3
	SymbolKindPackage       = 4
	SymbolKindClass         = 5
	SymbolKindMethod        = 6
	SymbolKindProperty      = 7
	SymbolKindField         = 8
	SymbolKindConstructor   = 9
	SymbolKindEnum          = 10
	SymbolKindInterface     = 11
	SymbolKindFunction      = 12
	SymbolKindVariable      = 13
	SymbolKindConstant      = 14
	SymbolKindString        = 15
	SymbolKindNumber        = 16
	SymbolKindBoolean       = 17
	SymbolKindArray         = 18
	SymbolKindObject        = 19
	SymbolKindKey           = 20
	SymbolKindNull          = 21
	SymbolKindEnumMember    = 22
	SymbolKindStruct        = 23
	SymbolKindEvent         = 24
	SymbolKindOperator      = 25
	SymbolKindTypeParameter = 26
)

// GetSymbolIcon returns the appropriate unicode symbol for a given symbol kind
func GetSymbolIcon(kind int) string {
	info := GetSymbolIconInfo(kind)
	return info.Icon
}

// GetSymbolIconInfo returns icon and color information for a symbol kind
func GetSymbolIconInfo(kind int) SymbolIconInfo {
	switch kind {
	case SymbolKindFile:
		return SymbolIconInfo{Icon: "◯", Color: "#89e051"}
	case SymbolKindModule:
		return SymbolIconInfo{Icon: "▢", Color: "#599eff"}
	case SymbolKindNamespace:
		return SymbolIconInfo{Icon: "▤", Color: "#2b7489"}
	case SymbolKindPackage:
		return SymbolIconInfo{Icon: "▣", Color: "#89e051"}
	case SymbolKindClass:
		return SymbolIconInfo{Icon: "◆", Color: "#f1e05a"}
	case SymbolKindMethod:
		return SymbolIconInfo{Icon: "▸", Color: "#b392f0"}
	case SymbolKindProperty:
		return SymbolIconInfo{Icon: "▹", Color: "#79b8ff"}
	case SymbolKindField:
		return SymbolIconInfo{Icon: "◈", Color: "#79b8ff"}
	case SymbolKindConstructor:
		return SymbolIconInfo{Icon: "◉", Color: "#89e051"}
	case SymbolKindEnum:
		return SymbolIconInfo{Icon: "☰", Color: "#f97583"}
	case SymbolKindInterface:
		return SymbolIconInfo{Icon: "◐", Color: "#79b8ff"}
	case SymbolKindFunction:
		return SymbolIconInfo{Icon: "ƒ", Color: "#b392f0"}
	case SymbolKindVariable:
		return SymbolIconInfo{Icon: "𝑥", Color: "#85e89d"}
	case SymbolKindConstant:
		return SymbolIconInfo{Icon: "𝐂", Color: "#ffab70"}
	case SymbolKindString:
		return SymbolIconInfo{Icon: "❝", Color: "#89e051"}
	case SymbolKindNumber:
		return SymbolIconInfo{Icon: "#", Color: "#79b8ff"}
	case SymbolKindBoolean:
		return SymbolIconInfo{Icon: "◩", Color: "#56b6c2"}
	case SymbolKindArray:
		return SymbolIconInfo{Icon: "[]", Color: "#f97583"}
	case SymbolKindObject:
		return SymbolIconInfo{Icon: "{}", Color: "#e1e4e8"}
	case SymbolKindKey:
		return SymbolIconInfo{Icon: "⚷", Color: "#d1d5da"}
	case SymbolKindNull:
		return SymbolIconInfo{Icon: "∅", Color: "#959da5"}
	case SymbolKindEnumMember:
		return SymbolIconInfo{Icon: "◉", Color: "#f97583"}
	case SymbolKindStruct:
		return SymbolIconInfo{Icon: "◧", Color: "#ffab70"}
	case SymbolKindEvent:
		return SymbolIconInfo{Icon: "⚡", Color: "#56b6c2"}
	case SymbolKindOperator:
		return SymbolIconInfo{Icon: "±", Color: "#d1d5da"}
	case SymbolKindTypeParameter:
		return SymbolIconInfo{Icon: "𝑇", Color: "#e36209"}
	default:
		return SymbolIconInfo{Icon: "○", Color: "#959da5"}
	}
}