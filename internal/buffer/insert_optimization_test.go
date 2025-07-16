package buffer

import (
	"testing"
)

func TestInsertNewlineCountOptimization(t *testing.T) {
	t.Run("newline count correctness after insert", func(t *testing.T) {
		testCases := []struct {
			name         string
			originalText string
			insertPos    int
			insertText   string
			expectedText string
		}{
			{
				name:         "insert text with newlines into text without newlines",
				originalText: "hello world",
				insertPos:    5,
				insertText:   "\nfoo\nbar\n",
				expectedText: "hello\nfoo\nbar\n world",
			},
			{
				name:         "insert text without newlines into text with newlines",
				originalText: "line1\nline2\nline3",
				insertPos:    6,
				insertText:   "INSERTED",
				expectedText: "line1\nINSERTEDline2\nline3",
			},
			{
				name:         "insert text with newlines into text with newlines",
				originalText: "first\nsecond\nthird",
				insertPos:    6,
				insertText:   "NEW\nLINE\n",
				expectedText: "first\nNEW\nLINE\nsecond\nthird",
			},
			{
				name:         "insert at beginning",
				originalText: "original\ntext",
				insertPos:    0,
				insertText:   "prefix\n",
				expectedText: "prefix\noriginal\ntext",
			},
			{
				name:         "insert at end",
				originalText: "original\ntext",
				insertPos:    13,
				insertText:   "\nsuffix",
				expectedText: "original\ntext\nsuffix",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				rope := NewRopeFromString(tc.originalText)

				// Count original newlines
				originalNewlines := countNewlines(tc.originalText)
				insertedNewlines := countNewlines(tc.insertText)
				expectedTotalNewlines := originalNewlines + insertedNewlines

				// Perform insertion
				rope.Insert(tc.insertPos, tc.insertText)

				// Verify text content is correct
				actualText := rope.String()
				if actualText != tc.expectedText {
					t.Errorf("Text mismatch: expected %q, got %q", tc.expectedText, actualText)
				}

				// Verify newline count is correct
				actualNewlines := rope.root.newlineCount
				if actualNewlines != expectedTotalNewlines {
					t.Errorf("Newline count mismatch: expected %d, got %d", expectedTotalNewlines, actualNewlines)
				}

				// Verify newline count matches actual count in text
				actualCountInText := countNewlines(actualText)
				if actualNewlines != actualCountInText {
					t.Errorf("Newline count inconsistency: node reports %d, actual text has %d", actualNewlines, actualCountInText)
				}
			})
		}
	})

	t.Run("optimization efficiency verification", func(t *testing.T) {
		// Create a large text with many newlines
		largeText := ""
		for i := 0; i < 1000; i++ {
			largeText += "This is a long line with content that should not be re-scanned\n"
		}

		rope := NewRopeFromString(largeText)
		originalNewlines := rope.root.newlineCount

		// Insert small text with newlines
		smallInsert := "small\ninsert\n"
		insertNewlines := countNewlines(smallInsert)

		rope.Insert(500, smallInsert) // Insert in middle

		// Verify newline count is correct
		expectedNewlines := originalNewlines + insertNewlines
		actualNewlines := rope.root.newlineCount

		if actualNewlines != expectedNewlines {
			t.Errorf("Newline count after optimization: expected %d, got %d", expectedNewlines, actualNewlines)
		}

		// Verify text integrity
		actualText := rope.String()
		actualCountInText := countNewlines(actualText)
		if actualNewlines != actualCountInText {
			t.Errorf("Newline count inconsistency after optimization: node reports %d, actual text has %d", actualNewlines, actualCountInText)
		}
	})
}
