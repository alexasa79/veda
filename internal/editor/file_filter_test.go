package editor

import (
	"testing"
	"veda/internal/proxy"
)

func TestShouldSkipFile(t *testing.T) {
	tests := []struct {
		name     string
		fileInfo proxy.FileInfo
		expected bool
		reason   string
	}{
		// Hidden files
		{
			name:     "hidden file",
			fileInfo: proxy.FileInfo{Path: "/home/user/.gitignore", Type: "file"},
			expected: true,
			reason:   "Should skip hidden files",
		},
		{
			name:     "file in hidden directory",
			fileInfo: proxy.FileInfo{Path: "/home/user/.config/settings.json", Type: "file"},
			expected: true,
			reason:   "Should skip files in hidden directories",
		},
		
		// Directories
		{
			name:     "directory",
			fileInfo: proxy.FileInfo{Path: "/home/user/src", Type: "directory"},
			expected: true,
			reason:   "Should skip directories",
		},
		
		// .d files
		{
			name:     "dependency file",
			fileInfo: proxy.FileInfo{Path: "/home/user/main.d", Type: "file"},
			expected: true,
			reason:   "Should skip .d dependency files",
		},
		
		// Binary files
		{
			name:     "object file",
			fileInfo: proxy.FileInfo{Path: "/home/user/main.o", Type: "file"},
			expected: true,
			reason:   "Should skip .o object files",
		},
		{
			name:     "executable",
			fileInfo: proxy.FileInfo{Path: "/home/user/program.exe", Type: "file"},
			expected: true,
			reason:   "Should skip .exe files",
		},
		{
			name:     "shared library",
			fileInfo: proxy.FileInfo{Path: "/home/user/lib.so", Type: "file"},
			expected: true,
			reason:   "Should skip .so files",
		},
		{
			name:     "archive",
			fileInfo: proxy.FileInfo{Path: "/home/user/lib.a", Type: "file"},
			expected: true,
			reason:   "Should skip .a archive files",
		},
		{
			name:     "python bytecode",
			fileInfo: proxy.FileInfo{Path: "/home/user/module.pyc", Type: "file"},
			expected: true,
			reason:   "Should skip .pyc files",
		},
		{
			name:     "image file",
			fileInfo: proxy.FileInfo{Path: "/home/user/photo.jpg", Type: "file"},
			expected: true,
			reason:   "Should skip image files",
		},
		{
			name:     "a.out",
			fileInfo: proxy.FileInfo{Path: "/home/user/a.out", Type: "file"},
			expected: true,
			reason:   "Should skip a.out files",
		},
		{
			name:     "core dump",
			fileInfo: proxy.FileInfo{Path: "/home/user/core", Type: "file"},
			expected: true,
			reason:   "Should skip core dump files",
		},
		{
			name:     "vd binary",
			fileInfo: proxy.FileInfo{Path: "/home/user/vd", Type: "file"},
			expected: true,
			reason:   "Should skip vd editor binary",
		},
		
		// Files that should NOT be skipped
		{
			name:     "source file",
			fileInfo: proxy.FileInfo{Path: "/home/user/main.c", Type: "file"},
			expected: false,
			reason:   "Should include source files",
		},
		{
			name:     "go file",
			fileInfo: proxy.FileInfo{Path: "/home/user/main.go", Type: "file"},
			expected: false,
			reason:   "Should include Go files",
		},
		{
			name:     "header file",
			fileInfo: proxy.FileInfo{Path: "/home/user/header.h", Type: "file"},
			expected: false,
			reason:   "Should include header files",
		},
		{
			name:     "text file",
			fileInfo: proxy.FileInfo{Path: "/home/user/README.md", Type: "file"},
			expected: false,
			reason:   "Should include text files",
		},
		{
			name:     "Makefile",
			fileInfo: proxy.FileInfo{Path: "/home/user/Makefile", Type: "file"},
			expected: false,
			reason:   "Should include Makefiles",
		},
		{
			name:     "README without extension",
			fileInfo: proxy.FileInfo{Path: "/home/user/README", Type: "file"},
			expected: false,
			reason:   "Should include README files without extension",
		},
		{
			name:     "LICENSE",
			fileInfo: proxy.FileInfo{Path: "/home/user/LICENSE", Type: "file"},
			expected: false,
			reason:   "Should include LICENSE files",
		},
		{
			name:     "case insensitive binary extension",
			fileInfo: proxy.FileInfo{Path: "/home/user/IMAGE.JPG", Type: "file"},
			expected: true,
			reason:   "Should skip files with binary extensions regardless of case",
		},
		{
			name:     "d source file not dependency",
			fileInfo: proxy.FileInfo{Path: "/home/user/module.d.ts", Type: "file"},
			expected: false,
			reason:   "Should not skip .d.ts TypeScript declaration files",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shouldSkipFile(tt.fileInfo)
			if result != tt.expected {
				t.Errorf("%s: expected %v, got %v (%s)", tt.name, tt.expected, result, tt.reason)
			}
		})
	}
}