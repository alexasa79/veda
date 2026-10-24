package editor

import (
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
	"veda/internal/buffer"
	"veda/internal/commands"
	"veda/internal/config"
	"veda/internal/proxy"
	"veda/internal/theme"
	"veda/internal/widgets"
)

func newSessionTestEditor(t *testing.T, text string) *Editor {
	t.Helper()
	e, err := NewWithScreen(NewMockScreen(100, 30))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(e.Close)
	b := buffer.NewBuffer(text, "scratch", "", config.IndentSpaces, 4)
	e.buffers[b.FilePath] = b
	e.bufferOrder = []string{b.FilePath}
	e.windows = []*Window{newWindow(b)}
	e.SetCurrentBuffer(b)
	return e
}

func roundTripSession(t *testing.T, source *Editor) *Editor {
	t.Helper()
	data, err := source.SessionState()
	if err != nil {
		t.Fatal(err)
	}
	target := newSessionTestEditor(t, "discard this")
	if err := target.RestoreSession(data); err != nil {
		t.Fatal(err)
	}
	return target
}

func TestSessionPreservesBuffersHistoryAndCaches(t *testing.T) {
	e := newSessionTestEditor(t, "hello\nsecond\nthird")
	b := e.CurrentBuffer()
	entry := buffer.NewInsertTextEntry(0, 5, " world")
	if _, err := b.History.Execute(b, entry); err != nil {
		t.Fatal(err)
	}
	if _, err := b.History.Execute(b, buffer.NewInsertTextEntry(0, 11, "!")); err != nil {
		t.Fatal(err)
	}
	if err := b.History.Undo(b); err != nil {
		t.Fatal(err)
	}
	b.IsModified = true
	b.CursorIterator = buffer.NewBufferIteratorAtLine(b, 2)
	b.ViewIterator = buffer.NewBufferIteratorAtLine(b, 1)
	b.CursorX = 2
	e.virtualCursors = []VirtualCursor{{X: 1, Y: 1}}
	e.registers = map[rune]string{'d': "deleted text"}
	e.dirCache["host:/project"] = []proxy.FileInfo{{Path: "host:/project/main.go"}}
	e.workingDirectory = "host:/project"
	e.searchPattern = "second"
	e.commandLine.AddToHistory(":w")
	e.searchLine.AddToHistory("second")
	e.navHistory = []NavigationEntry{{filePath: "scratch", cursorX: 3, cursorLineNumber: 1}}
	e.lastChange = entry
	e.lastCommand = &RepeatableCommand{Operator: ChangeOperator{}, TextObject: WordTextObject{modifier: ModifierInside, excludeTrailingSpace: true, Count: 2}}
	e.previousBuffer = b
	other := buffer.NewBuffer("hidden unsaved", "other", "", config.IndentTabs, 1)
	e.buffers["other"] = other
	e.bufferOrder = append(e.bufferOrder, "other")
	e.windows[0].Buffers = append(e.windows[0].Buffers, other)
	e.windows[0].Views = append(e.windows[0].Views, &SingleBufferView{Buf: other})
	e.windows[0].setActiveBuffer(b)
	e.windows = append(e.windows, newWindow(b))
	e.windows[1].CursorLine, e.windows[1].ViewLine = 2, 1

	r := roundTripSession(t, e)
	rb := r.CurrentBuffer()
	if rb.Rope.String() != "hello world\nsecond\nthird" || !rb.IsModified {
		t.Fatalf("text/modified lost: %q %v", rb.Rope.String(), rb.IsModified)
	}
	if rb.CursorX != 2 || rb.CursorIterator.LineNumber() != 2 || rb.ViewIterator.LineNumber() != 1 {
		t.Fatal("buffer cursor or viewport lost")
	}
	if r.windows[0].Buffer != r.windows[1].Buffer || r.previousBuffer != rb {
		t.Fatal("shared buffer identity lost")
	}
	if len(r.windows[0].Buffers) != 2 || r.buffers["other"].Rope.String() != "hidden unsaved" {
		t.Fatal("inactive buffer lost")
	}
	if len(r.virtualCursors) != 1 || r.registers['d'] != "deleted text" || len(r.dirCache["host:/project"]) != 1 || r.workingDirectory != "host:/project" {
		t.Fatal("session data lost")
	}
	if r.navHistory[0].cursorLineNumber != 1 || r.searchPattern != "second" {
		t.Fatal("navigation/search lost")
	}
	if history := r.commandLine.History(); len(history) != 1 || history[0] != "w" {
		t.Fatal("command history lost")
	}
	if history := r.searchLine.History(); len(history) != 1 || history[0] != "second" {
		t.Fatal("search history lost")
	}
	word := r.lastCommand.TextObject.(WordTextObject)
	if word.modifier != ModifierInside || !word.excludeTrailingSpace || word.Count != 2 {
		t.Fatalf("repeat state lost: %#v", word)
	}
	if err := rb.History.Redo(rb); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(rb.Rope.String(), "hello world!") {
		t.Fatal("redo history lost")
	}
	if err := rb.History.Undo(rb); err != nil {
		t.Fatal(err)
	}
	if err := rb.History.Undo(rb); err != nil {
		t.Fatal(err)
	}
	if rb.Rope.String() != "hello\nsecond\nthird" {
		t.Fatalf("undo history lost: %q", rb.Rope.String())
	}
}

