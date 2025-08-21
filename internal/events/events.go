package events

import "time"

// Event represents any system event (keyboard, mouse, resize, etc.)
type Event interface {
	When() time.Time // Timestamp when the event occurred
}

// KeyEvent represents a keyboard event
type KeyEvent struct {
	timestamp time.Time
	Key       Key
	Rune      rune
	Modifiers ModMask
}

func (e *KeyEvent) When() time.Time { return e.timestamp }

// NewKeyEvent creates a new key event
func NewKeyEvent(key Key, ch rune, mod ModMask) *KeyEvent {
	return &KeyEvent{
		timestamp: time.Now(),
		Key:       key,
		Rune:      ch,
		Modifiers: mod,
	}
}

// ResizeEvent represents a terminal/screen resize event
type ResizeEvent struct {
	timestamp time.Time
	Width     int
	Height    int
}

func (e *ResizeEvent) When() time.Time { return e.timestamp }

// NewResizeEvent creates a new resize event
func NewResizeEvent(width, height int) *ResizeEvent {
	return &ResizeEvent{
		timestamp: time.Now(),
		Width:     width,
		Height:    height,
	}
}

// MouseEvent represents a mouse event
type MouseEvent struct {
	timestamp time.Time
	X         int
	Y         int
	Button    ButtonMask
	Modifiers ModMask
}

func (e *MouseEvent) When() time.Time { return e.timestamp }

// NewMouseEvent creates a new mouse event
func NewMouseEvent(x, y int, button ButtonMask, mod ModMask) *MouseEvent {
	return &MouseEvent{
		timestamp: time.Now(),
		X:         x,
		Y:         y,
		Button:    button,
		Modifiers: mod,
	}
}

// Key represents a keyboard key
type Key int

const (
	KeyRune Key = iota // Regular character

	// Special keys
	KeyBackspace
	KeyBackspace2 // Alternative backspace
	KeyTab
	KeyBacktab
	KeyEnter
	KeyEscape
	KeyDelete
	KeyInsert
	KeySpace

	// Navigation keys
	KeyUp
	KeyDown
	KeyLeft
	KeyRight
	KeyHome
	KeyEnd
	KeyPgUp
	KeyPgDn

	// Function keys
	KeyF1
	KeyF2
	KeyF3
	KeyF4
	KeyF5
	KeyF6
	KeyF7
	KeyF8
	KeyF9
	KeyF10
	KeyF11
	KeyF12
	KeyF13
	KeyF14
	KeyF15
	KeyF16
	KeyF17
	KeyF18
	KeyF19
	KeyF20
	KeyF21
	KeyF22
	KeyF23
	KeyF24

	// Control keys
	KeyCtrlA
	KeyCtrlB
	KeyCtrlC
	KeyCtrlD
	KeyCtrlE
	KeyCtrlF
	KeyCtrlG
	KeyCtrlH
	KeyCtrlI
	KeyCtrlJ
	KeyCtrlK
	KeyCtrlL
	KeyCtrlM
	KeyCtrlN
	KeyCtrlO
	KeyCtrlP
	KeyCtrlQ
	KeyCtrlR
	KeyCtrlS
	KeyCtrlT
	KeyCtrlU
	KeyCtrlV
	KeyCtrlW
	KeyCtrlX
	KeyCtrlY
	KeyCtrlZ

	KeyCtrlSpace
	KeyCtrlBackslash
	KeyCtrlRightSq
	KeyCtrlCarat
	KeyCtrlUnderscore
)

// ModMask represents keyboard and mouse modifiers
type ModMask int

const (
	ModNone  ModMask = 0
	ModShift ModMask = 1 << iota
	ModCtrl
	ModAlt
	ModMeta
)

// ButtonMask represents mouse buttons
type ButtonMask int

const (
	ButtonNone ButtonMask = 0
	Button1    ButtonMask = 1 << iota
	Button2
	Button3
	Button4
	Button5
	Button6
	Button7
	Button8
	WheelUp    = Button4
	WheelDown  = Button5
	WheelLeft  = Button6
	WheelRight = Button7
)

