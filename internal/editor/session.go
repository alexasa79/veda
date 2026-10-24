package editor

import (
	"fmt"
	"sort"
	"veda/internal/buffer"
	"veda/internal/commands"
	"veda/internal/config"
	"veda/internal/events"
	"veda/internal/proxy"
	"veda/internal/theme"
	"veda/internal/widgets"
)

// Bump for incompatible snapshot changes. A rejected snapshot leaves the old
// worker alive, allowing the user to save before a full restart.
const sessionVersion = 1

type sessionView struct {
	Kind                            string
	Buffer, Left, Right, ActivePane int
	RealPath, Name                  string
}
type sessionWindow struct {
	Buffer                        int
	Buffers                       []int
	Views                         []sessionView
	ActiveView                    int
	ViewLine, CursorLine, CursorX int
}
type sessionTerminal struct {
	State    TerminalSessionState
	Emulator *TerminalEmulator
}
type editorSession struct {
	Version                                    int
	Config                                     *config.Config
	Buffers                                    []buffer.SessionBuffer
	Registry                                   map[string]int
	BufferOrder                                []string
	Windows                                    []sessionWindow
	ActiveWindow, LastWindow, PreviousBuffer   int
	WorkingDirectory                           string
	DirCache                                   map[string][]proxy.FileInfo
	VirtualCursors                             map[int][]VirtualCursor
	Registers                                  map[rune]string
	SearchPattern                              string
	SearchDirection                            bool
	Navigation                                 []NavigationEntry
	References                                 map[string]*ReferencesBuffer
	CurrentReferences, LastActiveSpecialBuffer string
	LSPStatus                                  map[string]string
	Terminals                                  map[string]sessionTerminal
	ShowContext, Fullscreen, ZenMode           bool
	MinWindowWidth                             int
	LastChange                                 buffer.HistoryEntry
	LastCommand                                *RepeatableCommand
	LastSimpleCommand                          string
	LastCommandCount                           int
	CommandHistory, SearchHistory              []string
}