func TestSessionPreservesEditorHistoryEntries(t *testing.T) {
	cases := []struct {
		name, before, after string
		entry               buffer.HistoryEntry
	}{
		{"paste", "abc", "aXYbc", &PasteCommand{position: 1, content: "XY", oldCursorX: 1}},
		{"rewrap", "one two", "one\ntwo", &RewrapHistoryEntry{startLine: 0, oldLines: []string{"one two"}, newLines: []string{"one", "two"}, startPos: 0, endPos: 7, newContent: "one\ntwo"}},
		{"table", "a|b", "a | b", &markdownTableHistoryEntry{oldLines: []string{"a|b"}, newLines: []string{"a | b"}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e := newSessionTestEditor(t, tc.before)
			if _, err := e.CurrentBuffer().History.Execute(e.CurrentBuffer(), tc.entry); err != nil {
				t.Fatal(err)
			}
			r := roundTripSession(t, e)
			b := r.CurrentBuffer()
			if b.Rope.String() != tc.after {
				t.Fatalf("got %q", b.Rope.String())
			}
			if err := b.History.Undo(b); err != nil {
				t.Fatal(err)
			}
			if b.Rope.String() != tc.before {
				t.Fatalf("undo: %q", b.Rope.String())
			}
			if err := b.History.Redo(b); err != nil {
				t.Fatal(err)
			}
			if b.Rope.String() != tc.after {
				t.Fatalf("redo: %q", b.Rope.String())
			}
		})
	}
}

func TestSessionPreservesInsertAndCompositeHistory(t *testing.T) {
	e := newSessionTestEditor(t, "abc")
	b := e.CurrentBuffer()
	b.History.StartInsertSession("i")
	_, _ = b.History.Execute(b, buffer.NewInsertTextEntry(0, 0, "hi "))
	b.History.EndInsertSession()
	ce := buffer.NewCompositeEntry("two cursors")
	ce.SetPreCursors(buffer.CursorPos{Line: 0, Char: 2}, []buffer.CursorPos{{Line: 0, Char: 4}})
	ce.SetPostCursors(buffer.CursorPos{Line: 0, Char: 3}, []buffer.CursorPos{{Line: 0, Char: 5}})
	ce.Add(buffer.NewInsertTextEntry(0, 0, "x"))
	_, _ = b.History.Execute(b, ce)
	r := roundTripSession(t, e)
	rb := r.CurrentBuffer()
	primary, virtual, ok := rb.History.GetTopUndoCompositePostCursors()
	if !ok || primary.Char != 3 || len(virtual) != 1 || virtual[0].Char != 5 {
		t.Fatal("composite cursor snapshot lost")
	}
	if err := rb.History.Undo(rb); err != nil {
		t.Fatal(err)
	}
	if err := rb.History.Undo(rb); err != nil {
		t.Fatal(err)
	}
	if rb.Rope.String() != "abc" {
		t.Fatalf("insert session undo: %q", rb.Rope.String())
	}
}

