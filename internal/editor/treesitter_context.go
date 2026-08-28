package editor

import (
	"strings"
	"veda/internal/buffer"
	"veda/internal/syntax"

	sitter "github.com/smacker/go-tree-sitter"
)

// getTreeSitterContext returns context lines using tree-sitter syntax tree
func getTreeSitterContext(buf *buffer.Buffer, highlighter *syntax.Highlighter, cursorLine int, viewStartLine int, viewEndLine int, maxLines int) []int {
	tree := highlighter.GetTree()
	if tree == nil {
		return nil
	}

	root := tree.RootNode()

	// Compute byte-column at cursor (CursorX is byte offset within the line)
	col := buf.CursorX
	if col < 0 {
		col = 0
	}
	// Clamp to current line length
	iter := buffer.NewBufferIteratorAtLine(buf, cursorLine)
	if !iter.IsEOF() {
		line := iter.Line()
		if col > len(line) {
			col = len(line)
		}
	}

	// Point query to get the deepest named node at the cursor
	p := sitter.Point{Row: uint32(cursorLine), Column: uint32(col)}
	n := root.NamedDescendantForPointRange(p, p)
	if n == nil {
		return nil
	}

	// Ascend and collect context-worthy ancestors
	lines := make([]int, 0, maxLines)
	seen := make(map[int]bool)
	language := highlighter.GetCurrentLanguage()

	for cur := n; cur != nil; cur = cur.Parent() {
		if isContextNode(cur, language) {
			lnum := -1
			if isFunctionNode(cur) {
				lnum = findFunctionDeclarationLine(buf, cur)
			} else {
				lnum = int(cur.StartPoint().Row)
			}
			if lnum >= 0 && (lnum < viewStartLine || lnum > viewEndLine) {
				if !seen[lnum] {
					seen[lnum] = true
					lines = append(lines, lnum)
				}
			}
		}
	}

	// Reverse so that outer-most ancestors appear first, then inner-most
	for i, j := 0, len(lines)-1; i < j; i, j = i+1, j-1 {
		lines[i], lines[j] = lines[j], lines[i]
	}
	// Keep only up to maxLines from the start (outer to inner)
	if len(lines) > maxLines {
		lines = lines[:maxLines]
	}
	return lines
}

// findContextNodes finds all nodes that contain the cursor line and are context-worthy
func findContextNodes(node *sitter.Node, language string, cursorLine int, viewStartLine int, viewEndLine int) []*sitter.Node {
	var result []*sitter.Node

	nodeStartLine := int(node.StartPoint().Row)
	nodeEndLine := int(node.EndPoint().Row)

	// Check if this node contains the cursor line
	if cursorLine >= nodeStartLine && cursorLine <= nodeEndLine {
		// Check if this is a context-worthy node type
		if isContextNode(node, language) {
			result = append(result, node)
		}

		// Recursively check children
		childCount := int(node.ChildCount())
		for i := 0; i < childCount; i++ {
			child := node.Child(i)
			childNodes := findContextNodes(child, language, cursorLine, viewStartLine, viewEndLine)
			result = append(result, childNodes...)
		}
	}

	return result
}

// isContextNode checks if a node type should be shown as context
func isContextNode(node *sitter.Node, language string) bool {
	switch node.Type() {
	// Preserve the existing context-node behavior for every language.
	case "function_definition", "struct_specifier", "enum_specifier",
		"if_statement", "for_statement", "while_statement", "do_statement",
		"switch_statement", "function_declaration", "method_declaration",
		"type_declaration", "select_statement":
		return true
	}

	if language != "cpp" {
		return false
	}

	switch node.Type() {
	case "class_specifier", "namespace_definition", "for_range_loop",
		"lambda_expression", "try_statement", "catch_clause", "template_declaration":
		return true
	}

	return false
}

// isFunctionNode checks if a node is a function definition
func isFunctionNode(node *sitter.Node) bool {
	nodeType := node.Type()
	return nodeType == "function_definition" || nodeType == "function_declaration" || nodeType == "method_declaration"
}

