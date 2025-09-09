package editor

import (
	"testing"
)

// getArgContext uses the existing ArgumentTextObject helpers to determine
// whether the cursor is inside an argument list and, if so, which argument
// index (0-based) it is in for the nearest enclosing () pair.
// It returns: inside(bool), argIndex(int, -1 if none).
func getArgContext(te *TestEditor, cursorLine, cursorChar int, funcName string) (bool, int) {
	ato := NewArgumentTextObject(ModifierInside)

	// Find the call parentheses for the target function name as reported by LSP.
	buf := te.CurrentBuffer()
	ol, oc, ok := findCallOpeningParenForName(ato, buf, cursorLine, cursorChar, funcName)
	if !ok {
		return false, -1
	}
	cl, cc := ato.findClosingFor(buf, ol, oc, '(', ')')
	if cl == -1 {
		return false, -1
	}

	// Parse arguments and locate the one containing the cursor.
	args := ato.parseArguments(buf, ol, oc, cl, cc)
	for i, arg := range args {
		if ato.positionInRange(cursorLine, cursorChar, arg.start, arg.end) {
			return true, i
		}
	}
	return true, -1 // inside call parens but not over any argument text
}

// findCallOpeningParenForName climbs outward to the opening '(' whose
// identifier to the left matches the provided function name (exact or suffix
// of a selector like pkg.Func). Returns (line, char, ok).
// The tests use the production findCallOpeningParenForName and helpers defined
// in signature.go within the same package.

// highlightAllowed determines if highlighting the current parameter should
// occur given the LSP-reported parameter count. When argIndex is within the
// reported parameter count, we highlight; otherwise we only show signature.
func highlightAllowed(argIndex, lspParamCount int) bool {
	return argIndex >= 0 && argIndex < lspParamCount
}

func TestSignatureArgContext_SimpleCases(t *testing.T) {
	// Content: f1(a, b, c)
	te := NewTestEditorWithContent(80, 24, "f1(a, b, c)")

	// Case: cursor on function name 'f1' -> outside of argument list
	inside, idx := getArgContext(te, 0, 1, "f1") // on '1' in f1
	if inside {
		t.Fatalf("expected outside of arg list when on function name; got inside with idx=%d", idx)
	}

	// Case: cursor on second argument 'b' -> index 1
	inside, idx = getArgContext(te, 0, 6, "f1") // position of 'b' in "f1(a, b, c)"
	if !inside {
		t.Fatal("expected inside arg list when cursor is on 'b'")
	}
	if idx != 1 {
		t.Fatalf("expected arg index 1 for 'b', got %d", idx)
	}
}

func TestSignatureArgContext_ParamCountVsPassed(t *testing.T) {
	// Content: f1(a, b)
	te := NewTestEditorWithContent(80, 24, "f1(a, b)")

	// Cursor on second arg 'b'
	inside, idx := getArgContext(te, 0, 6, "f1")
	if !inside || idx != 1 {
		t.Fatalf("expected inside with arg index 1; got inside=%v idx=%d", inside, idx)
	}

	// LSP reports more parameters than passed (e.g., expects 4 params)
	if !highlightAllowed(idx, 4) {
		t.Fatalf("expected highlight allowed when idx=%d < lspParams=4", idx)
	}

	// Now simulate fewer parameters reported than passed: lspParams=1
	if highlightAllowed(idx, 1) {
		t.Fatalf("expected highlight NOT allowed when idx=%d >= lspParams=1", idx)
	}
}

func TestSignatureArgContext_ExtraPassedArgs(t *testing.T) {
	// Content: f1(a, b, c, d)
	te := NewTestEditorWithContent(80, 24, "f1(a, b, c, d)")

	// Cursor on third arg 'c'
	inside, idx := getArgContext(te, 0, 9, "f1")
	if !inside || idx != 2 {
		t.Fatalf("expected inside with arg index 2; got inside=%v idx=%d", inside, idx)
	}

	// LSP says only 2 parameters are expected -> should not highlight
	if highlightAllowed(idx, 2) {
		t.Fatalf("expected NO highlight when idx=%d and lspParams=2", idx)
	}
}

func TestSignatureArgContext_NestedCallsAndParens(t *testing.T) {
	// Content: f1(a, g(h(x), y), (p+q)*r)
	content := "f1(a, g(h(x), y), (p+q)*r)"
	te := NewTestEditorWithContent(80, 24, content)

	// Cursor on 'x' inside h(x) -> still in 2nd argument of f1
	// index positions: f1( a ,  g ( h ( x ) , y ) ,  ( p + q ) * r )
	inside, idx := getArgContext(te, 0, nineIndex(content, 'x'), "f1")
	if !inside || idx != 1 {
		t.Fatalf("expected second argument (idx=1) for cursor in h(x); got inside=%v idx=%d", inside, idx)
	}

	// Cursor inside (p+q) -> third argument (idx=2)
	inside, idx = getArgContext(te, 0, idxOf(content, "(p+q)")+2, "f1") // on 'p'
	if !inside || idx != 2 {
		t.Fatalf("expected third argument (idx=2) for cursor in (p+q); got inside=%v idx=%d", inside, idx)
	}
}

func TestSignatureArgContext_MissingClosingParen(t *testing.T) {
	// Content with missing closing paren: f1(a, b
	te := NewTestEditorWithContent(80, 24, "f1(a, b")

	// On 'b' we should fail to find a complete enclosing () pair
	inside, idx := getArgContext(te, 0, 6, "f1")
	if inside {
		t.Fatalf("expected outside (no complete paren pair) but got inside with idx=%d", idx)
	}
}

// idxOf returns the first index of substr in s, or panics if not found.
func idxOf(s, substr string) int {
	pos := -1
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			pos = i
			break
		}
	}
	if pos < 0 {
		panic("substring not found in test content")
	}
	return pos
}

// nineIndex finds the first occurrence of the given rune in s and returns its index.
func nineIndex(s string, r rune) int {
	for i, ch := range s {
		if ch == r {
			return i
		}
	}
	panic("rune not found in test content")
}
