package editor

import (
	"strings"
	"testing"
	"veda/internal/commands"
	"veda/internal/events"
	"veda/internal/proxy"
	"veda/internal/widgets"
)

func TestPRReviewLinks(t *testing.T) {
	q := proxy.PRReviewRequest{Action: "file", URL: "https://github.com/acme/repo/pull/12", Base: strings.Repeat("a", 40), Head: strings.Repeat("b", 40), Path: "dir/a (test)#L2.go", OldPath: "old name.go", Status: "renamed"}
	dir := "remote:/work/my project"
	link := prLink(dir, q)
	gotDir, got, err := parsePRLink(link)
	if err != nil || gotDir != dir || got != q {
		t.Fatalf("round trip: %q %+v %v", gotDir, got, err)
	}
	te := NewTestEditorWithContent(100, 30, "[Diff]("+link+")")
	gotLink, line := te.getMarkdownLinkAtCursor()
	if gotLink != link || line != 0 {
		t.Fatalf("Markdown link: %q, %d", gotLink, line)
	}
	for _, invalid := range []string{"veda-pr:?action=delete&dir=/work", "veda-pr:?action=list", "https://example.com"} {
		if _, _, err := parsePRLink(invalid); err == nil {
			t.Errorf("accepted invalid link %q", invalid)
		}
	}
}

func TestPRReviewMarkdown(t *testing.T) {
	pr := proxy.PullRequest{Number: 12, Title: "Fix [parser]\ncrash", URL: "https://github.com/acme/repo/pull/12"}
	pr.Author.Login = "alice"
	result := &proxy.PRReviewResult{Requested: []proxy.PullRequest{pr}}
	_, text := formatPRReview("/work", proxy.PRReviewRequest{Action: "list"}, result)
	for _, want := range []string{"Review requested", "Assigned to me", "No PRs.", "#12 Fix ［parser］ crash", "alice", "refresh=1"} {
		if !strings.Contains(text, want) {
			t.Errorf("missing %q in %s", want, text)
		}
	}
	result.PR, result.Base, result.Head = pr, strings.Repeat("a", 40), strings.Repeat("b", 40)
	result.Files = []proxy.PRFile{{Path: "deleted.go", Status: "removed", Deletions: 5}, {Path: "new.go", OldPath: "old.go", Status: "renamed"}}
	_, text = formatPRReview("/work", proxy.PRReviewRequest{Action: "view", Reference: pr.URL}, result)
	if !strings.Contains(text, "old.go → new.go") || !strings.Contains(text, "+0 -5") {
		t.Fatal(text)
	}
	links := markdownLinkRegex.FindAllStringSubmatch(text, -1)
	if len(links) != 3 {
		t.Fatalf("expected refresh and two diff links: %v", links)
	}
	for _, link := range links[1:] {
		_, query, err := parsePRLink(link[2])
		if err != nil || query.Base != result.Base || query.Head != result.Head {
			t.Fatalf("un-pinned diff: %+v %v", query, err)
		}
	}
}

func TestPRDiffNavigationAndReadOnly(t *testing.T) {
	te := NewTestEditorWithContent(100, 30, "local unsaved content")
	original := te.CurrentBuffer()
	q := proxy.PRReviewRequest{Action: "file", Base: strings.Repeat("a", 40), Head: strings.Repeat("b", 40), Path: "file.go"}
	key := prLink("/work", q)
	te.showPRMarkdown("pr-list", "*Pull Requests*", "one\ntwo\nthree")
	te.ExecuteCommand("down")
	te.pushNavigationHistory()
	te.openPRDiff(key, q, computeDiff("one\nold\n", "one\nnew\n"), "/work/file.go")
	view := te.findDiffViewForBuffer(te.CurrentBuffer())
	if view == nil {
		t.Fatal("missing diff view")
	}
	for _, b := range view.AllBuffers() {
		if !b.IsReadOnly || b.RealPath != "/work/file.go" {
			t.Fatalf("snapshot can write a file: %+v", b)
		}
		te.switchToBuffer(b)
		before := b.Rope.String()
		for _, cmd := range []string{"enter-insert-mode", "delete-current-character", "toggle-stage-hunk", "toggle-stage-line", "drop-change-hunk", "drop-change-line", "w"} {
			if err := te.ExecuteCommand(cmd); err == nil {
				t.Errorf("%s should reject read-only diff", cmd)
			}
		}
		if b.Rope.String() != before {
			t.Fatal("snapshot was changed")
		}
	}
	te.runInvariants()
	if err := te.goBack(nil); err != nil {
		t.Fatal(err)
	}
	if te.CurrentBuffer().FilePath != "pr-list" || te.CurrentBuffer().CursorIterator.LineNumber() != 1 {
		t.Fatal("back lost PR list position")
	}
	if err := te.openPRLink(key); err != nil {
		t.Fatal(err)
	}
	if te.findDiffViewForBuffer(te.CurrentBuffer()) != view {
		t.Fatal("did not reuse diff")
	}
	if original.Rope.String() != "local unsaved content" {
		t.Fatal("local content changed")
	}
}

