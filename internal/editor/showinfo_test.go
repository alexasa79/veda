package editor

import (
	"testing"
	"time"
)

func TestShowInfo(t *testing.T) {
	te := NewTestEditor(80, 24)

	// Show an info message
	te.Editor.ShowInfo("Test info message")

	// Verify the error manager has the message with correct severity
	msg, severity := te.Editor.errorManager.GetCurrentBanner()

	if msg != "Test info message" {
		t.Errorf("Expected message 'Test info message', got %q", msg)
	}

	if severity != "info" {
		t.Errorf("Expected severity 'info', got %q", severity)
	}

	// Test with formatted message
	te.Editor.ShowInfo("File saved: %s", "/tmp/test.txt")

	msg, severity = te.Editor.errorManager.GetCurrentBanner()

	if msg != "File saved: /tmp/test.txt" {
		t.Errorf("Expected formatted message, got %q", msg)
	}

	if severity != "info" {
		t.Errorf("Expected severity 'info', got %q", severity)
	}

	// Test that message expires after duration
	time.Sleep(3 * time.Second) // Default duration is 2 seconds

	msg, severity = te.Editor.errorManager.GetCurrentBanner()

	if msg != "" || severity != "" {
		t.Errorf("Expected message to expire, but got msg=%q, severity=%q", msg, severity)
	}
}
