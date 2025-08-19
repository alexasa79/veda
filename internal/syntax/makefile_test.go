package syntax

import (
	"strings"
	"testing"
	"veda/internal/theme"
)

func TestMakefileSyntaxHighlighting(t *testing.T) {
	// Create a highlighter
	h := NewHighlighter()
	
	// Set language to makefile
	err := h.SetLanguage("makefile")
	if err != nil {
		t.Fatalf("Failed to set language to makefile: %v", err)
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
	
	// Test cases for different token types
	testCases := []struct {
		text      string
		tokenType TokenType
		partial   bool // If true, match substring instead of exact match
	}{
		{"# This is a comment", TokenTypeMakeComment, false},
		{"CC", TokenTypeMakeVariable, false},
		{"=", TokenTypeMakeOperator, false}, 
		{"gcc", TokenTypeText, true}, // Variable value content
		{"CFLAGS", TokenTypeMakeVariable, false},
		{":=", TokenTypeMakeOperator, false},
		{"all", TokenTypeMakeTarget, false},
		{":", TokenTypeOperator, false},
		{"program", TokenTypeMakeTarget, false},
		{"$(SOURCES)", TokenTypeMakeVariableRef, false},
		{"$(CC)", TokenTypeMakeVariableRef, false},
		{"rm -f program *.o", TokenTypeText, true}, // Command text
		{"@echo", TokenTypeText, true}, // Command with @
		{"\"Installing program...\"", TokenTypeMakeString, false},
		{"$(wildcard *.c)", TokenTypeMakeVariableRef, false},
		{".PHONY", TokenTypeMakeTarget, false},
	}
	
	// Check each test case
	for _, tc := range testCases {
		found := false
		var foundToken *Token
		
		// Find the token containing our test text
		lines := strings.Split(makefileCode, "\n")
		lineStart := 0
		
		for _, line := range lines {
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
		
		// Verify that tokens get appropriate styles
		switch tc.tokenType {
		case TokenTypeMakeTarget:
			if style != testTheme.SyntaxFunction {
				t.Errorf("Token %s should use SyntaxFunction style", tc.text)
			}
		case TokenTypeMakeVariable:
			if style != testTheme.SyntaxVariable {
				t.Errorf("Token %s should use SyntaxVariable style", tc.text)
			}
		case TokenTypeMakeVariableRef:
			if style != testTheme.SyntaxConstant {
				t.Errorf("Token %s should use SyntaxConstant style", tc.text)
			}
		case TokenTypeMakeString:
			if style != testTheme.SyntaxString {
				t.Errorf("Token %s should use SyntaxString style", tc.text)
			}
		case TokenTypeMakeComment:
			if style != testTheme.SyntaxComment {
				t.Errorf("Token %s should use SyntaxComment style", tc.text)
			}
		case TokenTypeMakeOperator:
			if style != testTheme.SyntaxOperator {
				t.Errorf("Token %s should use SyntaxOperator style", tc.text)
			}
		case TokenTypeOperator:
			if style != testTheme.SyntaxOperator {
				t.Errorf("Token %s should use SyntaxOperator style", tc.text)
			}
		}
	}
}

func TestMakefileFileDetection(t *testing.T) {
	h := NewHighlighter()
	
	testCases := []struct {
		filename      string
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
			
			if h.GetCurrentLanguage() != "makefile" {
				t.Errorf("Wrong language set for %s: got %s, want makefile", tc.filename, h.GetCurrentLanguage())
			}
			
			// Check that makefile language was loaded by verifying we can parse makefile code
			h.ParseFull([]byte("all: test\n\techo done"))
			// For makefile, currentTree should be nil since we don't use tree-sitter
			if h.currentTree != nil {
				t.Errorf("Makefile should not use tree-sitter parsing")
			}
			if h.currentSource == nil {
				t.Errorf("Makefile source should be stored")
			}
		} else {
			if h.GetCurrentLanguage() == "makefile" {
				t.Errorf("Makefile language incorrectly set for non-makefile: %s", tc.filename)
			}
		}
	}
}

func TestMakefileVariableReferences(t *testing.T) {
	h := NewHighlighter()
	h.SetLanguage("makefile")
	
	testCode := `$(VAR) ${VAR2} $@ $< $^`
	h.ParseFull([]byte(testCode))
	
	tokens := h.GetTokensForLine(0, len(testCode))
	
	expectedVarRefs := []string{"$(VAR)", "${VAR2}", "$@", "$<", "$^"}
	foundVarRefs := 0
	
	for _, token := range tokens {
		if token.TokenType == TokenTypeMakeVariableRef {
			tokenText := testCode[token.Start:token.End]
			found := false
			for _, expected := range expectedVarRefs {
				if tokenText == expected {
					found = true
					break
				}
			}
			if found {
				foundVarRefs++
			}
		}
	}
	
	if foundVarRefs != len(expectedVarRefs) {
		t.Errorf("Expected %d variable references, found %d", len(expectedVarRefs), foundVarRefs)
	}
}

func TestMakefileTargetAndDependencies(t *testing.T) {
	h := NewHighlighter()
	h.SetLanguage("makefile")
	
	testCode := `program: main.o utils.o`
	h.ParseFull([]byte(testCode))
	
	tokens := h.GetTokensForLine(0, len(testCode))
	
	// Should find target and colon operator
	hasTarget := false
	hasColon := false
	
	for _, token := range tokens {
		tokenText := testCode[token.Start:token.End]
		if token.TokenType == TokenTypeMakeTarget && tokenText == "program" {
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