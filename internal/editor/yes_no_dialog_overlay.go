package editor

import (
	"veda/internal/events"
	"veda/internal/widgets"
)

// YesNoDialogOverlay adapts a widgets.YesNoDialog to the overlay system
type YesNoDialogOverlay struct {
	dialog *widgets.YesNoDialog
}

func NewYesNoDialogOverlay(dialog *widgets.YesNoDialog) *YesNoDialogOverlay {
	return &YesNoDialogOverlay{dialog: dialog}
}

func (o *YesNoDialogOverlay) Draw(screen widgets.Screen) {
	o.dialog.Render(screen)
}

func (o *YesNoDialogOverlay) CheckContentUpdates() {
	// No dynamic content updates for simple dialog
}

func (o *YesNoDialogOverlay) HandleEvent(event interface{}) (bool, string) {
	if keyEv, ok := event.(*events.KeyEvent); ok {
		o.dialog.HandleEvent(keyEv)
	} else if _, ok := event.(*events.ResizeEvent); ok {
		// No-op; dialog will be redrawn at same position/size for now
	}
	if o.dialog.IsClosed() {
		return true, string(o.dialog.GetResult())
	}
	return false, ""
}

func (o *YesNoDialogOverlay) GetCursorPosition() (int, int) {
	// Dialog has no text input; place cursor off-screen
	return -1, -1
}
