package syntax

import (
	"bytes"
	"sort"

	sitter "github.com/smacker/go-tree-sitter"
)

// ByteRange is a half-open range of source byte offsets.
type ByteRange struct {
	Start int
	End   int
}

// FunctionHeaderRanges returns the parts of function headers overlapping the
// source range. Opening braces and single-line brace bodies are included.
func (h *Highlighter) FunctionHeaderRanges(start, end int) []ByteRange {
	if h.currentTree == nil || start >= end {
		return nil
	}
	var ranges []ByteRange
	var visit func(*sitter.Node, int)
	visit = func(node *sitter.Node, start int) {
		if int(node.EndByte()) <= start || int(node.StartByte()) >= end {
			return
		}
		if header, ok := h.functionHeader(node); ok {
			if header.Start < end && header.End > start {
				ranges = append(ranges, ByteRange{max(start, header.Start), min(end, header.End)})
				start = max(start, header.End)
				if start >= end {
					return
				}
			}
		}
		// Skip preceding siblings so rendering near EOF does not scan the file.
		count := int(node.NamedChildCount())
		first := sort.Search(count, func(i int) bool {
			return int(node.NamedChild(i).EndByte()) > start
		})
		for i := first; i < count; i++ {
			child := node.NamedChild(i)
			if int(child.StartByte()) >= end {
				break
			}
			visit(child, start)
		}
	}
	visit(h.currentTree.RootNode(), start)
	return ranges
}

func (h *Highlighter) functionHeader(node *sitter.Node) (ByteRange, bool) {
	switch node.Type() {
	case "template_declaration", "decorated_definition", "export_statement":
		for i := 0; i < int(node.NamedChildCount()); i++ {
			if header, ok := h.functionHeader(node.NamedChild(i)); ok {
				return ByteRange{int(node.StartByte()), header.End}, true
			}
		}
		return ByteRange{}, false
	case "function_definition", "function_declaration", "method_declaration",
		"function_expression", "generator_function_declaration", "generator_function",
		"method_definition", "arrow_function", "lambda_expression", "func_literal":
	default:
		return ByteRange{}, false
	}
	if h.GetCurrentLanguage() == "cpp" && node.Type() == "function_definition" {
		// C++ has no nested function definitions, but the parser can mistake
		// a loop macro followed by a block for one. Local class methods are valid.
	scope:
		for parent := node.Parent(); parent != nil; parent = parent.Parent() {
			switch parent.Type() {
			case "class_specifier", "struct_specifier", "union_specifier":
				break scope
			case "compound_statement":
				return ByteRange{}, false
			}
		}
	}
	body := node.ChildByFieldName("body")
	if body == nil || body.IsMissing() {
		return ByteRange{}, false
	}
	start, end := int(node.StartByte()), int(body.StartByte())
	if parent := node.Parent(); parent != nil {
		switch parent.Type() {
		case "template_declaration", "decorated_definition", "export_statement":
			start = int(parent.StartByte())
		}
	}
	if end < len(h.currentSource) && h.currentSource[end] == '{' {
		if body.StartPoint().Row == body.EndPoint().Row {
			end = int(body.EndByte())
		} else {
			end++
		}
	} else {
		end = start + len(bytes.TrimRight(h.currentSource[start:end], " \t\r\n"))
	}
	return ByteRange{start, end}, start < end
}
