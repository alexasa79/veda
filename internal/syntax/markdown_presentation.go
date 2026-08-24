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

type markdownLinePresentation struct {
	concealRanges []ConcealRange
	image         MarkdownImage
	hasImage      bool
}

type markdownPresentationIndex struct {
	lines      map[int]markdownLinePresentation
	lineStarts []int
	lineEnds   []int
}

// MarkdownImageForLine returns an image only when it is the line's sole
// non-whitespace Markdown element.
func (h *Highlighter) MarkdownImageForLine(lineStart, lineEnd int) (MarkdownImage, bool) {
	presentation, ok := h.markdownPresentationForLine(lineStart, lineEnd)
	if !ok {
		return MarkdownImage{}, false
	}
	return presentation.image, presentation.hasImage
}

// MarkdownConcealRangesForLine returns the Markdown syntax ranges that should
// not occupy screen cells in a rendered view. The returned ranges are sorted,
// merged, and relative to lineStart.
func (h *Highlighter) MarkdownConcealRangesForLine(lineStart, lineEnd int) []ConcealRange {
	presentation, ok := h.markdownPresentationForLine(lineStart, lineEnd)
	if !ok {
		return nil
	}
	return presentation.concealRanges
}

func (h *Highlighter) markdownPresentationForLine(lineStart, lineEnd int) (markdownLinePresentation, bool) {
	if h.currentLang != "markdown" || h.currentTree == nil || h.currentSource == nil ||
		lineStart < 0 || lineEnd < lineStart || lineEnd > len(h.currentSource) {
		return markdownLinePresentation{}, false
	}
	h.ensureMarkdownPresentationIndex()
	return h.markdownPresentationIndex.lines[lineStart], true
}

func (h *Highlighter) ensureMarkdownPresentationIndex() {
	if h.markdownPresentationIndex != nil {
		return
	}

	index := &markdownPresentationIndex{
		lines:      make(map[int]markdownLinePresentation),
		lineStarts: []int{0},
	}
	for i, b := range h.currentSource {
		if b == '\n' {
			index.lineStarts = append(index.lineStarts, i+1)
		}
	}
	index.lineEnds = make([]int, len(index.lineStarts))
	for i := range index.lineStarts {
		if i+1 < len(index.lineStarts) {
			index.lineEnds[i] = index.lineStarts[i+1] - 1
		} else {
			index.lineEnds[i] = len(h.currentSource)
		}
	}

	addRange := func(start, end int) {
		if start >= end {
			return
		}
		firstLine := index.lineIndexAt(start)
		lastLine := index.lineIndexAt(end - 1)
		for line := firstLine; line <= lastLine; line++ {
			lineStart := index.lineStarts[line]
			lineEnd := index.lineEnds[line]
			clippedStart := start
			if clippedStart < lineStart {
				clippedStart = lineStart
			}
			clippedEnd := end
			if clippedEnd > lineEnd {
				clippedEnd = lineEnd
			}
			if clippedStart < clippedEnd {
				presentation := index.lines[lineStart]
				presentation.concealRanges = append(presentation.concealRanges, ConcealRange{
					Start: clippedStart - lineStart,
					End:   clippedEnd - lineStart,
				})
				index.lines[lineStart] = presentation
			}
		}
	}

	var walk func(*sitter.Node)
	walk = func(node *sitter.Node) {
		nodeType := node.Type()
		if nodeType == "fenced_code_block" || nodeType == "indented_code_block" {
			return
		}
		start := int(node.StartByte())
		end := int(node.EndByte())
		switch node.Type() {
		case "emphasis_delimiter", "code_span_delimiter", "link_destination", "link_title":
			addRange(start, end)
		case "[", "]", "(", ")", "!":
			if hasMarkdownLinkAncestor(node) && !h.isMarkdownTaskCheckbox(node) {
				addRange(start, end)
			}
		case "#":
			line := index.lineIndexAt(start)
			lineStart := index.lineStarts[line]
			if start == lineStart {
				prefixEnd := markdownHeadingPrefixEnd(h.currentSource[lineStart:index.lineEnds[line]])
				addRange(lineStart, lineStart+prefixEnd)
			}
		case ">":
			line := index.lineIndexAt(start)
			lineStart := index.lineStarts[line]
			if start == lineStart {
				prefixEnd := markdownBlockquotePrefixEnd(h.currentSource[lineStart:index.lineEnds[line]])
				addRange(lineStart, lineStart+prefixEnd)
			}
		case "image":
			line := index.lineIndexAt(start)
			lineStart := index.lineStarts[line]
			lineEnd := index.lineEnds[line]
			contentStart := lineStart
			for contentStart < lineEnd && (h.currentSource[contentStart] == ' ' || h.currentSource[contentStart] == '\t') {
				contentStart++
			}
			contentEnd := lineEnd
			for contentEnd > contentStart && (h.currentSource[contentEnd-1] == ' ' || h.currentSource[contentEnd-1] == '\t') {
				contentEnd--
			}
			if start == contentStart && end == contentEnd {
				image := h.readMarkdownImage(node)
				if image.Destination != "" {
					presentation := index.lines[lineStart]
					presentation.image = image
					presentation.hasImage = true
					index.lines[lineStart] = presentation
				}
			}
		}

		for i := 0; i < int(node.ChildCount()); i++ {
			walk(node.Child(i))
		}
	}
	walk(h.currentTree.RootNode())

	for lineStart, presentation := range index.lines {
		presentation.concealRanges = mergeConcealRanges(presentation.concealRanges)
		index.lines[lineStart] = presentation
	}
	h.markdownPresentationIndex = index
}

func (index *markdownPresentationIndex) lineIndexAt(position int) int {
	line := sort.Search(len(index.lineStarts), func(i int) bool {
		return index.lineStarts[i] > position
	}) - 1
	if line < 0 {
		return 0
	}
	return line
}

func (h *Highlighter) readMarkdownImage(node *sitter.Node) MarkdownImage {
	var image MarkdownImage
	var walk func(*sitter.Node)
	walk = func(node *sitter.Node) {
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
			walk(node.Child(i))
		}
	}
	walk(node)
	return image
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
