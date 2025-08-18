package editor

import (
    "bytes"
    "unicode/utf8"
    "veda/internal/ui"
)

// TerminalEmulator is a minimal VT100/OSC renderer for a fixed-size grid.
// It supports:
// - Printable UTF-8, CR, LF, BS, TAB
// - CSI cursor movement (A/B/C/D), CUP/HVP (H/f), CHA (G)
// - CSI erase in line (K) and erase in display (J) [0/1/2]
// - SGR (m) ignored for now (no colors)
// - DECSET/DECRST (ESC [ ? ... h/l) ignored
// - OSC (ESC ] ... BEL or ST) ignored
// - Charset selects ESC ( B, etc. ignored
// This is intentionally minimal to make common shells legible.
type TerminalEmulator struct {
    width  int
    height int
    curX   int // 0-based column
    curY   int // 0-based row
    cells  []rune // row-major, len = width*height

    // Saved cursor
    savedX int
    savedY int

    // Attributes per cell (parallel to cells)
    fg    []ui.Color
    bg    []ui.Color
    attrs []ui.AttrMask

    // Current SGR state
    curFg    ui.Color
    curBg    ui.Color
    curAttrs ui.AttrMask

    // Scrollback of lines that scrolled off the top
    scrollback    []termLine
    maxScrollback int
}

func NewTerminalEmulator(cols, rows int) *TerminalEmulator {
    if cols < 1 { cols = 1 }
    if rows < 1 { rows = 1 }
    te := &TerminalEmulator{width: cols, height: rows}
    te.cells = make([]rune, te.width*te.height)
    for i := range te.cells { te.cells[i] = ' ' }
    te.fg = make([]ui.Color, te.width*te.height)
    te.bg = make([]ui.Color, te.width*te.height)
    te.attrs = make([]ui.AttrMask, te.width*te.height)
    for i := range te.fg { te.fg[i] = ui.ColorDefault; te.bg[i] = ui.ColorDefault; te.attrs[i] = ui.AttrNone }
    te.curFg, te.curBg, te.curAttrs = ui.ColorDefault, ui.ColorDefault, ui.AttrNone
    te.maxScrollback = 2000
    return te
}

// Cursor returns the current cursor column and row (0-based).
func (te *TerminalEmulator) Cursor() (int, int) {
    return te.curX, te.curY
}

func (te *TerminalEmulator) Resize(cols, rows int) {
    if cols < 1 { cols = 1 }
    if rows < 1 { rows = 1 }
    if cols == te.width && rows == te.height {
        return
    }
    newCells := make([]rune, cols*rows)
    for i := range newCells { newCells[i] = ' ' }
    newFg := make([]ui.Color, cols*rows)
    newBg := make([]ui.Color, cols*rows)
    newAttrs := make([]ui.AttrMask, cols*rows)
    for i := range newFg { newFg[i] = ui.ColorDefault; newBg[i] = ui.ColorDefault; newAttrs[i] = ui.AttrNone }
    minW := te.width
    if cols < minW { minW = cols }
    minH := te.height
    if rows < minH { minH = rows }
    for y := 0; y < minH; y++ {
        srcStart := y*te.width
        dstStart := y*cols
        copy(newCells[dstStart:dstStart+minW], te.cells[srcStart:srcStart+minW])
        copy(newFg[dstStart:dstStart+minW], te.fg[srcStart:srcStart+minW])
        copy(newBg[dstStart:dstStart+minW], te.bg[srcStart:srcStart+minW])
        copy(newAttrs[dstStart:dstStart+minW], te.attrs[srcStart:srcStart+minW])
    }
    te.width, te.height, te.cells = cols, rows, newCells
    te.fg, te.bg, te.attrs = newFg, newBg, newAttrs
    if te.curX >= te.width { te.curX = te.width - 1 }
    if te.curY >= te.height { te.curY = te.height - 1 }
}

func (te *TerminalEmulator) index(x, y int) int { return y*te.width + x }

func (te *TerminalEmulator) clampCursor() {
    if te.curX < 0 { te.curX = 0 }
    if te.curY < 0 { te.curY = 0 }
    if te.curX >= te.width { te.curX = te.width - 1 }
    if te.curY >= te.height { te.curY = te.height - 1 }
}

func (te *TerminalEmulator) clearLineMode(mode int) {
    switch mode {
    case 0:
        for x := te.curX; x < te.width; x++ { te.cells[te.index(x, te.curY)] = ' ' }
    case 1:
        for x := 0; x <= te.curX; x++ { te.cells[te.index(x, te.curY)] = ' ' }
    case 2:
        for x := 0; x < te.width; x++ { te.cells[te.index(x, te.curY)] = ' ' }
    }
}

