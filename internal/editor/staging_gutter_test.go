package editor

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestLineStagingGutter verifies staging specific lines updates git and gutter indicators,
// and that re-editing a staged line shows both changed and staged markers.
func TestLineStagingGutter(t *testing.T) {
	repo := NewGitTestRepo(t)
	run := func(name string, args ...string) {
		cmd := exec.Command(name, args...)
		cmd.Dir = repo
		cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=test", "GIT_AUTHOR_EMAIL=test@example.com", "GIT_COMMITTER_NAME=test", "GIT_COMMITTER_EMAIL=test@example.com")
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("command failed: %s %v\n%s", name, args, string(out))
		}
	}

	run("git", "init", ".")
	run("git", "config", "user.name", "Test User")
	run("git", "config", "user.email", "test@example.com")

	// Base file content and initial commit (9 lines to separate hunks >3 lines apart)
	base := "line1\nline2\nline3\nline4\nline5\nline6\nline7\nline8\nline9\n"
	path := filepath.Join(repo, "file.txt")
	if err := os.WriteFile(path, []byte(base), 0644); err != nil {
		t.Fatalf("write base: %v", err)
	}
	run("git", "add", "file.txt")
	run("git", "commit", "-m", "initial")

	// Modify two lines far apart (2 and 8)
	modified := "line1\nline2X\nline3\nline4\nline5\nline6\nline7\nline8X\nline9\n"
	if err := os.WriteFile(path, []byte(modified), 0644); err != nil {
		t.Fatalf("write modified: %v", err)
	}

	// Open editor with modified content
	te := NewTestEditorWithContent(100, 25, modified)
	// Re-key the buffer in editor maps under the real path so reload-file can find it
	oldKey := te.CurrentBuffer().FilePath
	te.CurrentBuffer().FilePath = path
	te.CurrentBuffer().RealPath = path
	te.buffers[path] = te.CurrentBuffer()
	delete(te.buffers, oldKey)

	// Load git changes via proxy directly (setup step)
	fc, err := te.Editor.proxyManager.GetFile(path)
	if err != nil {
		t.Fatalf("GetFile: %v", err)
	}
	te.CurrentBuffer().GitChanges = &fc.Changes

	// Stage lines 2 and 8 using keybinding "g space"
	// Move to line 2
	te.TypeKeys("down") // now on line 2 (1-based)
	te.TypeKeys("g space")
	// Move to line 8 (from 2 -> need 6 downs)
	te.TypeKeys("down")
	te.TypeKeys("down")
	te.TypeKeys("down")
	te.TypeKeys("down")
	te.TypeKeys("down")
	te.TypeKeys("down")
	te.TypeKeys("g space")

	// Verify staged sets include 2 and 8
	changes := te.CurrentBuffer().GitChanges
	if changes == nil {
		t.Fatalf("GitChanges is nil")
	}
	has := func(arr []int, v int) bool {
		for _, x := range arr {
			if x == v {
				return true
			}
		}
		return false
	}
	if !(has(changes.StagedModified, 2) || has(changes.StagedAdded, 2)) {
		t.Fatalf("expected line 2 to be staged; got %+v", changes)
	}
	if !(has(changes.StagedModified, 8) || has(changes.StagedAdded, 8)) {
		t.Fatalf("expected line 8 to be staged; got %+v", changes)
	}
	if has(changes.StagedModified, 3) || has(changes.StagedAdded, 3) {
		t.Fatalf("did not expect line 3 staged; got %+v", changes)
	}

	// Render to draw gutter
	te.render()
	// Helper to read gutter staged marker (rightmost indicator column)
	readGutterRune := func(oneBasedLine int) rune {
		lnw := te.CurrentBuffer().LineNumberWidth
		digits := lnw - 3
		if digits < 1 {
			digits = 1
		}
		x := digits + 2 // rightmost indicator column for staged
		// Compute screen Y
		contentStartY := len(te.CurrentBuffer().ContextCache)
		viewTop := te.CurrentBuffer().ViewIterator.LineNumber()
		screenY := contentStartY + (oneBasedLine - 1 - viewTop)
		r, _ := te.mockScreen.GetCell(x, screenY)
		return r
	}
	// Helper to read gutter change marker (leftmost indicator column)
	readLeftGutterRune := func(oneBasedLine int) rune {
		lnw := te.CurrentBuffer().LineNumberWidth
		digits := lnw - 3
		if digits < 1 {
			digits = 1
		}
		x := digits
		contentStartY := len(te.CurrentBuffer().ContextCache)
		viewTop := te.CurrentBuffer().ViewIterator.LineNumber()
		screenY := contentStartY + (oneBasedLine - 1 - viewTop)
		r, _ := te.mockScreen.GetCell(x, screenY)
		return r
	}
	// Right column shows '✓' for staged lines 2 and 4; not for 3
	if r := readGutterRune(2); r != '✓' {
		t.Fatalf("expected staged marker '✓' at line 2, got %q", r)
	}
	if r := readGutterRune(8); r != '✓' {
		t.Fatalf("expected staged marker '✓' at line 8, got %q", r)
	}
	if r := readGutterRune(3); r == '✓' {
		t.Fatalf("did not expect staged marker at line 3")
	}
	if r := readLeftGutterRune(2); r != '≋' {
		t.Fatalf("expected modified marker '≋' at line 2, got %q", r)
	}
	if r := readLeftGutterRune(8); r != '≋' {
		t.Fatalf("expected modified marker '≋' at line 8, got %q", r)
	}

	// Now modify a staged line again (line 2): enter insert, append 'Y', exit
	// Move cursor back to line 2
	te.TypeKeys("up")
	te.TypeKeys("up")
	if err := te.ExecuteCommand("enter-insert-mode"); err != nil {
		t.Fatal(err)
	}
	_ = te.InsertChar('Y')
	if err := te.ExecuteCommand("exit-insert-mode"); err != nil {
		t.Fatal(err)
	}

	// Reload git changes to reflect new unstaged diff on top of staged
	if err := te.ExecuteCommand("reload-file"); err != nil {
		t.Fatalf("reload-file after edit: %v", err)
	}
	te.render()

	// Expect both an unstaged change marker in left column and a staged marker in right column for line 2
	// Left column index: digits
	lnw := te.CurrentBuffer().LineNumberWidth
	digits := lnw - 3
	if digits < 1 {
		digits = 1
	}
	leftX := digits // leftmost indicator
	rightX := digits + 2
	contentStartY := len(te.CurrentBuffer().ContextCache)
	viewTop := te.CurrentBuffer().ViewIterator.LineNumber()
	y := contentStartY + (2 - 1 - viewTop)
	lch, _ := te.mockScreen.GetCell(leftX, y)
	rch, _ := te.mockScreen.GetCell(rightX, y)
	if lch != '≋' || rch != '✓' {
		t.Fatalf("expected modified '≋' (left) and staged (right) markers on line 2; got left=%q right=%q", lch, rch)
	}
}

