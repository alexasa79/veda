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
)

// GetFileIcon returns the appropriate unicode symbol for a file based on its name and type
func GetFileIcon(name string, fileType string) string {
	// Handle directories
	if fileType == "directory" {
		return IconDirectory
	}

	// Get file extension
	ext := strings.ToLower(filepath.Ext(name))

	// Map extensions to icons
	switch ext {
	// Go files
	case ".go":
		return IconGo

	// Python files
	case ".py", ".pyw", ".pyc", ".pyd", ".pyo":
		return IconPython

	// JavaScript/TypeScript files
	case ".js", ".jsx", ".ts", ".tsx", ".mjs":
		return IconJavaScript

	// Rust files
	case ".rs":
		return IconRust

	// C/C++ files
	case ".c", ".h":
		return IconC
	case ".cpp", ".cc", ".cxx", ".hpp", ".hxx":
		return IconCpp

	// Text files
	case ".txt", ".text":
		return IconText
	}

	return IconDefault
}