package syntax

import (
	"context"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/cpp"
)

// headerLanguage defaults ambiguous .h files to C. Only parsed C++ constructs
// count as evidence, so comments, strings and macro bodies do not affect detection.
func headerLanguage(content []byte) string {
	if len(content) == 0 {
		return "c"
	}
	parser := sitter.NewParser()
	defer parser.Close()
	parser.SetLanguage(cpp.GetLanguage())
	tree, err := parser.ParseCtx(context.Background(), nil, content)
	if err != nil {
		return "c"
	}
	defer tree.Close()
	if hasCppConstruct(tree.RootNode()) {
		return "cpp"
	}
	return "c"
}

func hasCppConstruct(node *sitter.Node) bool {
	switch node.Type() {
	case "comment", "string_literal", "char_literal", "raw_string_literal", "preproc_arg", "ERROR":
		return false
	case "class_specifier", "namespace_definition", "namespace_alias_definition",
		"template_declaration", "alias_declaration", "using_declaration",
		"qualified_identifier", "reference_declarator", "abstract_reference_declarator",
		"access_specifier":
		if !node.HasError() {
			return true
		}
	case "enum_specifier":
		for i := 0; i < int(node.ChildCount()); i++ {
			if childType := node.Child(i).Type(); childType == "class" || childType == "struct" {
				return true
			}
		}
	}
	for i := 0; i < int(node.NamedChildCount()); i++ {
		if hasCppConstruct(node.NamedChild(i)) {
			return true
		}
	}
	return false
}
