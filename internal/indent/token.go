package indent

// Location identifies a position in source text.
type Location struct {
	Offset int // rune offset from start of text
	Line   int // zero-based line number
	Column int // zero-based column (rune index within the line)
}

// TokenKind categorises lexical tokens recognised by the indentation engine.
type TokenKind int

const (
	TokenUnknown TokenKind = iota
	TokenWhitespace
	TokenNewline
	TokenIdentifier
	TokenKeyword
	TokenNumber
	TokenString
	TokenCharLiteral
	TokenOperator
	TokenOpenBrace
	TokenCloseBrace
	TokenOpenParen
	TokenCloseParen
	TokenOpenBracket
	TokenCloseBracket
	TokenIndent
	TokenDedent
	TokenSemicolon
	TokenColon
	TokenComma
	TokenQuestion
	TokenCommentStart
	TokenCommentEnd
	TokenCommentWord
	TokenLineCommentStart
	TokenPreprocessor
)

// Token represents a lexical item along with its origin locations.
type Token struct {
	Kind   TokenKind
	Lexeme string
	Start  Location
	End    Location
}

// Range represents a span of lines (inclusive) that should be re-indented.
type Range struct {
	StartLine int
	EndLine   int
}

// AnnotatedToken stores indentation metadata for a token.
type AnnotatedToken struct {
	Token             Token
	IndentLevel       int
	ContinuationLevel int
}

// Pair associates opening and closing token kinds.
type Pair struct {
	Open  TokenKind
	Close TokenKind
}

// Rules defines indentation semantics for a language.
type Rules struct {
	Anchors       []Pair
	Continuations []Pair
}

// Language exposes tokenisation and indentation rules for a language.
type Language interface {
	Tokenize(text []rune) ([]Token, error)
	Rules() Rules
	ExpressionRanges(tokens []Token, cursorLine, cursorColumn int) []Range
}
