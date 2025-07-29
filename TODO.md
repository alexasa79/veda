# Small improvements and bugs
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
- [ ] It may be a good idea to calculate the number of lines in each rope node, so that when we are
at a specific node, we could calculate the line number from the beginning of the node and not from
the beginning of the buffer. This is potentially tricky to implement because nodes do not start at
the beginning of the line. Perhaps we should introduce this invariant as well.
- [ ] Consider removing ByteAt from buffer.go.
- [ ] Change the interface how to set various settings in the editor. `SETTING=VALUE` should work,
instead of `set SETTING VALUE` borrowed from vim.
- [ ] Expose scroll offset as a configurable setting.
- [ ] Reimplement the way theme is structured. We want to separate code theme and the editor theme,
where editor theme is the editor elements (status bar, etc).
- [x] Add something that shows that we have an LSP for the current file.
- [ ] Organize commands in `handleVariableAssignment` in some way. Ideally, the command itself
should say if it has a completion.
- [ ] I have multiple references to Buffer.Rope.String() in the code. Each such reference serializes
the entire buffer, which makes it inefficient. Find a solution to this.
- [ ] When rendering file that ends with \n, the last line does not show line number.

# Features
- [x] Remote support
- [ ] Multi-line/multi-cursor support
- [x] Selection support
- [x] Go to specific line
- [x] Search
- [ ] Replace
- [ ] Find references
- [x] List methods in the file
- [x] List methods in the folder
- [ ] Render tab characters and trailing space characters.
- [ ] Completion
- [x] Help
- [x] Help for key bindings
- [ ] Indent command
- [ ] Rewrap paragraph
- [ ] Copy/paste, registers support
- [ ] Macros support
- [ ] Eye candy for icons and other elements in the fuzzy list widget