// SessionState exports only data. Buffer IDs preserve shared buffers between
// windows; iterators and callbacks are recreated by RestoreSession.
func (e *Editor) SessionState() ([]byte, error) {
	e.sessionMu.Lock()
	defer e.sessionMu.Unlock()
	s := editorSession{Version: sessionVersion, Config: e.config, Registry: make(map[string]int),
		BufferOrder: e.bufferOrder, ActiveWindow: e.activeWindow, LastWindow: e.lastWindow,
		WorkingDirectory: e.workingDirectory, DirCache: e.dirCache, VirtualCursors: make(map[int][]VirtualCursor),
		Registers: e.registers, SearchPattern: e.searchPattern, SearchDirection: e.searchDirection,
		Navigation: e.navHistory, References: e.referencesBuffers, LastActiveSpecialBuffer: e.lastActiveSpecialBuffer,
		LSPStatus: e.lspExtensionStatus, Terminals: make(map[string]sessionTerminal),
		ShowContext: e.showContext, Fullscreen: e.windowFullscreen, ZenMode: e.zenMode, MinWindowWidth: e.minActiveWindowWidth,
		LastChange: e.lastChange, LastCommand: e.lastCommand, LastSimpleCommand: e.lastSimpleCommand, LastCommandCount: e.lastCommandCount}
	s.CommandHistory, s.SearchHistory = e.commandLine.History(), e.searchLine.History()
	ids := make(map[*buffer.Buffer]int)
	add := func(b *buffer.Buffer) int {
		if b == nil {
			return 0
		}
		if id := ids[b]; id != 0 {
			return id
		}
		s.Buffers = append(s.Buffers, b.SessionState())
		id := len(s.Buffers)
		ids[b] = id
		vcs := e.virtualCursorsByBuffer[b]
		if b == e.CurrentBuffer() {
			vcs = e.virtualCursors
		}
		s.VirtualCursors[id] = vcs
		return id
	}
	var paths []string
	for path := range e.buffers {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	for _, path := range paths {
		s.Registry[path] = add(e.buffers[path])
	}
	viewState := func(view BufferView) (sessionView, error) {
		switch v := view.(type) {
		case *SingleBufferView:
			return sessionView{Kind: "single", Buffer: add(v.Buf)}, nil
		case *DiffBufferView:
			return sessionView{Kind: "diff", Left: add(v.LeftBuffer), Right: add(v.RightBuffer), ActivePane: v.ActivePane, RealPath: v.RealPath, Name: v.Name}, nil
		default:
			return sessionView{}, fmt.Errorf("cannot reload view %T", view)
		}
	}
	for _, w := range e.windows {
		sw := sessionWindow{Buffer: add(w.Buffer), ActiveView: -1, ViewLine: w.ViewLine, CursorLine: w.CursorLine, CursorX: w.CursorX}
		for _, b := range w.Buffers {
			sw.Buffers = append(sw.Buffers, add(b))
		}
		for i, view := range w.Views {
			v, err := viewState(view)
			if err != nil {
				return nil, err
			}
			sw.Views = append(sw.Views, v)
			if view == w.View {
				sw.ActiveView = i
			}
		}
		if w.View != nil && sw.ActiveView == -1 {
			v, err := viewState(w.View)
			if err != nil {
				return nil, err
			}
			sw.ActiveView = len(sw.Views)
			sw.Views = append(sw.Views, v)
		}
		s.Windows = append(s.Windows, sw)
	}
	s.PreviousBuffer = add(e.previousBuffer)
	if e.currentReferencesBuffer != nil {
		s.CurrentReferences = e.currentReferencesBuffer.BufferName
	}
	for path, ts := range e.terminalSessions {
		s.Terminals[path] = sessionTerminal{State: *ts, Emulator: ts.term}
	}
	return encodeSessionValues(s)
}

func (e *Editor) RestoreSession(data []byte) error {
	var s editorSession
	if err := decodeSessionValues(data, &s); err != nil {
		return fmt.Errorf("restore session: %w", err)
	}
	if s.Version != sessionVersion {
		return fmt.Errorf("unsupported session version %d", s.Version)
	}
	if s.Config == nil || s.Config.Theme == nil || len(s.Windows) == 0 || s.ActiveWindow < 0 || s.ActiveWindow >= len(s.Windows) {
		return fmt.Errorf("invalid session layout")
	}
	// Preserve the selected theme, but use definitions compiled into this worker.
	currentTheme := theme.GetThemeByName(s.Config.Theme.Name)
	if currentTheme == nil {
		return fmt.Errorf("session theme %q is unavailable", s.Config.Theme.Name)
	}
	s.Config.Theme = currentTheme
	buffers := make([]*buffer.Buffer, len(s.Buffers)+1)
	for i, state := range s.Buffers {
		if state.History == nil || state.CursorLine < 0 || state.ViewLine < 0 {
			return fmt.Errorf("invalid buffer session")
		}
		buffers[i+1] = state.Restore()
	}
	valid := func(id int) bool { return id >= 0 && id < len(buffers) }
	registry := make(map[string]*buffer.Buffer)
	for path, id := range s.Registry {
		if !valid(id) || id == 0 {
			return fmt.Errorf("invalid session buffer %q", path)
		}
		registry[path] = buffers[id]
	}
	var windows []*Window
	for _, sw := range s.Windows {
		if !valid(sw.Buffer) {
			return fmt.Errorf("invalid window buffer")
		}
		w := &Window{Buffer: buffers[sw.Buffer], ViewLine: sw.ViewLine, CursorLine: sw.CursorLine, CursorX: sw.CursorX}
		for _, id := range sw.Buffers {
			if !valid(id) || id == 0 {
				return fmt.Errorf("invalid window buffers")
			}
			w.Buffers = append(w.Buffers, buffers[id])
		}
		for _, sv := range sw.Views {
			var view BufferView
			switch sv.Kind {
			case "single":
				if !valid(sv.Buffer) || sv.Buffer == 0 {
					return fmt.Errorf("invalid single buffer view")
				}
				view = &SingleBufferView{Buf: buffers[sv.Buffer]}
			case "diff":
				if !valid(sv.Left) || !valid(sv.Right) || sv.Left == 0 || sv.Right == 0 || sv.ActivePane < 0 || sv.ActivePane > 1 {
					return fmt.Errorf("invalid diff view")
				}
				view = &DiffBufferView{LeftBuffer: buffers[sv.Left], RightBuffer: buffers[sv.Right], ActivePane: sv.ActivePane, RealPath: sv.RealPath, Name: sv.Name}
			default:
				return fmt.Errorf("unknown session view %q", sv.Kind)
			}
			w.Views = append(w.Views, view)
		}
		if sw.ActiveView >= 0 {
			if sw.ActiveView >= len(w.Views) {
				return fmt.Errorf("invalid active view")
			}
			w.View = w.Views[sw.ActiveView]
		}
		windows = append(windows, w)
	}
	if !valid(s.PreviousBuffer) {
		return fmt.Errorf("invalid previous buffer")
	}
	for id := range s.VirtualCursors {
		if !valid(id) || id == 0 {
			return fmt.Errorf("invalid virtual cursor buffer")
		}
	}
	// Install after validation, so a rejected snapshot never partly replaces state.
	e.config, e.buffers, e.bufferOrder, e.windows = s.Config, registry, s.BufferOrder, windows
	e.activeWindow, e.lastWindow, e.previousBuffer = s.ActiveWindow, s.LastWindow, buffers[s.PreviousBuffer]
	e.workingDirectory, e.dirCache = s.WorkingDirectory, s.DirCache
	e.virtualCursorsByBuffer = make(map[*buffer.Buffer][]VirtualCursor)
	for id, vcs := range s.VirtualCursors {
		e.virtualCursorsByBuffer[buffers[id]] = vcs
	}
	e.virtualCursors = e.virtualCursorsByBuffer[e.CurrentBuffer()]
	e.registers, e.searchPattern, e.searchDirection = s.Registers, s.SearchPattern, s.SearchDirection
	e.navHistory, e.referencesBuffers, e.lastActiveSpecialBuffer = s.Navigation, s.References, s.LastActiveSpecialBuffer
	e.currentReferencesBuffer = e.referencesBuffers[s.CurrentReferences]
	e.lspExtensionStatus = s.LSPStatus
	e.terminalSessions = make(map[string]*TerminalSessionState)
	for path, saved := range s.Terminals {
		ts := saved.State
		ts.term = saved.Emulator
		e.terminalSessions[path] = &ts
	}
	e.showContext, e.windowFullscreen, e.zenMode, e.minActiveWindowWidth = s.ShowContext, s.Fullscreen, s.ZenMode, s.MinWindowWidth
	e.lastChange, e.lastCommand, e.lastSimpleCommand, e.lastCommandCount = s.LastChange, s.LastCommand, s.LastSimpleCommand, s.LastCommandCount
	if e.lastCommand != nil {
		switch obj := e.lastCommand.TextObject.(type) {
		case SearchResultTextObject:
			obj.editor = e
			e.lastCommand.TextObject = obj
		case FunctionTextObject:
			obj.proxyManager = e.proxyManager
			e.lastCommand.TextObject = obj
		}
	}
	e.mode = commands.NormalMode
	for _, command := range s.CommandHistory {
		e.commandLine.AddToHistory(command)
	}
	for _, search := range s.SearchHistory {
		e.searchLine.AddToHistory(search)
	}
	e.completionMenu = widgets.NewCompletionMenu(e.config.Theme)
	e.keyHintsPopup = widgets.NewKeyHintsPopup(e.config.Theme)
	e.commandLine.SetStyle(e.config.Theme.GetStyle(e.config.Theme.CommandLine))
	e.searchLine.SetStyle(e.config.Theme.GetStyle(e.config.Theme.CommandLine))
	e.whitespaceStyle = newWhitespaceStyle(e.config.Theme.GetStyle(e.config.Theme.Whitespace))
	e.screen.SetStyle(e.config.Theme.GetStyle(e.config.Theme.Normal))
	for _, b := range buffers[1:] {
		b.OnContentChange = func() {
			e.notifyLSPFileChange()
			e.updateSearchMatchesOnContentChange()
			e.refreshReferencesInBackground()
		}
		path := b.RealPath
		if path == "" {
			path = b.FilePath
		}
		if b.IsInternal {
			_ = b.Highlighter.SetLanguage("markdown")
		} else {
			_ = b.Highlighter.SetLanguageFromFileAndContent(path, []byte(b.Rope.String()))
		}
		b.Highlighter.ParseFull([]byte(b.Rope.String()))
	}
	e.recomputeWindowsLayout()
	e.updateSearchMatches()
	e.startRestoredResources = true
	return nil
}

func (e *Editor) reloadEditor(args []string) error {
	if e.beginReload == nil {
		return fmt.Errorf("editor reload requires a vd session parent")
	}
	if len(args) != 0 {
		return fmt.Errorf("usage: reload")
	}
	if e.hasAsyncOperations() {
		return fmt.Errorf("wait for background operations to finish before reloading")
	}
	if err := e.beginReload(); err != nil {
		return err
	}
	e.reloadRequested, e.shouldExit = true, true
	return nil
}

func (e *Editor) EnableReload(begin func() error) { e.beginReload = begin }
func (e *Editor) ReloadRequested() bool           { return e.reloadRequested }

func (e *Editor) PrepareReload() ([]byte, error) {
	e.autoSaveManager.Stop()
	e.pendingSaves.Wait()
	e.stopTerminalPolling()
	e.resumeNotifications = e.proxyManager.PauseNotifications()
	for _, b := range e.buffers {
		b.History.EndInsertSession()
		b.History.EndComposite()
	}
	// Apply already completed operations before capturing their results.
	for {
		select {
		case ev := <-e.asyncEvents:
			e.handleEvent(ev)
		default:
			for path, ts := range e.terminalSessions {
				if !ts.Closed && ts.term != nil {
					e.handleEvent(events.NewCustomEvent("terminal-output", map[string]interface{}{"buffer": path, "data": ts.term.Render()}))
				}
			}
			return e.SessionState()
		}
	}
}

func (e *Editor) ResumeAfterReload(reason string) {
	if e.resumeNotifications != nil {
		e.resumeNotifications()
		e.resumeNotifications = nil
	}
	e.shouldExit, e.reloadRequested = false, false
	e.autoSaveManager.SetSaveCallback(e.performAutoSave)
	e.terminalPollStop = make(chan struct{})
	e.startRestoredResources = true
	e.ShowError("Reload failed: %s", reason)
}

func (e *Editor) stopTerminalPolling() {
	if e.terminalPollStop != nil {
		close(e.terminalPollStop)
		e.terminalPollWG.Wait()
		e.terminalPollStop = nil
	}
}

func (e *Editor) postTerminalEvent(ev events.Event) {
	select {
	case e.asyncEvents <- ev:
	case <-e.terminalPollStop:
	}
}
