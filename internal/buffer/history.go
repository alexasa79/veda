package buffer

// HistoryEntry represents an undoable buffer operation
type HistoryEntry interface {
	// Execute performs the operation
	Execute(buf *Buffer) error
	// Undo reverses the operation
	Undo(buf *Buffer) error
	// Description returns a human-readable description for debugging
	Description() string
}

// History manages undo and redo stacks for a buffer
type History struct {
	undoStack []HistoryEntry
	redoStack []HistoryEntry

	// For tracking insert mode sessions
	currentComposite *CompositeEntry
	inInsertMode     bool
}

// NewHistory creates a new history manager
func NewHistory() *History {
	return &History{
		undoStack: make([]HistoryEntry, 0),
		redoStack: make([]HistoryEntry, 0),
	}
}

// Execute adds an entry to history and executes it
// Returns the entry that was added to history (might be nil if in insert mode)
func (h *History) Execute(buf *Buffer, entry HistoryEntry) (HistoryEntry, error) {
	// If we're in insert mode, add to composite entry
	if h.inInsertMode && h.currentComposite != nil {
		h.currentComposite.Add(entry)
		return nil, entry.Execute(buf)
	}

	// Execute the entry
	if err := entry.Execute(buf); err != nil {
		return nil, err
	}

	// Add to undo stack
	h.undoStack = append(h.undoStack, entry)

	// Clear redo stack when new entry is executed
	h.redoStack = h.redoStack[:0]

	return entry, nil
}

// Undo performs undo operation
func (h *History) Undo(buf *Buffer) error {
	if len(h.undoStack) == 0 {
		return nil // Nothing to undo
	}

	// Pop from undo stack
	entry := h.undoStack[len(h.undoStack)-1]
	h.undoStack = h.undoStack[:len(h.undoStack)-1]

	// Undo the entry
	if err := entry.Undo(buf); err != nil {
		// On error, restore to undo stack
		h.undoStack = append(h.undoStack, entry)
		return err
	}

	// Push to redo stack
	h.redoStack = append(h.redoStack, entry)

	return nil
}

// Redo performs redo operation
func (h *History) Redo(buf *Buffer) error {
	if len(h.redoStack) == 0 {
		return nil // Nothing to redo
	}

	// Pop from redo stack
	entry := h.redoStack[len(h.redoStack)-1]
	h.redoStack = h.redoStack[:len(h.redoStack)-1]

	// Execute the entry
	if err := entry.Execute(buf); err != nil {
		// On error, restore to redo stack
		h.redoStack = append(h.redoStack, entry)
		return err
	}

	// Push back to undo stack
	h.undoStack = append(h.undoStack, entry)

	return nil
}

// StartInsertSession begins tracking entries for insert mode
func (h *History) StartInsertSession() {
	h.inInsertMode = true
	h.currentComposite = NewCompositeEntry("Insert mode session")
}

// EndInsertSession finalizes the insert mode session
// Returns the composite entry if there were changes, nil otherwise
func (h *History) EndInsertSession() HistoryEntry {
	if !h.inInsertMode {
		return nil
	}

	h.inInsertMode = false

	var result HistoryEntry
	// Only add to history if there were actual changes
	if h.currentComposite != nil && len(h.currentComposite.Entries) > 0 {
		h.undoStack = append(h.undoStack, h.currentComposite)
		// Clear redo stack when new entry is added
		h.redoStack = h.redoStack[:0]
		result = h.currentComposite
	}

	h.currentComposite = nil
	return result
}

// CompositeEntry groups multiple entries into one
type CompositeEntry struct {
	description string
	Entries     []HistoryEntry
}

// NewCompositeEntry creates a new composite entry
func NewCompositeEntry(description string) *CompositeEntry {
	return &CompositeEntry{
		description: description,
		Entries:     make([]HistoryEntry, 0),
	}
}

// Add adds an entry to the composite
func (ce *CompositeEntry) Add(entry HistoryEntry) {
	ce.Entries = append(ce.Entries, entry)
}

