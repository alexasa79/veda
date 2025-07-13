package buffer

import "strings"

const (
	maxLeafSize = 4096
	minLeafSize = 2048
)

// Node is the interface for all nodes in the rope.
type Node interface {
	// Length returns the total length of the text stored under this node.
	Length() int
	// String returns the full text of the node.
	String() string
	insert(pos int, text string) Node
}

// Leaf represents a leaf node in the rope, containing a string fragment.
type Leaf struct {
	text string
}

// NewLeaf creates a new leaf node.
func NewLeaf(text string) *Leaf {
	return &Leaf{text: text}
}

func (l *Leaf) Length() int {
	return len(l.text)
}

func (l *Leaf) String() string {
	return l.text
}

// ConcatNode represents an internal node that concatenates two sub-nodes.
type ConcatNode struct {
	left, right Node
	// weight is the length of the text in the left child.
	weight int
}

// NewConcatNode creates a new concatenation node.
func NewConcatNode(left, right Node) *ConcatNode {
	return &ConcatNode{
		left:   left,
		right:  right,
		weight: left.Length(),
	}
}

func (n *ConcatNode) Length() int {
	return n.weight + n.right.Length()
}

func (n *ConcatNode) String() string {
	var sb strings.Builder
	sb.WriteString(n.left.String())
	sb.WriteString(n.right.String())
	return sb.String()
}

// Rope is the main data structure for the editor's text content.
// It implements a rope data structure for efficient text manipulation.
type Rope struct {
	root Node
}

// NewRopeFromString creates a new rope from a string.
// This is a simple implementation that doesn't create a balanced tree yet.
func NewRopeFromString(text string) *Rope {
	return &Rope{
		root: NewLeaf(text),
	}
}

func (r *Rope) String() string {
	if r.root == nil {
		return ""
	}
	return r.root.String()
}

func (r *Rope) Length() int {
	if r.root == nil {
		return 0
	}
	return r.root.Length()
}

// Lines returns all lines in the rope.
func (r *Rope) Lines() []string {
	return strings.Split(r.String(), "\n")
}

// Insert inserts text at a given position in the rope.
func (r *Rope) Insert(pos int, text string) {
	if r.root == nil {
		r.root = NewLeaf(text)
		return
	}
	r.root = r.root.insert(pos, text)
}

// ByteAt returns the byte at the given position in the rope.
// Returns 0 if position is out of bounds.
func (r *Rope) ByteAt(pos int) byte {
	if r.root == nil || pos < 0 || pos >= r.Length() {
		return 0
	}
	return r.byteAt(r.root, pos)
}

// byteAt is a helper method that recursively finds the byte at position
func (r *Rope) byteAt(node Node, pos int) byte {
	switch n := node.(type) {
	case *Leaf:
		if pos >= 0 && pos < len(n.text) {
			return n.text[pos]
		}
		return 0
	case *ConcatNode:
		if pos < n.weight {
			return r.byteAt(n.left, pos)
		}
		return r.byteAt(n.right, pos-n.weight)
	default:
		return 0
	}
}

// insert is the recursive helper for the Insert method.
func (n *ConcatNode) insert(pos int, text string) Node {
	// TODO: Implement rebalancing
	if pos < n.weight {
		n.left = n.left.insert(pos, text)
	} else {
		n.right = n.right.insert(pos-n.weight, text)
	}
	n.weight = n.left.Length()
	return n
}

func (l *Leaf) insert(pos int, text string) Node {
	// TODO: Implement rebalancing
	l.text = l.text[:pos] + text + l.text[pos:]
	if l.Length() > maxLeafSize {
		// Split the leaf
		splitPos := l.Length() / 2
		return NewConcatNode(
			NewLeaf(l.text[:splitPos]),
			NewLeaf(l.text[splitPos:]),
		)
	}
	return l
}
