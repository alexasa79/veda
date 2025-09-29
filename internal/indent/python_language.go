package indent

import (
	"sort"
	"strings"
	"unicode"
)

// PythonLanguage implements indentation analysis for Python source.
type PythonLanguage struct{}

// NewPythonLanguage constructs a Python language specification.
func NewPythonLanguage() *PythonLanguage {
	return &PythonLanguage{}
}

// Rules returns anchor and continuation pairings suitable for Python.
func (p *PythonLanguage) Rules() Rules {
	return Rules{
		Anchors: []Pair{{Open: TokenIndent, Close: TokenDedent}},
		Continuations: []Pair{
			{Open: TokenOpenParen, Close: TokenCloseParen},
			{Open: TokenOpenBracket, Close: TokenCloseBracket},
			{Open: TokenOpenBrace, Close: TokenCloseBrace},
		},
	}
}

// Tokenize produces tokens for Python source.
func (p *PythonLanguage) Tokenize(src []rune) ([]Token, error) {
	tk := newPythonTokenizer(src)
	return tk.scan()
}

// ExpressionRanges identifies the expression span for reindent-expression.
func (p *PythonLanguage) ExpressionRanges(tokens []Token, cursorLine, cursorColumn int) []Range {
	type pair struct {
		openIdx  int
		closeIdx int
		kind     TokenKind
	}

	var parenPairs []pair
	var indentPairs []pair
	var parenStack []int
	var indentStack []int

	for idx, tok := range tokens {
		switch tok.Kind {
		case TokenOpenParen, TokenOpenBracket, TokenOpenBrace:
			parenStack = append(parenStack, idx)
		case TokenCloseParen, TokenCloseBracket, TokenCloseBrace:
			if len(parenStack) == 0 {
				continue
			}
			openIdx := parenStack[len(parenStack)-1]
			parenStack = parenStack[:len(parenStack)-1]
			parenPairs = append(parenPairs, pair{openIdx: openIdx, closeIdx: idx, kind: tokens[openIdx].Kind})
		case TokenIndent:
			indentStack = append(indentStack, idx)
		case TokenDedent:
			if len(indentStack) == 0 {
				continue
			}
			openIdx := indentStack[len(indentStack)-1]
			indentStack = indentStack[:len(indentStack)-1]
			indentPairs = append(indentPairs, pair{openIdx: openIdx, closeIdx: idx, kind: TokenIndent})
		}
	}

	selectParenRange := func(tokenIdx int) (Range, bool) {
		best := Range{StartLine: cursorLine, EndLine: cursorLine}
		bestSpan := 1 << 30
		bestKind := TokenUnknown
		for _, pr := range parenPairs {
			if pr.openIdx <= tokenIdx && tokenIdx <= pr.closeIdx {
				op := tokens[pr.openIdx]
				cl := tokens[pr.closeIdx]
				if cursorLine < op.Start.Line || cursorLine > cl.Start.Line {
					continue
				}
				span := cl.Start.Line - op.Start.Line
				if span < 0 {
					span = 0
				}
				if span < bestSpan || (span == bestSpan && prefersKind(pr.kind, bestKind)) {
					bestSpan = span
					bestKind = pr.kind
					best = Range{StartLine: op.Start.Line, EndLine: cl.Start.Line}
				}
			}
		}
		if bestSpan != 1<<30 {
			return best, true
		}
		return Range{}, false
	}

	selectIndentRange := func(tokenIdx int) (Range, bool) {
		best := Range{StartLine: cursorLine, EndLine: cursorLine}
		bestSpan := 1 << 30
		for _, pr := range indentPairs {
			if pr.openIdx <= tokenIdx && tokenIdx < pr.closeIdx {
				start := tokens[pr.openIdx].Start.Line
				end := tokens[pr.closeIdx].Start.Line - 1
				if end < start {
					end = start
				}
				if cursorLine < start || cursorLine > end {
					continue
				}
				span := end - start
				if span < 0 {
					span = 0
				}
				if span < bestSpan {
					bestSpan = span
					best = Range{StartLine: start, EndLine: end}
				}
			}
		}
		if bestSpan != 1<<30 {
			return best, true
		}
		return Range{}, false
	}

	resultSet := make(map[[2]int]struct{})
	hadTokenOnLine := false
	for idx, tok := range tokens {
		if tok.Start.Line > cursorLine {
			break
		}
		if tok.Start.Line == cursorLine {
			hadTokenOnLine = true
			if r, ok := selectParenRange(idx); ok {
				resultSet[[2]int{r.StartLine, r.EndLine}] = struct{}{}
				continue
			}
			if r, ok := selectIndentRange(idx); ok {
				resultSet[[2]int{r.StartLine, r.EndLine}] = struct{}{}
				continue
			}
			resultSet[[2]int{cursorLine, cursorLine}] = struct{}{}
		}
	}

	if !hadTokenOnLine {
		// fall back to block based on indentation if cursor line is empty.
		if r, ok := selectIndentRange(len(tokens) - 1); ok {
			resultSet[[2]int{r.StartLine, r.EndLine}] = struct{}{}
		} else {
			resultSet[[2]int{cursorLine, cursorLine}] = struct{}{}
		}
	}

	if len(resultSet) == 0 {
		resultSet[[2]int{cursorLine, cursorLine}] = struct{}{}
	}

	ranges := make([]Range, 0, len(resultSet))
	for key := range resultSet {
		ranges = append(ranges, Range{StartLine: key[0], EndLine: key[1]})
	}
	sort.Slice(ranges, func(i, j int) bool {
		if ranges[i].StartLine == ranges[j].StartLine {
			return ranges[i].EndLine < ranges[j].EndLine
		}
		return ranges[i].StartLine < ranges[j].StartLine
	})
	return ranges
}

