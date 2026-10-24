package editor

import (
	"strings"

	"veda/internal/buffer"
	"veda/internal/syntax"
	"veda/internal/ui"
)

func (e *Editor) functionHeaderRanges(x, line, start, end int, style ui.Style) []syntax.ByteRange {
	buf := e.CurrentBuffer()
	if buf == nil || buf.Highlighter == nil || line < 0 || x < buf.LineNumberWidth {
		return nil
	}
	// Existing cursor, context, and diff backgrounds take precedence.
	if style.Bg != e.config.Theme.GetStyle(e.config.Theme.Normal).Bg {
		return nil
	}
	if line < len(buf.DiffHints) && buf.DiffHints[line].LineType != buffer.DiffLineContext {
		return nil
	}
	return buf.Highlighter.FunctionHeaderRanges(start, end)
}

// Fill rows containing only header text and whitespace, including inline bodies.
func (e *Editor) functionHeaderLineBackground(x, line, start int, text string, style ui.Style) (ui.Color, bool) {
	end := start + max(1, len(text))
	for _, header := range e.functionHeaderRanges(x, line, start, end, style) {
		first := min(len(text), header.Start-start)
		last := min(len(text), header.End-start)
		if strings.TrimSpace(text[:first]) == "" && strings.TrimSpace(text[last:]) == "" {
			return e.config.Theme.FunctionHeaderBackground(), true
		}
	}
	return ui.ColorDefault, false
}
