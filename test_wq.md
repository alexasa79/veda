# Testing :wq Command

The `:wq` command has been successfully implemented in Veda!

## What it does:
1. Writes the current buffer to disk (like `:w`)
2. Closes the current buffer (like `:q`)
3. If it's the last buffer, exits the editor

## Usage:
- Type `:wq` in command mode and press Enter
- The file will be saved and the buffer will close
- If multiple buffers are open, you'll switch to the next buffer
- If it's the only buffer, the editor will exit

## Implementation details:
- Added `writeAndQuit` function in `file_operations.go`
- Registered the command in `commands.go`
- Updated help documentation in `help.go`

This follows the vim convention where `:wq` is a common way to save and exit.