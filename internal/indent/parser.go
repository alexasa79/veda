package indent

import "fmt"

// Analyzer produces indentation annotations for a token stream.
type Analyzer struct {
	tokens []Token
	rules  Rules
}

// NewAnalyzer constructs Analyzer for the provided tokens and language rules.
func NewAnalyzer(tokens []Token, rules Rules) *Analyzer {
	return &Analyzer{
		tokens: tokens,
		rules:  rules,
	}
}

// Analyze computes indentation metadata for each token.
func (a *Analyzer) Analyze() ([]AnnotatedToken, error) {
	anchorPairs := make(map[TokenKind]TokenKind, len(a.rules.Anchors))
	anchorClosers := make(map[TokenKind]TokenKind, len(a.rules.Anchors))
	for _, p := range a.rules.Anchors {
		anchorPairs[p.Open] = p.Close
		anchorClosers[p.Close] = p.Open
	}

	continuationPairs := make(map[TokenKind]TokenKind, len(a.rules.Continuations))
	continuationClosers := make(map[TokenKind]TokenKind, len(a.rules.Continuations))
	for _, p := range a.rules.Continuations {
		continuationPairs[p.Open] = p.Close
		continuationClosers[p.Close] = p.Open
	}

	var anchorStack []TokenKind
	var continuationStack []TokenKind

	annotated := make([]AnnotatedToken, 0, len(a.tokens))

	for _, tok := range a.tokens {
		switch tok.Kind {
		case TokenCloseBrace, TokenCloseParen, TokenCloseBracket:
			if opener, ok := anchorClosers[tok.Kind]; ok {
				if len(anchorStack) == 0 || anchorStack[len(anchorStack)-1] != opener {
					return nil, fmt.Errorf("indent: unmatched closing anchor %v at %v", tok.Kind, tok.Start)
				}
				anchorStack = anchorStack[:len(anchorStack)-1]
			}
			if opener, ok := continuationClosers[tok.Kind]; ok {
				if len(continuationStack) == 0 || continuationStack[len(continuationStack)-1] != opener {
					return nil, fmt.Errorf("indent: unmatched closing continuation %v at %v", tok.Kind, tok.Start)
				}
				continuationStack = continuationStack[:len(continuationStack)-1]
			}
		}

		annotated = append(annotated, AnnotatedToken{
			Token:             tok,
			IndentLevel:       len(anchorStack),
			ContinuationLevel: len(continuationStack),
		})

		switch tok.Kind {
		case TokenOpenBrace, TokenOpenParen, TokenOpenBracket:
			if _, ok := anchorPairs[tok.Kind]; ok {
				anchorStack = append(anchorStack, tok.Kind)
			}
			if _, ok := continuationPairs[tok.Kind]; ok {
				continuationStack = append(continuationStack, tok.Kind)
			}
		}
	}

	if len(anchorStack) > 0 {
		return nil, fmt.Errorf("indent: unmatched opening anchor %v", anchorStack[len(anchorStack)-1])
	}
	if len(continuationStack) > 0 {
		return nil, fmt.Errorf("indent: unmatched opening continuation %v", continuationStack[len(continuationStack)-1])
	}

	return annotated, nil
}
