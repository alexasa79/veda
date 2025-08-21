package widgets

import (
	"strings"
	"testing"
	"veda/internal/events"
	"veda/internal/ui"
)

// MockScreenWriter implements ScreenWriter for testing
type MockScreenWriter struct {
	content map[struct{ x, y int }]rune
}

func NewMockScreenWriter() *MockScreenWriter {
	return &MockScreenWriter{
		content: make(map[struct{ x, y int }]rune),
	}
}

func (m *MockScreenWriter) SetContent(x, y int, mainc rune, combc []rune, style ui.Style) {
	m.content[struct{ x, y int }{x, y}] = mainc
}

func TestCommandLineHistory(t *testing.T) {
	cl := NewCommandLineWidget(0, 0, 80)

	// Add some commands to history
	cl.AddToHistory(":quit")
	cl.AddToHistory(":open file.txt")
	cl.AddToHistory(":tabstop=4")

	// Set initial text
	cl.SetText(":")
	cl.SetCursor(1)

	// Test navigating up through history
	upKey := &events.KeyEvent{Key: events.KeyUp}
	cl.HandleKey(upKey)
	if cl.GetText() != ":tabstop=4" {
		t.Errorf("Expected ':tabstop=4', got '%s'", cl.GetText())
	}

	cl.HandleKey(upKey)
	if cl.GetText() != ":open file.txt" {
		t.Errorf("Expected ':open file.txt', got '%s'", cl.GetText())
	}

	cl.HandleKey(upKey)
	if cl.GetText() != ":quit" {
		t.Errorf("Expected ':quit', got '%s'", cl.GetText())
	}

	// Test navigating down
	downKey := &events.KeyEvent{Key: events.KeyDown}
	cl.HandleKey(downKey)
	if cl.GetText() != ":open file.txt" {
		t.Errorf("Expected ':open file.txt', got '%s'", cl.GetText())
	}
}

func TestCommandLineCompletion(t *testing.T) {
	cl := NewCommandLineWidget(0, 0, 80)

	// Set up a simple completion function
	cl.SetCompletionFunc(func(prefix string) []string {
		commands := []string{"quit", "open", "save", "open-file"}
		var completions []string
		for _, cmd := range commands {
			if strings.HasPrefix(cmd, prefix) {
				completions = append(completions, cmd)
			}
		}
		return completions
	})

	// Test cycling through multiple completions
	cl.SetText(":o")
	cl.SetCursor(2)

	tabKey := &events.KeyEvent{Key: events.KeyTab}

	// First tab: should complete to first match (open)
	cl.HandleKey(tabKey)
	if cl.GetText() != ":open" {
		t.Errorf("First tab: Expected ':open', got '%s'", cl.GetText())
	}

	// Without typing anything else, second tab should cycle to next match (open-file)
	cl.HandleKey(tabKey)
	if cl.GetText() != ":open-file" {
		t.Errorf("Second tab: Expected ':open-file', got '%s'", cl.GetText())
	}

	// Third tab: should cycle back to first match (open)
	cl.HandleKey(tabKey)
	if cl.GetText() != ":open" {
		t.Errorf("Third tab: Expected ':open', got '%s'", cl.GetText())
	}

	// Test unique completion (need fresh widget to reset state)
	cl2 := NewCommandLineWidget(0, 0, 80)
	cl2.SetCompletionFunc(func(prefix string) []string {
		commands := []string{"quit", "open", "save", "open-file"}
		var completions []string
		for _, cmd := range commands {
			if strings.HasPrefix(cmd, prefix) {
				completions = append(completions, cmd)
			}
		}
		return completions
	})

	cl2.SetText(":q")
	cl2.SetCursor(2)
	cl2.HandleKey(tabKey)

	if cl2.GetText() != ":quit" {
		t.Errorf("Expected ':quit', got '%s'", cl2.GetText())
	}
}

func TestCommandLineCompletionReset(t *testing.T) {
	cl := NewCommandLineWidget(0, 0, 80)

	// Set up a completion function for show commands
	cl.SetCompletionFunc(func(prefix string) []string {
		commands := []string{"show-buffers", "show-lsp-buffer-go", "show-signature"}
		var completions []string
		for _, cmd := range commands {
			if strings.HasPrefix(cmd, prefix) {
				completions = append(completions, cmd)
			}
		}
		return completions
	})

	// Test that typing more characters resets cycling
	cl.SetText(":show")
	cl.SetCursor(5)

	tabKey := &events.KeyEvent{Key: events.KeyTab}

	// First tab: should complete to first match
	cl.HandleKey(tabKey)
	if cl.GetText() != ":show-buffers" {
		t.Errorf("First tab: Expected ':show-buffers', got '%s'", cl.GetText())
	}

	// Type more characters
	cl.SetText(":show-sig")
	cl.SetCursor(9)

	// Tab should now complete to show-signature
	cl.HandleKey(tabKey)
	if cl.GetText() != ":show-signature" {
		t.Errorf("After typing: Expected ':show-signature', got '%s'", cl.GetText())
	}
}

func TestHistoryNoDuplicates(t *testing.T) {
	cl := NewCommandLineWidget(0, 0, 80)

	// Add the same command multiple times
	cl.AddToHistory(":quit")
	cl.AddToHistory(":quit")
	cl.AddToHistory(":quit")

	// Add a different command
	cl.AddToHistory(":open")

	// Navigate history
	cl.SetText(":")
	upKey := &events.KeyEvent{Key: events.KeyUp}

	cl.HandleKey(upKey)
	if cl.GetText() != ":open" {
		t.Errorf("Expected ':open', got '%s'", cl.GetText())
	}

	cl.HandleKey(upKey)
	if cl.GetText() != ":quit" {
		t.Errorf("Expected ':quit', got '%s'", cl.GetText())
	}

	// Should stay at quit (no duplicates)
	cl.HandleKey(upKey)
	if cl.GetText() != ":quit" {
		t.Errorf("Expected to stay at ':quit', got '%s'", cl.GetText())
	}
}
