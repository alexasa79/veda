package editor

import (
	"fmt"
	"strings"
	"veda/internal/buffer"
	"veda/internal/proxy"
	"veda/internal/syntax"
	"veda/internal/ui"
)

// A nil state disables the preview. Keep only the active buffer's comparison.
type changePreviewState struct {
	buffer     *buffer.Buffer
	generation uint64
	changes    *proxy.FileChanges
	pending    bool
	diff       *buffer.DiffResult
}

type changePreviewBlock struct {
	lines       []string
	focus       int
	first, last int // Current buffer lines, 0-based.
	deleted     bool
}

func (b *changePreviewBlock) title() string {
	if b.deleted {
		return fmt.Sprintf("Before · HEAD · deletion at L%d", b.first+1)
	}
	if b.first == b.last {
		return fmt.Sprintf("Before · HEAD · replaces L%d", b.first+1)
	}
	return fmt.Sprintf("Before · HEAD · replaces L%d–L%d", b.first+1, b.last+1)
}

func (e *Editor) toggleChangePreview(args []string) error {
	if e.changePreview == nil {
		e.changePreview = &changePreviewState{}
		e.ShowTransientInfo("Change preview enabled (HEAD)")
	} else {
		e.changePreview = nil
		e.ShowTransientInfo("Change preview disabled")
	}
	return nil
}

// Fetch and compare snapshots off the UI thread. Moving the cursor reuses the
// comparison; edits and refreshed Git metadata invalidate it.
func (e *Editor) changePreviewDiff(buf *buffer.Buffer) *buffer.DiffResult {
	state := e.changePreview
	if state.pending {
		return nil
	}
	if state.buffer == buf && state.generation == buf.Generation && state.changes == buf.GitChanges {
		return state.diff
	}
	if e.proxyManager == nil {
		return nil
	}
	request := &changePreviewState{
		buffer: buf, generation: buf.Generation, changes: buf.GitChanges, pending: true,
	}
	e.changePreview = request
	path := buf.RealPath
	if path == "" {
		path = buf.FilePath
	}
	content := buf.Rope.String()
	go func() {
		previous, err := e.proxyManager.GetGitFileVersion(path, "HEAD")
		var diff *buffer.DiffResult
		if err == nil {
			diff = e.computeDiffViaProxy(previous, content)
		}
		e.Post(func(e *Editor) {
			if e.changePreview == request {
				request.pending = false
				request.diff = diff
			}
		})
	}()
	return nil
}

// Select whole change blocks rather than claiming an exact old/new line match.
// A deletion is attached to the following line, or the last line at EOF.
func changePreviewLines(diff *buffer.DiffResult, cursorLine, lineCount int) *changePreviewBlock {
	if diff == nil {
		return nil
	}
	for start := 0; start < len(diff.RightLines); {
		if diff.RightLines[start].LineType == buffer.DiffLineContext {
			start++
			continue
		}
		end := start
		first, last := -1, -1
		var old []string
		for end < len(diff.RightLines) && diff.RightLines[end].LineType != buffer.DiffLineContext {
			left, right := diff.LeftLines[end], diff.RightLines[end]
			if !left.IsFillerLine {
				old = append(old, left.Content)
			}
			if !right.IsFillerLine {
				if first < 0 {
					first = right.RightLine
				}
				last = right.RightLine
			}
			end++
		}
		deleted := first < 0
		if deleted {
			first = max(0, lineCount-1)
			if end < len(diff.RightLines) {
				first = diff.RightLines[end].RightLine
			}
			last = first
		}
		if cursorLine >= first && cursorLine <= last && len(old) > 0 {
			return &changePreviewBlock{lines: old, focus: min(cursorLine-first, len(old)-1), first: first, last: last, deleted: deleted}
		}
		start = end
	}
	return nil
}

func changePreviewRows(lines []string, width int) []WrappedLine {
	wrapper := NewWrapCalculator(width, 0)
	var rows []WrappedLine
	offset := 0
	for _, line := range lines {
		// Match the editor's four-column display of tabs.
		line = strings.ReplaceAll(line, "\t", "    ")
		for _, wrapped := range wrapper.WrapLine(line, 0) {
			wrapped.StartChar += offset
			wrapped.EndChar += offset
			rows = append(rows, wrapped)
		}
		offset += len(line) + 1
	}
	return rows
}

// The marker and preview share Git's deletion anchor and text. Do not infer
// deleted text from a second alignment, which can disagree on repeated lines.
func deletedTextAtLine(buf *buffer.Buffer, line int) []string {
	if buf.GitChanges == nil {
		return nil
	}
	var lines []string
	for _, hunk := range buf.GitChanges.DeletedHunks {
		if min(hunk.Line-1, buf.LineCount-1) == line {
			lines = append(lines, hunk.Lines...)
		}
	}
	return lines
}

