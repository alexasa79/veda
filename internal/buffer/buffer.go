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

// Buffer is the main data structure for the editor's text.
type Buffer struct {
	root Node
}

// NewBufferFromString creates a new buffer from a string.
// This is a simple implementation that doesn't create a balanced tree yet.
func NewBufferFromString(text string) *Buffer {
	return &Buffer{
		root: NewLeaf(text),
	}
}

func (b *Buffer) String() string {
	if b.root == nil {
		return ""
	}
	return b.root.String()
}

func (b *Buffer) Length() int {
	if b.root == nil {
		return 0
	}
	return b.root.Length()
}

// Lines returns all lines in the buffer.
func (b *Buffer) Lines() []string {
	return strings.Split(b.String(), "\n")
}

// Line returns the line at the given index.
func (b *Buffer) Line(i int) string {
	lines := b.Lines()
	if i < 0 || i >= len(lines) {
		return ""
	}
	return lines[i]
}

// Insert inserts text at a given position in the buffer.
func (b *Buffer) Insert(pos int, text string) {
	if b.root == nil {
		b.root = NewLeaf(text)
		return
	}
	b.root = b.root.insert(pos, text)
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