func TestSessionPreservesDiffAndTerminalDisplay(t *testing.T) {
	e := newSessionTestEditor(t, "left")
	left := e.CurrentBuffer()
	right := buffer.NewBuffer("right", "right", "", config.IndentSpaces, 4)
	right.ViewIterator, right.CursorIterator = buffer.NewBufferIterator(right), buffer.NewBufferIterator(right)
	e.buffers["right"] = right
	diff := &DiffBufferView{LeftBuffer: left, RightBuffer: right, ActivePane: 1, RealPath: "host:/file.go"}
	e.windows[0] = &Window{Buffer: right, Buffers: []*buffer.Buffer{left, right}, View: diff, Views: []BufferView{diff}}
	term := NewTerminalEmulator(5, 2)
	term.Write([]byte("one\r\ntwo\r\n\x1b[31mred"))
	e.terminalSessions["terminal"] = &TerminalSessionState{SessionID: "keep-alive", WorkingDir: "host:/project", Cols: 5, Rows: 2, term: term, Banner: "shell"}
	r := roundTripSession(t, e)
	dv, ok := r.windows[0].View.(*DiffBufferView)
	if !ok || dv.ActivePane != 1 || dv.RightBuffer != r.CurrentBuffer() || dv.LeftBuffer != r.buffers["scratch"] {
		t.Fatal("diff view lost")
	}
	ts := r.terminalSessions["terminal"]
	if ts.SessionID != "keep-alive" || ts.term.Render() != term.Render() || ts.term.ScrollbackSize() != term.ScrollbackSize() {
		t.Fatal("terminal session/display lost")
	}
	fg, _, _ := ts.term.StyleAtGlobal(0, term.ScrollbackSize()+1)
	want, _, _ := term.StyleAtGlobal(0, term.ScrollbackSize()+1)
	if fg != want {
		t.Fatal("terminal colors lost")
	}
}

func TestSessionRejectsIncompatibleDataWithoutChangingEditor(t *testing.T) {
	e := newSessionTestEditor(t, "keep me")
	data, err := e.SessionState()
	if err != nil {
		t.Fatal(err)
	}
	var s editorSession
	if err := decodeSessionValues(data, &s); err != nil {
		t.Fatal(err)
	}
	s.Version++
	data, err = encodeSessionValues(s)
	if err != nil {
		t.Fatal(err)
	}
	if e.RestoreSession(data) == nil {
		t.Fatal("accepted incompatible session")
	}
	if e.CurrentBuffer().Rope.String() != "keep me" {
		t.Fatal("failed restore modified editor")
	}
	if e.RestoreSession([]byte("broken")) == nil {
		t.Fatal("accepted corrupt session")
	}
}

func TestReloadCommandGuards(t *testing.T) {
	e := newSessionTestEditor(t, "unsaved")
	if e.reloadEditor(nil) == nil {
		t.Fatal("reload without parent accepted")
	}
	e.EnableReload(func() error { return nil })
	e.asyncOperations["save"] = "saving"
	if e.reloadEditor(nil) == nil {
		t.Fatal("reload during background operation accepted")
	}
	delete(e.asyncOperations, "save")
	if err := e.reloadEditor(nil); err != nil {
		t.Fatal(err)
	}
	if !e.ReloadRequested() || !e.shouldExit {
		t.Fatal("reload not requested")
	}
	e.mode = commands.NormalMode
}