func (te *TerminalEmulator) clearDisplayMode(mode int) {
    switch mode {
    case 0:
        te.clearLineMode(0)
        for y := te.curY + 1; y < te.height; y++ {
            for x := 0; x < te.width; x++ { te.cells[te.index(x, y)] = ' ' }
        }
    case 1:
        for y := 0; y < te.curY; y++ {
            for x := 0; x < te.width; x++ { te.cells[te.index(x, y)] = ' ' }
        }
        te.clearLineMode(1)
    case 2:
        for i := range te.cells { te.cells[i] = ' ' }
    }
}

func (te *TerminalEmulator) newline() {
    if te.curY == te.height-1 {
        // Push the top line into scrollback before shifting
        te.pushTopLineToScrollback()
        copy(te.cells[0:te.width*(te.height-1)], te.cells[te.width:])
        copy(te.fg[0:te.width*(te.height-1)], te.fg[te.width:])
        copy(te.bg[0:te.width*(te.height-1)], te.bg[te.width:])
        copy(te.attrs[0:te.width*(te.height-1)], te.attrs[te.width:])
        for x := 0; x < te.width; x++ {
            idx := te.index(x, te.height-1)
            te.cells[idx] = ' '
            te.fg[idx] = ui.ColorDefault
            te.bg[idx] = ui.ColorDefault
            te.attrs[idx] = ui.AttrNone
        }
    } else {
        te.curY++
    }
}

type termLine struct {
    cells []rune
    fg    []ui.Color
    bg    []ui.Color
    attrs []ui.AttrMask
}

func (te *TerminalEmulator) pushTopLineToScrollback() {
    // Capture row 0
    start := 0
    end := te.width
    line := termLine{
        cells: make([]rune, te.width),
        fg:    make([]ui.Color, te.width),
        bg:    make([]ui.Color, te.width),
        attrs: make([]ui.AttrMask, te.width),
    }
    copy(line.cells, te.cells[start:end])
    copy(line.fg, te.fg[start:end])
    copy(line.bg, te.bg[start:end])
    copy(line.attrs, te.attrs[start:end])
    te.scrollback = append(te.scrollback, line)
    if te.maxScrollback > 0 && len(te.scrollback) > te.maxScrollback {
        // Drop oldest
        drop := len(te.scrollback) - te.maxScrollback
        if drop > 0 {
            te.scrollback = te.scrollback[drop:]
        }
    }
}