// findFunctionDeclarationLine finds the start of a function declaration
// This handles multi-line function signatures
func findFunctionDeclarationLine(buf *buffer.Buffer, node *sitter.Node) int {
	// For function definitions, we need to find the declarator node
	// which contains the function name and parameters
	childCount := int(node.ChildCount())

	// First, try to find the function declarator directly
	var declaratorNode *sitter.Node
	for i := 0; i < childCount; i++ {
		child := node.Child(i)
		childType := child.Type()

		// Look for the declarator that contains the function name
		if childType == "function_declarator" {
			declaratorNode = child
			break
		} else if childType == "declarator" {
			// Check if this declarator contains a function_declarator
			declaratorNode = findFunctionDeclaratorInNode(child)
			if declaratorNode != nil {
				break
			}
		}

		// For Go, the function name is usually an identifier at the top level
		if node.Type() == "function_declaration" && childType == "identifier" {
			return int(child.StartPoint().Row)
		}
	}

	// If we found a function declarator, find the line with the function name
	if declaratorNode != nil {
		// The function name is usually in an identifier node within the declarator
		funcNameLine := findFunctionNameLine(declaratorNode)
		if funcNameLine >= 0 {
			return funcNameLine
		}
		// Fallback to declarator start line
		return int(declaratorNode.StartPoint().Row)
	}

	// Fallback: scan lines to find the one with the function name
	// This handles cases where tree-sitter structure might be different
	return findFunctionNameByScanning(buf, node)
}

// findFunctionDeclaratorInNode recursively searches for a function_declarator node
func findFunctionDeclaratorInNode(node *sitter.Node) *sitter.Node {
	if node.Type() == "function_declarator" {
		return node
	}

	childCount := int(node.ChildCount())
	for i := 0; i < childCount; i++ {
		child := node.Child(i)
		if result := findFunctionDeclaratorInNode(child); result != nil {
			return result
		}
	}

	return nil
}

// findFunctionNameLine finds the line containing the function name within a declarator
func findFunctionNameLine(declaratorNode *sitter.Node) int {
	// Look for an identifier node which typically contains the function name
	childCount := int(declaratorNode.ChildCount())
	for i := 0; i < childCount; i++ {
		child := declaratorNode.Child(i)
		if child.Type() == "identifier" || child.Type() == "field_identifier" {
			return int(child.StartPoint().Row)
		}
		// Sometimes the identifier is nested deeper
		if child.Type() == "pointer_declarator" || child.Type() == "parenthesized_declarator" {
			if line := findFunctionNameLine(child); line >= 0 {
				return line
			}
		}
	}
	return -1
}

// findFunctionNameByScanning scans the function node's text to find the line with the function name
func findFunctionNameByScanning(buf *buffer.Buffer, node *sitter.Node) int {
	startLine := int(node.StartPoint().Row)
	endLine := int(node.EndPoint().Row)

	// Scan lines looking for the function name
	// In C/C++, the function name line typically has:
	// - An identifier followed by '('
	// - Not just type keywords
	for line := startLine; line <= endLine && line < buf.LineCount; line++ {
		iter := buffer.NewBufferIteratorAtLine(buf, line)
		if iter.IsEOF() {
			continue
		}

		lineText := strings.TrimSpace(iter.Line())

		// Skip empty lines and lines that are just type specifiers
		if lineText == "" || lineText == "{" {
			continue
		}

		// Check if this line contains a function name pattern
		// Look for identifier followed by '(' either on same line or next line
		if strings.Contains(lineText, "(") {
			// This line has opening parenthesis, likely the function name line
			// Make sure it's not just a cast or type
			if !isJustTypeDeclaration(lineText) {
				return line
			}
		} else if line < endLine {
			// Check if next line starts with '('
			nextIter := buffer.NewBufferIteratorAtLine(buf, line+1)
			if !nextIter.IsEOF() {
				nextLine := strings.TrimSpace(nextIter.Line())
				if strings.HasPrefix(nextLine, "(") {
					// Current line likely has function name
					if !isJustTypeDeclaration(lineText) {
						return line
					}
				}
			}
		}
	}

	// Fallback to node start line
	return startLine
}

// isJustTypeDeclaration checks if a line contains only type keywords/specifiers
func isJustTypeDeclaration(line string) bool {
	// Common C type keywords and specifiers
	typeKeywords := []string{
		"static", "const", "inline", "extern", "volatile",
		"void", "char", "int", "long", "short", "float", "double",
		"unsigned", "signed", "struct", "union", "enum", "*",
	}

	// Remove pointer symbols for checking
	checkLine := strings.ReplaceAll(line, "*", " ")
	checkLine = strings.ReplaceAll(checkLine, "const", " ")

	// If line only contains type keywords and whitespace, it's just a type
	words := strings.Fields(checkLine)
	if len(words) == 0 {
		return true
	}

	// Check if all words are type keywords
	allTypeKeywords := true
	for _, word := range words {
		isTypeKeyword := false
		for _, keyword := range typeKeywords {
			if word == keyword {
				isTypeKeyword = true
				break
			}
		}
		if !isTypeKeyword {
			allTypeKeywords = false
			break
		}
	}

	return allTypeKeywords
}
