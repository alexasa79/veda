package ui

// Style represents text styling attributes
type Style struct {
	Fg    Color
	Bg    Color
	Attrs AttrMask
}

// Color represents a color value
type Color int32

const (
	// ColorDefault is the default color
	ColorDefault Color = -1
	
	// Basic colors (0-15)
	ColorBlack Color = iota
	ColorRed
	ColorGreen
	ColorYellow
	ColorBlue
	ColorMagenta
	ColorCyan
	ColorWhite
	ColorBrightBlack
	ColorBrightRed
	ColorBrightGreen
	ColorBrightYellow
	ColorBrightBlue
	ColorBrightMagenta
	ColorBrightCyan
	ColorBrightWhite
)

// NewRGBColor creates a color from RGB values
func NewRGBColor(r, g, b int32) Color {
	return Color(1<<24 | (r&0xff)<<16 | (g&0xff)<<8 | (b&0xff))
}

// RGB returns the RGB values of a color
func (c Color) RGB() (r, g, b int32) {
	if c < 0 {
		return 0, 0, 0
	}
	if c&(1<<24) != 0 {
		// RGB color
		return (int32(c) >> 16) & 0xff, (int32(c) >> 8) & 0xff, int32(c) & 0xff
	}
	// Palette color - return approximate RGB values
	switch c {
	case ColorBlack:
		return 0, 0, 0
	case ColorRed:
		return 205, 0, 0
	case ColorGreen:
		return 0, 205, 0
	case ColorYellow:
		return 205, 205, 0
	case ColorBlue:
		return 0, 0, 238
	case ColorMagenta:
		return 205, 0, 205
	case ColorCyan:
		return 0, 205, 205
	case ColorWhite:
		return 229, 229, 229
	default:
		return 0, 0, 0
	}
}

// AttrMask represents text attributes
type AttrMask int

const (
	AttrBold AttrMask = 1 << iota
	AttrBlink
	AttrReverse
	AttrUnderline
	AttrDim
	AttrItalic
	AttrStrikeThrough
	AttrInvalid AttrMask = 1 << 31
	AttrNone    AttrMask = 0
)

// CursorStyle represents cursor appearance
type CursorStyle int

const (
	CursorStyleDefault CursorStyle = iota
	CursorStyleBlinkingBlock
	CursorStyleSteadyBlock
	CursorStyleBlinkingUnderline
	CursorStyleSteadyUnderline
	CursorStyleBlinkingBar
	CursorStyleSteadyBar
)

// Event represents any screen event
type Event interface {
	When() int64
}

// EventKey represents a keyboard event
type EventKey struct {
	timestamp int64
	Key       Key
	Rune      rune
	Modifiers ModMask
}

func (e *EventKey) When() int64 { return e.timestamp }

// EventResize represents a screen resize event
type EventResize struct {
	timestamp int64
	Width     int
	Height    int
}

func (e *EventResize) When() int64 { return e.timestamp }

// Key represents keyboard keys
type Key int16

const (
	KeyRune Key = iota
	KeyUp
	KeyDown
	KeyRight
	KeyLeft
	KeyUpLeft
	KeyUpRight
	KeyDownLeft
	KeyDownRight
	KeyCenter
	KeyPgUp
	KeyPgDn
	KeyHome
	KeyEnd
	KeyInsert
	KeyDelete
	KeyHelp
	KeyExit
	KeyClear
	KeyCancel
	KeyPrint
	KeyPause
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
	KeyF25
	KeyF26
	KeyF27
	KeyF28
	KeyF29
	KeyF30
	KeyF31
	KeyF32
	KeyF33
	KeyF34
	KeyF35
	KeyF36
	KeyF37
	KeyF38
	KeyF39
	KeyF40
	KeyF41
	KeyF42
	KeyF43
	KeyF44
	KeyF45
	KeyF46
	KeyF47
	KeyF48
	KeyF49
	KeyF50
	KeyF51
	KeyF52
	KeyF53
	KeyF54
	KeyF55
	KeyF56
	KeyF57
	KeyF58
	KeyF59
	KeyF60
	KeyF61
	KeyF62
	KeyF63
	KeyF64
	KeyCtrlSpace
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
	KeyCtrlLeftSq
	KeyCtrlBackslash
	KeyCtrlRightSq
	KeyCtrlCarat
	KeyCtrlUnderscore
	KeyBackspace
	KeyBackspace2
	KeyTab
	KeyEsc
	KeyEscape
	KeyEnter
	KeySpace
	KeyBacktab2
)

// Special key aliases
const (
	KeyBacktab = KeyBacktab2
)

// ModMask represents keyboard modifiers
type ModMask int16

const (
	ModShift ModMask = 1 << iota
	ModCtrl
	ModAlt
	ModMeta
	ModNone ModMask = 0
)

// NewEventKey creates a new key event
func NewEventKey(k Key, ch rune, mod ModMask) *EventKey {
	return &EventKey{
		timestamp: 0,
		Key:       k,
		Rune:      ch,
		Modifiers: mod,
	}
}

// NewEventResize creates a new resize event
func NewEventResize(width, height int) *EventResize {
	return &EventResize{
		timestamp: 0,
		Width:     width,
		Height:    height,
	}
}

// StyleDefault is the default style
var StyleDefault = Style{
	Fg: ColorDefault,
	Bg: ColorDefault,
	Attrs: AttrNone,
}

// Decompose returns the components of a style
func (s Style) Decompose() (fg Color, bg Color, attr AttrMask) {
	return s.Fg, s.Bg, s.Attrs
}

// Foreground returns a new style with the foreground color set
func (s Style) Foreground(c Color) Style {
	s.Fg = c
	return s
}

// Background returns a new style with the background color set
func (s Style) Background(c Color) Style {
	s.Bg = c
	return s
}

// Attributes returns a new style with the attributes set
func (s Style) Attributes(a AttrMask) Style {
	s.Attrs = a
	return s
}