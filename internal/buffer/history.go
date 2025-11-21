package buffer

import "unicode"

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
	currentComposite  *InsertModeEntry
	inInsertMode      bool
	insertInitCommand string // Command that initiated current insert mode

	// Generic composite for grouping non-insert multi-step operations (e.g., multi-cursor)
	activeComposite *CompositeEntry
}

// NewHistory creates a new history manager
func NewHistory() *History {
	return &History{
		undoStack: make([]HistoryEntry, 0),
		redoStack: make([]HistoryEntry, 0),
	}
}

// SnapshotUndo returns up to n most recent undo entries' descriptions (newest first).
func (h *History) SnapshotUndo(n int) []string {
	if n <= 0 {
		n = len(h.undoStack)
	}
	if n > len(h.undoStack) {
		n = len(h.undoStack)
	}
	out := make([]string, 0, n)
	for i := 0; i < n; i++ {
		e := h.undoStack[len(h.undoStack)-1-i]
		out = append(out, e.Description())
	}
	return out
}

// SnapshotRedo returns up to n most recent redo entries' descriptions (newest first).
func (h *History) SnapshotRedo(n int) []string {
	if n <= 0 {
		n = len(h.redoStack)
	}
	if n > len(h.redoStack) {
		n = len(h.redoStack)
	}
	out := make([]string, 0, n)
	for i := 0; i < n; i++ {
		e := h.redoStack[len(h.redoStack)-1-i]
		out = append(out, e.Description())
	}
	return out
}

// Execute adds an entry to history and executes it
// Returns the entry that was added to history (might be nil if in insert mode)
func (h *History) Execute(buf *Buffer, entry HistoryEntry) (HistoryEntry, error) {
	// If we're in insert mode, add to composite entry
	if h.inInsertMode && h.currentComposite != nil {
		h.currentComposite.Add(entry)
		return nil, entry.Execute(buf)
	}

	// If a generic composite is active, accumulate entries into it
	if h.activeComposite != nil {
		h.activeComposite.Add(entry)
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

	// Special handling for insert-mode entries: undo only the last "word" chunk.
	if ime, ok := entry.(*InsertModeEntry); ok {
		chunk, err := ime.undoLastWord(buf)
		if err != nil {
			// Restore stack on error
			h.undoStack = append(h.undoStack, entry)
			return err
		}
		// If chunk is empty, nothing to do
		if chunk == nil {
			// Put entry back where it was
			h.undoStack = append(h.undoStack, entry)
			return nil
		}

		// Push the undone chunk to redo so it can be replayed.
		h.redoStack = append(h.redoStack, chunk)

		// If there are still entries remaining, keep the insert session on the undo stack.
		if len(ime.Entries) > 0 {
			h.undoStack = append(h.undoStack, ime)
		}
		return nil
	}

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

	// Push back to undo stack. If both this entry and the current top are
	// insert-mode entries from the same session type, merge them to keep a
	// single insert entry.
	if ime, ok := entry.(*InsertModeEntry); ok && len(h.undoStack) > 0 {
		if prev, okPrev := h.undoStack[len(h.undoStack)-1].(*InsertModeEntry); okPrev && prev.InitCommand == ime.InitCommand {
			prev.Entries = append(prev.Entries, ime.Entries...)
			return nil
		}
	}

	h.undoStack = append(h.undoStack, entry)

	return nil
}

// StartInsertSession begins tracking entries for insert mode
func (h *History) StartInsertSession(initCommand string) {
	h.inInsertMode = true
	h.insertInitCommand = initCommand
	h.currentComposite = NewInsertModeEntry("Insert mode session", initCommand)
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
		// If a generic composite is active, add the insert session into it
		// instead of pushing a separate undo unit. This ensures operations
		// like multi-cursor 'change' are undone in a single step.
		if h.activeComposite != nil {
			h.activeComposite.Add(h.currentComposite)
			result = h.currentComposite
		} else {
			h.undoStack = append(h.undoStack, h.currentComposite)
			// Clear redo stack when new entry is added
			h.redoStack = h.redoStack[:0]
			result = h.currentComposite
		}
	}

	h.currentComposite = nil
	return result
}

// StartComposite begins a generic composite entry for grouping multiple history
// operations into a single undo/redo unit (e.g., multi-cursor delete/change).
func (h *History) StartComposite(description string) {
	// If already in a composite, allow simple nesting by keeping the outer one.
	if h.activeComposite == nil {
		h.activeComposite = NewCompositeEntry(description)
	}
}

