package editor

import (
    "testing"
    "veda/internal/ui"
)

// This test exercises the rendering path for terminal buffers by simulating
// emulator writes with SGR and verifying styles exposed to the renderer.
func TestTerminalEmulatorSGRColors(t *testing.T) {
    te := NewTerminalEmulator(80, 3)
    // Write: bold white username, yellow @host, blue path, magenta branch
    seq := []byte("\x1b[1;37muser\x1b[0m \x1b[33m@host\x1b[0m \x1b[34m~/project\x1b[0m (\x1b[35mmain\x1b[0m)\n")
    te.Write(seq)

    // Check positions for expected colors/attributes
    chk := func(x, y int, wantFg ui.Color, wantAttr ui.AttrMask) {
        fg, _, attrs := te.StyleAt(x, y)
        if fg != wantFg {
            t.Fatalf("at (%d,%d) fg=%v want=%v", x, y, fg, wantFg)
        }
        if attrs&wantAttr != wantAttr {
            t.Fatalf("at (%d,%d) attrs=%v want has %v", x, y, attrs, wantAttr)
        }
    }
    // 'u' in user at x=0
    chk(0, 0, ui.ColorWhite, ui.AttrBold)
    // space after user should be default
    fg, bg, attrs := te.StyleAt(4, 0)
    if fg != ui.ColorDefault || attrs != ui.AttrNone || bg != ui.ColorDefault {
        t.Fatalf("space after user not default: fg=%v bg=%v attrs=%v", fg, bg, attrs)
    }
    // '@' in @host should be yellow; find first '@'
    atX := -1
    for x := 0; x < 80; x++ {
        // locate by rune scan in Render text to be precise
        if r := te.cells[te.index(x, 0)]; r == '@' { atX = x; break }
    }
    if atX == -1 { t.Fatalf("did not find '@' on line 0") }
    chk(atX, 0, ui.ColorYellow, ui.AttrNone)
    // '~' in path should be blue; find first '~'
    tildeX := -1
    for x := 0; x < 80; x++ {
        if r := te.cells[te.index(x, 0)]; r == '~' { tildeX = x; break }
    }
    if tildeX == -1 { t.Fatalf("did not find '~' on line 0") }
    chk(tildeX, 0, ui.ColorBlue, ui.AttrNone)
    // 'm' in main should be magenta somewhere
    foundMagenta := false
    for x := 0; x < 80; x++ {
        fg, _, _ := te.StyleAt(x, 0)
        if fg == ui.ColorMagenta {
            foundMagenta = true
            break
        }
    }
    if !foundMagenta {
        t.Fatalf("expected to find magenta in line for branch name")
    }
}
