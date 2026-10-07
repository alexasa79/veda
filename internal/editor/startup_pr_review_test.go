package editor

import (
	"os"
	"testing"
	"veda/internal/proxy"
)

func TestOpenPullRequestsStartsWithoutFileList(t *testing.T) {
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	te := NewTestEditor(80, 24)
	t.Cleanup(te.Close)
	calls := 0
	// Isolate startup routing from GitHub authentication and network access.
	te.commandRegistry.Register("review-prs", func(args []string) error {
		calls++
		if te.workingDirectory != dir {
			t.Fatalf("working directory = %q, want %q", te.workingDirectory, dir)
		}
		q := proxy.PRReviewRequest{Action: "list"}
		name, content := formatPRReview(dir, q, &proxy.PRReviewResult{})
		te.showPRMarkdown(prLink(dir, q), name, content)
		return nil
	}, false, false)
	te.OpenPullRequests(dir)
	if calls != 0 {
		t.Fatal("PR list loaded before startup")
	}
	te.handlePendingPath()
	te.WaitForAsyncOperations()
	if calls != 1 {
		t.Fatalf("PR list opened %d times", calls)
	}
	if te.renderingState.IsOverlayActive() {
		t.Fatal("file picker opened")
	}
	if te.CurrentBuffer().DisplayName() != "*Pull Requests*" {
		t.Fatalf("unexpected startup buffer: %s", te.CurrentBuffer().DisplayName())
	}
	if te.pendingPath != "" || te.pendingDirectoryCommand != "" {
		t.Fatal("startup request was not cleared")
	}
}

func TestOpenPullRequestsDefersRemotePath(t *testing.T) {
	te := NewTestEditor(80, 24)
	t.Cleanup(te.Close)
	te.OpenPullRequests("host:/project")
	if te.pendingPath != "host:/project" || te.pendingDirectoryCommand != "review-prs" {
		t.Fatal("remote startup request lost")
	}
}