// Execute executes all entries in order
func (ce *CompositeEntry) Execute(buf *Buffer) error {
	for _, entry := range ce.Entries {
		if err := entry.Execute(buf); err != nil {
			return err
		}
	}
	return nil
}

// Undo reverses all entries in reverse order
func (ce *CompositeEntry) Undo(buf *Buffer) error {
	// Undo in reverse order
	for i := len(ce.Entries) - 1; i >= 0; i-- {
		if err := ce.Entries[i].Undo(buf); err != nil {
			return err
		}
	}
	return nil
}

// Description returns the composite entry description
func (ce *CompositeEntry) Description() string {
	return ce.description
}

// InsertTextEntry represents text insertion
type InsertTextEntry struct {
	Line      int    // Line number where insertion starts
	Character int    // Character position in line
	Text      string // Text to insert
}

// NewInsertTextEntry creates a new insert text entry
func NewInsertTextEntry(line, character int, text string) *InsertTextEntry {
	return &InsertTextEntry{
		Line:      line,
		Character: character,
		Text:      text,
	}
}

// Execute inserts the text
func (ie *InsertTextEntry) Execute(buf *Buffer) error {
	// Use the optimized line-based iterator
	iter := NewBufferIteratorAtLine(buf, ie.Line)
	pos := iter.Offset + ie.Character

	// Insert the text at the calculated position
	buf.Insert(pos, ie.Text)

	// Update cursor position
	buf.CursorIterator = NewBufferIteratorAtLine(buf, ie.Line)
	buf.CursorX = ie.Character + len(ie.Text)

	return nil
}

// Undo removes the inserted text
func (ie *InsertTextEntry) Undo(buf *Buffer) error {
	// Use the optimized line-based iterator
	iter := NewBufferIteratorAtLine(buf, ie.Line)
	pos := iter.Offset + ie.Character

	// Delete the inserted text
	buf.Delete(pos, pos + len(ie.Text))

	// After delete, if buffer is empty, ensure it has a valid rope
	if buf.Rope.root == nil {
		buf.Rope = NewRopeFromString("")
	}

	// Restore cursor position
	buf.CursorIterator = NewBufferIteratorAtLine(buf, ie.Line)
	buf.CursorX = ie.Character

	return nil
}

// Description returns entry description
func (ie *InsertTextEntry) Description() string {
	if len(ie.Text) == 1 {
		return "Insert char: " + ie.Text
	}
	return "Insert text: " + ie.Text
}

// DeleteTextEntry represents text deletion
type DeleteTextEntry struct {
	Line      int    // Line number where deletion starts
	Character int    // Character position in line
	Text      string // Text that was deleted (for undo)
}

// NewDeleteTextEntry creates a new delete text entry
func NewDeleteTextEntry(line, character int, text string) *DeleteTextEntry {
	return &DeleteTextEntry{
		Line:      line,
		Character: character,
		Text:      text,
	}
}

// Execute deletes the text (for redo)
func (de *DeleteTextEntry) Execute(buf *Buffer) error {
	// Use the optimized line-based iterator
	iter := NewBufferIteratorAtLine(buf, de.Line)
	pos := iter.Offset + de.Character

	// Delete the text
	buf.Delete(pos, pos + len(de.Text))

	// After delete, if buffer is empty, ensure it has a valid rope
	if buf.Rope.root == nil {
		buf.Rope = NewRopeFromString("")
	}

	// Update cursor position
	buf.CursorIterator = NewBufferIteratorAtLine(buf, de.Line)
	buf.CursorX = de.Character

	return nil
}

// Undo restores the deleted text
func (de *DeleteTextEntry) Undo(buf *Buffer) error {
	// Use the optimized line-based iterator
	iter := NewBufferIteratorAtLine(buf, de.Line)
	pos := iter.Offset + de.Character

	// Restore the deleted text
	buf.Insert(pos, de.Text)

	// Restore cursor position
	buf.CursorIterator = NewBufferIteratorAtLine(buf, de.Line)
	buf.CursorX = de.Character

	return nil
}

// Description returns entry description
func (de *DeleteTextEntry) Description() string {
	if len(de.Text) == 1 {
		return "Delete char: " + de.Text
	}
	return "Delete text: " + de.Text
}