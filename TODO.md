# Bugs
- [ ] Allow going into search and command modes and exit the visual mode when it happens.
- [ ] Getting completion on empty space does something weird.
- [ ] References window is not rendered as markdown.
- [ ] Entering command that does not exist does not indicate anything.
- [ ] The error message for not found string looks aweful - red on gray background. Also, it should
  probably appear above the status bar.

# Features
- [ ] Multi-line/multi-cursor support
- [ ] Replace
- [ ] Indent command
- [ ] Registers support (for copy paste)
- [ ] Macros support
- [ ] Eye candy for icons and other elements in the fuzzy list widget

# Small improvements
- [ ] It may be a good idea to calculate the number of lines in each rope node, so that when we are
at a specific node, we could calculate the line number from the beginning of the node and not from
the beginning of the buffer. This is potentially tricky to implement because nodes do not start at
the beginning of the line. Perhaps we should introduce this invariant as well.
- [ ] Consider removing ByteAt from buffer.go.
- [ ] Expose scroll offset as a configurable setting.
- [ ] Reimplement the way theme is structured. We want to separate code theme and the editor theme,
where editor theme is the editor elements (status bar, etc).
- [ ] Organize commands in `handleVariableAssignment` in some way. Ideally, the command itself
should say if it has a completion.
- [ ] I have multiple references to Buffer.Rope.String() in the code. Each such reference serializes
the entire buffer, which makes it inefficient. Find a solution to this.

# Finished
## Features
- [x] Remote support
- [x] Selection support
- [x] Go to specific line
- [x] Search
- [x] List methods in the file
- [x] List methods in the folder
- [x] Help
- [x] Help for key bindings
- [x] Copy/paste
- [x] Add syntax highlighting for Markdown
- [x] Rewrap paragraph
- [x] Support :wq
- [x] Render tab characters and trailing space characters.
- [x] Find references
- [x] Completion
- [x] Add vertical rules at 78, 80, 100 and 120 characters.
- [x] Local history support
- [x] Pressing "c left" or "c right" does not work.

## Improvements and bugs
- [x] Remove `drawLine` (does not seem to be used anymore) and rename `drawLineFromIterator` to
`drawLine`.
- [x] getLineTokensFromIterator should not exist. Tokens should be a map from iterator to slice of
tokens and not a map from line index to slice of tokens.
- [x] In saveCurrentBufferState we should save the iterator and not the line number.
- [x] Get rid of cursorY in Editor. We're going to use an iterator instead.
- [x] Get rid of the rope walker business.
- [x] There is a cursor iterator in the editor and there is also one in the buffer. I wonder if we
need one in the buffer.
- [x] Remove the interface from the Rope. I do not see why we would need any abstraction there.
- [x] Add a generation number to the iterator as a way to make sure that all iterators are
invalidated whenever the buffer is changed.
- [x] Change the interface how to set various settings in the editor. `SETTING=VALUE` should work.
instead of `set SETTING VALUE` borrowed from vim.
- [x] Add something that shows that we have an LSP for the current file.
- [x] Visual mode does not work.
- [x] Open directory, press escape, crash.
- [x] When rendering file that ends with \n, the last line does not show line number.
- [x] Entering command that does not exist closes the editor.
- [x] Not supporting resizing the editor.
- [x] Also, completion does not show progress.
- [x] New files are not saved.
- [x] After paste, the newly pasted code is not highlighted.