// HasActiveComposite reports whether a generic composite is currently open
func (h *History) HasActiveComposite() bool {
	return h.activeComposite != nil
}

// EndComposite finalizes the generic composite and pushes it to history if non-empty.
// Returns the composite entry if added, nil otherwise.
func (h *History) EndComposite() HistoryEntry {
	if h.activeComposite == nil {
		return nil
	}
	var result HistoryEntry
	// Push if there are entries OR if we have cursor snapshots, so editor-level
	// state changes (like virtual cursors) can be undone/redone.
	if len(h.activeComposite.Entries) > 0 || h.activeComposite.hasCursorSnapshot || h.activeComposite.hasPostCursorSnapshot {
		h.undoStack = append(h.undoStack, h.activeComposite)
		// Clear redo stack when a new entry is added
		h.redoStack = h.redoStack[:0]
		result = h.activeComposite
	}
	h.activeComposite = nil
	return result
}

// GetTopRedoCompositeCursors returns the cursor snapshot from the most recent redo entry if available
func (h *History) GetTopRedoCompositeCursors() (CursorPos, []CursorPos, bool) {
	if len(h.redoStack) == 0 {
		return CursorPos{}, nil, false
	}
	if ce, ok := h.redoStack[len(h.redoStack)-1].(*CompositeEntry); ok {
		primary, virtual, okSnap := ce.GetPreCursors()
		if !okSnap {
			return CursorPos{}, nil, false
		}
		return primary, virtual, true
	}
	if ime, ok := h.redoStack[len(h.redoStack)-1].(*InsertModeEntry); ok && ime.CompositeEntry != nil {
		primary, virtual, okSnap := ime.CompositeEntry.GetPreCursors()
		if !okSnap {
			return CursorPos{}, nil, false
		}
		return primary, virtual, true
	}
	return CursorPos{}, nil, false
}

// GetTopUndoCompositePostCursors returns the post-operation cursor snapshot
// from the most recent undo entry if available. Useful after Redo.
func (h *History) GetTopUndoCompositePostCursors() (CursorPos, []CursorPos, bool) {
	if len(h.undoStack) == 0 {
		return CursorPos{}, nil, false
	}
	if ce, ok := h.undoStack[len(h.undoStack)-1].(*CompositeEntry); ok {
		primary, virtual, okSnap := ce.GetPostCursors()
		if !okSnap {
			return CursorPos{}, nil, false
		}
		return primary, virtual, true
	}
	if ime, ok := h.undoStack[len(h.undoStack)-1].(*InsertModeEntry); ok && ime.CompositeEntry != nil {
		primary, virtual, okSnap := ime.CompositeEntry.GetPostCursors()
		if !okSnap {
			return CursorPos{}, nil, false
		}
		return primary, virtual, true
	}
	return CursorPos{}, nil, false
}

// SetActiveCompositeCursors sets a cursor snapshot on the active composite, if any
func (h *History) SetActiveCompositeCursors(primary CursorPos, virtual []CursorPos) {
	if h.activeComposite != nil {
		h.activeComposite.SetPreCursors(primary, virtual)
	}
}

// SetActiveCompositePostCursors stores the post-operation cursor snapshot on the active composite, if any
func (h *History) SetActiveCompositePostCursors(primary CursorPos, virtual []CursorPos) {
	if h.activeComposite != nil {
		h.activeComposite.SetPostCursors(primary, virtual)
	}
}

// CompositeEntry groups multiple entries into one
type CompositeEntry struct {
	description           string
	Entries               []HistoryEntry
	prePrimary            CursorPos
	preVirtual            []CursorPos
	hasCursorSnapshot     bool
	postPrimary           CursorPos
	postVirtual           []CursorPos
	hasPostCursorSnapshot bool
}

// NewCompositeEntry creates a new composite entry
func NewCompositeEntry(description string) *CompositeEntry {
	return &CompositeEntry{
		description: description,
		Entries:     make([]HistoryEntry, 0),
	}
}

// CursorPos represents a cursor position by line and character index
type CursorPos struct {
	Line int
	Char int
}

