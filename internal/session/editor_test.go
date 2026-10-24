package session

import (
	"os"
	"strings"
	"testing"
	"time"
	"veda/internal/buffer"
	"veda/internal/editor"
	"veda/internal/events"
	"veda/internal/proxy"
)

func TestSupervisorReloadsEditorWithUnsavedTextAndUndo(t *testing.T) {
	screen := &observedScreen{MockScreen: editor.NewMockScreen(80, 24), input: make(chan events.Event, 128), frames: make(chan string, 128)}
	manager, err := proxy.NewProxyManager()
	if err != nil {
		t.Fatal(err)
	}
	executable, _ := os.Executable()
	done := make(chan error, 1)
	go func() {
		_, err := Run(screen, manager, executable, []string{"-test.run=^TestEditorWorkerProcess$"})
		done <- err
	}()
	waitFor := func(text string) {
		t.Helper()
		timer := time.NewTimer(15 * time.Second)
		defer timer.Stop()
		last := ""
		for {
			select {
			case line := <-screen.frames:
				last = line
				if strings.Contains(line, text) {
					return
				}
			case err := <-done:
				t.Fatalf("editor exited before %q: %v", text, err)
			case <-timer.C:
				t.Fatalf("editor did not render %q; last line: %q", text, last)
			}
		}
	}
	// Input may be queued before the first frame, just as during a real reload.
	typeKeys := func(text string) {
		for _, ch := range text {
			if ch == '\n' {
				screen.input <- events.NewKeyEvent(events.KeyEnter, 0, events.ModNone)
			} else {
				screen.input <- events.NewKeyEvent(events.KeyRune, ch, events.ModNone)
			}
		}
	}
	typeKeys(":new-buffer\niunsaved")
	screen.input <- events.NewKeyEvent(events.KeyEscape, 0, events.ModNone)
	waitFor("unsaved")
	typeKeys(":reload\n")
	typeKeys("0iX")
	screen.input <- events.NewKeyEvent(events.KeyEscape, 0, events.ModNone)
	waitFor("Xunsaved restored")
	typeKeys(":qa\n")
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("editor failed to quit after reload")
	}
}

func TestEditorWorkerProcess(t *testing.T) {
	if os.Getenv("VEDA_WORKER") != "1" {
		return
	}
	w, err := OpenWorker()
	if err != nil {
		t.Fatal(err)
	}
	e, err := editor.NewWithScreen(w)
	if err != nil {
		t.Fatal(err)
	}
	if len(w.State) != 0 {
		if err := e.RestoreSession(w.State); err != nil {
			w.ReportError(err)
			os.Exit(1)
		}
		b := e.CurrentBuffer()
		if b == nil || b.Rope.String() != "unsaved" || !b.IsModified {
			t.Fatal("unsaved text did not survive reload")
		}
		// Entering command mode records a cursor-only undo unit too.
		undos := 0
		for b.Rope.String() == "unsaved" && undos < 5 {
			if err := b.History.Undo(b); err != nil {
				t.Fatal(err)
			}
			undos++
		}
		if b.Rope.String() != "" {
			t.Fatalf("undo after reload: %q", b.Rope.String())
		}
		for i := 0; i < undos; i++ {
			if err := b.History.Redo(b); err != nil {
				t.Fatal(err)
			}
		}
		if b.Rope.String() != "unsaved" {
			t.Fatalf("redo after reload: %q", b.Rope.String())
		}
		b.Insert(b.Length(), " restored")
		b.CursorIterator = buffer.NewBufferIterator(b)
		b.ViewIterator = buffer.NewBufferIterator(b)
	}
	e.EnableReload(w.Pause)
	if err := w.Ready(); err != nil {
		t.Fatal(err)
	}
	for {
		e.Run()
		if !e.ReloadRequested() {
			break
		}
		data, err := e.PrepareReload()
		if err != nil {
			w.CancelReload()
			e.ResumeAfterReload(err.Error())
			continue
		}
		committed, reason := w.Reload(data)
		if committed {
			e.Close()
			w.Close()
			os.Exit(0)
		}
		e.ResumeAfterReload(reason)
	}
	e.Close()
	w.Close()
	os.Exit(0)
}
