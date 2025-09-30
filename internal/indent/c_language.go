package indent

import (
	"fmt"
	"sort"
	"unicode"
)

// CLanguage implements Language for C-like syntax.
type CLanguage struct{}

// NewCLanguage constructs a C-like language specification.
func NewCLanguage() *CLanguage {
	return &CLanguage{}
}

// Rules returns anchor and continuation pairing suitable for C-like languages.
func (c *CLanguage) Rules() Rules {
	return Rules{
		Anchors: []Pair{{Open: TokenOpenBrace, Close: TokenCloseBrace}},
		Continuations: []Pair{
			{Open: TokenOpenParen, Close: TokenCloseParen},
			{Open: TokenOpenBracket, Close: TokenCloseBracket},
		},
	}
}

// Tokenize produces tokens for a C-like language.
func (c *CLanguage) Tokenize(src []rune) ([]Token, error) {
	tk := newTokenizer(src)
	return tk.scan()
}

func (c *CLanguage) ExpressionRanges(tokens []Token, cursorLine, cursorColumn int) []Range {
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

func isSkippableToken(kind TokenKind) bool {
	switch kind {
	case TokenWhitespace, TokenNewline, TokenCommentStart, TokenCommentEnd, TokenCommentWord, TokenLineCommentStart:
		return true
	default:
		return false
	}
}

func isBoundaryToken(kind TokenKind) bool {
	switch kind {
	case TokenSemicolon, TokenKeyword, TokenOpenBrace, TokenCloseBrace, TokenOpenParen, TokenCloseParen, TokenOpenBracket, TokenCloseBracket, TokenPreprocessor:
		return true
	default:
		return false
	}
}

func startAfterBoundary(tok Token) int {
	line := tok.Start.Line + 1
	if line < 0 {
		return 0
	}
	return line
}

func endBeforeBoundary(tok Token) int {
	switch tok.Kind {
	case TokenSemicolon, TokenCloseBrace, TokenCloseParen, TokenCloseBracket:
		return tok.Start.Line
	default:
		return tok.Start.Line - 1
	}
}

func locationBeforeLocation(loc Location, line, column int) bool {
	if loc.Line != line {
		return loc.Line < line
	}
	return loc.Column < column
}

func locationBeforeCursor(line, column int, loc Location) bool {
	if line != loc.Line {
		return line < loc.Line
	}
	return column < loc.Column
}

func locationEqual(loc Location, line, column int) bool {
	return loc.Line == line && loc.Column == column
}

func locationEqualLoc(line, column int, loc Location) bool {
	return line == loc.Line && column == loc.Column
}

func locationEqualPos(line1, column1 int, line2, column2 int) bool {
	return line1 == line2 && column1 == column2
}

func prefersKind(candidate, current TokenKind) bool {
	if current == TokenUnknown {
		return true
	}
	if candidate == TokenOpenBrace && current != TokenOpenBrace {
		return true
	}
	return false
}

type tokenizer struct {
	src    []rune
	length int
	index  int
	offset int
	line   int
	column int
	tokens []Token
}

func newTokenizer(src []rune) *tokenizer {
	return &tokenizer{
		src:    src,
		length: len(src),
	}
}

func (t *tokenizer) scan() ([]Token, error) {
	for t.index < t.length {
		r := t.src[t.index]
		switch {
		case r == '\n':
			t.emitStandaloneNewline()
		case r == '\r':
			// Handle Windows newline \r\n
			t.emitStandaloneNewline()
		case unicode.IsSpace(r):
			start := t.mark()
			for t.index < t.length && unicode.IsSpace(t.src[t.index]) && t.src[t.index] != '\n' && t.src[t.index] != '\r' {
				t.advance()
			}
			end := t.location()
			lexeme := string(t.src[start.Offset:t.index])
			t.tokens = append(t.tokens, Token{Kind: TokenWhitespace, Lexeme: lexeme, Start: start, End: end})
		case r == '#':
			t.tokens = append(t.tokens, t.scanPreprocessor())
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
		case r == '?':
			t.emitSingle(TokenQuestion, "?")
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

func (t *tokenizer) scanBlockComment() ([]Token, error) {
	startLoc := t.mark()
	// emit comment start token
	t.advanceN(2) // consume /*
	commentStart := Token{
		Kind:   TokenCommentStart,
		Lexeme: "/*",
		Start:  startLoc,
		End:    t.location(),
	}

	var tokens = []Token{commentStart}

	var wordStart *Location

	flushWord := func(end Location) {
		if wordStart == nil {
			return
		}
		lexeme := string(t.src[wordStart.Offset:end.Offset])
		if len(lexeme) > 0 {
			tokens = append(tokens, Token{
				Kind:   TokenCommentWord,
				Lexeme: lexeme,
				Start:  *wordStart,
				End:    end,
			})
		}
		wordStart = nil
	}

	for t.index < t.length {
		r := t.src[t.index]
		if r == '*' && t.peek('/') {
			endLoc := t.location()
			t.advanceN(2)
			flushWord(endLoc)
			commentEnd := Token{
				Kind:   TokenCommentEnd,
				Lexeme: "*/",
				Start:  endLoc,
				End:    t.location(),
			}
			tokens = append(tokens, commentEnd)
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

func (t *tokenizer) scanLineComment() []Token {
	startLoc := t.mark()
	t.advanceN(2)
	start := Token{
		Kind:   TokenLineCommentStart,
		Lexeme: "//",
		Start:  startLoc,
		End:    t.location(),
	}

	var tokens = []Token{start}
	var wordStart *Location

	flush := func(end Location) {
		if wordStart == nil {
			return
		}
		lexeme := string(t.src[wordStart.Offset:end.Offset])
		if len(lexeme) > 0 {
			tokens = append(tokens, Token{
				Kind:   TokenCommentWord,
				Lexeme: lexeme,
				Start:  *wordStart,
				End:    end,
			})
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

func (t *tokenizer) scanIdentifier() Token {
	start := t.mark()
	t.advance()
	for t.index < t.length && isIdentifierPart(t.src[t.index]) {
		t.advance()
	}
	end := t.location()
	lexeme := string(t.src[start.Offset:t.index])
	kind := TokenIdentifier
	if isKeyword(lexeme) {
		kind = TokenKeyword
	}
	return Token{Kind: kind, Lexeme: lexeme, Start: start, End: end}
}

func (t *tokenizer) scanNumber() Token {
	start := t.mark()
	t.advance()
	for t.index < t.length {
		r := t.src[t.index]
		if unicode.IsDigit(r) || r == '.' || r == 'x' || r == 'X' || unicode.IsLetter(r) {
			t.advance()
		} else {
			break
		}
	}
	return Token{Kind: TokenNumber, Lexeme: string(t.src[start.Offset:t.index]), Start: start, End: t.location()}
}

func (t *tokenizer) scanString(kind TokenKind) Token {
	start := t.mark()
	terminator := t.src[t.index]
	t.advance() // consume opening quote
	for t.index < t.length {
		r := t.src[t.index]
		t.advance()
		if r == '\\' && t.index < t.length {
			// skip escaped char
			t.advance()
			continue
		}
		if r == terminator {
			break
		}
	}
	return Token{Kind: kind, Lexeme: string(t.src[start.Offset:t.index]), Start: start, End: t.location()}
}

func (t *tokenizer) scanOperator() Token {
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
		// Stop before structural tokens that should be separate
		if r == '(' || r == ')' || r == '[' || r == ']' || r == '{' || r == '}' || r == ';' || r == ':' || r == ',' {
			break
		}
		t.advance()
	}
	return Token{Kind: TokenOperator, Lexeme: string(t.src[start.Offset:t.index]), Start: start, End: t.location()}
}

func (t *tokenizer) scanPreprocessor() Token {
	start := t.mark()
	t.advance() // consume '#'
	lastNonSpace := '#'
	for t.index < t.length {
		r := t.src[t.index]
		if r == '\n' || r == '\r' {
			if lastNonSpace == '\\' {
				// Line continuation, consume newline sequence and continue scanning.
				t.consumeNewline()
				// Skip leading whitespace on the continued line to keep directive content compact.
				for t.index < t.length {
					if t.src[t.index] == '\n' || t.src[t.index] == '\r' {
						break
					}
					if !unicode.IsSpace(t.src[t.index]) {
						break
					}
					t.advance()
				}
				lastNonSpace = 0
				continue
			}
			break
		}
		t.advance()
		if !unicode.IsSpace(r) {
			lastNonSpace = r
		}
	}
	return Token{Kind: TokenPreprocessor, Lexeme: string(t.src[start.Offset:t.index]), Start: start, End: t.location()}
}

func (t *tokenizer) emitSingle(kind TokenKind, lexeme string) {
	start := t.mark()
	t.advance()
	t.tokens = append(t.tokens, Token{Kind: kind, Lexeme: lexeme, Start: start, End: t.location()})
}

func (t *tokenizer) emitNewlineToken(tokens *[]Token) {
	start := t.mark()
	lexeme := t.consumeNewline()
	*tokens = append(*tokens, Token{Kind: TokenNewline, Lexeme: lexeme, Start: start, End: t.location()})
}

func (t *tokenizer) emitStandaloneNewline() {
	start := t.mark()
	lexeme := t.consumeNewline()
	t.tokens = append(t.tokens, Token{Kind: TokenNewline, Lexeme: lexeme, Start: start, End: t.location()})
}

func (t *tokenizer) consumeNewline() string {
	if t.index >= t.length {
		return ""
	}
	r := t.src[t.index]
	t.advance()
	lexeme := string(r)
	lineAdvanced := r == '\n'
	if r == '\r' && t.index < t.length && t.src[t.index] == '\n' {
		lexeme += "\n"
		t.advance()
		lineAdvanced = true
	}
	if !lineAdvanced {
		// Treat solitary carriage return as newline
		t.line++
		t.column = 0
	}
	return lexeme
}

func (t *tokenizer) mark() Location {
	return Location{Offset: t.offset, Line: t.line, Column: t.column}
}

func (t *tokenizer) location() Location {
	return Location{Offset: t.offset, Line: t.line, Column: t.column}
}

func (t *tokenizer) advance() {
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

func (t *tokenizer) advanceN(n int) {
	for i := 0; i < n; i++ {
		t.advance()
	}
}

func (t *tokenizer) peek(expected rune) bool {
	return t.index+1 < t.length && t.src[t.index+1] == expected
}

func isIdentifierStart(r rune) bool {
	return unicode.IsLetter(r) || r == '_'
}

func isIdentifierPart(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_'
}

var cKeywords = map[string]struct{}{
	"auto":     {},
	"break":    {},
	"case":     {},
	"char":     {},
	"const":    {},
	"continue": {},
	"default":  {},
	"do":       {},
	"double":   {},
	"else":     {},
	"enum":     {},
	"extern":   {},
	"float":    {},
	"for":      {},
	"goto":     {},
	"if":       {},
	"inline":   {},
	"int":      {},
	"long":     {},
	"register": {},
	"restrict": {},
	"return":   {},
	"short":    {},
	"signed":   {},
	"sizeof":   {},
	"static":   {},
	"struct":   {},
	"switch":   {},
	"typedef":  {},
	"union":    {},
	"unsigned": {},
	"void":     {},
	"volatile": {},
	"while":    {},
}

func isKeyword(word string) bool {
	_, ok := cKeywords[word]
	return ok
}
