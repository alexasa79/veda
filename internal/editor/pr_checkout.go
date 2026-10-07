package editor

import (
	"fmt"
	"strings"
	"veda/internal/buffer"
	"veda/internal/proxy"
	"veda/internal/utils"
)

const prCheckoutOperation = "pr-checkout"

// Navigation must prefer a real file buffer over the aligned diff rows, even
// when the definition is in the file currently being reviewed.
func (e *Editor) ensurePRWorkingBuffer(path, content string) error {
	if path == "" {
		return nil
	}
	if b := e.buffers[path]; b != nil {
		if b.IsModified {
			return fmt.Errorf("save or close unsaved changes in %s before opening its PR diff", path)
		}
		if !b.HasFile() || b.Rope.String() != content {
			b.RealPath, b.IsReadOnly = path, false
			e.reloadPRWorkingBuffer(b, content)
		}
		return nil
	}
	b := buffer.NewBuffer(content, path, path, e.config.DefaultIndentType, e.config.DefaultIndentWidth)
	e.buffers[path] = b
	e.bufferOrder = append(e.bufferOrder, path)
	w := e.windows[e.activeWindow]
	w.Buffers = append(w.Buffers, b)
	w.Views = append(w.Views, &SingleBufferView{Buf: b})
	return nil
}

func (e *Editor) preparePRCheckout() ([]string, error) {
	if e.hasUnsavedChanges() {
		return nil, fmt.Errorf("save or close unsaved file buffers before checking out a PR")
	}
	if e.hasAsyncOperations() {
		return nil, fmt.Errorf("wait for current operations to finish before checking out a PR")
	}
	var files []string
	for _, b := range e.buffers {
		if b.HasFile() && !e.isDiffBuffer(b) && !b.IsInternal {
			files = append(files, b.RealPath)
		}
	}
	e.startAsyncOperation(prCheckoutOperation, "loading PR review")
	return files, nil
}

func (e *Editor) applyPRCheckout(result *proxy.PRCheckoutResult) {
	if result == nil || result.Root == "" {
		return
	}
	// Any older diff may now refer to another branch. Preserve its contents,
	// but prevent editing, staging, or navigation through stale file associations.
	for _, b := range e.buffers {
		if e.isDiffBuffer(b) && (b.RealPath == result.Root || strings.HasPrefix(b.RealPath, result.Root+"/")) {
			b.RealPath = ""
			b.IsReadOnly = true
		}
	}
	for _, file := range result.Files {
		for _, b := range e.buffers {
			if e.isDiffBuffer(b) || b.RealPath != file.Path {
				continue
			}
			utils.Assert(!b.IsModified, "buffer modified during PR checkout")
			if file.Error != "" {
				// Keep the old contents available, but detach a removed/unreadable file.
				b.RealPath = ""
				b.IsReadOnly = true
				e.ShowInfo("Checkout: %s is no longer available: %s", file.Path, file.Error)
				continue
			}
			e.reloadPRWorkingBuffer(b, file.Content)
		}
	}
	// The proxy stopped the old servers; the next navigation starts fresh ones.
	e.lspExtensionStatus = make(map[string]string)
	for _, w := range e.windows {
		if w.Buffer != nil {
			w.ViewLine = w.Buffer.ViewIterator.LineNumber()
			w.CursorLine = w.Buffer.CursorIterator.LineNumber()
			w.CursorX = w.Buffer.CursorX
		}
	}
}

func (e *Editor) reloadPRWorkingBuffer(b *buffer.Buffer, content string) {
	line := b.CursorIterator.LineNumber()
	b.Rope = buffer.NewRopeFromString(content)
	b.Generation++
	b.MarkModified()
	b.MarkSaved()
	b.History = buffer.NewHistory()
	b.CursorIterator = buffer.NewBufferIteratorAtLine(b, min(line, b.LineCount-1))
	b.CursorX = min(b.CursorX, len(b.CursorIterator.Line()))
	b.ViewIterator = buffer.NewBufferIterator(b)
	b.ChangedLines = make(map[int]bool)
	b.GitChanges, b.Diagnostics, b.FoldingRanges = nil, nil, nil
	b.CurrentSignature, b.ContextCache = nil, nil
	b.GitBranch = ""
	b.UpdateLineNumberWidth()
	delete(e.virtualCursorsByBuffer, b)
	if e.CurrentBuffer() == b {
		e.virtualCursors = nil
	}
	if b.Highlighter != nil {
		b.Highlighter.ParseFull([]byte(content))
	}
}
