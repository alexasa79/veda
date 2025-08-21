package editor

import (
	"strings"
	"testing"
	"veda/internal/buffer"
	"veda/internal/syntax"
)

func TestArrowSyntaxHighlightingBug(t *testing.T) {
	// This test reproduces a bug where typing "class" at line 15 in main.py
	// causes the "->" in line 18 to change color, while "def" stays correct

	t.Run("arrow operator loses color when class is typed above", func(t *testing.T) {
		// Read the actual main.py content to reproduce the exact scenario
		pythonCode := `#!/usr/bin/env python3
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
    tasks = [
        Task(
            title="Implement user authentication",
            description="Add login/logout functionality with JWT tokens",
            priority=Priority.HIGH,
            due_date=datetime(2024, 12, 31)
        ),
        Task(
            title="Write unit tests",
            description="Achieve 80% code coverage",
            priority=Priority.MEDIUM
        ),
        Task(
            title="Update documentation",
            description="Add API documentation and usage examples",
            priority=Priority.LOW,
            tags=["docs", "api"]
        )
    ]
    return tasks`

		// Create test editor with Python content
		te := NewTestEditorWithContent(80, 24, pythonCode)
		te.buffer.FilePath = "main.py"

		// Initialize syntax highlighting
		te.highlighter = syntax.NewHighlighter()
		err := te.highlighter.SetLanguageFromFile("main.py")
		if err != nil {
			t.Skipf("Skipping test: Python syntax highlighting not available: %v", err)
		}
		te.highlighter.ParseFull([]byte(te.buffer.Rope.String()))

		// Force render to ensure syntax highlighting is applied
		te.render()

		// Find the line containing the arrow operator "-> List[Task]:"
		lines := strings.Split(te.GetBufferContent(), "\n")
		var arrowLineNum int = -1

		for i, line := range lines {
			if strings.Contains(line, "-> List[Task]:") {
				arrowLineNum = i
				break
			}
		}

		if arrowLineNum == -1 {
			t.Fatal("Could not find '-> List[Task]:' in the original content")
		}

		t.Logf("Found arrow operator on line %d: %s", arrowLineNum, lines[arrowLineNum])

		// Get the expected operator style from the theme
		expectedOperatorStyle := te.config.Theme.GetStyle(te.config.Theme.SyntaxOperator)
		expectedOpFg, _, _ := expectedOperatorStyle.Decompose()

		// Find the "->" operator on its line and get its initial style
		initialArrowStyle := getCellStyle(te, arrowLineNum, "->")
		if initialArrowStyle == nil {
			t.Fatal("Could not find '->' operator on its line before making changes")
		}

		initialArrowFg, _, _ := initialArrowStyle.Decompose()
		t.Logf("Before typing 'class': arrow has color %v, expected operator color %v", initialArrowFg, expectedOpFg)

		// Navigate to line 15 (where user wants to type "class")
		te.SetCursorPosition(0, 15)

		// Enter insert mode and open a new line by pressing 'o'
		te.ExecuteCommand("open-line-below")

		// Type "class"
		for _, ch := range "class" {
			te.InsertChar(ch)
		}

		// Exit insert mode
		te.PressKey(KeyEscape, 0)

		// Force render after the edit
		te.render()

		// Debug: Print buffer content after typing "class"
		t.Logf("Buffer content after typing 'class' at line 15:")
		updatedLines := strings.Split(te.GetBufferContent(), "\n")
		for i, line := range updatedLines {
			if i >= 14 && i <= 19 { // Show context around the change
				t.Logf("Line %d: %s", i, line)
			}
		}

		// Find the arrow line again (it may have shifted down)
		var newArrowLineNum int = -1

		for i, line := range updatedLines {
			if strings.Contains(line, "-> List[Task]:") {
				newArrowLineNum = i
				break
			}
		}

		if newArrowLineNum == -1 {
			t.Fatal("Could not find '-> List[Task]:' after typing 'class'")
		}

		t.Logf("After change: arrow operator now on line %d", newArrowLineNum)

		// Check that the arrow operator still has the correct color
		arrowStyleAfter := getCellStyle(te, newArrowLineNum, "->")
		if arrowStyleAfter == nil {
			t.Fatal("Could not find '->' operator after typing 'class'")
		}

		arrowFgAfter, _, _ := arrowStyleAfter.Decompose()

		if arrowFgAfter != expectedOpFg {
			t.Errorf("BUG REPRODUCED: '->' operator lost its color after typing 'class' above it. Got %v, expected %v", arrowFgAfter, expectedOpFg)

			// Debug: Check what tokens the highlighter has for the arrow line
			arrowIter := buffer.NewBufferIteratorAtLine(te.buffer, newArrowLineNum)
			arrowLineStart := arrowIter.Position()
			arrowLineEnd := arrowIter.LineEnd()
			arrowLineContent := arrowIter.Line()
			arrowTokens := te.highlighter.GetTokensForLine(arrowLineStart, arrowLineEnd)
			t.Logf("Arrow line content: %q", arrowLineContent)
			t.Logf("Tokens for arrow line (line %d, range %d-%d): %+v", newArrowLineNum, arrowLineStart, arrowLineEnd, arrowTokens)

			// Try manually calling updateSyntaxHighlighting and re-render to see if that fixes it
			t.Log("Testing if manual updateSyntaxHighlighting fixes the issue...")
			te.updateSyntaxHighlighting()
			te.render()

			arrowStyleAfterUpdate := getCellStyle(te, newArrowLineNum, "->")
			if arrowStyleAfterUpdate != nil {
				arrowFgAfterUpdate, _, _ := arrowStyleAfterUpdate.Decompose()
				if arrowFgAfterUpdate == expectedOpFg {
					t.Log("SUCCESS: Manual updateSyntaxHighlighting fixes the arrow issue")
				} else {
					t.Logf("FAILURE: Even manual updateSyntaxHighlighting doesn't fix the arrow. Got %v, expected %v", arrowFgAfterUpdate, expectedOpFg)

					// Check tokens after manual update
					arrowTokensAfterUpdate := te.highlighter.GetTokensForLine(arrowLineStart, arrowLineEnd)
					t.Logf("Tokens for arrow line after manual update: %+v", arrowTokensAfterUpdate)
				}
			}
		} else {
			t.Logf("'->' operator still has correct color after typing 'class': %v", arrowFgAfter)
		}

		// Also verify that "def" keyword stayed correct (unlike the previous bug)
		var defLineNum int = -1

		for i, line := range updatedLines {
			if strings.Contains(line, "def create_sample_tasks") {
				defLineNum = i
				break
			}
		}

		if defLineNum != -1 {
			defStyle := getCellStyle(te, defLineNum, "def")
			if defStyle != nil {
				expectedKeywordStyle := te.config.Theme.GetStyle(te.config.Theme.SyntaxKeyword)
				expectedKeywordFg, _, _ := expectedKeywordStyle.Decompose()
				defFg, _, _ := defStyle.Decompose()

				if defFg == expectedKeywordFg {
					t.Logf("GOOD: 'def' keyword maintained correct color: %v", defFg)
				} else {
					t.Logf("UNEXPECTED: 'def' keyword also lost color: got %v, expected %v", defFg, expectedKeywordFg)
				}
			}
		}
	})

	t.Run("fresh parse with class and arrow", func(t *testing.T) {
		// Test with content that already has "class" to see if the issue is with fresh parsing
		freshCode := `#!/usr/bin/env python3
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

class SomeClass:

def create_sample_tasks() -> List[Task]:
    """Create some sample tasks for demonstration."""
    pass`

		// Create a fresh editor with this content
		te := NewTestEditorWithContent(80, 24, freshCode)
		te.buffer.FilePath = "main.py"

		// Initialize syntax highlighting
		te.highlighter = syntax.NewHighlighter()
		err := te.highlighter.SetLanguageFromFile("main.py")
		if err != nil {
			t.Skipf("Skipping test: Python syntax highlighting not available: %v", err)
		}
		te.highlighter.ParseFull([]byte(freshCode))

		// Force render
		te.render()

		// Find the arrow line
		lines := strings.Split(freshCode, "\n")
		var arrowLineNum int = -1

		for i, line := range lines {
			if strings.Contains(line, "-> List[Task]:") {
				arrowLineNum = i
				break
			}
		}

		if arrowLineNum == -1 {
			t.Fatal("Could not find '-> List[Task]:' in fresh content")
		}

		t.Logf("Fresh parse - arrow line content: %q", lines[arrowLineNum])

		// Check if the arrow gets highlighted correctly in this fresh parse
		arrowStyle := getCellStyle(te, arrowLineNum, "->")
		expectedOperatorStyle := te.config.Theme.GetStyle(te.config.Theme.SyntaxOperator)
		expectedOpFg, _, _ := expectedOperatorStyle.Decompose()

		if arrowStyle != nil {
			arrowFg, _, _ := arrowStyle.Decompose()
			t.Logf("Fresh parse - arrow color: %v, expected: %v", arrowFg, expectedOpFg)
			if arrowFg == expectedOpFg {
				t.Log("Fresh parse: arrow has correct color - so the issue is with incremental updates")
			} else {
				t.Log("Fresh parse: arrow has wrong color - the issue might be with the Python content structure")
			}
		} else {
			t.Log("Fresh parse: could not find arrow style")
		}

		// Check tokens for the arrow line in fresh parse
		arrowIter := buffer.NewBufferIteratorAtLine(te.buffer, arrowLineNum)
		arrowLineStart := arrowIter.Position()
		arrowLineEnd := arrowIter.LineEnd()
		arrowTokens := te.highlighter.GetTokensForLine(arrowLineStart, arrowLineEnd)
		t.Logf("Fresh parse - tokens for arrow line: %+v", arrowTokens)
	})
}
