package terminal

import (
	"io"
	"os"
	"strings"
	"testing"

	"veda/internal/ui"
)

func TestShowClearsBeforeDrawingLastColumn(t *testing.T) {
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	defer writer.Close()
	term := &Terminal{
		width: 5, height: 1, tty: writer,
		frontBuffer: make([]string, 1), backBuffer: make([]string, 1),
	}
	screen := NewScreenAdapter(term)
	for x, ch := range "09:59" {
		screen.SetContent(x, 0, ch, nil, ui.StyleDefault)
	}
	screen.Show()
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	output, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	clock := strings.Index(text, "09:59")
	if clock < 0 {
		t.Fatalf("clock missing from terminal output: %q", text)
	}
	// After filling a row, the terminal cursor stays in its last column.
	// Clearing from there would erase the final digit we just wrote.
	if strings.Contains(text[clock:], ClearToEOL) {
		t.Fatalf("clear-to-end-of-line after the clock erases its last digit: %q", text)
	}
}
