package editor

import (
	"strings"
	"veda/internal/events"
	"veda/internal/ui"
	"veda/internal/widgets"
)

type prPrompt struct{ input *widgets.LineEditor }

func (p *prPrompt) Draw(screen widgets.Screen) {
	w, h := screen.Size()
	width := min(64, w)
	x, y := (w-width)/2, max(0, (h-4)/2)
	for row := y; row < min(y+4, h); row++ {
		for col := x; col < x+width; col++ {
			screen.SetContent(col, row, ' ', nil, ui.StyleDefault)
		}
	}
	for row, text := range []string{"Review PR — number or URL", "", "Enter: open   Escape: cancel"} {
		for col, r := range []rune(text) {
			if col >= width || y+row >= h {
				break
			}
			screen.SetContent(x+col, y+row, r, nil, ui.StyleDefault)
		}
	}
	p.input.SetPosition(x, min(y+1, h-1))
	p.input.SetWidth(max(1, width))
	p.input.Draw(screen)
}

func (p *prPrompt) CheckContentUpdates()          {}
func (p *prPrompt) GetCursorPosition() (int, int) { return p.input.GetCursorPosition() }
func (p *prPrompt) HandleEvent(event interface{}) (bool, string) {
	if key, ok := event.(*events.KeyEvent); ok {
		switch key.Key {
		case events.KeyEscape:
			return true, ""
		case events.KeyEnter:
			return true, strings.TrimSpace(p.input.GetText())
		default:
			p.input.HandleKey(key)
		}
	}
	return false, ""
}

func (e *Editor) promptPRReview() {
	result := e.renderingState.SetOverlay(&prPrompt{input: widgets.NewLineEditor(0, 0, 64)})
	go func() {
		if text := <-result; text != "" {
			e.Post(func(ed *Editor) {
				if err := ed.reviewPR([]string{text}); err != nil {
					ed.ShowError("%v", err)
				}
			})
		}
	}()
}
