# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview
Veda is a terminal-based text editor written in Go with vim-like features. It uses a Python proxy process for file system operations and implements a rope data structure for efficient text manipulation.

## Commands

### Development
```bash
# Run the editor
go run main.go

# Build the editor
go build -o veda

# Run all tests
go test ./...

# Run specific package tests (e.g., buffer tests)
go test ./internal/buffer

# Format code
go fmt ./...

# Check for issues
go vet ./...
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
- Normal mode: Arrow keys for navigation, 'A' opens file browser
- Command mode: Triggered by ':', supports commands like `:q` or `:quit`
- The editor loads test data (100 lines) on startup for scrolling demonstration

### Testing Strategy
Currently only the buffer package has tests. When adding new functionality:
- Unit tests go in `*_test.go` files in the same package
- Use standard Go testing conventions
- Run tests with `go test ./internal/package_name`