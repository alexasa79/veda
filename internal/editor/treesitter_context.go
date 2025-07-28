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

	// Get the root node
	root := tree.RootNode()
	
	// Find nodes that contain the cursor line and should be shown as context
	contextNodes := findContextNodes(root, cursorLine, viewStartLine, viewEndLine)
	
	// Convert nodes to line numbers
	var lineNumbers []int
	lineMap := make(map[int]bool)
	
	for _, node := range contextNodes {
		startLine := int(node.StartPoint().Row)
		
		// For function definitions, we want to show the complete signature
		if isFunctionNode(node) {
			// Get the line that contains the function name and parameters
			funcLine := findFunctionDeclarationLine(buf, node)
			if funcLine >= 0 && (funcLine < viewStartLine || funcLine > viewEndLine) && !lineMap[funcLine] {
				lineMap[funcLine] = true
				lineNumbers = append(lineNumbers, funcLine)
			}
		} else if startLine < viewStartLine || startLine > viewEndLine {
			// For other nodes, use the start line
			if !lineMap[startLine] {
				lineMap[startLine] = true
				lineNumbers = append(lineNumbers, startLine)
			}
		}
	}
	
	// Limit to maxLines
	if len(lineNumbers) > maxLines {
		lineNumbers = lineNumbers[len(lineNumbers)-maxLines:]
	}
	
	return lineNumbers
}

// findContextNodes finds all nodes that contain the cursor line and are context-worthy
func findContextNodes(node *sitter.Node, cursorLine int, viewStartLine int, viewEndLine int) []*sitter.Node {
	var result []*sitter.Node
	
	nodeStartLine := int(node.StartPoint().Row)
	nodeEndLine := int(node.EndPoint().Row)
	
	// Check if this node contains the cursor line
	if cursorLine >= nodeStartLine && cursorLine <= nodeEndLine {
		// Check if this is a context-worthy node type
		if isContextNode(node) {
			result = append(result, node)
		}
		
		// Recursively check children
		childCount := int(node.ChildCount())
		for i := 0; i < childCount; i++ {
			child := node.Child(i)
			childNodes := findContextNodes(child, cursorLine, viewStartLine, viewEndLine)
			result = append(result, childNodes...)
		}
	}
	
	return result
}

// isContextNode checks if a node type should be shown as context
func isContextNode(node *sitter.Node) bool {
	nodeType := node.Type()
	
	// Context-worthy node types for different languages
	contextTypes := map[string]bool{
		// C/C++ context nodes
		"function_definition":     true,
		"struct_specifier":        true,
		"enum_specifier":          true,
		"if_statement":            true,
		"for_statement":           true,
		"while_statement":         true,
		"do_statement":            true,
		"switch_statement":        true,
		"compound_statement":      false, // Skip compound statements to avoid braces
		
		// Go context nodes
		"function_declaration":    true,
		"method_declaration":      true,
		"type_declaration":        true,
		"select_statement":        true,
	}
	
	return contextTypes[nodeType]
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
	
	for i := 0; i < childCount; i++ {
		child := node.Child(i)
		childType := child.Type()
		
		// Look for the declarator or similar node that contains the function signature
		if childType == "function_declarator" || childType == "declarator" || 
		   childType == "identifier" || childType == "field_identifier" {
			// Return the start line of this node
			return int(child.StartPoint().Row)
		}
		
		// For Go, the function name is usually an identifier at the top level
		if node.Type() == "function_declaration" && childType == "identifier" {
			return int(child.StartPoint().Row)
		}
	}
	
	// If we can't find a specific declarator, look for the first line
	// that's not a storage class specifier or type
	startLine := int(node.StartPoint().Row)
	iter := buffer.NewBufferIteratorAtLine(buf, startLine)
	
	if !iter.IsEOF() {
		line := strings.TrimSpace(iter.Line())
		// If the line doesn't start with '{', it's likely part of the signature
		if !strings.HasPrefix(line, "{") {
			return startLine
		}
		
		// Otherwise, check previous lines for the signature
		for i := 1; i <= 5 && startLine-i >= 0; i++ {
			prevIter := buffer.NewBufferIteratorAtLine(buf, startLine-i)
			if !prevIter.IsEOF() {
				prevLine := strings.TrimSpace(prevIter.Line())
				// Look for lines with parentheses (function parameters)
				if strings.Contains(prevLine, "(") || strings.Contains(prevLine, ")") {
					// Find the start of the function declaration
					for j := i; j <= 5 && startLine-j >= 0; j++ {
						checkIter := buffer.NewBufferIteratorAtLine(buf, startLine-j)
						if !checkIter.IsEOF() {
							checkLine := strings.TrimSpace(checkIter.Line())
							// If we find a line that looks like it starts a function
							// (has a type or function name)
							if !strings.Contains(checkLine, ")") && 
							   !strings.HasSuffix(checkLine, ",") &&
							   !strings.HasPrefix(checkLine, "{") {
								return startLine - j
							}
						}
					}
					return startLine - i
				}
			}
		}
	}
	
	return startLine
}