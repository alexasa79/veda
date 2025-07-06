package utils

import (
	"path/filepath"
	"strings"
)

// Common unicode symbols for files and directories
const (
	// Directory symbols
	IconDirectory     = "📁" // U+1F4C1
	IconDirectoryOpen = "📂" // U+1F4C2
	
	// Programming language symbols
	IconGo         = "\ue627" // Go gopher icon
	IconPython     = "🐍"     // U+1F40D
	IconJavaScript = "📜"     // U+1F4DC
	IconRust       = "🦀"     // U+1F980
	IconC          = "Ⓒ"      // U+24B8
	IconCpp        = "C++"
	
	// File type symbols
	IconText       = "📄"  // U+1F4C4
	IconMarkdown   = "📝"  // U+1F4DD
	IconJSON       = "{ }"
	IconYAML       = "📋"  // U+1F4CB
	IconConfig     = "⚙️"  // U+2699
	IconImage      = "🖼️"  // U+1F5BC
	IconVideo      = "🎬"  // U+1F3AC
	IconAudio      = "🎵"  // U+1F3B5
	IconArchive    = "📦"  // U+1F4E6
	IconExecutable = "⚡"  // U+26A1
	IconDefault    = "📄"  // U+1F4C4
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
	
	// Markdown files
	case ".md", ".markdown":
		return IconMarkdown
	
	// Data files
	case ".json":
		return IconJSON
	case ".yaml", ".yml":
		return IconYAML
	case ".toml", ".ini", ".cfg", ".conf", ".config":
		return IconConfig
	
	// Image files
	case ".png", ".jpg", ".jpeg", ".gif", ".bmp", ".svg", ".ico", ".webp":
		return IconImage
	
	// Video files
	case ".mp4", ".avi", ".mov", ".mkv", ".flv", ".wmv", ".webm":
		return IconVideo
	
	// Audio files
	case ".mp3", ".wav", ".flac", ".aac", ".ogg", ".wma", ".m4a":
		return IconAudio
	
	// Archive files
	case ".zip", ".tar", ".gz", ".bz2", ".xz", ".rar", ".7z":
		return IconArchive
	
	// Executable files
	case ".exe", ".app", ".deb", ".rpm":
		return IconExecutable
	}
	
	// Special file names
	switch strings.ToLower(name) {
	case "makefile", "dockerfile", "docker-compose.yml", "docker-compose.yaml":
		return IconConfig
	case "readme", "readme.md", "readme.txt":
		return IconMarkdown
	case ".gitignore", ".dockerignore":
		return IconConfig
	}
	
	// Check if it's a script without extension
	if fileType == "file" && (strings.HasPrefix(name, ".") || ext == "") {
		// Hidden files or files without extension often are config files
		return IconConfig
	}
	
	return IconDefault
}