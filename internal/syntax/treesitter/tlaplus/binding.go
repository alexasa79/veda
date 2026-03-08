package tlaplus

// #include "tree_sitter/parser.h"
// const TSLanguage *tree_sitter_tlaplus(void);
import "C"
import (
	"unsafe"

	sitter "github.com/smacker/go-tree-sitter"
)

// GetLanguage returns the tree-sitter language for TLA+ specifications.
func GetLanguage() *sitter.Language {
	ptr := unsafe.Pointer(C.tree_sitter_tlaplus())
	return sitter.NewLanguage(ptr)
}
