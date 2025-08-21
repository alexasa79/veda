package editor

import (
	"strings"
	"testing"
)

func TestDDEmptyLineBug(t *testing.T) {
	t.Run("dd on empty line should delete the line", func(t *testing.T) {
		// Reproduce the exact scenario from artifacts/python_test_project/main.py
		// Line 16 is an empty line that dd should be able to delete
		content := `#!/usr/bin/env python3
"""
Main entry point for the task management system.
"""

import sys
from typing import List
from datetime import datetime

from models.task import Task, Priority
from models.project import Project
from services.task_manager import TaskManager
from utils.formatter import ConsoleFormatter
from utils.validators import validate_date_string


def create_sample_tasks() -> List[Task]:
    """Create some sample tasks for demonstration."""
    pass`

		// Create test editor with this content
		te := NewTestEditorWithContent(80, 24, content)

		// Navigate to line 16 (0-indexed line 15) - the empty line
		te.SetCursorPosition(0, 15)

		// Verify we're on an empty line
		currentLine := te.buffer.CursorIterator.Line()
		if strings.TrimSpace(currentLine) != "" {
			t.Fatalf("Expected to be on empty line, but line contains: %q", currentLine)
		}
		t.Logf("Cursor is on empty line: %q", currentLine)

		// Count lines before deletion
		linesBefore := strings.Split(te.GetBufferContent(), "\n")
		lineCountBefore := len(linesBefore)
		t.Logf("Lines before dd: %d", lineCountBefore)

		// Execute dd command
		te.TypeKeys("dd")

		// Count lines after deletion
		linesAfter := strings.Split(te.GetBufferContent(), "\n")
		lineCountAfter := len(linesAfter)
		t.Logf("Lines after dd: %d", lineCountAfter)

		// Verify that exactly one line was deleted
		if lineCountAfter != lineCountBefore-1 {
			t.Errorf("Expected line count to decrease by 1. Before: %d, After: %d",
				lineCountBefore, lineCountAfter)

			// Debug: show the content before and after
			t.Logf("Content before dd:")
			for i, line := range linesBefore {
				marker := ""
				if i == 15 {
					marker = " <-- target line"
				}
				t.Logf("  %d: %q%s", i, line, marker)
			}

			t.Logf("Content after dd:")
			for i, line := range linesAfter {
				t.Logf("  %d: %q", i, line)
			}
		}

		// Also verify the cursor position is reasonable
		cursorLine := te.buffer.CursorIterator.LineNumber()
		if cursorLine < 0 || cursorLine >= len(linesAfter) {
			t.Errorf("Cursor position %d is out of bounds for %d lines", cursorLine, len(linesAfter))
		}
	})

	t.Run("dd on various empty line scenarios", func(t *testing.T) {
		testCases := []struct {
			name    string
			content string
			lineNum int
		}{
			{
				name:    "empty line at beginning",
				content: "\nline1\nline2",
				lineNum: 0,
			},
			{
				name:    "empty line in middle",
				content: "line1\n\nline3",
				lineNum: 1,
			},
			{
				name:    "empty line at end",
				content: "line1\n\n", // Line 0: "line1", Line 1: ""
				lineNum: 1,           // Delete the empty line at end (line 1)
			},
			{
				name:    "line with only spaces",
				content: "line1\n   \nline3",
				lineNum: 1,
			},
			{
				name:    "line with only tab",
				content: "line1\n\t\nline3",
				lineNum: 1,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				te := NewTestEditorWithContent(80, 24, tc.content)
				te.SetCursorPosition(0, tc.lineNum)

				linesBefore := strings.Split(te.GetBufferContent(), "\n")
				lineCountBefore := len(linesBefore)

				// Show what line we're targeting
				targetLine := ""
				if tc.lineNum < len(linesBefore) {
					targetLine = linesBefore[tc.lineNum]
				}
				t.Logf("Deleting line %d: %q", tc.lineNum, targetLine)

				te.TypeKeys("dd")

				linesAfter := strings.Split(te.GetBufferContent(), "\n")
				lineCountAfter := len(linesAfter)

				if lineCountAfter != lineCountBefore-1 {
					t.Errorf("Expected line count to decrease by 1. Before: %d, After: %d",
						lineCountBefore, lineCountAfter)
				}
			})
		}
	})
}
