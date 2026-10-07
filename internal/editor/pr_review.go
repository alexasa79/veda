package editor

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"veda/internal/buffer"
	"veda/internal/proxy"
	"veda/internal/syntax"
)

const prLinkPrefix = "veda-pr:"

var prReferencePattern = regexp.MustCompile(`^(?:[1-9][0-9]*|https://[^/]+/[^/]+/[^/]+/pull/[1-9][0-9]*/?(?:[?#].*)?)$`)

func prLink(dir string, q proxy.PRReviewRequest) string {
	v := url.Values{"dir": {dir}, "action": {q.Action}}
	for k, s := range map[string]string{"reference": q.Reference, "url": q.URL, "base": q.Base, "head": q.Head, "path": q.Path, "old_path": q.OldPath, "status": q.Status} {
		if s != "" {
			v.Set(k, s)
		}
	}
	return prLinkPrefix + "?" + v.Encode()
}

func parsePRLink(link string) (string, proxy.PRReviewRequest, error) {
	u, err := url.Parse(link)
	if err != nil {
		return "", proxy.PRReviewRequest{}, err
	}
	v := u.Query()
	q := proxy.PRReviewRequest{Action: v.Get("action"), Reference: v.Get("reference"), URL: v.Get("url"), Base: v.Get("base"), Head: v.Get("head"), Path: v.Get("path"), OldPath: v.Get("old_path"), Status: v.Get("status")}
	if u.Scheme != "veda-pr" || v.Get("dir") == "" || (q.Action != "list" && q.Action != "view" && q.Action != "file") {
		return "", q, fmt.Errorf("invalid PR review link")
	}
	return v.Get("dir"), q, nil
}

func (e *Editor) reviewPRs(args []string) error {
	return e.loadPRReview(e.workingDirectory, proxy.PRReviewRequest{Action: "list"}, false)
}

func (e *Editor) reviewPR(args []string) error {
	if len(args) == 0 {
		e.promptPRReview()
		return nil
	}
	if len(args) != 1 || !prReferencePattern.MatchString(args[0]) {
		return fmt.Errorf("usage: review-pr <number or HTTPS PR URL>")
	}
	return e.loadPRReview(e.workingDirectory, proxy.PRReviewRequest{Action: "view", Reference: args[0]}, false)
}

func (e *Editor) openPRLink(link string) error {
	dir, q, err := parsePRLink(link)
	if err != nil {
		return err
	}
	u, _ := url.Parse(link)
	return e.loadPRReview(dir, q, u.Query().Get("refresh") != "1")
}

func (e *Editor) loadPRReview(dir string, q proxy.PRReviewRequest, reuse bool) error {
	key := prLink(dir, q)
	bufferKey := key
	if q.Action == "file" {
		bufferKey += "@current"
	}
	if reuse && q.Action != "view" {
		if b := e.buffers[bufferKey]; b != nil {
			if q.Action != "file" || b.HasFile() {
				e.pushNavigationHistory()
				e.switchToBuffer(b)
				return nil
			}
		}
	}
	var openFiles []string
	if q.Action == "view" {
		var err error
		openFiles, err = e.preparePRCheckout()
		if err != nil {
			return err
		}
	}
	e.pushNavigationHistory()
	if q.Action != "view" {
		e.startAsyncOperation(key, "loading PR review")
	}
	go func() {
		result, err := e.proxyManager.ReviewPR(dir, q)
		var checkout *proxy.PRCheckoutResult
		if err == nil && q.Action == "view" {
			e.Post(func(ed *Editor) {
				ed.startAsyncOperation(prCheckoutOperation, "checking out PR and submodules")
			})
			e.pendingSaves.Wait()
			checkout, err = e.proxyManager.CheckoutPR(dir, result.PR.URL, result.Head, openFiles)
		}
		var diff *buffer.DiffResult
		if err == nil && q.Action == "file" && !result.Binary {
			diff = e.computeDiffViaProxy(result.Left, result.Right)
		}
		e.Post(func(ed *Editor) {
			defer ed.endAsyncOperation(key)
			if q.Action == "view" {
				defer ed.endAsyncOperation(prCheckoutOperation)
				ed.applyPRCheckout(checkout)
			}
			if err != nil {
				ed.ShowError("%v", err)
				return
			}
			if q.Action == "file" {
				if result.Binary {
					ed.ShowInfo("Binary file: %s", q.Path)
					return
				}
				if err := ed.ensurePRWorkingBuffer(result.NavigationPath, result.Right); err != nil {
					ed.ShowError("%v", err)
					return
				}
				ed.openPRDiff(key, q, diff, result.NavigationPath)
				return
			}
			reviewDir := dir
			if checkout != nil && checkout.Root != "" {
				reviewDir = checkout.Root
			}
			name, content := formatPRReview(reviewDir, q, result)
			ed.showPRMarkdown(key, name, content)
		})
	}()
	return nil
}