// Write consumes terminal bytes and updates the grid.
func (te *TerminalEmulator) Write(data []byte) {
    i := 0
    for i < len(data) {
        b := data[i]
        // Handle ESC sequences first so ESC (0x1b) isn't consumed by control-char branch
        if b == 0x1b { // ESC
            i++
            if i >= len(data) { break }
            b2 := data[i]
            if b2 == '[' { // CSI
                i++
                params := []int{}
                cur := 0
                hasCur := false
                priv := false
                for i < len(data) {
                    c := data[i]
                    i++
                    if c == '?' { priv = true; continue }
                    if c >= '0' && c <= '9' { cur = cur*10 + int(c-'0'); hasCur = true; continue }
                    if c == ';' { if hasCur { params = append(params, cur) } else { params = append(params, 0) }; cur = 0; hasCur = false; continue }
                    if hasCur { params = append(params, cur) }
            switch c {
            case 'm':
                // SGR: set graphics rendition
                if len(params) == 0 { params = []int{0} }
                iParam := 0
                for iParam < len(params) {
                    p := params[iParam]
                    iParam++
                    switch p {
                    case 0:
                        te.curFg = ui.ColorDefault
                        te.curBg = ui.ColorDefault
                        te.curAttrs = ui.AttrNone
                case 1:
                        te.curAttrs |= ui.AttrBold
                    case 2:
                        te.curAttrs |= ui.AttrDim
                    case 3:
                        te.curAttrs |= ui.AttrItalic
                    case 4:
                        te.curAttrs |= ui.AttrUnderline
                    case 5:
                        te.curAttrs |= ui.AttrBlink
                    case 7:
                        te.curAttrs |= ui.AttrReverse
                    case 9:
                        te.curAttrs |= ui.AttrStrikeThrough
                case 22:
                        te.curAttrs &^= ui.AttrBold | ui.AttrDim
                        // Some apps use 22 after setting bright colors; keep colors as-is
                    case 23:
                        te.curAttrs &^= ui.AttrItalic
                    case 24:
                        te.curAttrs &^= ui.AttrUnderline
                    case 25:
                        te.curAttrs &^= ui.AttrBlink
                    case 27:
                        te.curAttrs &^= ui.AttrReverse
                    case 29:
                        te.curAttrs &^= ui.AttrStrikeThrough
                    // Foreground 30-37, 90-97
                    case 30,31,32,33,34,35,36,37, 90,91,92,93,94,95,96,97:
                        te.curFg = ui.NewSGRColor(int32(p))
                        if p == 94 {
                            LL.Trace("SGR fg=94 -> stored raw SGR color")
                        }
                    case 39:
                        te.curFg = ui.ColorDefault
                    // Background 40-47, 100-107
                    case 40,41,42,43,44,45,46,47, 100,101,102,103,104,105,106,107:
                        te.curBg = ui.NewSGRColor(int32(p))
                    case 49:
                        te.curBg = ui.ColorDefault
                    case 38, 48:
                        // Extended color: 38;5;n or 38;2;r;g;b (same for 48)
                        isFg := (p == 38)
                        if iParam < len(params) {
                            mode := params[iParam]; iParam++
                            if mode == 5 && iParam < len(params) {
                                n := params[iParam]; iParam++
                                c := ui.New256Color(int32(n))
                                if isFg { te.curFg = c } else { te.curBg = c }
                            } else if mode == 2 && iParam+2 < len(params) {
                                r := params[iParam]; g := params[iParam+1]; b := params[iParam+2]
                                iParam += 3
                                c := ui.NewRGBColor(int32(r), int32(g), int32(b))
                                if isFg { te.curFg = c } else { te.curBg = c }
                            }
                        }
                    default:
                        // Ignore unknown SGR codes
                    }
                }
                    case 'H', 'f':
                        row, col := 1, 1
                        if len(params) >= 1 && params[0] > 0 { row = params[0] }
                        if len(params) >= 2 && params[1] > 0 { col = params[1] }
                        if row < 1 { row = 1 }
                        if col < 1 { col = 1 }
                        if row > te.height { row = te.height }
                        if col > te.width { col = te.width }
                        te.curY, te.curX = row-1, col-1
                    case 'A':
                        n := 1; if len(params) > 0 && params[0] > 0 { n = params[0] }
                        te.curY -= n; te.clampCursor()
                    case 'B':
                        n := 1; if len(params) > 0 && params[0] > 0 { n = params[0] }
                        te.curY += n; te.clampCursor()
                    case 'C':
                        n := 1; if len(params) > 0 && params[0] > 0 { n = params[0] }
                        te.curX += n; te.clampCursor()
                    case 'D':
                        n := 1; if len(params) > 0 && params[0] > 0 { n = params[0] }
                        te.curX -= n; te.clampCursor()
                    case 'G':
                        col := 1; if len(params) > 0 && params[0] > 0 { col = params[0] }
                        if col < 1 { col = 1 }
                        if col > te.width { col = te.width }
                        te.curX = col - 1
                    case 'K':
                        mode := 0; if len(params) > 0 { mode = params[0] }
                        te.clearLineMode(mode)
                    case 'J':
                        mode := 0; if len(params) > 0 { mode = params[0] }
                        te.clearDisplayMode(mode)
                    case 's':
                        te.savedX, te.savedY = te.curX, te.curY
                    case 'u':
                        te.curX, te.curY = te.savedX, te.savedY; te.clampCursor()
                    default:
                        _ = priv // ignore private modes like ?2004h/l
                    }
                    break
                }
                continue
            }
            if b2 == ']' { // OSC: ignore until BEL or ST
                i++
                for i < len(data) {
                    if data[i] == 0x07 { i++; break }
                    if data[i] == 0x1b && i+1 < len(data) && data[i+1] == '\\' { i += 2; break }
                    i++
                }
                continue
            }
            // Single-ESC sequences (non-CSI/OSC)
            switch b2 {
            // Charset designation: ESC ( X, ESC ) X, ESC * X, ESC + X, ESC - X, ESC . X
            case '(', ')', '*', '+', '-', '.':
                // Skip selector byte and the following designation byte
                // Current index i points at the selector; advance past both
                i++
                if i < len(data) { i++ }
                continue
            // DEC save/restore cursor
            case '7':
                te.savedX, te.savedY = te.curX, te.curY
                continue
            case '8':
                te.curX, te.curY = te.savedX, te.savedY; te.clampCursor()
                continue
            // Index / Reverse Index / Next Line
            case 'D': // IND: move down one line
                te.newline()
                continue
            case 'M': // RI: move up one line
                te.curY--
                te.clampCursor()
                continue
            case 'E': // NEL: CR+LF
                te.curX = 0
                te.newline()
                continue
            case 'c': // RIS: reset
                for i2 := range te.cells { te.cells[i2] = ' ' }
                te.curX, te.curY = 0, 0
                continue
            default:
                // Skip this single char and continue
                i++
                continue
            }
        }
        if b < 0x20 {
            switch b {
            case '\r':
                te.curX = 0
                case '\n':
                // In bracketed paste mode, shells often output a visible return glyph; ignore printing of '⏎' runes from control-newlines.
                te.newline()
            case '\b':
                if te.curX > 0 { te.curX-- }
            case '\t':
                nextTab := (te.curX/8 + 1) * 8
                if nextTab >= te.width { nextTab = te.width - 1 }
                te.curX = nextTab
            case 0x07:
                // BEL: ignore
            }
            i++
            continue
        }
        // Printable UTF-8
        r, size := rune(b), 1
        if b >= 0x80 {
            r, size = utf8.DecodeRune(data[i:])
            if r == utf8.RuneError && size == 1 { r = '�' }
        }
        // Filter out the RETURN SYMBOL (⏎) which some prompts echo; do not render as a glyph
        if r == '\u23CE' {
            i += size
            continue
        }
        // Determine display width (handle common wide runes like emoji)
        w := runeDisplayWidth(r)
        if w < 1 { w = 1 }
        // Place rune and styles; if double-width, mark next cell as styled space
        idx := te.index(te.curX, te.curY)
        te.cells[idx] = r
        te.fg[idx] = te.curFg
        te.bg[idx] = te.curBg
        te.attrs[idx] = te.curAttrs
        if w == 2 {
            if te.curX+1 < te.width {
                idx2 := te.index(te.curX+1, te.curY)
                te.cells[idx2] = ' '
                te.fg[idx2] = te.curFg
                te.bg[idx2] = te.curBg
                te.attrs[idx2] = te.curAttrs
            }
        }
        te.curX += w
        if te.curX >= te.width {
            te.curX = 0
            te.newline()
        }
        i += size
    }
}