func (e *Editor) drawChangePreview(win *Window, screen Screen) {
	buf := win.Buffer
	if e.changePreview == nil || buf == nil || buf.IsInternal || !buf.HasFile() || e.renderingState.IsOverlayActive() {
		return
	}
	width, height := screen.Size()
	if width < 24 || height < 8 {
		return
	}
	block := &changePreviewBlock{lines: deletedTextAtLine(buf, win.CursorLine), first: win.CursorLine, last: win.CursorLine, deleted: true}
	if len(block.lines) == 0 {
		block = changePreviewLines(e.changePreviewDiff(buf), win.CursorLine, buf.LineCount)
	}
	if block == nil {
		return
	}
	lines, focus := block.lines, block.focus
	// Measure visible code and pinned context in the active window.
	realScreen, realBuffer := e.screen, e.CurrentBuffer()
	savedView, savedCursor, savedX := buf.ViewIterator, buf.CursorIterator, buf.CursorX
	e.screen = screen
	e.SetCurrentBuffer(buf)
	buf.ViewIterator = buffer.NewBufferIteratorAtLine(buf, win.ViewLine)
	buf.CursorIterator = buffer.NewBufferIteratorAtLine(buf, win.CursorLine)
	buf.CursorX = win.CursorX
	contentStart := e.getContentStartY()
	// Window content starts at zero, but pinned context is painted over its
	// first rows. Reserve those rows independently of the cursor offset.
	top := max(e.getContentStartY(), len(buf.ContextCache)) + 2
	visibleLines := e.CalculateWrappedLines()
	buf.ViewIterator, buf.CursorIterator, buf.CursorX = savedView, savedCursor, savedX
	e.screen = realScreen
	e.SetCurrentBuffer(realBuffer)

	bottom := height - 2
	y := top
	maxHeight := (bottom - top) / 2
	if maxHeight < 3 {
		return
	}
	// Grow into the unused right side of the rows the preview could occupy.
	// Keep one column between existing text and the expanded panel.
	textRight := buf.LineNumberWidth
	for i, line := range visibleLines {
		row := contentStart + i
		if row >= y && row < y+maxHeight {
			textRight = max(textRight, buf.LineNumberWidth+wrapTextWidth(strings.TrimRight(line.Text, " \t")))
		}
	}
	panelWidth := min(width-4, max(82, width-textRight-4))
	rows := changePreviewRows(lines, panelWidth-2)
	focusRow := len(changePreviewRows(lines[:focus], panelWidth-2))
	panelHeight := min(len(rows)+2, maxHeight)
	x := width - panelWidth - 3
	style := e.config.Theme.GetStyle(e.config.Theme.Normal)
	border := style
	border.Attrs |= ui.AttrBold
	for row := 0; row < panelHeight; row++ {
		for col := 0; col < panelWidth; col++ {
			ch := ' '
			if row == 0 || row == panelHeight-1 {
				ch = '─'
			} else if col == 0 || col == panelWidth-1 {
				ch = '│'
			}
			cellStyle := style
			if ch != ' ' {
				cellStyle = border
			}
			screen.SetContent(x+col, y+row, ch, nil, cellStyle)
		}
	}
	screen.SetContent(x, y, '╭', nil, border)
	screen.SetContent(x+panelWidth-1, y, '╮', nil, border)
	screen.SetContent(x, y+panelHeight-1, '╰', nil, border)
	screen.SetContent(x+panelWidth-1, y+panelHeight-1, '╯', nil, border)
	title := " " + block.title()
	if len(rows) > panelHeight-2 {
		title += " (more)"
	}
	title += " "
	for i, ch := range []rune(title) {
		if i >= panelWidth-2 {
			break
		}
		screen.SetContent(x+1+i, y, ch, nil, border)
	}
	start := max(0, min(focusRow-(panelHeight-2)/2, len(rows)-(panelHeight-2)))
	var tokens []syntax.Token
	if buf.Highlighter != nil {
		text := strings.ReplaceAll(strings.Join(lines, "\n"), "\t", "    ")
		tokens = buf.Highlighter.GetTokensForString(text)
	}
	tokenIndex := 0
	for row, wrapped := range rows[start:min(len(rows), start+panelHeight-2)] {
		col := 0
		for offset, ch := range wrapped.Text {
			w := wrapRuneWidth(ch)
			if col+w > panelWidth-2 {
				break
			}
			position := wrapped.StartChar + offset
			for tokenIndex < len(tokens) && tokens[tokenIndex].End <= position {
				tokenIndex++
			}
			cellStyle := style
			if tokenIndex < len(tokens) && tokens[tokenIndex].Start <= position {
				cellStyle = e.config.Theme.GetStyle(syntax.GetStyleForToken(tokens[tokenIndex].TokenType, e.config.Theme)).Background(style.Bg)
			}
			screen.SetContent(x+1+col, y+1+row, ch, nil, cellStyle)
			col += w
		}
	}
}