func TestPRMarkdownPresentationAndRefresh(t *testing.T) {
	te := NewTestEditorWithContent(40, 12, "")
	key := prLink("/work", proxy.PRReviewRequest{Action: "list"})
	te.showPRMarkdown(key, "*Pull Requests*", "[First PR]("+key+")")
	te.renderPane()
	if got := markdownPresentationScreenLine(te, 0); got != "First PR" {
		t.Fatalf("visible link: %q", got)
	}
	te.showPRMarkdown(key, "*Pull Requests*", "[Updated PR]("+key+")")
	te.renderPane()
	if got := markdownPresentationScreenLine(te, 0); got != "Updated PR" {
		t.Fatalf("stale page: %q", got)
	}
}

func TestPRDeletedFileDiff(t *testing.T) {
	te := NewTestEditorWithContent(80, 24, "local")
	q := proxy.PRReviewRequest{Action: "file", Base: strings.Repeat("a", 40), Head: strings.Repeat("b", 40), Path: "deleted.go", Status: "removed"}
	te.openPRDiff(prLink("/work", q), q, computeDiff("deleted\nfile\n", ""), "")
	if err := te.ExecuteCommand("next-item"); err != nil {
		t.Fatal(err)
	}
	te.runInvariants()
	if err := te.closeDiffBufferPair(te.CurrentBuffer().FilePath); err != nil {
		t.Fatal(err)
	}
	te.runInvariants()
	if te.CurrentBuffer().Rope.String() != "local" {
		t.Fatal("closing review lost local buffer")
	}
}

func TestPRReviewPrompt(t *testing.T) {
	te := NewTestEditorWithContent(80, 24, "text")
	if !te.commandRegistry.Has("review-pr") || !te.commandRegistry.Has("review-prs") {
		t.Fatal("commands missing")
	}
	if err := te.reviewPR([]string{"-1"}); err == nil {
		t.Fatal("accepted invalid number")
	}
	p := &prPrompt{input: widgets.NewLineEditor(0, 0, 64)}
	p.input.SetText(" 123 ")
	done, result := p.HandleEvent(&events.KeyEvent{Key: events.KeyEnter})
	if !done || result != "123" {
		t.Fatalf("submit: %v %q", done, result)
	}
	done, result = p.HandleEvent(&events.KeyEvent{Key: events.KeyEscape})
	if !done || result != "" {
		t.Fatal("cancel did not discard input")
	}
	for _, r := range []rune{'s', 'P'} {
		cmd, complete := te.keyMapper.ProcessKey(&events.KeyEvent{Key: events.KeyRune, Rune: r}, commands.NormalMode)
		if r == 'P' && (!complete || cmd != "review-prs") {
			t.Fatalf("shortcut: %q %v", cmd, complete)
		}
	}
}

func TestPRCommandURLWithQuery(t *testing.T) {
	te := NewTestEditorWithContent(80, 24, "")
	want := "https://github.com/acme/repo/pull/12?diff=split"
	var got []string
	te.commandRegistry.Register("review-pr", func(args []string) error { got = args; return nil }, false, false)
	te.commandLine.SetText(":review-pr " + want)
	te.executeCommand()
	if len(got) != 1 || got[0] != want {
		t.Fatalf("URL interpreted as variable assignment: %v", got)
	}
}

func TestPRChangesCompactLinks(t *testing.T) {
	te := NewTestEditorWithContent(80, 24, "local")
	q := proxy.PRReviewRequest{Action: "view", Reference: "12"}
	result := &proxy.PRReviewResult{
		PR:   proxy.PullRequest{Number: 12, URL: "https://github.com/acme/repo/pull/12"},
		Base: strings.Repeat("a", 40), Head: strings.Repeat("b", 40),
		Files: []proxy.PRFile{
			{Path: "include/example.h", Status: "modified", Additions: 7, Deletions: 7},
			{Path: "src/example (new).go", OldPath: "src/old.go", Status: "renamed", Additions: 2, Deletions: 1},
		},
	}
	name, content := formatPRReview("host:/project", q, result)
	te.showPRMarkdown(prLink("host:/project", q), name, content)
	page := te.CurrentBuffer()
	text := page.Rope.String()
	want := "## Changes (2 files)\n[include/example.h](include/example.h#diff) +7 -7 [M]\n[src/old.go → src/example (new).go](src/example%20%28new%29.go#diff) +2 -1 [R]\n"
	if !strings.HasSuffix(text, want) {
		t.Fatalf("unexpected changes layout:\n%s", text)
	}
	if strings.Contains(text, "veda-pr:") {
		t.Fatal("routing details leaked into Markdown")
	}
	for _, link := range markdownLinkRegex.FindAllStringSubmatch(text, -1) {
		target := page.MarkdownLinkTargets[link[2]]
		dir, request, err := parsePRLink(target)
		if err != nil || dir != "host:/project" {
			t.Fatalf("invalid target %q: %v", target, err)
		}
		if request.Action != "file" {
			continue
		}
		if request.Base != result.Base || request.Head != result.Head {
			t.Fatal("snapshot IDs lost")
		}
		// Preload the snapshot so following the link verifies routing without GitHub.
		te.openPRDiff(target, request, computeDiff("old\n", "new\n"), "host:/project/"+request.Path)
		diff := te.CurrentBuffer()
		te.switchToBuffer(page)
		if err := te.openMarkdownLink(link[2], 0); err != nil {
			t.Fatal(err)
		}
		if te.CurrentBuffer() != diff {
			t.Fatal("short link did not open its PR snapshot")
		}
		te.switchToBuffer(page)
	}
}
