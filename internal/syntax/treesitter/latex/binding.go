package latex

// #include "tree_sitter/parser.h"
// const TSLanguage *tree_sitter_latex(void);
import "C"
import (
	"unsafe"

	sitter "github.com/smacker/go-tree-sitter"
)

// GetLanguage returns the tree-sitter language for LaTeX documents.
func GetLanguage() *sitter.Language {
	ptr := unsafe.Pointer(C.tree_sitter_latex())
	return sitter.NewLanguage(ptr)
}
