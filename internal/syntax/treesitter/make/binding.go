package makefile

// #include "tree_sitter/parser.h"
// const TSLanguage *tree_sitter_make(void);
import "C"
import (
	"unsafe"

	sitter "github.com/smacker/go-tree-sitter"
)

// GetLanguage returns the tree-sitter language for Makefiles.
func GetLanguage() *sitter.Language {
	ptr := unsafe.Pointer(C.tree_sitter_make())
	return sitter.NewLanguage(ptr)
}
