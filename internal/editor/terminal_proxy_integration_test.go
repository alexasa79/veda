package editor

import (
	"bytes"
	"testing"
	"time"
	"veda/internal/proxy"
	"veda/internal/ui"
)

// Starts a real PTY via proxy, prints colored text from bash, feeds bytes
// into TerminalEmulator and asserts per-cell style.
func TestTerminalProxyColorSGR(t *testing.T) {
	// Start proxy
	pm, err := proxy.NewProxyManager()
	if err != nil {
		t.Skipf("proxy not available: %v", err)
		return
	}
	defer pm.Close()

	cols, rows := 80, 24
	// Start bash without rc/profile for deterministic output
	sid, _, err := pm.StartTerminal("", []string{"bash", "--noprofile", "--norc"}, map[string]string{"TERM": "xterm-256color"}, cols, rows)
	if err != nil || sid == "" {
		t.Skipf("terminal not available: %v", err)
		return
	}
	defer pm.CloseTerminal("", sid)

	// Nudge to get prompt
	_ = pm.WriteTerminal("", sid, []byte{'\r'})

	// Clear screen, home, move to row 10 col 10, set red, print RED, reset
	cmd := "printf $'\\e[2J\\e[H\\e[10;10H\\e[31mRED\\e[0m\\n'\r"
	if err := pm.WriteTerminal("", sid, []byte(cmd)); err != nil {
		t.Fatalf("write terminal failed: %v", err)
	}

	te := NewTerminalEmulator(cols, rows)

	// Read loop until we observe RED or timeout
	deadline := time.Now().Add(3 * time.Second)
	var seen bool
	for time.Now().Before(deadline) {
		data, _, _, err := pm.ReadTerminal("", sid, 65536)
		if err != nil {
			// small delay and continue
			time.Sleep(20 * time.Millisecond)
			continue
		}
		if len(data) > 0 {
			te.Write(data)
			if bytes.Contains(data, []byte("RED")) {
				seen = true
				break
			}
		} else {
			time.Sleep(20 * time.Millisecond)
		}
	}
	if !seen {
		t.Fatalf("did not observe RED in terminal output within timeout")
	}

	// Expect 'R' at (col=9,row=9) zero-based
	x, y := 9, 9
	fg, _, attrs := te.StyleAt(x, y)
	// SGR code 31 (red) should be stored as NewSGRColor(31)
	expectedRed := ui.NewSGRColor(31)
	if fg != expectedRed {
		t.Fatalf("expected SGR red (31) at (%d,%d), got fg=%v (expected %v)", x, y, fg, expectedRed)
	}
	if attrs != ui.AttrNone && (attrs&ui.AttrBold) == 0 { // allow bold or none
		t.Fatalf("unexpected attrs at (%d,%d): %v", x, y, attrs)
	}
}