// pythonTokenizer converts Python source text into tokens.
type pythonTokenizer struct {
	src         []rune
	length      int
	index       int
	offset      int
	line        int
	column      int
	tokens      []Token
	indentStack []int
	atLineStart bool
	parenDepth  int
}

func newPythonTokenizer(src []rune) *pythonTokenizer {
	return &pythonTokenizer{
		src:         src,
		length:      len(src),
		indentStack: []int{0},
		atLineStart: true,
	}
}

func (t *pythonTokenizer) scan() ([]Token, error) {
	for t.index < t.length {
		if t.atLineStart {
			t.consumeIndentation()
		}
		if t.index >= t.length {
			break
		}
		r := t.src[t.index]
		switch {
		case r == '\n':
			t.emitStandaloneNewline()
			t.atLineStart = true
		case r == '\r':
			t.emitStandaloneNewline()
			t.atLineStart = true
		case r == '#':
			token := t.scanHashComment()
			t.tokens = append(t.tokens, token)
		case unicode.IsSpace(r):
			start := t.mark()
			for t.index < t.length {
				r = t.src[t.index]
				if r == '\n' || r == '\r' || !unicode.IsSpace(r) {
					break
				}
				t.advance()
			}
			t.tokens = append(t.tokens, Token{Kind: TokenWhitespace, Lexeme: string(t.src[start.Offset:t.index]), Start: start, End: t.location()})
		case isIdentifierStart(r):
			t.tokens = append(t.tokens, t.scanIdentifier())
		case unicode.IsDigit(r):
			t.tokens = append(t.tokens, t.scanNumber())
		case r == '"' || r == '\'':
			t.tokens = append(t.tokens, t.scanString())
		case r == '(':
			t.emitSingle(TokenOpenParen, "(")
			t.parenDepth++
		case r == ')':
			t.emitSingle(TokenCloseParen, ")")
			if t.parenDepth > 0 {
				t.parenDepth--
			}
		case r == '[':
			t.emitSingle(TokenOpenBracket, "[")
			t.parenDepth++
		case r == ']':
			t.emitSingle(TokenCloseBracket, "]")
			if t.parenDepth > 0 {
				t.parenDepth--
			}
		case r == '{':
			t.emitSingle(TokenOpenBrace, "{")
			t.parenDepth++
		case r == '}':
			t.emitSingle(TokenCloseBrace, "}")
			if t.parenDepth > 0 {
				t.parenDepth--
			}
		case r == ':':
			t.emitSingle(TokenColon, ":")
		case r == ',':
			t.emitSingle(TokenComma, ",")
		case r == ';':
			t.emitSingle(TokenSemicolon, ";")
		default:
			t.tokens = append(t.tokens, t.scanOperator())
		}
		if r != '\n' && r != '\r' {
			t.atLineStart = false
		}
	}

	for len(t.indentStack) > 1 {
		t.indentStack = t.indentStack[:len(t.indentStack)-1]
		loc := t.location()
		t.tokens = append(t.tokens, Token{Kind: TokenDedent, Start: loc, End: loc})
	}

	return t.tokens, nil
}

