package fish

// #include "tree_sitter/parser.h"
// TSLanguage *tree_sitter_fish(void);
import "C"
import (
	"unsafe"

	sitter "github.com/smacker/go-tree-sitter"
)

// GetLanguage returns the tree-sitter language for fish scripts.
func GetLanguage() *sitter.Language {
	ptr := unsafe.Pointer(C.tree_sitter_fish())
	return sitter.NewLanguage(ptr)
}