// runeDisplayWidth returns column width of a rune (1 or 2) for common wide ranges.
// This is a simplified wcwidth: treats East Asian Wide and emoji ranges as width 2.
func runeDisplayWidth(r rune) int {
    // Control and combining are treated as width 1 in our grid (we render controls separately)
    if r == 0 {
        return 0
    }
    // Common wide ranges (CJK, Hangul, Fullwidth forms, and emoji blocks)
    if (r >= 0x1100 && r <= 0x115F) || // Hangul Jamo init. consonants
        r == 0x2329 || r == 0x232A ||
        (r >= 0x2E80 && r <= 0xA4CF) || // CJK ... Yi
        (r >= 0xAC00 && r <= 0xD7A3) || // Hangul Syllables
        (r >= 0xF900 && r <= 0xFAFF) || // CJK Compatibility Ideographs
        (r >= 0xFE10 && r <= 0xFE19) || // Vertical forms
        (r >= 0xFE30 && r <= 0xFE6F) || // CJK Compatibility Forms
        (r >= 0xFF00 && r <= 0xFF60) || // Fullwidth Forms
        (r >= 0xFFE0 && r <= 0xFFE6) || // Fullwidth symbols
        (r >= 0x1F300 && r <= 0x1FAFF) { // Emoji and symbols blocks
        return 2
    }
    return 1
}

// Render returns the current grid as text joined with newlines.
// Trailing spaces on each line are trimmed for readability.
func (te *TerminalEmulator) Render() string {
    var buf bytes.Buffer
    // First render scrollback lines
    for si := 0; si < len(te.scrollback); si++ {
        end := te.width
        // Trim trailing spaces with default bg
        for end > 0 && te.scrollback[si].cells[end-1] == ' ' && te.scrollback[si].bg[end-1] == ui.ColorDefault { end-- }
        for x := 0; x < end; x++ { buf.WriteRune(te.scrollback[si].cells[x]) }
        buf.WriteByte('\n')
    }
    for y := 0; y < te.height; y++ {
        end := te.width
        for end > 0 && te.cells[te.index(end-1, y)] == ' ' && te.bg[te.index(end-1, y)] == ui.ColorDefault { end-- }
        for x := 0; x < end; x++ { buf.WriteRune(te.cells[te.index(x, y)]) }
        if y != te.height-1 { buf.WriteByte('\n') }
    }
    return buf.String()
}