func (t *pythonTokenizer) consumeIndentation() {
	startIdx := t.index
	startLoc := t.mark()
	indentColumns := 0
	spacesConsumed := 0
	tabsConsumed := 0

	for t.index < t.length {
		r := t.src[t.index]
		if r == ' ' {
			indentColumns++
			spacesConsumed++
			t.advance()
		} else if r == '\t' {
			width := 8 - (indentColumns % 8)
			indentColumns += width
			tabsConsumed++
			t.advance()
		} else {
			break
		}
	}

	if t.index >= t.length {
		return
	}

	r := t.src[t.index]
	if r == '\n' || r == '\r' {
		t.atLineStart = true
		return
	}
	if r == '#' {
		return
	}
	if t.parenDepth > 0 {
		return
	}

	current := t.indentStack[len(t.indentStack)-1]
	switch {
	case indentColumns > current:
		t.indentStack = append(t.indentStack, indentColumns)
		lexeme := string(t.src[startIdx : startIdx+spacesConsumed+tabsConsumed])
		t.tokens = append(t.tokens, Token{Kind: TokenIndent, Lexeme: lexeme, Start: startLoc, End: t.location()})
	case indentColumns < current:
		for len(t.indentStack) > 1 && indentColumns < t.indentStack[len(t.indentStack)-1] {
			t.indentStack = t.indentStack[:len(t.indentStack)-1]
			loc := t.location()
			t.tokens = append(t.tokens, Token{Kind: TokenDedent, Start: loc, End: loc})
		}
		if indentColumns > t.indentStack[len(t.indentStack)-1] {
			t.indentStack = append(t.indentStack, indentColumns)
			t.tokens = append(t.tokens, Token{Kind: TokenIndent, Start: startLoc, End: t.location()})
		}
	}
}

func (t *pythonTokenizer) scanIdentifier() Token {
	start := t.mark()
	t.advance()
	for t.index < t.length && isIdentifierPart(t.src[t.index]) {
		t.advance()
	}
	lexeme := string(t.src[start.Offset:t.index])
	kind := TokenIdentifier
	if isPythonKeyword(lexeme) {
		kind = TokenKeyword
	}
	return Token{Kind: kind, Lexeme: lexeme, Start: start, End: t.location()}
}

func (t *pythonTokenizer) scanNumber() Token {
	start := t.mark()
	t.advance()
	for t.index < t.length {
		r := t.src[t.index]
		if unicode.IsDigit(r) || r == '.' || r == '_' || unicode.IsLetter(r) {
			t.advance()
		} else {
			break
		}
	}
	return Token{Kind: TokenNumber, Lexeme: string(t.src[start.Offset:t.index]), Start: start, End: t.location()}
}

func (t *pythonTokenizer) scanString() Token {
	start := t.mark()
	quote := t.src[t.index]
	triple := false
	if t.peek(quote) && t.peekN(2, quote) {
		triple = true
		t.advanceN(3)
	} else {
		t.advance()
	}

	for t.index < t.length {
		r := t.src[t.index]
		if r == '\\' {
			t.advance()
			if t.index < t.length {
				t.advance()
			}
			continue
		}
		if triple {
			if r == quote && t.peek(quote) && t.peekN(2, quote) {
				t.advanceN(3)
				break
			}
			t.advance()
		} else {
			t.advance()
			if r == quote || r == '\n' || r == '\r' {
				break
			}
		}
	}

	return Token{Kind: TokenString, Lexeme: string(t.src[start.Offset:t.index]), Start: start, End: t.location()}
}

