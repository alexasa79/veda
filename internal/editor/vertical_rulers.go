package editor

import "veda/internal/ui"

// Draw rulers only in blank cells, preserving text and the cell's background.
func (e *Editor) setTextCell(x, y int, ch rune, combining []rune, style ui.Style) {
	if ch == ' ' && x >= e.CurrentBuffer().LineNumberWidth {
		column := x - e.CurrentBuffer().LineNumberWidth
		for _, ruler := range e.config.VerticalRulers {
			if column == ruler {
				ch = '│'
				style = style.Foreground(e.config.Theme.GetStyle(e.config.Theme.VerticalRuler).Fg)
				break
			}
		}
	}
	e.screen.SetContent(x, y, ch, combining, style)
}