func TestPrepareReloadWaitsForAutoSaveAndPendingWrites(t *testing.T) {
	e := newSessionTestEditor(t, "before")
	started, release, writesStarted := make(chan struct{}), make(chan struct{}), make(chan struct{})
	var releaseOnce sync.Once
	defer releaseOnce.Do(func() { close(release) })
	e.autoSaveManager.SetSaveCallback(func(string) {
		close(started)
		<-release
		e.pendingSaves.Add(1)
		close(writesStarted)
	})
	go e.autoSaveManager.requestAutoSave("scratch")
	<-started
	type result struct {
		data []byte
		err  error
	}
	done := make(chan result, 1)
	go func() { data, err := e.PrepareReload(); done <- result{data, err} }()
	select {
	case <-done:
		t.Fatal("snapshot did not wait for an active auto-save callback")
	case <-time.After(30 * time.Millisecond):
	}
	releaseOnce.Do(func() { close(release) })
	<-writesStarted
	select {
	case <-done:
		t.Fatal("snapshot did not wait for a pending write")
	case <-time.After(30 * time.Millisecond):
	}
	b := e.CurrentBuffer()
	b.Insert(b.Length(), " after")
	e.pendingSaves.Done()
	select {
	case result := <-done:
		if result.err != nil {
			t.Fatal(result.err)
		}
		restored := newSessionTestEditor(t, "")
		if err := restored.RestoreSession(result.data); err != nil {
			t.Fatal(err)
		}
		if restored.CurrentBuffer().Rope.String() != "before after" {
			t.Fatal("snapshot missed completed write")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("snapshot did not finish after writes completed")
	}
}

func TestSessionRebindsRepeatObjects(t *testing.T) {
	for _, kind := range []string{"search", "function"} {
		t.Run(kind, func(t *testing.T) {
			e := newSessionTestEditor(t, "word word")
			e.searchPattern = "word"
			var obj TextObject
			if kind == "search" {
				obj = NewSearchResultTextObject(e)
			} else {
				obj = NewFunctionTextObject(ModifierInside, e.proxyManager)
			}
			e.lastCommand = &RepeatableCommand{Operator: DeleteOperator{}, TextObject: obj}
			r := roundTripSession(t, e)
			switch obj := r.lastCommand.TextObject.(type) {
			case SearchResultTextObject:
				if obj.editor != r || obj.searchPattern != "word" {
					t.Fatal("search repeat uses old editor")
				}
			case FunctionTextObject:
				if obj.proxyManager != r.proxyManager || obj.modifier != ModifierInside {
					t.Fatal("function repeat uses old proxy manager")
				}
			default:
				t.Fatalf("unexpected repeat object %T", obj)
			}
		})
	}
}

func TestSessionReloadUsesCurrentThemeDefinitions(t *testing.T) {
	for _, name := range theme.GetThemeNames() {
		t.Run(name, func(t *testing.T) {
			e := newSessionTestEditor(t, "theme reload")
			e.config.Theme = theme.GetThemeByName(name)
			// Represent definitions compiled into the previous worker.
			e.config.Theme.Normal.Foreground = "#123456"
			e.config.Theme.CommandLine.Foreground = "#234567"
			e.config.Theme.Whitespace.Foreground = "#345678"
			e.config.Theme.VerticalRuler.Foreground = "#456789"
			e.config.ScrollOffset = 3
			e.config.VerticalRulers = []int{72, 96}
			e.config.RelativeLineNumbers = true
			want := *e.config
			want.Theme = theme.GetThemeByName(name)

			r := roundTripSession(t, e)
			if !reflect.DeepEqual(r.config, &want) {
				t.Fatal("reload must keep user settings and use current theme definitions")
			}
			if e.config.Theme.Normal.Foreground != "#123456" {
				t.Fatal("reload changed the old worker's theme")
			}
			screen := r.screen.(*MockScreen)
			if screen.defaultStyle != want.Theme.GetStyle(want.Theme.Normal) {
				t.Fatal("screen retained the previous theme")
			}
			for _, line := range []*widgets.CommandLineWidget{r.commandLine, r.searchLine} {
				line.SetPosition(0, 0)
				line.SetText(":test")
				line.Draw(screen)
				if screen.cells[0][0].Style != want.Theme.GetStyle(want.Theme.CommandLine) {
					t.Fatal("command/search line retained the previous theme")
				}
			}
		})
	}
}

func TestSessionReloadRejectsMissingTheme(t *testing.T) {
	e := newSessionTestEditor(t, "keep this")
	e.config.Theme.Name = "removed-theme"
	data, err := e.SessionState()
	if err != nil {
		t.Fatal(err)
	}
	target := newSessionTestEditor(t, "unchanged")
	if err := target.RestoreSession(data); err == nil {
		t.Fatal("reload accepted an unavailable theme")
	}
	if target.CurrentBuffer().Rope.String() != "unchanged" {
		t.Fatal("rejected theme replaced editor state")
	}
}

type sessionActivationScreen struct {
	*MockScreen
	beforeShow func()
}

func (s *sessionActivationScreen) Show() {
	s.beforeShow()
	s.MockScreen.Show()
}

func TestSessionReloadResumesAutoSaveAfterActivation(t *testing.T) {
	for _, failed := range []bool{false, true} {
		name := "successful"
		if failed {
			name = "failed"
		}
		t.Run(name, func(t *testing.T) {
			e := newSessionTestEditor(t, "scratch")
			e.CurrentBuffer().IsModified = true
			for _, path := range []string{"active.txt", "host:/inactive.txt", "clean.txt"} {
				b := buffer.NewBuffer("unsaved", path, path, config.IndentSpaces, 4)
				b.IsModified = path != "clean.txt"
				e.buffers[path] = b
				e.bufferOrder = append(e.bufferOrder, path)
				e.windows[0].Buffers = append(e.windows[0].Buffers, b)
				e.windows[0].Views = append(e.windows[0].Views, &SingleBufferView{Buf: b})
				if b.IsModified {
					e.autoSaveManager.MarkDirty(path)
				}
			}
			e.SetCurrentBuffer(e.buffers["active.txt"])
			data, err := e.PrepareReload()
			if err != nil {
				t.Fatal(err)
			}
			r := e
			if failed {
				r.ResumeAfterReload("test rejection")
			} else {
				r = newSessionTestEditor(t, "discard")
				if err := r.RestoreSession(data); err != nil {
					t.Fatal(err)
				}
			}
			saved := make(chan string, 4)
			r.autoSaveManager.SetSaveCallback(func(path string) { saved <- path })
			r.autoSaveManager.minEditDelay = 10 * time.Millisecond
			r.screen = &sessionActivationScreen{MockScreen: NewMockScreen(100, 30), beforeShow: func() {
				r.autoSaveManager.mutex.RLock()
				defer r.autoSaveManager.mutex.RUnlock()
				if len(r.autoSaveManager.timers) != 0 {
					t.Error("autosave resumed before the first frame was accepted")
				}
			}}
			r.shouldExit = true // Exercise initial rendering and activation without waiting for input.
			r.Run()
			want := map[string]bool{"active.txt": true, "host:/inactive.txt": true}
			for len(want) > 0 {
				select {
				case path := <-saved:
					if !want[path] {
						t.Fatalf("unexpected autosave for %q", path)
					}
					delete(want, path)
				case <-time.After(time.Second):
					t.Fatalf("autosave did not resume for %v", want)
				}
			}
			r.autoSaveManager.Stop()
			select {
			case path := <-saved:
				t.Fatalf("unexpected extra autosave for %q", path)
			default:
			}
		})
	}
}
