package terminal

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestKittyImageLifecycleCommands(t *testing.T) {
	terminal := &Terminal{}
	placement := KittyImagePlacement{
		Key: "readme:4", Path: "/project/diagram.png", X: 3, Y: 2, Columns: 30, Rows: 8,
	}
	terminal.pendingKittyImages = []KittyImagePlacement{placement}

	first := terminal.updateKittyImages()
	encodedPath := base64.StdEncoding.EncodeToString([]byte(placement.Path))
	if !strings.Contains(first, "a=t,f=100,t=f") || !strings.Contains(first, encodedPath) {
		t.Fatalf("first frame did not transmit the PNG path: %q", first)
	}
	if !strings.Contains(first, "\x1b[3;4H") || !strings.Contains(first, "c=30,r=8,C=1") {
		t.Fatalf("first frame did not place the image: %q", first)
	}

	if unchanged := terminal.updateKittyImages(); unchanged != "" {
		t.Fatalf("unchanged frame emitted commands: %q", unchanged)
	}

	placement.Y = 4
	terminal.pendingKittyImages = []KittyImagePlacement{placement}
	moved := terminal.updateKittyImages()
	if strings.Contains(moved, "a=t") || !strings.Contains(moved, "\x1b[5;4H") {
		t.Fatalf("moving placement emitted unexpected commands: %q", moved)
	}

	terminal.pendingKittyImages = nil
	removed := terminal.updateKittyImages()
	if !strings.Contains(removed, "a=d,d=I") {
		t.Fatalf("removed image did not free its data: %q", removed)
	}
}