// SetPreCursors sets the cursor snapshot on the composite entry
func (ce *CompositeEntry) SetPreCursors(primary CursorPos, virtual []CursorPos) {
	ce.prePrimary = primary
	if virtual != nil {
		ce.preVirtual = make([]CursorPos, len(virtual))
		copy(ce.preVirtual, virtual)
	} else {
		ce.preVirtual = nil
	}
	ce.hasCursorSnapshot = true
}

// GetPreCursors returns the stored cursor snapshot if present
func (ce *CompositeEntry) GetPreCursors() (CursorPos, []CursorPos, bool) {
	if !ce.hasCursorSnapshot {
		return CursorPos{}, nil, false
	}
	if ce.preVirtual != nil {
		out := make([]CursorPos, len(ce.preVirtual))
		copy(out, ce.preVirtual)
		return ce.prePrimary, out, true
	}
	// Even if virtual is nil, primary may still be meaningful
	return ce.prePrimary, nil, true
}

// SetPostCursors stores the cursor snapshot after the composite operation
func (ce *CompositeEntry) SetPostCursors(primary CursorPos, virtual []CursorPos) {
	ce.postPrimary = primary
	if virtual != nil {
		ce.postVirtual = make([]CursorPos, len(virtual))
		copy(ce.postVirtual, virtual)
	} else {
		ce.postVirtual = nil
	}
	ce.hasPostCursorSnapshot = true
}

// GetPostCursors returns the stored post-operation cursor snapshot if present
func (ce *CompositeEntry) GetPostCursors() (CursorPos, []CursorPos, bool) {
	if !ce.hasPostCursorSnapshot {
		return CursorPos{}, nil, false
	}
	if ce.postVirtual != nil {
		out := make([]CursorPos, len(ce.postVirtual))
		copy(out, ce.postVirtual)
		return ce.postPrimary, out, true
	}
	return ce.postPrimary, nil, true
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

// InsertModeEntry wraps a CompositeEntry with insert mode context
type InsertModeEntry struct {
	*CompositeEntry
	InitCommand string // The command that initiated insert mode ("i", "a", "A", "o", "O")
}

// undoLastWord undoes the most recent word (plus trailing whitespace) within this insert session.
// It returns a new InsertModeEntry containing the undone chunk for redo.
func (ime *InsertModeEntry) undoLastWord(buf *Buffer) (*InsertModeEntry, error) {
	if len(ime.Entries) == 0 {
		return nil, nil
	}

	var removed []HistoryEntry
	cutoff := len(ime.Entries)
	sawNonWhitespace := false

	for i := len(ime.Entries) - 1; i >= 0; i-- {
		entry := ime.Entries[i]
		if err := entry.Undo(buf); err != nil {
			return nil, err
		}
		removed = append(removed, entry)

		hasWS, hasNonWS := entryTextKinds(entry)
		if hasNonWS {
			sawNonWhitespace = true
		}
		cutoff = i

		if (hasWS && sawNonWhitespace) || i == 0 {
			break
		}
	}

	// Keep the remaining prefix of the session
	ime.Entries = ime.Entries[:cutoff]

	if len(removed) == 0 {
		return nil, nil
	}

	// removed entries were gathered from end to start; reverse to original order for redo
	for l, r := 0, len(removed)-1; l < r; l, r = l+1, r-1 {
		removed[l], removed[r] = removed[r], removed[l]
	}

	chunk := NewInsertModeEntry("Insert mode session", ime.InitCommand)
	chunk.Entries = append(chunk.Entries, removed...)
	return chunk, nil
}

func entryTextKinds(entry HistoryEntry) (hasWhitespace bool, hasNonWhitespace bool) {
	switch e := entry.(type) {
	case *InsertTextEntry:
		return classifyText(e.Text)
	case *DeleteTextEntry:
		return classifyText(e.Text)
	default:
		return false, false
	}
}

func classifyText(text string) (hasWhitespace bool, hasNonWhitespace bool) {
	for _, r := range text {
		if unicode.IsSpace(r) {
			hasWhitespace = true
		} else {
			hasNonWhitespace = true
		}
	}
	return
}

// NewInsertModeEntry creates a new insert mode entry
func NewInsertModeEntry(description string, initCommand string) *InsertModeEntry {
	return &InsertModeEntry{
		CompositeEntry: NewCompositeEntry(description),
		InitCommand:    initCommand,
	}
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
	buf.Delete(pos, pos+len(ie.Text))

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
	buf.Delete(pos, pos+len(de.Text))

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
