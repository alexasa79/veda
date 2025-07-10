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
2. **Python Proxy** (`veda-proxy.py`): JSON-RPC service for file operations, automatically started by the editor
3. **Buffer System** (`internal/buffer/`): Rope data structure for efficient text operations
4. **Widget System** (`internal/widgets/`): UI components including dual-pane file browser and command line editor

### Key Design Patterns
- **Modal Editing**: Vim-like modes (Normal, Command)
- **Event-Driven UI**: tcell event loop handles keyboard/mouse input
- **IPC Communication**: JSON-RPC between Go and Python over stdin/stdout
- **Component-Based UI**: Widgets manage their own rendering and input handling

### Navigation and Commands
- Normal mode: Arrow keys for navigation, 'space s f' opens file browser, 'w' moves to next word, 'b' moves to previous word
- Command mode: Triggered by ':', supports commands like `:q` or `:quit`
- The editor loads test data (100 lines) on startup for scrolling demonstration

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