func (t *pythonTokenizer) scanOperator() Token {
	start := t.mark()
	t.advance()
	for t.index < t.length {
		r := t.src[t.index]
		if unicode.IsSpace(r) || unicode.IsLetter(r) || unicode.IsDigit(r) || r == '\n' || r == '\r' {
			break
		}
		if r == '#' {
			break
		}
		if strings.ContainsRune("()[]{}:,;", r) {
			break
		}
		t.advance()
	}
	return Token{Kind: TokenOperator, Lexeme: string(t.src[start.Offset:t.index]), Start: start, End: t.location()}
}

func (t *pythonTokenizer) scanHashComment() Token {
	start := t.mark()
	for t.index < t.length {
		r := t.src[t.index]
		if r == '\n' || r == '\r' {
			break
		}
		t.advance()
	}
	return Token{Kind: TokenLineCommentStart, Lexeme: string(t.src[start.Offset:t.index]), Start: start, End: t.location()}
}

func (t *pythonTokenizer) emitSingle(kind TokenKind, lexeme string) {
	start := t.mark()
	t.advance()
	t.tokens = append(t.tokens, Token{Kind: kind, Lexeme: lexeme, Start: start, End: t.location()})
}

func (t *pythonTokenizer) emitStandaloneNewline() {
	start := t.mark()
	lexeme := t.consumeNewline()
	t.tokens = append(t.tokens, Token{Kind: TokenNewline, Lexeme: lexeme, Start: start, End: t.location()})
}

func (t *pythonTokenizer) mark() Location {
	return Location{Offset: t.offset, Line: t.line, Column: t.column}
}

func (t *pythonTokenizer) location() Location {
	return Location{Offset: t.offset, Line: t.line, Column: t.column}
}

func (t *pythonTokenizer) advance() {
	if t.index >= t.length {
		return
	}
	r := t.src[t.index]
	t.index++
	t.offset++
	if r == '\n' {
		t.line++
		t.column = 0
		t.atLineStart = true
	} else {
		t.column++
	}
}

func (t *pythonTokenizer) advanceN(n int) {
	for i := 0; i < n; i++ {
		t.advance()
	}
}

func (t *pythonTokenizer) peek(expected rune) bool {
	return t.index+1 < t.length && t.src[t.index+1] == expected
}

func (t *pythonTokenizer) peekN(n int, expected rune) bool {
	return t.index+n < t.length && t.src[t.index+n] == expected
}

func (t *pythonTokenizer) consumeNewline() string {
	if t.index >= t.length {
		return ""
	}
	r := t.src[t.index]
	t.advance()
	lexeme := string(r)
	if r == '\r' && t.index < t.length && t.src[t.index] == '\n' {
		lexeme += "\n"
		t.advance()
	}
	return lexeme
}

var pythonKeywords = map[string]struct{}{
	"False":    {},
	"None":     {},
	"True":     {},
	"and":      {},
	"as":       {},
	"assert":   {},
	"async":    {},
	"await":    {},
	"break":    {},
	"class":    {},
	"continue": {},
	"def":      {},
	"del":      {},
	"elif":     {},
	"else":     {},
	"except":   {},
	"finally":  {},
	"for":      {},
	"from":     {},
	"global":   {},
	"if":       {},
	"import":   {},
	"in":       {},
	"is":       {},
	"lambda":   {},
	"nonlocal": {},
	"not":      {},
	"or":       {},
	"pass":     {},
	"raise":    {},
	"return":   {},
	"try":      {},
	"while":    {},
	"with":     {},
	"yield":    {},
}

func isPythonKeyword(word string) bool {
	_, ok := pythonKeywords[word]
	return ok
}
