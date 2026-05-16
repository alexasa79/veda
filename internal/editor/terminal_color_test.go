package editor

import (
	"strings"
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
	// 'u' in user at x=0 - should be SGR white (37) with bold
	chk(0, 0, ui.NewSGRColor(37), ui.AttrBold)
	// space after user should be default
	fg, bg, attrs := te.StyleAt(4, 0)
	if fg != ui.ColorDefault || attrs != ui.AttrNone || bg != ui.ColorDefault {
		t.Fatalf("space after user not default: fg=%v bg=%v attrs=%v", fg, bg, attrs)
	}
	// '@' in @host should be SGR yellow (33); find first '@'
	atX := -1
	for x := 0; x < 80; x++ {
		// locate by rune scan in Render text to be precise
		if r := te.cells[te.index(x, 0)]; r == '@' {
			atX = x
			break
		}
	}
	if atX == -1 {
		t.Fatalf("did not find '@' on line 0")
	}
	chk(atX, 0, ui.NewSGRColor(33), ui.AttrNone)
	// '~' in path should be SGR blue (34); find first '~'
	tildeX := -1
	for x := 0; x < 80; x++ {
		if r := te.cells[te.index(x, 0)]; r == '~' {
			tildeX = x
			break
		}
	}
	if tildeX == -1 {
		t.Fatalf("did not find '~' on line 0")
	}
	chk(tildeX, 0, ui.NewSGRColor(34), ui.AttrNone)
	// 'm' in main should be SGR magenta (35) somewhere
	expectedMagenta := ui.NewSGRColor(35)
	foundMagenta := false
	for x := 0; x < 80; x++ {
		fg, _, _ := te.StyleAt(x, 0)
		if fg == expectedMagenta {
			foundMagenta = true
			break
		}
	}
	if !foundMagenta {
		t.Fatalf("expected to find SGR magenta (35) in line for branch name")
	}
}

func TestTerminalEmulatorColonSGRDoesNotRenderParameters(t *testing.T) {
	te := NewTerminalEmulator(40, 3)
	te.Write([]byte("\x1b[4:1munder\x1b[4:0m plain"))

	rendered := te.Render()
	if strings.Contains(rendered, "4:1m") || strings.Contains(rendered, "4:0m") {
		t.Fatalf("rendered CSI parameters as text: %q", rendered)
	}
	if !strings.Contains(rendered, "under plain") {
		t.Fatalf("missing expected text: %q", rendered)
	}

	_, _, attrs := te.StyleAt(0, 0)
	if attrs&ui.AttrUnderline == 0 {
		t.Fatalf("expected underline attr on first character, got %v", attrs)
	}
}

func TestTerminalEmulatorDeviceAttributeResponses(t *testing.T) {
	te := NewTerminalEmulator(40, 3)

	response := te.WriteWithResponse([]byte("\x1b[c"))
	if string(response) != "\x1b[?1;2c" {
		t.Fatalf("primary DA response = %q, want %q", response, "\x1b[?1;2c")
	}
	if strings.TrimSpace(te.Render()) != "" {
		t.Fatalf("primary DA should not render text: %q", te.Render())
	}

	response = te.WriteWithResponse([]byte("\x1b[>c"))
	if string(response) != "\x1b[>0;0;0c" {
		t.Fatalf("secondary DA response = %q, want %q", response, "\x1b[>0;0;0c")
	}
}

func TestTerminalEmulatorDCSDoesNotRender(t *testing.T) {
	te := NewTerminalEmulator(80, 3)
	te.Write([]byte("before\x1bP+q696e646e\x1b\\after"))

	rendered := te.Render()
	if strings.Contains(rendered, "+q696e646e") {
		t.Fatalf("rendered DCS payload as text: %q", rendered)
	}
	if !strings.Contains(rendered, "beforeafter") {
		t.Fatalf("missing surrounding text: %q", rendered)
	}
}
