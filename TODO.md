# Bugs
- [ ] Multiple cursors appear in both windows.
- [ ] panic when closing the editor:
panic: runtime error: invalid memory address or nil pointer dereference
[signal SIGSEGV: segmentation violation code=0x2 addr=0xd0 pc=0x1008af7f8]

goroutine 15 [running]:
veda/internal/editor.(*Editor).writeFile.func1()
        /Users/sasandl/hobbies/veda/internal/editor/file_operations.go:964 +0xe8
created by veda/internal/editor.(*Editor).writeFile in goroutine 1
        /Users/sasandl/hobbies/veda/internal/editor/file_operations.go:951 +0x3b8
- [ ] Closing buffer also closes the window. The buffer in this case was a
      terminal.
