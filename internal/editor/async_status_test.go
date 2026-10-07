package editor

import (
	"strings"
	"testing"
)

func TestPRCheckoutStatusShowsOperationAndClears(t *testing.T) {
	te := NewTestEditorWithContent(100, 24, "")
	t.Cleanup(te.Close)
	status := func() string {
		te.render()
		return te.GetScreenContent()[te.getStatusBarY()]
	}
	if _, err := te.preparePRCheckout(); err != nil {
		t.Fatal(err)
	}
	if got := status(); !strings.Contains(got, "loading PR review") {
		t.Fatalf("missing loading status: %q", got)
	}
	// Connection messages and other operations must not hide checkout progress.
	te.connectionStatus = "Connected"
	te.startAsyncOperation("another-request", "loading other data")
	te.startAsyncOperation(prCheckoutOperation, "checking out PR and submodules")
	if got := status(); !strings.Contains(got, "checking out PR and submodules") {
		t.Fatalf("missing checkout status: %q", got)
	}
	te.endAsyncOperation(prCheckoutOperation)
	if got := status(); strings.Contains(got, "checking out PR") || !strings.Contains(got, "loading other data") {
		t.Fatalf("incorrect status after checkout ends: %q", got)
	}
	te.endAsyncOperation("another-request")
	if got := status(); strings.Contains(got, "loading other data") || !strings.Contains(got, "Connected") {
		t.Fatalf("incorrect status after all operations end: %q", got)
	}
}