// StyleAt returns the style at cell (x,y)
func (te *TerminalEmulator) StyleAt(x, y int) (ui.Color, ui.Color, ui.AttrMask) {
    if x < 0 || y < 0 || x >= te.width || y >= te.height { return ui.ColorDefault, ui.ColorDefault, ui.AttrNone }
    idx := te.index(x, y)
    return te.fg[idx], te.bg[idx], te.attrs[idx]
}

// Dimensions returns the current grid width and height.
func (te *TerminalEmulator) Dimensions() (int, int) { return te.width, te.height }

// TotalRows returns total rows including scrollback and current grid.
func (te *TerminalEmulator) TotalRows() int { return len(te.scrollback) + te.height }

// ScrollbackSize returns number of scrollback lines stored.
func (te *TerminalEmulator) ScrollbackSize() int { return len(te.scrollback) }

// StyleAtGlobal returns style at global row (including scrollback)
func (te *TerminalEmulator) StyleAtGlobal(x, globalY int) (ui.Color, ui.Color, ui.AttrMask) {
    if x < 0 || x >= te.width || globalY < 0 { return ui.ColorDefault, ui.ColorDefault, ui.AttrNone }
    sb := len(te.scrollback)
    if globalY < sb {
        line := te.scrollback[globalY]
        if x >= len(line.fg) { return ui.ColorDefault, ui.ColorDefault, ui.AttrNone }
        return line.fg[x], line.bg[x], line.attrs[x]
    }
    y := globalY - sb
    if y < 0 || y >= te.height { return ui.ColorDefault, ui.ColorDefault, ui.AttrNone }
    idx := te.index(x, y)
    return te.fg[idx], te.bg[idx], te.attrs[idx]
}

// mapSGRBasicColor maps SGR basic/bright codes to UI colors
func mapSGRBasicColor(p int) ui.Color {
    switch p {
    case 30: return ui.ColorBlack
    case 31: return ui.ColorRed
    case 32: return ui.ColorGreen
    case 33: return ui.ColorYellow
    case 34: return ui.ColorBlue
    case 35: return ui.ColorMagenta
    case 36: return ui.ColorCyan
    case 37: return ui.ColorWhite
    case 90: return ui.ColorBrightBlack
    case 91: return ui.ColorBrightRed
    case 92: return ui.ColorBrightGreen
    case 93: return ui.ColorBrightYellow
    case 94: return ui.ColorBrightBlue
    case 95: return ui.ColorBrightMagenta
    case 96: return ui.ColorBrightCyan
    case 97: return ui.ColorBrightWhite
    case 40: return ui.ColorBlack
    case 41: return ui.ColorRed
    case 42: return ui.ColorGreen
    case 43: return ui.ColorYellow
    case 44: return ui.ColorBlue
    case 45: return ui.ColorMagenta
    case 46: return ui.ColorCyan
    case 47: return ui.ColorWhite
    case 100: return ui.ColorBrightBlack
    case 101: return ui.ColorBrightRed
    case 102: return ui.ColorBrightGreen
    case 103: return ui.ColorBrightYellow
    case 104: return ui.ColorBrightBlue
    case 105: return ui.ColorBrightMagenta
    case 106: return ui.ColorBrightCyan
    case 107: return ui.ColorBrightWhite
    default:
        return ui.ColorDefault
    }
}

// mapSGR256Color maps 0-255 xterm color index to a UI color approximation.
func mapSGR256Color(n int) ui.Color {
    if n < 0 { n = 0 }
    if n > 255 { n = 255 }
    // 0-15: basic + bright
    if n < 16 {
        basic := []ui.Color{
            ui.ColorBlack, ui.ColorRed, ui.ColorGreen, ui.ColorYellow,
            ui.ColorBlue, ui.ColorMagenta, ui.ColorCyan, ui.ColorWhite,
            ui.ColorBrightBlack, ui.ColorBrightRed, ui.ColorBrightGreen, ui.ColorBrightYellow,
            ui.ColorBrightBlue, ui.ColorBrightMagenta, ui.ColorBrightCyan, ui.ColorBrightWhite,
        }
        return basic[n]
    }
    // 16-231: 6x6x6 color cube
    if n >= 16 && n <= 231 {
        idx := n - 16
        r := (idx / 36) % 6
        g := (idx / 6) % 6
        b := idx % 6
        // Map 0..5 to 0..255
        toRGB := func(v int) int32 {
            if v == 0 { return 0 }
            return int32(55 + 40*v)
        }
        return ui.NewRGBColor(toRGB(r), toRGB(g), toRGB(b))
    }
    // 232-255: grayscale
    gray := 8 + 10*(n-232)
    return ui.NewRGBColor(int32(gray), int32(gray), int32(gray))
}
