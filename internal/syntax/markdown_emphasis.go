package syntax

import (
	"unicode"
	"unicode/utf8"
	"veda/internal/utils"

	sitter "github.com/smacker/go-tree-sitter"
)

// hasValidMarkdownEmphasisDelimiters checks the CommonMark flanking rules for
// an emphasis node. Tree-sitter has already paired the delimiter runs; this
// rejects pairs whose opening or closing run cannot act as that delimiter.
func hasValidMarkdownEmphasisDelimiters(node *sitter.Node, source []byte) bool {
	start := int(node.StartByte())
	end := int(node.EndByte())
	utils.Assert(start >= 0 && start < end && end <= len(source), "invalid Markdown emphasis bounds")

	delimiter := source[start]
	utils.Assert(delimiter == '*' || delimiter == '_', "invalid Markdown emphasis delimiter")
	utils.Assert(source[end-1] == delimiter, "Markdown emphasis delimiters do not match")

	openStart, openEnd := markdownDelimiterRun(source, start, delimiter)
	closeStart, closeEnd := markdownDelimiterRun(source, end-1, delimiter)
	openLeft, openRight, openBeforePunctuation, _ := markdownFlanking(source, openStart, openEnd)
	closeLeft, closeRight, _, closeAfterPunctuation := markdownFlanking(source, closeStart, closeEnd)

	if delimiter == '*' {
		return openLeft && closeRight
	}

	canOpen := openLeft && (!openRight || openBeforePunctuation)
	canClose := closeRight && (!closeLeft || closeAfterPunctuation)
	return canOpen && canClose
}

func markdownDelimiterRun(source []byte, at int, delimiter byte) (int, int) {
	start := at
	for start > 0 && source[start-1] == delimiter {
		start--
	}
	end := at + 1
	for end < len(source) && source[end] == delimiter {
		end++
	}
	return start, end
}

func markdownFlanking(source []byte, start, end int) (left, right, beforePunctuation, afterPunctuation bool) {
	before, hasBefore := runeBefore(source, start)
	after, hasAfter := runeAfter(source, end)
	beforeWhitespace := !hasBefore || isCommonMarkWhitespace(before)
	afterWhitespace := !hasAfter || isCommonMarkWhitespace(after)
	beforePunctuation = hasBefore && isMarkdownPunctuation(before)
	afterPunctuation = hasAfter && isMarkdownPunctuation(after)
	left = !afterWhitespace && (!afterPunctuation || beforeWhitespace || beforePunctuation)
	right = !beforeWhitespace && (!beforePunctuation || afterWhitespace || afterPunctuation)
	return
}

func runeBefore(source []byte, at int) (rune, bool) {
	if at == 0 {
		return 0, false
	}
	r, _ := utf8.DecodeLastRune(source[:at])
	return r, true
}

func runeAfter(source []byte, at int) (rune, bool) {
	if at == len(source) {
		return 0, false
	}
	r, _ := utf8.DecodeRune(source[at:])
	return r, true
}

func isMarkdownPunctuation(r rune) bool {
	return unicode.IsPunct(r) || unicode.IsSymbol(r)
}

func isCommonMarkWhitespace(r rune) bool {
	switch r {
	case '\t', '\n', '\f', '\r':
		return true
	}
	return unicode.Is(unicode.Zs, r)
}
