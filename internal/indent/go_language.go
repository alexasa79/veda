package indent

import (
	"fmt"
	"sort"
	"strings"
	"unicode"
)

// GoLanguage implements Language for Go syntax.
type GoLanguage struct{}

// NewGoLanguage constructs a Go language specification.
func NewGoLanguage() *GoLanguage {
	return &GoLanguage{}
}

// Rules returns anchor and continuation pairing suitable for Go.
func (g *GoLanguage) Rules() Rules {
	return Rules{
		Anchors: []Pair{{Open: TokenOpenBrace, Close: TokenCloseBrace}},
		Continuations: []Pair{
			{Open: TokenOpenParen, Close: TokenCloseParen},
			{Open: TokenOpenBracket, Close: TokenCloseBracket},
		},
	}
}

// Tokenize produces tokens for Go source code.
func (g *GoLanguage) Tokenize(src []rune) ([]Token, error) {
	tk := newGoTokenizer(src)
	return tk.scan()
}

func (g *GoLanguage) ExpressionRanges(tokens []Token, cursorLine, cursorColumn int) []Range {
	type pair struct {
		openIdx  int
		closeIdx int
	}

	var stack []pair
	var pairs []pair
	for idx, tok := range tokens {
		switch tok.Kind {
		case TokenOpenBrace, TokenOpenParen, TokenOpenBracket:
			stack = append(stack, pair{openIdx: idx})
		case TokenCloseBrace, TokenCloseParen, TokenCloseBracket:
			if len(stack) == 0 {
				continue
			}
			frame := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			frame.closeIdx = idx
			pairs = append(pairs, frame)
		}
	}

	add := func(set map[[2]int]struct{}, start, end int) {
		if start < 0 {
			start = 0
		}
		if end < start {
			end = start
		}
		set[[2]int{start, end}] = struct{}{}
	}

	rangeFromPairs := func(tokenIdx int) (start, end int, ok bool) {
		bestSpan := 1 << 30
		bestKind := TokenUnknown
		for _, pr := range pairs {
			if pr.openIdx <= tokenIdx && tokenIdx <= pr.closeIdx {
				op := tokens[pr.openIdx]
				cl := tokens[pr.closeIdx]
				if op.Start.Line > cursorLine || cl.Start.Line < cursorLine {
					continue
				}
				span := cl.Start.Line - op.Start.Line
				if span < 0 {
					span = 0
				}
				kind := tokens[pr.openIdx].Kind
				if span < bestSpan || (span == bestSpan && prefersKind(kind, bestKind)) {
					bestSpan = span
					bestKind = kind
					start = op.Start.Line
					end = cl.Start.Line
					ok = true
				}
			}
		}
		return
	}

	boundaryRangeForToken := func(tokenIdx int) (start, end int) {
		current := tokens[tokenIdx]
		start = current.Start.Line
		end = current.Start.Line

		for i := tokenIdx - 1; i >= 0; i-- {
			tok := tokens[i]
			if isSkippableToken(tok.Kind) {
				continue
			}
			if tok.Start.Line == current.Start.Line {
				start = tok.Start.Line
				continue
			}
			if isBoundaryToken(tok.Kind) {
				candidate := startAfterBoundary(tok)
				if candidate > start {
					candidate = start
				}
				start = candidate
				break
			}
			start = tok.Start.Line
		}

		for i := tokenIdx + 1; i < len(tokens); i++ {
			tok := tokens[i]
			if isSkippableToken(tok.Kind) {
				continue
			}
			if tok.Start.Line == current.Start.Line {
				end = tok.Start.Line
				continue
			}
			if isBoundaryToken(tok.Kind) {
				candidate := endBeforeBoundary(tok)
				if candidate < start {
					candidate = start
				}
				end = candidate
				break
			}
			end = tok.Start.Line
		}

		if end < start {
			end = start
		}
		return
	}

	selectForToken := func(tokenIdx int) (start, end int, ok bool) {
		boundaryStart, boundaryEnd := boundaryRangeForToken(tokenIdx)
		if pairStart, pairEnd, okPairs := rangeFromPairs(tokenIdx); okPairs {
			if pairStart >= boundaryStart && pairEnd <= boundaryEnd {
				return pairStart, pairEnd, true
			}
		}
		return boundaryStart, boundaryEnd, true
	}

	results := make(map[[2]int]struct{})
	hadTokenOnLine := false
	for idx, tok := range tokens {
		if tok.Start.Line > cursorLine {
			break
		}
		if tok.Start.Line == cursorLine {
			hadTokenOnLine = true
			if start, end, ok := selectForToken(idx); ok {
				add(results, start, end)
			}
		}
	}

	if !hadTokenOnLine {
		bestSpan := 1 << 30
		bestKind := TokenUnknown
		best := struct{ start, end int }{cursorLine, cursorLine}
		found := false
		for _, pr := range pairs {
			op := tokens[pr.openIdx]
			cl := tokens[pr.closeIdx]
			if op.Start.Line <= cursorLine && cursorLine <= cl.Start.Line {
				span := cl.Start.Line - op.Start.Line
				if span < 0 {
					span = 0
				}
				kind := tokens[pr.openIdx].Kind
				if !found || span < bestSpan || (span == bestSpan && prefersKind(kind, bestKind)) {
					bestSpan = span
					bestKind = kind
					best = struct{ start, end int }{op.Start.Line, cl.Start.Line}
					found = true
				}
			}
		}
		if found {
			add(results, best.start, best.end)
		} else {
			add(results, cursorLine, cursorLine)
		}
	}

	if len(results) == 0 {
		add(results, cursorLine, cursorLine)
	}

	ranges := make([]Range, 0, len(results))
	for key := range results {
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

// Tokenizer implementation (adapted from c_language).
type goTokenizer struct {
	src    []rune
	length int
	index  int
	offset int
	line   int
	column int
	tokens []Token
}

func newGoTokenizer(src []rune) *goTokenizer {
	return &goTokenizer{src: src, length: len(src)}
}

func (t *goTokenizer) scan() ([]Token, error) {
	for t.index < t.length {
		r := t.src[t.index]
		switch {
		case r == '\n':
			t.emitStandaloneNewline()
		case r == '\r':
			t.emitStandaloneNewline()
		case unicode.IsSpace(r):
			start := t.mark()
			for t.index < t.length && unicode.IsSpace(t.src[t.index]) && t.src[t.index] != '\n' && t.src[t.index] != '\r' {
				t.advance()
			}
			t.tokens = append(t.tokens, Token{Kind: TokenWhitespace, Lexeme: string(t.src[start.Offset:t.index]), Start: start, End: t.location()})
		case r == '/' && t.peek('*'):
			tokens, err := t.scanBlockComment()
			if err != nil {
				return nil, err
			}
			t.tokens = append(t.tokens, tokens...)
		case r == '/' && t.peek('/'):
			tokens := t.scanLineComment()
			t.tokens = append(t.tokens, tokens...)
		case isIdentifierStart(r):
			t.tokens = append(t.tokens, t.scanIdentifier())
		case unicode.IsDigit(r):
			t.tokens = append(t.tokens, t.scanNumber())
		case r == '"':
			t.tokens = append(t.tokens, t.scanString(TokenString))
		case r == '\'':
			t.tokens = append(t.tokens, t.scanString(TokenCharLiteral))
		case r == '`':
			t.tokens = append(t.tokens, t.scanRawString())
		case r == '{':
			t.emitSingle(TokenOpenBrace, "{")
		case r == '}':
			t.emitSingle(TokenCloseBrace, "}")
		case r == '(':
			t.emitSingle(TokenOpenParen, "(")
		case r == ')':
			t.emitSingle(TokenCloseParen, ")")
		case r == '[':
			t.emitSingle(TokenOpenBracket, "[")
		case r == ']':
			t.emitSingle(TokenCloseBracket, "]")
		case r == ';':
			t.emitSingle(TokenSemicolon, ";")
		case r == ':':
			t.emitSingle(TokenColon, ":")
		case r == ',':
			t.emitSingle(TokenComma, ",")
		default:
			t.tokens = append(t.tokens, t.scanOperator())
		}
	}
	return t.tokens, nil
}

func (t *goTokenizer) scanBlockComment() ([]Token, error) {
	startLoc := t.mark()
	t.advanceN(2)
	commentStart := Token{Kind: TokenCommentStart, Lexeme: "/*", Start: startLoc, End: t.location()}

	var tokens = []Token{commentStart}
	var wordStart *Location

	flushWord := func(end Location) {
		if wordStart == nil {
			return
		}
		lexeme := string(t.src[wordStart.Offset:end.Offset])
		if len(lexeme) > 0 {
			tokens = append(tokens, Token{Kind: TokenCommentWord, Lexeme: lexeme, Start: *wordStart, End: end})
		}
		wordStart = nil
	}

	for t.index < t.length {
		r := t.src[t.index]
		if r == '*' && t.peek('/') {
			endLoc := t.location()
			t.advanceN(2)
			flushWord(endLoc)
			tokens = append(tokens, Token{Kind: TokenCommentEnd, Lexeme: "*/", Start: endLoc, End: t.location()})
			return tokens, nil
		}
		if r == '\n' {
			endLoc := t.location()
			flushWord(endLoc)
			t.emitNewlineToken(&tokens)
			continue
		}
		if unicode.IsSpace(r) {
			endLoc := t.location()
			flushWord(endLoc)
			t.advance()
			continue
		}
		if wordStart == nil {
			loc := t.location()
			wordStart = &loc
		}
		t.advance()
	}
	return nil, fmt.Errorf("unterminated comment starting at line %d column %d", startLoc.Line, startLoc.Column)
}

func (t *goTokenizer) scanLineComment() []Token {
	startLoc := t.mark()
	t.advanceN(2)
	start := Token{Kind: TokenLineCommentStart, Lexeme: "//", Start: startLoc, End: t.location()}

	var tokens = []Token{start}
	var wordStart *Location

	flush := func(end Location) {
		if wordStart == nil {
			return
		}
		lexeme := string(t.src[wordStart.Offset:end.Offset])
		if len(lexeme) > 0 {
			tokens = append(tokens, Token{Kind: TokenCommentWord, Lexeme: lexeme, Start: *wordStart, End: end})
		}
		wordStart = nil
	}

	for t.index < t.length {
		r := t.src[t.index]
		if r == '\n' || r == '\r' {
			endLoc := t.location()
			flush(endLoc)
			return tokens
		}
		if unicode.IsSpace(r) {
			endLoc := t.location()
			flush(endLoc)
			t.advance()
			continue
		}
		if wordStart == nil {
			loc := t.location()
			wordStart = &loc
		}
		t.advance()
	}
	flush(t.location())
	return tokens
}

func (t *goTokenizer) scanIdentifier() Token {
	start := t.mark()
	t.advance()
	for t.index < t.length && isIdentifierPart(t.src[t.index]) {
		t.advance()
	}
	lexeme := string(t.src[start.Offset:t.index])
	kind := TokenIdentifier
	if isGoKeyword(lexeme) {
		kind = TokenKeyword
	}
	return Token{Kind: kind, Lexeme: lexeme, Start: start, End: t.location()}
}

func (t *goTokenizer) scanNumber() Token {
	start := t.mark()
	t.advance()
	for t.index < t.length {
		r := t.src[t.index]
		if unicode.IsDigit(r) || r == '.' || r == 'x' || r == 'X' || r == '_' || unicode.IsLetter(r) {
			t.advance()
		} else {
			break
		}
	}
	return Token{Kind: TokenNumber, Lexeme: string(t.src[start.Offset:t.index]), Start: start, End: t.location()}
}

func (t *goTokenizer) scanString(kind TokenKind) Token {
	start := t.mark()
	terminator := t.src[t.index]
	t.advance()
	for t.index < t.length {
		r := t.src[t.index]
		t.advance()
		if r == '\\' && t.index < t.length {
			t.advance()
			continue
		}
		if r == terminator {
			break
		}
	}
	return Token{Kind: kind, Lexeme: string(t.src[start.Offset:t.index]), Start: start, End: t.location()}
}

func (t *goTokenizer) scanRawString() Token {
	start := t.mark()
	t.advance() // consume opening backtick
	for t.index < t.length {
		r := t.src[t.index]
		t.advance()
		if r == '`' {
			break
		}
	}
	return Token{Kind: TokenString, Lexeme: string(t.src[start.Offset:t.index]), Start: start, End: t.location()}
}

func (t *goTokenizer) scanOperator() Token {
	start := t.mark()
	t.advance()
	for t.index < t.length {
		r := t.src[t.index]
		if unicode.IsSpace(r) || unicode.IsLetter(r) || unicode.IsDigit(r) || r == '\n' || r == '\r' {
			break
		}
		if r == '/' && (t.peek('*') || t.peek('/')) {
			break
		}
		if strings.ContainsRune("()[]{}:,;`,", r) {
			break
		}
		t.advance()
	}
	return Token{Kind: TokenOperator, Lexeme: string(t.src[start.Offset:t.index]), Start: start, End: t.location()}
}

func (t *goTokenizer) emitSingle(kind TokenKind, lexeme string) {
	start := t.mark()
	t.advance()
	t.tokens = append(t.tokens, Token{Kind: kind, Lexeme: lexeme, Start: start, End: t.location()})
}

func (t *goTokenizer) emitNewlineToken(tokens *[]Token) {
	start := t.mark()
	lexeme := t.consumeNewline()
	*tokens = append(*tokens, Token{Kind: TokenNewline, Lexeme: lexeme, Start: start, End: t.location()})
}

func (t *goTokenizer) emitStandaloneNewline() {
	start := t.mark()
	lexeme := t.consumeNewline()
	t.tokens = append(t.tokens, Token{Kind: TokenNewline, Lexeme: lexeme, Start: start, End: t.location()})
}

func (t *goTokenizer) consumeNewline() string {
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

func (t *goTokenizer) mark() Location {
	return Location{Offset: t.offset, Line: t.line, Column: t.column}
}

func (t *goTokenizer) location() Location {
	return Location{Offset: t.offset, Line: t.line, Column: t.column}
}

func (t *goTokenizer) advance() {
	if t.index >= t.length {
		return
	}
	r := t.src[t.index]
	t.index++
	t.offset++
	if r == '\n' {
		t.line++
		t.column = 0
	} else {
		t.column++
	}
}

func (t *goTokenizer) advanceN(n int) {
	for i := 0; i < n; i++ {
		t.advance()
	}
}

func (t *goTokenizer) peek(expected rune) bool {
	return t.index+1 < t.length && t.src[t.index+1] == expected
}

var goKeywords = map[string]struct{}{
	"break":       {},
	"case":        {},
	"chan":        {},
	"const":       {},
	"continue":    {},
	"default":     {},
	"defer":       {},
	"else":        {},
	"fallthrough": {},
	"for":         {},
	"func":        {},
	"go":          {},
	"goto":        {},
	"if":          {},
	"import":      {},
	"interface":   {},
	"map":         {},
	"package":     {},
	"range":       {},
	"return":      {},
	"select":      {},
	"struct":      {},
	"switch":      {},
	"type":        {},
	"var":         {},
}

func isGoKeyword(word string) bool {
	_, ok := goKeywords[word]
	return ok
}
