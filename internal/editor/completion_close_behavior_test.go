package editor

import (
	"testing"
	"veda/internal/commands"
	"veda/internal/events"
	"veda/internal/widgets"
)

// helper to prime a visible completion menu with simple items
func primeCompletionMenu(te *TestEditor, items []widgets.CompletionItem) {
	te.completionMenu.SetItems(items)
	te.completionMenu.SetPosition(0, 0)
	te.completionMenu.Show()
}

func TestCompletionMenuClosesOnSpaceAndTab(t *testing.T) {
	te := NewTestEditorWithContent(80, 24, "")
	te.mode = commands.InsertMode

	items := []widgets.CompletionItem{{Label: "hello", InsertText: "hello"}, {Label: "help", InsertText: "help"}}
	primeCompletionMenu(te, items)
	if !te.completionMenu.IsVisible() {
		t.Fatalf("expected completion menu to be visible")
	}

	// Press space -> should close (send via TypeKeys to include rune)
	te.TypeKeys(" ")
	if te.completionMenu.IsVisible() {
		t.Fatalf("expected completion menu to close on space")
	}

	// Reopen and press Tab -> should close and allow indentation handling
	primeCompletionMenu(te, items)
	te.PressKey(events.KeyTab, 0)
	if te.completionMenu.IsVisible() {
		t.Fatalf("expected completion menu to close on tab")
	}
}

func TestCompletionMenuClosesOnExactMatchTyped(t *testing.T) {
	te := NewTestEditorWithContent(80, 24, "")
	te.mode = commands.InsertMode

	items := []widgets.CompletionItem{{Label: "hello", InsertText: "hello"}, {Label: "world", InsertText: "world"}}
	primeCompletionMenu(te, items)

	te.TypeKeys("hello")
	if te.completionMenu.IsVisible() {
		t.Fatalf("expected completion menu to close after exact match typed")
	}
}

func TestCompletionMenuClosesOnLeftRight(t *testing.T) {
	te := NewTestEditorWithContent(80, 24, "")
	te.mode = commands.InsertMode

	items := []widgets.CompletionItem{{Label: "foo", InsertText: "foo"}}
	primeCompletionMenu(te, items)

	te.PressKey(events.KeyLeft, 0)
	if te.completionMenu.IsVisible() {
		t.Fatalf("expected completion menu to close on left arrow")
	}

	primeCompletionMenu(te, items)
	te.PressKey(events.KeyRight, 0)
	if te.completionMenu.IsVisible() {
		t.Fatalf("expected completion menu to close on right arrow")
	}
}

func TestCompletionMenuClosesOnPunctuation(t *testing.T) {
    te := NewTestEditorWithContent(80, 24, "")
    te.mode = commands.InsertMode

    items := []widgets.CompletionItem{{Label: "foo", InsertText: "foo"}}
    primeCompletionMenu(te, items)

    // Type a dot '.' which is non-word punctuation
    te.TypeKeys(".")
    if te.completionMenu.IsVisible() {
        t.Fatalf("expected completion menu to close on punctuation")
    }
}

func TestCompletionMenuHidesWhenNoPrefixMatch(t *testing.T) {
    te := NewTestEditorWithContent(80, 24, "")
    te.mode = commands.InsertMode

    items := []widgets.CompletionItem{{Label: "hello", InsertText: "hello"}, {Label: "help", InsertText: "help"}}
    primeCompletionMenu(te, items)

    // Type a prefix that matches none of the items
    te.TypeKeys("xz")
    if te.completionMenu.IsVisible() {
        t.Fatalf("expected completion menu to hide when no prefix matches")
    }
}