// TestLineStagingGutter_ModifiedAfterLineShift ensures a replacement stays marked
// as modified even when earlier insertions shift line numbers.
func TestLineStagingGutter_ModifiedAfterLineShift(t *testing.T) {
	repo := NewGitTestRepo(t)
	run := func(name string, args ...string) {
		cmd := exec.Command(name, args...)
		cmd.Dir = repo
		cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=test", "GIT_AUTHOR_EMAIL=test@example.com", "GIT_COMMITTER_NAME=test", "GIT_COMMITTER_EMAIL=test@example.com")
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("command failed: %s %v\n%s", name, args, string(out))
		}
	}

	run("git", "init", ".")
	run("git", "config", "user.name", "Test User")
	run("git", "config", "user.email", "test@example.com")

	baseLines := []string{
		"line1",
		"line2",
		"line3",
		"line4",
		"line5",
		"line6",
		"line7",
		"line8",
		"line9",
		"line10",
	}
	base := strings.Join(baseLines, "\n") + "\n"
	path := filepath.Join(repo, "file.txt")
	if err := os.WriteFile(path, []byte(base), 0644); err != nil {
		t.Fatalf("write base: %v", err)
	}
	run("git", "add", "file.txt")
	run("git", "commit", "-m", "initial")

	modLines := []string{
		"line1",
		"line2",
		"insA",
		"insB",
		"line3",
		"line4",
		"line5",
		"line6",
		"line7",
		"line8X",
		"line9",
		"line10",
	}
	modified := strings.Join(modLines, "\n") + "\n"
	if err := os.WriteFile(path, []byte(modified), 0644); err != nil {
		t.Fatalf("write modified: %v", err)
	}

	var changedLine int
	for i, line := range modLines {
		if line == "line8X" {
			changedLine = i + 1
			break
		}
	}
	if changedLine == 0 {
		t.Fatalf("could not locate changed line in modified content")
	}

	te := NewTestEditorWithContent(100, 25, modified)
	oldKey := te.CurrentBuffer().FilePath
	te.CurrentBuffer().FilePath = path
	te.CurrentBuffer().RealPath = path
	te.buffers[path] = te.CurrentBuffer()
	delete(te.buffers, oldKey)

	fc, err := te.Editor.proxyManager.GetFile(path)
	if err != nil {
		t.Fatalf("GetFile: %v", err)
	}
	te.CurrentBuffer().GitChanges = &fc.Changes

	te.render()

	readLeftGutterRune := func(oneBasedLine int) rune {
		lnw := te.CurrentBuffer().LineNumberWidth
		digits := lnw - 3
		if digits < 1 {
			digits = 1
		}
		x := digits
		contentStartY := len(te.CurrentBuffer().ContextCache)
		viewTop := te.CurrentBuffer().ViewIterator.LineNumber()
		screenY := contentStartY + (oneBasedLine - 1 - viewTop)
		r, _ := te.mockScreen.GetCell(x, screenY)
		return r
	}

	if r := readLeftGutterRune(changedLine); r != '≋' {
		t.Fatalf("expected modified marker '≋' at line %d, got %q", changedLine, r)
	}
}

