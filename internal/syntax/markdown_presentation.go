package syntax

import (
	"sort"

	sitter "github.com/smacker/go-tree-sitter"
)

// ConcealRange is a half-open, line-relative byte range containing Markdown
// source syntax that is omitted from the rendered presentation.
type ConcealRange struct {
	Start int
	End   int
}

type MarkdownImage struct {
	Alt         string
	Destination string
}

// MarkdownImageForLine returns an image only when it is the line's sole
// non-whitespace Markdown element.
func (h *Highlighter) MarkdownImageForLine(lineStart, lineEnd int) (MarkdownImage, bool) {
	if h.currentLang != "markdown" || h.currentTree == nil || h.currentSource == nil ||
		lineStart < 0 || lineEnd < lineStart || lineEnd > len(h.currentSource) {
		return MarkdownImage{}, false
	}

	contentStart := lineStart
	for contentStart < lineEnd && (h.currentSource[contentStart] == ' ' || h.currentSource[contentStart] == '\t') {
		contentStart++
	}
	contentEnd := lineEnd
	for contentEnd > contentStart && (h.currentSource[contentEnd-1] == ' ' || h.currentSource[contentEnd-1] == '\t') {
		contentEnd--
	}

	var imageNode *sitter.Node
	var findImage func(*sitter.Node)
	findImage = func(node *sitter.Node) {
		if imageNode != nil || int(node.EndByte()) <= contentStart || int(node.StartByte()) >= contentEnd {
			return
		}
		if node.Type() == "image" && int(node.StartByte()) == contentStart && int(node.EndByte()) == contentEnd {
			imageNode = node
			return
		}
		for i := 0; i < int(node.ChildCount()); i++ {
			findImage(node.Child(i))
		}
	}
	findImage(h.currentTree.RootNode())
	if imageNode == nil {
		return MarkdownImage{}, false
	}

	var image MarkdownImage
	var readParts func(*sitter.Node)
	readParts = func(node *sitter.Node) {
		start := int(node.StartByte())
		end := int(node.EndByte())
		if start >= 0 && end <= len(h.currentSource) {
			switch node.Type() {
			case "link_text", "image_description":
				image.Alt = string(h.currentSource[start:end])
			case "link_destination":
				image.Destination = string(h.currentSource[start:end])
			}
		}
		for i := 0; i < int(node.ChildCount()); i++ {
			readParts(node.Child(i))
		}
	}
	readParts(imageNode)
	return image, image.Destination != ""
}

// MarkdownConcealRangesForLine returns the Markdown syntax ranges that should
// not occupy screen cells in a rendered view. The returned ranges are sorted,
// merged, and relative to lineStart.
func (h *Highlighter) MarkdownConcealRangesForLine(lineStart, lineEnd int) []ConcealRange {
	if h.currentLang != "markdown" || h.currentTree == nil || lineStart < 0 || lineEnd < lineStart {
		return nil
	}

	var ranges []ConcealRange
	addNode := func(node *sitter.Node) {
		start := int(node.StartByte())
		end := int(node.EndByte())
		if start < lineStart {
			start = lineStart
		}
		if end > lineEnd {
			end = lineEnd
		}
		if start < end {
			ranges = append(ranges, ConcealRange{Start: start - lineStart, End: end - lineStart})
		}
	}

	var walk func(*sitter.Node)
	walk = func(node *sitter.Node) {
		if int(node.EndByte()) <= lineStart || int(node.StartByte()) >= lineEnd {
			return
		}

		switch node.Type() {
		case "emphasis_delimiter", "code_span_delimiter", "link_destination", "link_title":
			addNode(node)
		case "[", "]", "(", ")", "!":
			if hasMarkdownLinkAncestor(node) && !h.isMarkdownTaskCheckbox(node) {
				addNode(node)
			}
		}

		for i := 0; i < int(node.ChildCount()); i++ {
			walk(node.Child(i))
		}
	}
	walk(h.currentTree.RootNode())

	line := h.currentSource[lineStart:lineEnd]
	for _, token := range h.GetTokensForLine(lineStart, lineEnd) {
		if token.Start != 0 {
			continue
		}
		switch token.TokenType {
		case TokenTypeMarkdownH1, TokenTypeMarkdownH2, TokenTypeMarkdownH3,
			TokenTypeMarkdownH4, TokenTypeMarkdownH5, TokenTypeMarkdownH6:
			end := markdownHeadingPrefixEnd(line)
			if end > 0 {
				ranges = append(ranges, ConcealRange{Start: 0, End: end})
			}
		case TokenTypeMarkdownBlockquote:
			end := markdownBlockquotePrefixEnd(line)
			if end > 0 {
				ranges = append(ranges, ConcealRange{Start: 0, End: end})
			}
		}
	}

	return mergeConcealRanges(ranges)
}

func (h *Highlighter) isMarkdownTaskCheckbox(node *sitter.Node) bool {
	for parent := node.Parent(); parent != nil; parent = parent.Parent() {
		if parent.Type() != "shortcut_link" {
			continue
		}
		start := int(parent.StartByte())
		end := int(parent.EndByte())
		if start < 0 || end > len(h.currentSource) {
			return false
		}
		text := string(h.currentSource[start:end])
		return text == "[ ]" || text == "[x]" || text == "[X]"
	}
	return false
}

func hasMarkdownLinkAncestor(node *sitter.Node) bool {
	for parent := node.Parent(); parent != nil; parent = parent.Parent() {
		switch parent.Type() {
		case "inline_link", "shortcut_link", "full_reference_link", "collapsed_reference_link", "image":
			return true
		}
	}
	return false
}

func markdownHeadingPrefixEnd(line []byte) int {
	i := 0
	for i < len(line) && i < 6 && line[i] == '#' {
		i++
	}
	if i == 0 || i >= len(line) || (line[i] != ' ' && line[i] != '\t') {
		return 0
	}
	for i < len(line) && (line[i] == ' ' || line[i] == '\t') {
		i++
	}
	return i
}

func markdownBlockquotePrefixEnd(line []byte) int {
	i := 0
	for i < len(line) && line[i] == '>' {
		i++
		if i < len(line) && line[i] == ' ' {
			i++
		}
	}
	return i
}

func mergeConcealRanges(ranges []ConcealRange) []ConcealRange {
	if len(ranges) == 0 {
		return nil
	}
	sort.Slice(ranges, func(i, j int) bool {
		if ranges[i].Start == ranges[j].Start {
			return ranges[i].End < ranges[j].End
		}
		return ranges[i].Start < ranges[j].Start
	})

	merged := ranges[:1]
	for _, current := range ranges[1:] {
		last := &merged[len(merged)-1]
		if current.Start <= last.End {
			if current.End > last.End {
				last.End = current.End
			}
			continue
		}
		merged = append(merged, current)
	}
	return merged
}
