# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview
Veda is a terminal-based text editor written in Go with vim-like features. It uses a Python proxy process for file system operations and implements a rope data structure for efficient text manipulation. The compiled binary is named `vd`.

## Commands

### Makefile Targets
- Use these commands to build the project:
```bash
# Build the editor binary 'vd' (default target)
make
# or
make all

# Run all tests
make test

# Clean build artifacts (removes the 'vd' binary)
make clean
```

## Architecture

### Core Components
1. **Go Editor Core** (`main.go`, `internal/editor/`): Terminal UI using tcell/v2
   - `editor.go`: Main editor struct and core functionality (1,219 lines)
   - `edit_operations.go`: Text editing operations (insertChar, backspace, delete, etc.)
   - `search.go`: Search and find functionality
   - `scroll.go`: Viewport scrolling operations
   - `mode_management.go`: Mode switching between Normal, Insert, Command, Visual modes
   - `layout.go`: Layout calculations and screen area management
   - `async_operations.go`: Async operation tracking and spinner animations
   - `command_execution.go`: Command completion functionality
2. **Python Proxy** (`veda-proxy.py`): JSON-RPC service for file operations and LSP server management, automatically started by the editor
3. **Buffer System** (`internal/buffer/`): Rope data structure for efficient text operations
4. **Widget System** (`internal/widgets/`): UI components including dual-pane file browser and command line editor
5. **LSP Integration** (`LSPManager` in proxy): Manages language servers for code intelligence features

### Key Design Patterns
- **Modal Editing**: Vim-like modes (Normal, Insert, Command)
- **Event-Driven UI**: tcell like event loop handles keyboard/mouse input
- **IPC Communication**: JSON-RPC between Go and Python over stdin/stdout
- **Component-Based UI**: Widgets manage their own rendering and input handling
- **Single-Line Status Bar**: Minimal status bar showing mode, filename, and cursor position

### Navigation and Commands
- Normal mode: Arrow keys for navigation, 'space s f' opens file browser, 'w' moves to next word, 'b' moves to previous word, '0' or Home moves to beginning of line, '$' or End moves to end of line, 'g d' goes to definition (LSP). Block cursor.
- Insert mode: Press 'i' to enter, 'Escape' to exit. Type to insert text, 'Enter' for newline, 'Backspace' to delete. Arrow keys work for cursor movement while staying in insert mode. Home/End keys move to beginning/end of line. Bar cursor.
- Command mode: Triggered by ':', supports commands like `:q` or `:quit`. Block cursor.
- The editor loads test data (100 lines) on startup for scrolling demonstration

### Undo/Redo System
- **Undo**: Press 'u' in normal mode to undo the last change
- **Redo**: Press 'U' (shift-u) in normal mode to redo an undone change
- **Command History**: All text modifications are tracked as commands for undo/redo (per buffer)
- **Insert Mode Sessions**: All changes made during a single insert mode session (from entering to exiting insert mode) are grouped as one undo unit
- **Important**: When implementing new text-modifying commands, they MUST support undo/redo by implementing the Command interface

### Language Server Protocol (LSP) Support
- **Go to Definition**: Press 'g d' in normal mode to navigate to the definition of the symbol under the cursor
- **Supported Languages**:
  - Go (requires `gopls` - install with `go install golang.org/x/tools/gopls@latest`)
  - C/C++ (requires `clangd` - install from your package manager or https://clangd.llvm.org/)
- **Automatic Server Management**: LSP servers start automatically when needed and stop when the editor exits

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

## Memory Notes
- When working with the Editor's buffer, assert that the buffer field is always initialized and populated. No need to add extra null checks.
- When using Buffer and Iterator objects, any change to the buffer invalidates all current iterators pointing to the buffer. To work with these primitives, remember the line number of the iterator before making the change and then call NewBufferIteratorAtLine to restore the iterator.
- `tracker.go` implements a time tracker that we use for performance measurements. The typical use pattern is to create a performance tracker object. Then to call `t0 := tracker.Start()`, followed by `t0 = tracker.ReportSince(..., t0)`.
- Always look for implementation that reduces the amount of state we have to keep. For instance if we have a boolean that tells us if the state is present and a pointer to the state's structure, we can remove the boolean and check if the pointer is nil.

## Implementation Guidelines
- **Undo/Redo Support**: All new commands that modify text MUST implement undo/redo functionality. This includes:
  - Storing necessary state to reverse the operation (e.g., deleted text, cursor positions)
  - Implementing the Command interface with Execute() and Undo() methods
  - Ensuring commands handle buffer iterator invalidation correctly
  - Testing undo/redo behavior for edge cases
- **Adding New Commands**: When adding new commands to the editor:
  - Register the command in `internal/editor/commands.go` in the `registerCommands()` function
  - Add key bindings in `internal/editor/keybindings.go` in the `setupDefaultBindings()` function if needed
  - **IMPORTANT**: Update the help documentation in `internal/editor/help.go` to include the new command
  - Add the command to the appropriate category in the help text with its key binding and description
- **Code Organization**: The editor package is modularly organized:
  - Text editing operations should go in `edit_operations.go`
  - Search-related functionality should go in `search.go`
  - Scrolling and viewport management should go in `scroll.go`
  - Mode switching logic should go in `mode_management.go`
  - Layout calculations should go in `layout.go`
  - Keep `editor.go` focused on core editor struct and high-level coordination

## Claude Operational Guidelines
- You are not allowed to leave the root directory of the project. All commands, experiments, everything you do, must be done in the root directory of the project.
- You are not allowed to change the current directory.

## Operational Guidelines
- Stop telling the user he is right. Be cautious with this because the user may not be right. You should be sceptical about what the user asks and suggests. Do not thank user for every input. This is annoying.

## Permissions and Capabilities
- You are allowed to run ls commands.

## Proxy Implementation Notes
- All access to the file is implemented via proxy process. The proxy is written in Python. It has a client side implemented in proxy.go.

## Project Language Notes
- The project is written in Go. The only file written in other programming language is veda-proxy.py. This is the only non-Go file this project will ever have.

## Editor Interaction Notes
- This is an editor project. It is an interactive program and you cannot run it yourself. If you want to test and run the actual editor, ask the user to do it for you and report back the results.

## Debugging difficult problems
When debugging difficult problems, the user can run tests for you. Ask the user to test things for you. The editor has debug log (veda.log), that you can check.