// TestHunkStaging_MultiLineReplaceLargeFile sets up a larger file and performs a 1→2
// line replacement inside a single hunk, then attempts to stage the affected lines.
// This reproduces the known overlapping-hunks issue in the patch builder.
func TestHunkStaging_MultiLineReplaceLargeFile(t *testing.T) {
	// This test validates staging of a 1→2 split within a single hunk on a large file

	repo := NewGitTestRepo(t)
	run := func(name string, args ...string) {
		cmd := exec.Command(name, args...)
		cmd.Dir = repo
		cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=test", "GIT_AUTHOR_EMAIL=test@example.com", "GIT_COMMITTER_NAME=test", "GIT_COMMITTER_EMAIL=test@example.com")
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("command failed: %s %v\n%s", name, args, string(out))
		}
	}

	// Init repo
	run("git", "init", ".")
	run("git", "config", "user.name", "Test User")
	run("git", "config", "user.email", "test@example.com")

	// Build a larger base file (300+ lines) with a function signature at ~line 151
	var baseLines []string
	for i := 1; i <= 150; i++ {
		baseLines = append(baseLines, "line "+fmt.Sprint(i))
	}
	baseLines = append(baseLines, "int compute(int a, int b) {") // line 151
	baseLines = append(baseLines, "\treturn a + b;")
	baseLines = append(baseLines, "}")
	for i := 154; i <= 320; i++ {
		baseLines = append(baseLines, "line "+fmt.Sprint(i))
	}
	base := strings.Join(baseLines, "\n") + "\n"

	path := filepath.Join(repo, "bigfile.c")
	if err := os.WriteFile(path, []byte(base), 0644); err != nil {
		t.Fatalf("write base: %v", err)
	}
	run("git", "add", filepath.Base(path))
	run("git", "commit", "-m", "initial")

	// Modify: split the function signature line (151) into two lines (1→2 replace)
	var modLines []string
	for i := 1; i <= 150; i++ {
		modLines = append(modLines, "line "+fmt.Sprint(i))
	}
	modLines = append(modLines, "int compute(int a,")
	modLines = append(modLines, "\t\tint b) {")
	modLines = append(modLines, "\treturn a + b;")
	modLines = append(modLines, "}")
	for i := 154; i <= 320; i++ {
		modLines = append(modLines, "line "+fmt.Sprint(i))
	}
	modified := strings.Join(modLines, "\n") + "\n"
	if err := os.WriteFile(path, []byte(modified), 0644); err != nil {
		t.Fatalf("write modified: %v", err)
	}

	// Compute working-file line numbers of the two new lines (1-based)
	var addLine1, addLine2 int
	for idx, s := range modLines {
		if s == "int compute(int a," {
			addLine1 = idx + 1
		}
		if s == "\t\tint b) {" {
			addLine2 = idx + 1
		}
	}
	if addLine1 == 0 || addLine2 == 0 {
		t.Fatalf("could not locate added lines in modified content")
	}

	// Start editor and wire buffer to real path
	te := NewTestEditorWithContent(120, 30, modified)
	oldKey := te.CurrentBuffer().FilePath
	te.CurrentBuffer().FilePath = path
	te.CurrentBuffer().RealPath = path
	te.buffers[path] = te.CurrentBuffer()
	delete(te.buffers, oldKey)

	// Load initial git changes for the buffer
	fc, err := te.Editor.proxyManager.GetFile(path)
	if err != nil {
		t.Fatalf("GetFile: %v", err)
	}
	te.CurrentBuffer().GitChanges = &fc.Changes

	// Jump to the first added line using command mode and stage the hunk via keybinding ("g h")
	te.TypeKeys(":" + fmt.Sprint(addLine1) + "\n")
	te.TypeKeys("g h")

	// Refresh from git to verify
	fc2, err := te.Editor.proxyManager.GetFile(path)
	if err != nil {
		t.Fatalf("GetFile after staging: %v", err)
	}
	te.CurrentBuffer().GitChanges = &fc2.Changes
	changes := te.CurrentBuffer().GitChanges
	has := func(arr []int, v int) bool {
		for _, x := range arr {
			if x == v {
				return true
			}
		}
		return false
	}
	if !(has(changes.StagedModified, addLine1) || has(changes.StagedAdded, addLine1)) {
		// Dump *Messages* buffer for debugging
		if mbuf, ok := te.buffers["*Messages*"]; ok && mbuf.Rope != nil {
			t.Logf("Messages:\n%s", mbuf.Rope.String())
		}
		t.Fatalf("expected line %d to be staged; got %+v", addLine1, changes)
	}
	if !(has(changes.StagedModified, addLine2) || has(changes.StagedAdded, addLine2)) {
		if mbuf, ok := te.buffers["*Messages*"]; ok && mbuf.Rope != nil {
			t.Logf("Messages:\n%s", mbuf.Rope.String())
		}
		t.Fatalf("expected line %d to be staged; got %+v", addLine2, changes)
	}
}

