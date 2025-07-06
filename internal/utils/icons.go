package utils

import (
	"path/filepath"
	"strings"
)

// Common unicode symbols for files and directories
const (
	// Directory symbols (Nerd Font)
	IconDirectory     = "" // nf-fa-folder
	IconDirectoryOpen = "" // nf-fa-folder_open
	
	// Programming language symbols (Nerd Font)
	IconGo         = "\ue627" // nf-dev-go (Go gopher)
	IconPython     = ""      // nf-dev-python
	IconJavaScript = ""      // nf-dev-javascript
	IconRust       = ""      // nf-dev-rust
	IconC          = ""      // nf-custom-c
	IconCpp        = ""      // nf-custom-cpp
	
	// File type symbols (Nerd Font)
	IconText       = ""  // nf-fa-file_text
	IconMarkdown   = ""  // nf-dev-markdown
	IconJSON       = ""  // nf-seti-json
	IconYAML       = ""  // nf-seti-yml
	IconConfig     = ""  // nf-fa-cog
	IconImage      = ""  // nf-fa-file_image_o
	IconVideo      = ""  // nf-fa-file_video_o
	IconAudio      = ""  // nf-fa-file_audio_o
	IconArchive    = ""  // nf-fa-file_archive_o
	IconExecutable = ""  // nf-oct-zap
	IconDefault    = ""  // nf-fa-file
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
	
	// Special file names with specific icons
	switch strings.ToLower(name) {
	case "makefile":
		return "" // nf-seti-makefile
	case "dockerfile":
		return "" // nf-seti-docker
	case "docker-compose.yml", "docker-compose.yaml":
		return "" // nf-seti-docker
	case "readme", "readme.md", "readme.txt":
		return "" // nf-seti-readme
	case ".gitignore":
		return "" // nf-dev-git
	case ".dockerignore":
		return "" // nf-seti-docker
	case "package.json":
		return "" // nf-seti-npm
	case "go.mod", "go.sum":
		return "\ue627" // Go gopher
	}
	
	// Check if it's a script without extension
	if fileType == "file" && (strings.HasPrefix(name, ".") || ext == "") {
		// Hidden files or files without extension often are config files
		return IconConfig
	}
	
	return IconDefault
}