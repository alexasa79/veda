package editor

import (
	"strings"
	"veda/internal/buffer"
	"veda/internal/commands"
)

// getSignatureY returns the Y position where signature should be drawn
func (e *Editor) getSignatureY() int {
	// Signature should sit directly above the status bar when present.
	return e.getStatusBarY() - e.getSignatureLines()
}

// getErrorOverlayLines returns how many lines are currently occupied by
// banners/diagnostics above the status bar.
func (e *Editor) getErrorOverlayLines() int {
	// Transient info banner has highest priority and uses one line
	if e.transientInfoActive && e.transientInfoMessage != "" {
		return 1
	}
	// Current banner (info or error) uses one line while visible
	if msg, _ := e.errorManager.GetCurrentBanner(); msg != "" {
		return 1
	}
	// Otherwise, diagnostics for the current line may take multiple lines
	if e.CurrentBuffer() != nil {
		cursorLine := e.CurrentBuffer().CursorIterator.LineNumber() + 1
		if diag, ok := e.CurrentBuffer().GetDiagnosticForLine(cursorLine); ok {
			return e.getDiagnosticLineCount(diag.Message)
		}
	}
	return 0
}

// getSignatureLines returns the number of lines needed for signature display
func (e *Editor) getSignatureLines() int {
	if e.CurrentBuffer().CurrentSignature == nil {
		return 0
	}
	// For now, we'll use 1 line for the signature
	// Later we can expand this if we need multi-line signatures
	return 1
}

// requestSignatureHelpAsync sends an asynchronous request for signature help
func (e *Editor) requestSignatureHelpAsync() {
	// Only works if we have a file open
	if !e.CurrentBuffer().HasFile() {
		return
	}

	// Get cursor position
	line := e.CurrentBuffer().CursorIterator.LineNumber()
	col := e.CurrentBuffer().CursorX

	LL.Debug("Requesting async signature help for %s at %d:%d", e.CurrentBuffer().FilePath, line, col)

	// Launch goroutine to fetch signature help
	go func(generation uint64) {
		sigHelp, err := e.proxyManager.GetSignatureHelp(e.CurrentBuffer().FilePath, line, col)

		// Only apply response if generation matches current state
		if generation == e.CurrentBuffer().SignatureGeneration {
			if err != nil {
				LL.Debug("LSP signature help error for %s: %v", e.CurrentBuffer().FilePath, err)
				e.CurrentBuffer().CurrentSignature = nil
			} else {
				// Parse and store the signature help
				e.parseSignatureHelpResponse(e.CurrentBuffer(), sigHelp)
			}
		} else {
			// Stale response - ignore it
			LL.Debug("Ignoring stale signature help response: gen=%d, current=%d",
				generation, e.CurrentBuffer().SignatureGeneration)
		}
	}(e.CurrentBuffer().SignatureGeneration)
}

// showSignature initiates signature help display
func (e *Editor) showSignature(args []string) error {
	// Increment generation for this new request
	e.CurrentBuffer().SignatureGeneration++

	// Request signature help from LSP
	e.requestSignatureHelpAsync()

	return nil
}

// checkSignatureUpdate checks if we should re-request signature help based on cursor position
func (e *Editor) checkSignatureUpdate() {
	if e.inPasteMode {
		return
	}

	// In insert mode, check if we should show/hide signature help automatically
	if e.mode == commands.InsertMode {
		// If signature is already visible, auto-close using the anchored call check
		if e.CurrentBuffer().CurrentSignature != nil {
			insideAnchor := e.isInsideSignatureCall()
			if !insideAnchor {
				// If anchor fails but heuristic says we're inside, keep signature.
				if !e.isLikelyInsideCallOnCurrentLine() {
					e.CurrentBuffer().CurrentSignature = nil
					return
				}
			} else {
				// Anchor is valid; keep active parameter in sync
				e.recomputeSignatureActiveParameter(e.CurrentBuffer())
			}
			// Trigger refresh on comma or ')'
			if !e.CurrentBuffer().CursorIterator.IsEOF() {
				line := e.CurrentBuffer().CursorIterator.Line()
				if e.CurrentBuffer().CursorX < len(line) {
					ch := line[e.CurrentBuffer().CursorX]
					if ch == ',' || ch == ')' {
						e.CurrentBuffer().SignatureGeneration++
						e.requestSignatureHelpAsync()
					}
				}
			}
			return
		}

		// Otherwise, consider requesting signature when inside parens (heuristic)
		if e.isLikelyInsideCallOnCurrentLine() {
			e.CurrentBuffer().SignatureGeneration++
			e.requestSignatureHelpAsync()
		}
		return
	}

	// In non-insert modes, only check if we have an active signature
	if e.CurrentBuffer().CurrentSignature == nil {
		return
	}

	// If we've moved away from the anchored call, clear the signature.
	if !e.isInsideSignatureCall() && !e.isLikelyInsideCallOnCurrentLine() {
		e.CurrentBuffer().CurrentSignature = nil
		return
	}

	// We're still inside the call context; keep active parameter in sync
	e.recomputeSignatureActiveParameter(e.CurrentBuffer())

	// Check if current character is comma or closing paren
	if !e.CurrentBuffer().CursorIterator.IsEOF() {
		line := e.CurrentBuffer().CursorIterator.Line()
		if e.CurrentBuffer().CursorX < len(line) {
			ch := line[e.CurrentBuffer().CursorX]
			if ch == ',' || ch == ')' {
				// Increment generation and re-request
				e.CurrentBuffer().SignatureGeneration++
				e.requestSignatureHelpAsync()
			}
		}
	}
}