// Test partial selection within a replace block (1→2 split: stage only first line)
func TestHunkStaging_PartialSelectionWithinReplaceLargeFile(t *testing.T) {
	repo := NewGitTestRepo(t)
	run := func(name string, args ...string) {
		cmd := exec.Command(name, args...)
		cmd.Dir = repo
		cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=test", "GIT_AUTHOR_EMAIL=test@example.com", "GIT_COMMITTER_NAME=test", "GIT_COMMITTER_EMAIL=test@example.com")
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("command failed: %s %v\n%s", name, args, string(out))
		}
	}

	run("git", "init", ".")
	run("git", "config", "user.name", "Test User")
	run("git", "config", "user.email", "test@example.com")

	// Base large file
	var baseLines []string
	for i := 1; i <= 150; i++ {
		baseLines = append(baseLines, "line "+fmt.Sprint(i))
	}
	baseLines = append(baseLines, "int f(int a, int b) {")
	baseLines = append(baseLines, "\treturn a + b;")
	baseLines = append(baseLines, "}")
	for i := 154; i <= 320; i++ {
		baseLines = append(baseLines, "line "+fmt.Sprint(i))
	}
	base := strings.Join(baseLines, "\n") + "\n"

	path := filepath.Join(repo, "partial.c")
	if err := os.WriteFile(path, []byte(base), 0644); err != nil {
		t.Fatalf("write base: %v", err)
	}
	run("git", "add", filepath.Base(path))
	run("git", "commit", "-m", "initial")

	// Modify: 1→2 line split
	var modLines []string
	for i := 1; i <= 150; i++ {
		modLines = append(modLines, "line "+fmt.Sprint(i))
	}
	modLines = append(modLines, "int f(int a,")
	modLines = append(modLines, "\t\tint b) {")
	modLines = append(modLines, "\treturn a + b;")
	modLines = append(modLines, "}")
	for i := 154; i <= 320; i++ {
		modLines = append(modLines, "line "+fmt.Sprint(i))
	}
	modified := strings.Join(modLines, "\n") + "\n"
	if err := os.WriteFile(path, []byte(modified), 0644); err != nil {
		t.Fatalf("write modified: %v", err)
	}

	var addLine1, addLine2 int
	for idx, s := range modLines {
		if s == "int f(int a," {
			addLine1 = idx + 1
		}
		if s == "\t\tint b) {" {
			addLine2 = idx + 1
		}
	}
	if addLine1 == 0 || addLine2 == 0 {
		t.Fatalf("could not find added lines")
	}

	te := NewTestEditorWithContent(120, 30, modified)
	oldKey := te.CurrentBuffer().FilePath
	te.CurrentBuffer().FilePath = path
	te.CurrentBuffer().RealPath = path
	te.buffers[path] = te.CurrentBuffer()
	delete(te.buffers, oldKey)

	// Load git changes via proxy to initialize buffer state
	fc, err := te.Editor.proxyManager.GetFile(path)
	if err != nil {
		t.Fatalf("GetFile: %v", err)
	}
	te.CurrentBuffer().GitChanges = &fc.Changes

	// Jump to the first added line via command mode and stage only that line ("g space")
	te.TypeKeys(":" + fmt.Sprint(addLine1) + "\n")
	te.TypeKeys("g space")
	// Refresh from git to verify
	fc2, err := te.Editor.proxyManager.GetFile(path)
	if err != nil {
		t.Fatalf("GetFile after staging: %v", err)
	}
	te.CurrentBuffer().GitChanges = &fc2.Changes
	changes := te.CurrentBuffer().GitChanges

	has := func(arr []int, v int) bool {
		for _, x := range arr {
			if x == v {
				return true
			}
		}
		return false
	}
	if !(has(changes.StagedModified, addLine1) || has(changes.StagedAdded, addLine1)) {
		if mbuf, ok := te.buffers["*Messages*"]; ok && mbuf.Rope != nil {
			t.Logf("Messages:\n%s", mbuf.Rope.String())
		}
		t.Fatalf("expected line %d to be staged; got %+v", addLine1, changes)
	}
	if has(changes.StagedModified, addLine2) || has(changes.StagedAdded, addLine2) {
		t.Fatalf("did not expect line %d to be staged; got %+v", addLine2, changes)
	}
}

