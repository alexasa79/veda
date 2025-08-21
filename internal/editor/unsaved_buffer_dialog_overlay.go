package editor

import (
	"veda/internal/events"
	"veda/internal/widgets"
)

// UnsavedBufferDialogOverlay handles dialog responses for unsaved buffer warnings
type UnsavedBufferDialogOverlay struct {
	dialog   *widgets.YesNoDialog
	fileName string
	action   string
	multiple bool
	editor   *Editor
}

// NewUnsavedBufferDialogOverlay creates a new unsaved buffer dialog overlay
func NewUnsavedBufferDialogOverlay(dialog *widgets.YesNoDialog, fileName, action string, multiple bool, editor *Editor) *UnsavedBufferDialogOverlay {
	return &UnsavedBufferDialogOverlay{
		dialog:   dialog,
		fileName: fileName,
		action:   action,
		multiple: multiple,
		editor:   editor,
	}
}

func (o *UnsavedBufferDialogOverlay) Draw(screen widgets.Screen) {
	o.dialog.Render(screen)
}

func (o *UnsavedBufferDialogOverlay) CheckContentUpdates() {
	// No dynamic content updates for dialog
}

func (o *UnsavedBufferDialogOverlay) HandleEvent(event interface{}) (bool, string) {
	if keyEv, ok := event.(*events.KeyEvent); ok {
		o.dialog.HandleEvent(keyEv)
	} else if _, ok := event.(*events.ResizeEvent); ok {
		// No-op; dialog will be redrawn at same position/size for now
	}

	if o.dialog.IsClosed() {
		result := o.dialog.GetResult()

		// Handle the user's choice
		switch result {
		case widgets.DialogResultYes:
			// User chose to quit without saving
			if o.multiple {
				// Quit all without saving
				o.editor.actuallyQuit(nil)
			} else {
				// Close current buffer without saving
				if o.action == "quit all" {
					o.editor.actuallyQuit(nil)
				} else {
					o.editor.closeBuffer(nil)
				}
			}
		case widgets.DialogResultNo, widgets.DialogResultCancel:
			// User chose not to quit, do nothing (stay in editor)
		}

		return true, string(result)
	}
	return false, ""
}

func (o *UnsavedBufferDialogOverlay) GetCursorPosition() (int, int) {
	// Dialog has no text input; place cursor off-screen
	return -1, -1
}