// isLikelyInsideCallOnCurrentLine checks for an unmatched '(' to the left of the
// cursor on the current line, accounting for same-line paren balancing.
// This is a lightweight heuristic to decide whether signature help should remain visible.
func (e *Editor) isLikelyInsideCallOnCurrentLine() bool {
	if e.CurrentBuffer() == nil || e.CurrentBuffer().CursorIterator.IsEOF() {
		return false
	}

	line := e.CurrentBuffer().CursorIterator.Line()
	if len(line) == 0 {
		return false
	}

	// Clamp cursor position within the line bounds (treat positions beyond end as end)
	cursorX := e.CurrentBuffer().CursorX
	if cursorX < 0 {
		cursorX = 0
	}
	if cursorX >= len(line) {
		cursorX = len(line)
	}

	// Walk left from cursor, tracking paren depth on this line.
	// depth counts unmatched ')' we've seen while scanning left.
	depth := 0
	for i := cursorX - 1; i >= 0; i-- {
		switch line[i] {
		case ')':
			depth++
		case '(':
			if depth > 0 {
				depth--
			} else {
				// Found an unmatched '('
				return true
			}
		}
	}
	return false
}

// isInsideSignatureCall checks if the cursor is within the parentheses of the
// function call that matches the currently shown signature's function name.
// If anchoring fails, it falls back to the lightweight heuristic.
func (e *Editor) isInsideSignatureCall() bool {
	buf := e.CurrentBuffer()
	if buf == nil || buf.CurrentSignature == nil || buf.CurrentSignature.Label == "" {
		return false
	}
	funcName := extractFunctionNameFromLabel(buf.CurrentSignature.Label)
	if funcName == "" {
		return e.isLikelyInsideCallOnCurrentLine()
	}
	ato := NewArgumentTextObject(ModifierInside)
	cLine := buf.CursorIterator.LineNumber()
	cChar := buf.CursorX
	ol, oc, ok := findCallOpeningParenForName(ato, buf, cLine, cChar, funcName)
	if !ok {
		// If we can't find the paren anchor, consider being on the function name as inside
		if onName, _, _ := isCursorOnFunctionName(buf, cLine, cChar, funcName); onName {
			return true
		}
		return false
	}
	cl, cc := ato.findClosingFor(buf, ol, oc, '(', ')')
	if cl == -1 {
		return false
	}
	// Include the function name token to the left of '('
	nameStartLine, nameStartChar := ol, oc
	if pl, pc, ok := prevNonSpace(buf, ol, oc); ok {
		// Scan left over identifier-like characters and dots
		iter := buffer.NewBufferIteratorAtLine(buf, pl)
		line := iter.Line()
		start := pc
		for start >= 0 {
			b := line[start]
			if (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9') || b == '_' || b == '.' {
				start--
				continue
			}
			break
		}
		nameStartLine, nameStartChar = pl, start+1
	}
	// Visible anywhere from function name start through the closing paren (inclusive)
	return posLE(nameStartLine, nameStartChar, cLine, cChar) && posLE(cLine, cChar, cl, cc)
}

func posLT(l1, c1, l2, c2 int) bool {
	if l1 != l2 {
		return l1 < l2
	}
	return c1 < c2
}

func posLE(l1, c1, l2, c2 int) bool {
	if l1 != l2 {
		return l1 < l2
	}
	return c1 <= c2
}

// isCursorOnFunctionName checks if the cursor is over an identifier token that
// matches the given function name (or selector suffix). Returns whether it
// matches and the token span [line,charStart]..[line,charEnd).
func isCursorOnFunctionName(buf *buffer.Buffer, line, char int, funcName string) (bool, int, int) {
	if buf == nil {
		return false, 0, 0
	}
	iter := buffer.NewBufferIteratorAtLine(buf, line)
	if iter.IsEOF() {
		return false, 0, 0
	}
	s := iter.Line()
	if char < 0 {
		char = 0
	}
	if char > len(s) {
		char = len(s)
	}
	// If at end, move left one for token
	idx := char
	if idx == len(s) && idx > 0 {
		idx = idx - 1
	}
	// Expand left over ident-like
	start := idx
	for start > 0 {
		b := s[start-1]
		if (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9') || b == '_' || b == '.' {
			start--
			continue
		}
		break
	}
	// Expand right over ident-like
	end := idx
	for end < len(s) {
		b := s[end]
		if (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9') || b == '_' || b == '.' {
			end++
			continue
		}
		break
	}
	if end <= start {
		return false, start, end
	}
	tok := s[start:end]
	if tok == funcName || (len(tok) > len(funcName) && strings.HasSuffix(tok, "."+funcName)) {
		return true, start, end
	}
	return false, start, end
}