// Test staging of a pure insertion line
func TestHunkStaging_PureInsertionLargeFile(t *testing.T) {
	repo := NewGitTestRepo(t)
	run := func(name string, args ...string) {
		cmd := exec.Command(name, args...)
		cmd.Dir = repo
		cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=test", "GIT_AUTHOR_EMAIL=test@example.com", "GIT_COMMITTER_NAME=test", "GIT_COMMITTER_EMAIL=test@example.com")
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("command failed: %s %v\n%s", name, args, string(out))
		}
	}
	run("git", "init", ".")
	run("git", "config", "user.name", "Test User")
	run("git", "config", "user.email", "test@example.com")

	// Base file
	var baseLines []string
	for i := 1; i <= 200; i++ {
		baseLines = append(baseLines, "base "+fmt.Sprint(i))
	}
	base := strings.Join(baseLines, "\n") + "\n"
	path := filepath.Join(repo, "insert.txt")
	if err := os.WriteFile(path, []byte(base), 0644); err != nil {
		t.Fatalf("write base: %v", err)
	}
	run("git", "add", filepath.Base(path))
	run("git", "commit", "-m", "initial")

	// Modify: insert one line at ~161
	var modLines []string
	for i := 1; i <= 160; i++ {
		modLines = append(modLines, "base "+fmt.Sprint(i))
	}
	modLines = append(modLines, "INSERTED LINE")
	for i := 161; i <= 200; i++ {
		modLines = append(modLines, "base "+fmt.Sprint(i))
	}
	modified := strings.Join(modLines, "\n") + "\n"
	if err := os.WriteFile(path, []byte(modified), 0644); err != nil {
		t.Fatalf("write modified: %v", err)
	}

	var insLine int
	for idx, s := range modLines {
		if s == "INSERTED LINE" {
			insLine = idx + 1
			break
		}
	}
	if insLine == 0 {
		t.Fatalf("could not find inserted line")
	}

	te := NewTestEditorWithContent(120, 30, modified)
	oldKey := te.CurrentBuffer().FilePath
	te.CurrentBuffer().FilePath = path
	te.CurrentBuffer().RealPath = path
	te.buffers[path] = te.CurrentBuffer()
	delete(te.buffers, oldKey)

	// Load changes and stage the inserted line via keybinding
	fc, err := te.Editor.proxyManager.GetFile(path)
	if err != nil {
		t.Fatalf("GetFile: %v", err)
	}
	te.CurrentBuffer().GitChanges = &fc.Changes
	// Jump directly to the inserted line using command mode
	te.TypeKeys(":" + fmt.Sprint(insLine) + "\n")
	te.TypeKeys("g space")
	changes := te.CurrentBuffer().GitChanges

	has := func(arr []int, v int) bool {
		for _, x := range arr {
			if x == v {
				return true
			}
		}
		return false
	}
	if !(has(changes.StagedAdded, insLine) || has(changes.StagedModified, insLine)) {
		if mbuf, ok := te.buffers["*Messages*"]; ok && mbuf.Rope != nil {
			t.Logf("Messages:\n%s", mbuf.Rope.String())
		}
		t.Fatalf("expected inserted line %d to be staged; got %+v", insLine, changes)
	}
}

