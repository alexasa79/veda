# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview
Veda is a terminal-based text editor written in Go with vim-like features. It uses a Python proxy process for file system operations and implements a rope data structure for efficient text manipulation.

## Commands

### Makefile Targets
```bash
# Build the editor (default target)
make
# or
make all

# Run all tests
make test

# Clean build artifacts
make clean
```

## Architecture

### Core Components
1. **Go Editor Core** (`main.go`, `internal/editor/editor.go`): Terminal UI using tcell/v2
2. **Python Proxy** (`veda-proxy.py`): JSON-RPC service for file operations and LSP server management, automatically started by the editor
3. **Buffer System** (`internal/buffer/`): Rope data structure for efficient text operations
4. **Widget System** (`internal/widgets/`): UI components including dual-pane file browser and command line editor
5. **LSP Integration** (`LSPManager` in proxy): Manages language servers for code intelligence features

### Key Design Patterns
- **Modal Editing**: Vim-like modes (Normal, Insert, Command)
- **Event-Driven UI**: tcell event loop handles keyboard/mouse input
- **IPC Communication**: JSON-RPC between Go and Python over stdin/stdout
- **Component-Based UI**: Widgets manage their own rendering and input handling
- **Single-Line Status Bar**: Minimal status bar showing mode, filename, and cursor position

### Navigation and Commands
- Normal mode: Arrow keys for navigation, 'space s f' opens file browser, 'w' moves to next word, 'b' moves to previous word, '0' or Home moves to beginning of line, '$' or End moves to end of line, 'g d' goes to definition (LSP). Block cursor.
- Insert mode: Press 'i' to enter, 'Escape' to exit. Type to insert text, 'Enter' for newline, 'Backspace' to delete. Arrow keys work for cursor movement while staying in insert mode. Home/End keys move to beginning/end of line. Bar cursor.
- Command mode: Triggered by ':', supports commands like `:q` or `:quit`. Block cursor.
- The editor loads test data (100 lines) on startup for scrolling demonstration

### Language Server Protocol (LSP) Support
- **Go to Definition**: Press 'g d' in normal mode to navigate to the definition of the symbol under the cursor
- **Supported Languages**: 
  - Go (requires `gopls` - install with `go install golang.org/x/tools/gopls@latest`)
  - C/C++ (requires `clangd` - install from your package manager or https://clangd.llvm.org/)
- **Automatic Server Management**: LSP servers start automatically when needed and stop when the editor exits

### Known Issues and Workarounds
- 'A' no longer opens file browser. To open file browser type 'space s f'.

### Testing Framework
The project includes a comprehensive unit testing framework:
- **Screen Abstraction**: `Screen` interface allows swapping between real terminal and mock implementation
- **MockScreen**: Test implementation that captures rendered output and cursor position
- **TestEditor**: Wrapper providing test-friendly methods like `ExecuteCommand()` and `GetCursorPosition()`
- **Assertion Helpers**: Functions like `AssertCursorAt()` and `AssertBufferEquals()` for verifying editor state

Example test:
```go
te := NewTestEditorWithContent(80, 24, "Hello, World!")
te.ExecuteCommand("right")
AssertCursorAt(t, te, 1, 0)
```

### Testing Strategy
Currently the buffer package and editor package have tests. When adding new functionality:
- Unit tests go in `*_test.go` files in the same package
- Use standard Go testing conventions
- Run tests with `go test ./internal/package_name` or use `make test` to run all tests
- Use the testing framework for editor behavior tests (cursor movement, editing operations, etc.)
```