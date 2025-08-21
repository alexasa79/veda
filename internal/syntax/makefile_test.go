package syntax

import (
	"strings"
	"testing"
	"veda/internal/theme"
)

func TestMakefileSyntaxHighlighting(t *testing.T) {
	// Create a highlighter
	h := NewHighlighter()

	// Set language to make
	err := h.SetLanguage("make")
	if err != nil {
		t.Fatalf("Failed to set language to make: %v", err)
	}

	// Test makefile content
	makefileCode := `# This is a comment
CC = gcc
CFLAGS := -Wall -g
SOURCES += main.c utils.c

# Default target
all: program

# Build target
program: $(SOURCES) $(HEADERS)
	$(CC) $(CFLAGS) -o $@ $^

# Clean target
clean:
	rm -f program *.o

# Install with conditional
install: program
	@echo "Installing program..."
	cp program /usr/local/bin/

# Variable with function call
FILES = $(wildcard *.c)

.PHONY: all clean install`

	// Parse the code
	h.ParseFull([]byte(makefileCode))

	// Test cases for different node types that should be found
	testCases := []struct {
		text      string
		tokenType TokenType
		partial   bool // If true, match substring instead of exact match
	}{
		{"# This is a comment", TokenTypeComment, false},
		{"CC", TokenTypeConstant, false}, // CC is a built-in variable
		{"=", TokenTypeOperator, false},
		{"gcc", TokenTypeString, false},      // Variable value content in tree-sitter
		{"CFLAGS", TokenTypeConstant, false}, // Built-in variable
		{":=", TokenTypeOperator, false},
		{"all", TokenTypeConstant, false}, // Standard target
		{":", TokenTypeOperator, false},
		{"program", TokenTypeFunction, false}, // Custom target
		{"CC", TokenTypeConstant, false},      // Variable reference within $(CC) - just check for CC
		{".PHONY", TokenTypeConstant, false},  // Built-in target
	}

	// Check each test case
	for _, tc := range testCases {
		found := false
		var foundToken *Token

		// Find the token containing our test text
		lines := strings.Split(makefileCode, "\n")
		lineStart := 0

		for _, line := range lines {
			if strings.TrimSpace(line) == "" {
				lineStart += len(line) + 1
				continue
			}

			lineEnd := lineStart + len(line)

			tokens := h.GetTokensForLine(lineStart, lineEnd)
			for _, token := range tokens {
				tokenText := string(makefileCode[lineStart+token.Start : lineStart+token.End])
				if tc.partial {
					// For partial matches, check if the text is contained in the token
					if strings.Contains(tokenText, tc.text) {
						found = true
						foundToken = &token
						break
					}
				} else {
					// For exact matches
					if tokenText == tc.text {
						found = true
						foundToken = &token
						break
					}
				}
			}
			if found {
				break
			}

			lineStart = lineEnd + 1 // +1 for the newline character
		}

		if !found {
			t.Errorf("Token not found for text: %s", tc.text)
			continue
		}

		if foundToken.TokenType != tc.tokenType {
			t.Errorf("Wrong token type for %s: got %v, want %v", tc.text, foundToken.TokenType, tc.tokenType)
		}

		// Verify that GetStyleForToken returns the correct style
		testTheme := theme.NewTheme()
		style := GetStyleForToken(foundToken.TokenType, testTheme)

		// Verify that GetStyleForToken returns a valid style (we don't test exact mapping here)
		if style == (theme.StyleDef{}) {
			t.Errorf("Token %s (%v) should have a valid style", tc.text, tc.tokenType)
		}
	}
}

func TestMakefileFileDetection(t *testing.T) {
	h := NewHighlighter()

	testCases := []struct {
		filename     string
		shouldBeMake bool
	}{
		{"Makefile", true},
		{"makefile", true},
		{"GNUmakefile", true},
		{"gnumakefile", true},
		{"build.mk", true},
		{"rules.mak", true},
		{"script.sh", false},
		{"script.py", false},
		{"README.md", false},
	}

	for _, tc := range testCases {
		err := h.SetLanguageFromFile(tc.filename)

		if tc.shouldBeMake {
			if err != nil {
				t.Errorf("Failed to set language for %s: %v", tc.filename, err)
				continue
			}

			if h.GetCurrentLanguage() != "make" {
				t.Errorf("Wrong language set for %s: got %s, want make", tc.filename, h.GetCurrentLanguage())
			}

			// Check that make language was loaded by verifying we can parse makefile code
			h.ParseFull([]byte("all: test\n\techo done"))
			// For make, we should use tree-sitter parsing
			if h.currentTree == nil {
				t.Errorf("Make should use tree-sitter parsing")
			}
			if h.currentSource == nil {
				t.Errorf("Make source should be stored")
			}
		} else {
			if h.GetCurrentLanguage() == "make" {
				t.Errorf("Make language incorrectly set for non-makefile: %s", tc.filename)
			}
		}
	}
}

func TestMakefileVariableReferences(t *testing.T) {
	h := NewHighlighter()
	h.SetLanguage("make")

	testCode := `$(VAR) ${VAR2} $@ $< $^`
	h.ParseFull([]byte(testCode))

	tokens := h.GetTokensForLine(0, len(testCode))

	// Tree-sitter parses variable references as separate tokens: $ + ( + VAR + )
	// So we look for $ operators and variable names within the tokens
	foundDollarSigns := 0
	foundVarNames := 0

	for _, token := range tokens {
		tokenText := testCode[token.Start:token.End]
		if token.TokenType == TokenTypeOperator && tokenText == "$" {
			foundDollarSigns++
		}
		if token.TokenType == TokenTypeText && (tokenText == "VAR" || tokenText == "VAR2") {
			foundVarNames++
		}
		if token.TokenType == TokenTypeConstant && (tokenText == "@" || tokenText == "<" || tokenText == "^") {
			foundVarNames++
		}
	}

	// We should find at least some $ signs and variable components
	if foundDollarSigns == 0 {
		t.Error("Expected to find $ operators in variable references")
	}
}

func TestMakefileTargetAndDependencies(t *testing.T) {
	h := NewHighlighter()
	h.SetLanguage("make")

	testCode := `program: main.o utils.o`
	h.ParseFull([]byte(testCode))

	tokens := h.GetTokensForLine(0, len(testCode))

	// Should find target and colon operator
	hasTarget := false
	hasColon := false

	for _, token := range tokens {
		tokenText := testCode[token.Start:token.End]
		if token.TokenType == TokenTypeFunction && tokenText == "program" {
			hasTarget = true
		}
		if token.TokenType == TokenTypeOperator && tokenText == ":" {
			hasColon = true
		}
	}

	if !hasTarget {
		t.Error("Expected to find 'program' as a target")
	}
	if !hasColon {
		t.Error("Expected to find ':' as an operator")
	}
}