// Test that unstage reverses a staging operation
func TestUnstageLines_SymmetryLargeFile(t *testing.T) {
	repo := NewGitTestRepo(t)
	run := func(name string, args ...string) {
		cmd := exec.Command(name, args...)
		cmd.Dir = repo
		cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=test", "GIT_AUTHOR_EMAIL=test@example.com", "GIT_COMMITTER_NAME=test", "GIT_COMMITTER_EMAIL=test@example.com")
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("command failed: %s %v\n%s", name, args, string(out))
		}
	}

	run("git", "init", ".")
	run("git", "config", "user.name", "Test User")
	run("git", "config", "user.email", "test@example.com")

	// Base large file
	var baseLines []string
	for i := 1; i <= 120; i++ {
		baseLines = append(baseLines, "L "+fmt.Sprint(i))
	}
	base := strings.Join(baseLines, "\n") + "\n"
	path := filepath.Join(repo, "sym.c")
	if err := os.WriteFile(path, []byte(base), 0644); err != nil {
		t.Fatalf("write base: %v", err)
	}
	run("git", "add", filepath.Base(path))
	run("git", "commit", "-m", "initial")

	// Modify: add two lines (pure insertion block)
	var modLines []string
	for i := 1; i <= 60; i++ {
		modLines = append(modLines, "L "+fmt.Sprint(i))
	}
	modLines = append(modLines, "A1")
	modLines = append(modLines, "A2")
	for i := 61; i <= 120; i++ {
		modLines = append(modLines, "L "+fmt.Sprint(i))
	}
	modified := strings.Join(modLines, "\n") + "\n"
	if err := os.WriteFile(path, []byte(modified), 0644); err != nil {
		t.Fatalf("write modified: %v", err)
	}

	// Determine lines
	var l1, l2 int
	for idx, s := range modLines {
		if s == "A1" {
			l1 = idx + 1
		}
		if s == "A2" {
			l2 = idx + 1
		}
	}
	if l1 == 0 || l2 == 0 {
		t.Fatalf("could not find inserted range")
	}

	te := NewTestEditorWithContent(120, 30, modified)
	oldKey := te.CurrentBuffer().FilePath
	te.CurrentBuffer().FilePath = path
	te.CurrentBuffer().RealPath = path
	te.buffers[path] = te.CurrentBuffer()
	delete(te.buffers, oldKey)

	// Load changes
	fc, err := te.Editor.proxyManager.GetFile(path)
	if err != nil {
		t.Fatalf("GetFile: %v", err)
	}
	te.CurrentBuffer().GitChanges = &fc.Changes
	// Move to first line and stage both lines via key ("g space" on each)
	for i := 1; i < l1; i++ {
		te.TypeKeys("down")
	}
	te.TypeKeys("g space")
	te.TypeKeys("down")
	te.TypeKeys("g space")
	// Now unstage the same two lines via key (toggle each with "g space")
	te.TypeKeys("up")
	te.TypeKeys("g space")
	te.TypeKeys("up")
	te.TypeKeys("g space")
	changes := te.CurrentBuffer().GitChanges

	has := func(arr []int, v int) bool {
		for _, x := range arr {
			if x == v {
				return true
			}
		}
		return false
	}
	if has(changes.StagedAdded, l1) || has(changes.StagedModified, l1) || has(changes.StagedAdded, l2) || has(changes.StagedModified, l2) {
		t.Fatalf("expected lines to be unstaged; got %+v", changes)
	}
}
