package buffer

import (
	"bytes"
	"encoding/gob"
	"veda/internal/proxy"
)

func init() {
	gob.Register(&InsertTextEntry{})
	gob.Register(&DeleteTextEntry{})
	gob.Register(&CompositeEntry{})
	gob.Register(&InsertModeEntry{})
}

func encodeState(values ...interface{}) ([]byte, error) {
	var data bytes.Buffer
	enc := gob.NewEncoder(&data)
	for _, value := range values {
		if err := enc.Encode(value); err != nil {
			return nil, err
		}
	}
	return data.Bytes(), nil
}

func decodeState(data []byte, values ...interface{}) error {
	dec := gob.NewDecoder(bytes.NewReader(data))
	for _, value := range values {
		if err := dec.Decode(value); err != nil {
			return err
		}
	}
	return nil
}

func (h *History) GobEncode() ([]byte, error) {
	// Active groups belong to an unfinished command, never a reload boundary.
	return encodeState(h.undoStack, h.redoStack)
}
func (h *History) GobDecode(data []byte) error { return decodeState(data, &h.undoStack, &h.redoStack) }
func (ce *CompositeEntry) GobEncode() ([]byte, error) {
	return encodeState(ce.description, ce.Entries, ce.prePrimary, ce.preVirtual, ce.hasCursorSnapshot, ce.postPrimary, ce.postVirtual, ce.hasPostCursorSnapshot)
}
func (ce *CompositeEntry) GobDecode(data []byte) error {
	return decodeState(data, &ce.description, &ce.Entries, &ce.prePrimary, &ce.preVirtual, &ce.hasCursorSnapshot, &ce.postPrimary, &ce.postVirtual, &ce.hasPostCursorSnapshot)
}
func (ime *InsertModeEntry) GobEncode() ([]byte, error) {
	return encodeState(ime.CompositeEntry, ime.InitCommand)
}
func (ime *InsertModeEntry) GobDecode(data []byte) error {
	return decodeState(data, &ime.CompositeEntry, &ime.InitCommand)
}

// SessionBuffer contains durable buffer data. Iterators, syntax trees and
// callbacks are reconstructed in the new process.
type SessionBuffer struct {
	Text                                                string
	FilePath, BufferName, RealPath                      string
	IsModified, IsReadOnly, IsInternal                  bool
	CursorX, CursorLine, ViewLine                       int
	Indentation                                         IndentationInfo
	Generation                                          uint64
	History                                             *History
	VisualAnchorLine, VisualAnchorX, VisualAnchorOffset int
	GitChanges                                          *proxy.FileChanges
	GitBranch                                           string
	Diagnostics                                         []proxy.Diagnostic
	FoldingRanges                                       []proxy.FoldingRange
	MarkdownLinkTargets                                 map[string]string
	DiffHints                                           []DiffLine
	ChangedLines                                        map[int]bool
}

func (b *Buffer) SessionState() SessionBuffer {
	s := SessionBuffer{Text: b.Rope.String(), FilePath: b.FilePath, BufferName: b.BufferName, RealPath: b.RealPath,
		IsModified: b.IsModified, IsReadOnly: b.IsReadOnly, IsInternal: b.IsInternal, CursorX: b.CursorX,
		Indentation: b.Indentation, Generation: b.Generation, History: b.History,
		VisualAnchorLine: b.VisualAnchorLine, VisualAnchorX: b.VisualAnchorX, VisualAnchorOffset: b.VisualAnchorOffset,
		GitChanges: b.GitChanges, GitBranch: b.GitBranch, Diagnostics: b.Diagnostics, FoldingRanges: b.FoldingRanges,
		MarkdownLinkTargets: b.MarkdownLinkTargets, DiffHints: b.DiffHints, ChangedLines: b.ChangedLines}
	if b.CursorIterator != nil {
		s.CursorLine = b.CursorIterator.LineNum
	}
	if b.ViewIterator != nil {
		s.ViewLine = b.ViewIterator.LineNum
	}
	return s
}

func (s SessionBuffer) Restore() *Buffer {
	b := NewBuffer(s.Text, s.FilePath, s.RealPath, s.Indentation.Type, s.Indentation.Width)
	b.BufferName, b.IsModified, b.IsReadOnly, b.IsInternal = s.BufferName, s.IsModified, s.IsReadOnly, s.IsInternal
	b.Generation, b.CursorX = s.Generation, s.CursorX
	b.CursorIterator = NewBufferIteratorAtLine(b, s.CursorLine)
	b.ViewIterator = NewBufferIteratorAtLine(b, s.ViewLine)
	b.History = s.History
	b.VisualAnchorLine, b.VisualAnchorX, b.VisualAnchorOffset = s.VisualAnchorLine, s.VisualAnchorX, s.VisualAnchorOffset
	b.GitChanges, b.GitBranch, b.Diagnostics, b.FoldingRanges = s.GitChanges, s.GitBranch, s.Diagnostics, s.FoldingRanges
	b.MarkdownLinkTargets, b.DiffHints, b.ChangedLines = s.MarkdownLinkTargets, s.DiffHints, s.ChangedLines
	return b
}