// recomputeSignatureActiveParameter updates ActiveParameter using buffer context
// and the LSP-provided signature label as an anchor for the function name.
// If it cannot reliably determine the current argument index or it's out of
// bounds, it disables highlighting by setting ActiveParameter to -1.
func (e *Editor) recomputeSignatureActiveParameter(buf *buffer.Buffer) {
	if buf == nil || buf.CurrentSignature == nil || buf.CurrentSignature.Label == "" {
		return
	}

	funcName := extractFunctionNameFromLabel(buf.CurrentSignature.Label)
	if funcName == "" {
		return // keep LSP-provided value
	}

	// Locate the call parentheses for funcName nearest to the cursor.
	ato := NewArgumentTextObject(ModifierInside)
	cursorLine := buf.CursorIterator.LineNumber()
	cursorChar := buf.CursorX

	ol, oc, ok := findCallOpeningParenForName(ato, buf, cursorLine, cursorChar, funcName)
	if !ok {
		return // no reliable anchor; keep existing
	}
	cl, cc := ato.findClosingFor(buf, ol, oc, '(', ')')
	if cl == -1 {
		return // no matching close; keep existing
	}

	// Parse arguments within these parens and find current argument.
	args := ato.parseArguments(buf, ol, oc, cl, cc)
	idx := -1
	for i, a := range args {
		if ato.positionInRange(cursorLine, cursorChar, a.start, a.end) {
			idx = i
			break
		}
	}

	if idx >= 0 {
		if idx < len(buf.CurrentSignature.Parameters) {
			buf.CurrentSignature.ActiveParameter = idx
		} else {
			// We are in the call but beyond reported params -> disable highlight
			buf.CurrentSignature.ActiveParameter = -1
		}
	}
}

// extractFunctionNameFromLabel tries to obtain the function name from the
// signature label by taking the identifier immediately before the first '('.
func extractFunctionNameFromLabel(label string) string {
	idx := strings.Index(label, "(")
	if idx <= 0 {
		return ""
	}
	// Walk left skipping whitespace
	i := idx - 1
	for i >= 0 && (label[i] == ' ' || label[i] == '\t') {
		i--
	}
	end := i + 1
	for i >= 0 && isIdentChar(label[i]) {
		i--
	}
	start := i + 1
	// If this looks like a selector, return the suffix after the last dot/colon
	name := label[start:end]
	// Split on '.' or ':' and return last token
	sep := -1
	for j := len(name) - 1; j >= 0; j-- {
		if name[j] == '.' || name[j] == ':' {
			sep = j
			break
		}
	}
	if sep >= 0 && sep+1 < len(name) {
		return name[sep+1:]
	}
	return name
}

// findCallOpeningParenForName finds the nearest opening '(' whose identifier
// directly to the left matches funcName (or ends with ".funcName").
func findCallOpeningParenForName(ato *ArgumentTextObject, buf *buffer.Buffer, startLine, startChar int, funcName string) (int, int, bool) {
	line, char := startLine, startChar
	for {
		ol, oc := ato.findOpeningFor(buf, line, char, '(', ')')
		if ol == -1 {
			return 0, 0, false
		}
		name := readIdentLeft(buf, ol, oc)
		if name == funcName || (len(name) > len(funcName) && strings.HasSuffix(name, "."+funcName)) {
			return ol, oc, true
		}
		// Move search just before this opener to look for an outer one.
		line, char = ol, oc-1
	}
}

// readIdentLeft returns the identifier-like token (letters/digits/_/.) directly
// to the left of position (line,char), ignoring whitespace.
func readIdentLeft(buf *buffer.Buffer, line, char int) string {
	// Find previous non-space char
	pl, pc, ok := prevNonSpace(buf, line, char)
	if !ok {
		return ""
	}
	iter := buffer.NewBufferIteratorAtLine(buf, pl)
	s := iter.Line()
	// Collect in reverse
	var rev []byte
	for i := pc; i >= 0; i-- {
		b := s[i]
		if (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9') || b == '_' || b == '.' {
			rev = append(rev, b)
			continue
		}
		break
	}
	// Reverse
	for i, j := 0, len(rev)-1; i < j; i, j = i+1, j-1 {
		rev[i], rev[j] = rev[j], rev[i]
	}
	return string(rev)
}

// prevNonSpace finds the previous non-space character before (line,char).
func prevNonSpace(buf *buffer.Buffer, line, char int) (int, int, bool) {
	for l := line; l >= 0; l-- {
		iter := buffer.NewBufferIteratorAtLine(buf, l)
		if iter.IsEOF() {
			return 0, 0, false
		}
		s := iter.Line()
		c := char - 1
		if l != line {
			c = len(s) - 1
		}
		for ; c >= 0; c-- {
			r := s[c]
			if r != ' ' && r != '\t' && r != '\n' {
				return l, c, true
			}
		}
		char = 0
	}
	return 0, 0, false
}