// String returns a string representation of the key
func (k Key) String() string {
	switch k {
	case KeyRune:
		return "Rune"
	case KeyBackspace, KeyBackspace2:
		return "Backspace"
	case KeyTab:
		return "Tab"
	case KeyBacktab:
		return "Backtab"
	case KeyEnter:
		return "Enter"
	case KeyEscape:
		return "Escape"
	case KeyDelete:
		return "Delete"
	case KeyInsert:
		return "Insert"
	case KeySpace:
		return "Space"
	case KeyUp:
		return "Up"
	case KeyDown:
		return "Down"
	case KeyLeft:
		return "Left"
	case KeyRight:
		return "Right"
	case KeyHome:
		return "Home"
	case KeyEnd:
		return "End"
	case KeyPgUp:
		return "PageUp"
	case KeyPgDn:
		return "PageDown"
	case KeyF1:
		return "F1"
	case KeyF2:
		return "F2"
	case KeyF3:
		return "F3"
	case KeyF4:
		return "F4"
	case KeyF5:
		return "F5"
	case KeyF6:
		return "F6"
	case KeyF7:
		return "F7"
	case KeyF8:
		return "F8"
	case KeyF9:
		return "F9"
	case KeyF10:
		return "F10"
	case KeyF11:
		return "F11"
	case KeyF12:
		return "F12"
	case KeyCtrlA:
		return "Ctrl+A"
	case KeyCtrlB:
		return "Ctrl+B"
	case KeyCtrlC:
		return "Ctrl+C"
	case KeyCtrlD:
		return "Ctrl+D"
	case KeyCtrlE:
		return "Ctrl+E"
	case KeyCtrlF:
		return "Ctrl+F"
	case KeyCtrlG:
		return "Ctrl+G"
	case KeyCtrlH:
		return "Ctrl+H"
	case KeyCtrlI:
		return "Ctrl+I"
	case KeyCtrlJ:
		return "Ctrl+J"
	case KeyCtrlK:
		return "Ctrl+K"
	case KeyCtrlL:
		return "Ctrl+L"
	case KeyCtrlM:
		return "Ctrl+M"
	case KeyCtrlN:
		return "Ctrl+N"
	case KeyCtrlO:
		return "Ctrl+O"
	case KeyCtrlP:
		return "Ctrl+P"
	case KeyCtrlQ:
		return "Ctrl+Q"
	case KeyCtrlR:
		return "Ctrl+R"
	case KeyCtrlS:
		return "Ctrl+S"
	case KeyCtrlT:
		return "Ctrl+T"
	case KeyCtrlU:
		return "Ctrl+U"
	case KeyCtrlV:
		return "Ctrl+V"
	case KeyCtrlW:
		return "Ctrl+W"
	case KeyCtrlX:
		return "Ctrl+X"
	case KeyCtrlY:
		return "Ctrl+Y"
	case KeyCtrlZ:
		return "Ctrl+Z"
	case KeyCtrlSpace:
		return "Ctrl+Space"
	default:
		return "Unknown"
	}
}

// CustomEvent represents a custom application event
type CustomEvent struct {
	timestamp time.Time
	Type      string
	Data      interface{}
}

func (e *CustomEvent) When() time.Time { return e.timestamp }

// NewCustomEvent creates a new custom event
func NewCustomEvent(eventType string, data interface{}) *CustomEvent {
	return &CustomEvent{
		timestamp: time.Now(),
		Type:      eventType,
		Data:      data,
	}
}

// PasteEvent represents a bracketed paste marker event
// Start is true for paste start (ESC[200~), false for paste end (ESC[201~)
type PasteEvent struct {
	timestamp time.Time
	Start     bool
}

func (e *PasteEvent) When() time.Time { return e.timestamp }

// NewPasteEvent creates a new paste event
func NewPasteEvent(start bool) *PasteEvent {
	return &PasteEvent{
		timestamp: time.Now(),
		Start:     start,
	}
}