// Keep externally supplied titles and paths on one Markdown line, outside link syntax.
func prLabel(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	return strings.NewReplacer("[", "［", "]", "］", "*", "∗", "`", "'", "\\", "/").Replace(s)
}

func formatPRReview(dir string, q proxy.PRReviewRequest, result *proxy.PRReviewResult) (string, string) {
	var b strings.Builder
	fmt.Fprintf(&b, "[Refresh](%s&refresh=1)\n\n", prLink(dir, q))
	if q.Action == "list" {
		b.WriteString("# Pull Requests\n\nCurrent repository · open PRs\n\n")
		for _, group := range []struct {
			name string
			prs  []proxy.PullRequest
		}{{"Review requested", result.Requested}, {"Assigned to me", result.Assigned}} {
			fmt.Fprintf(&b, "## %s\n\n", group.name)
			if len(group.prs) == 0 {
				b.WriteString("No PRs.\n\n")
			}
			for _, pr := range group.prs {
				link := prLink(dir, proxy.PRReviewRequest{Action: "view", Reference: pr.URL})
				fmt.Fprintf(&b, "[#%d %s](%s) — %s\n\n", pr.Number, prLabel(pr.Title), link, prLabel(pr.Author.Login))
			}
			if len(group.prs) == 1000 {
				b.WriteString("Showing the first 1000 PRs.\n\n")
			}
		}
		b.WriteString("Use :review-pr <number or URL> to open another PR.\n")
		return "*Pull Requests*", b.String()
	}
	pr := result.PR
	fmt.Fprintf(&b, "# PR #%d: %s\n\nAuthor: %s\n\n%s\n\n", pr.Number, prLabel(pr.Title), prLabel(pr.Author.Login), pr.URL)
	fmt.Fprintf(&b, "Base: %s\nHead: %s\n\n", result.Base, result.Head)
	if pr.Body != "" {
		b.WriteString(pr.Body + "\n\n")
	}
	fmt.Fprintf(&b, "## Changes (%d files)\n", len(result.Files))
	for _, f := range result.Files {
		label := f.Path
		if f.OldPath != "" {
			label = f.OldPath + " → " + f.Path
		}
		link := prLink(dir, proxy.PRReviewRequest{Action: "file", URL: pr.URL, Base: result.Base, Head: result.Head, Path: f.Path, OldPath: f.OldPath, Status: f.Status})
		status := map[string]string{"modified": "M", "added": "A", "removed": "D", "renamed": "R", "copied": "C"}[f.Status]
		if status == "" {
			status = f.Status
		}
		fmt.Fprintf(&b, "[%s](%s) +%d -%d [%s]\n", prLabel(label), link, f.Additions, f.Deletions, status)
	}
	return fmt.Sprintf("*PR #%d: %s*", pr.Number, prLabel(pr.Title)), b.String()
}

func (e *Editor) showPRMarkdown(key, name, content string) {
	content, targets := compactPRLinks(content)
	e.saveCurrentBufferState()
	b := e.buffers[key]
	if b == nil {
		b = buffer.NewBuffer(content, key, name, e.config.DefaultIndentType, e.config.DefaultIndentWidth)
		e.buffers[key] = b
		e.bufferOrder = append(e.bufferOrder, key)
	} else {
		b.Rope = buffer.NewRopeFromString(content)
		b.Generation++
		b.CursorIterator = buffer.NewBufferIterator(b)
		b.ViewIterator = buffer.NewBufferIterator(b)
		b.CursorX = 0
	}
	b.BufferName, b.RealPath = name, ""
	b.MarkdownLinkTargets = targets
	b.UpdateLineNumberWidth()
	b.IsReadOnly, b.IsInternal = true, true
	b.Highlighter = syntax.NewHighlighter()
	b.Highlighter.SetLanguageFromFile("review.md")
	b.Highlighter.ParseFull([]byte(content))
	e.switchToBuffer(b)
}

// Keep routing details out of the displayed Markdown, scoped to this page.
func compactPRLinks(content string) (string, map[string]string) {
	targets := make(map[string]string)
	content = markdownLinkRegex.ReplaceAllStringFunc(content, func(link string) string {
		parts := markdownLinkRegex.FindStringSubmatch(link)
		u, err := url.Parse(parts[2])
		if err != nil || u.Scheme != "veda-pr" {
			return link
		}
		_, q, err := parsePRLink(parts[2])
		if err != nil {
			return link
		}
		short := "#refresh"
		if q.Action == "file" {
			short = (&url.URL{Path: q.Path, Fragment: "diff"}).String()
		} else if q.Action == "view" && u.Query().Get("refresh") != "1" {
			short = q.Reference
		} else if u.Query().Get("refresh") != "1" {
			return link
		}
		short = strings.NewReplacer("(", "%28", ")", "%29").Replace(short)
		targets[short] = parts[2]
		return "[" + parts[1] + "](" + short + ")"
	})
	return content, targets
}

func (e *Editor) openPRDiff(key string, q proxy.PRReviewRequest, diff *buffer.DiffResult, navigationPath string) {
	if existing := e.buffers[key+"@current"]; existing != nil {
		for _, b := range e.findDiffViewForBuffer(existing).AllBuffers() {
			b.RealPath = navigationPath
		}
		e.switchToBuffer(existing)
		return
	}
	e.saveCurrentBufferState()
	makePane := func(suffix, label string, lines []buffer.DiffLine) *buffer.Buffer {
		b := buffer.NewBuffer(createBufferContent(lines), key+suffix, q.Path+" ("+label+")", e.config.DefaultIndentType, e.config.DefaultIndentWidth)
		b.RealPath = navigationPath
		b.IsReadOnly, b.IsInternal = true, true
		b.DiffHints = lines
		b.Highlighter = syntax.NewHighlighter()
		b.Highlighter.SetLanguageFromFile(q.Path)
		b.Highlighter.ParseFull([]byte(b.Rope.String()))
		e.buffers[b.FilePath] = b
		e.bufferOrder = append(e.bufferOrder, b.FilePath)
		return b
	}
	left := makePane("@HEAD", "base "+q.Base[:7], diff.LeftLines)
	right := makePane("@current", "PR "+q.Head[:7], diff.RightLines)
	view := &DiffBufferView{LeftBuffer: left, RightBuffer: right, ActivePane: 1, RealPath: key, Name: q.Path + " [PR diff]"}
	w := e.windows[e.activeWindow]
	w.Views = append(w.Views, view)
	w.addBufferToWindow(left)
	w.addBufferToWindow(right)
	w.View, w.Buffer = view, right
	e.switchToBuffer(right)
	for i, line := range diff.RightLines {
		if line.LineType != buffer.DiffLineContext {
			right.CursorIterator = buffer.NewBufferIteratorAtLine(right, i)
			break
		}
	}
	view.SyncCursors()
	e.scrollToCenter()
}
