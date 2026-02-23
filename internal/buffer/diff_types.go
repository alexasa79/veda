package buffer

// DiffLineType indicates the type of difference
type DiffLineType int

const (
	DiffLineContext  DiffLineType = iota // Unchanged line
	DiffLineAdded                        // Line added in right file
	DiffLineDeleted                      // Line removed from left file
	DiffLineModified                     // Line changed between files
)

// DiffLine represents a single line in the diff with its status
type DiffLine struct {
	Content      string
	LineType     DiffLineType
	LeftLine     int         // Line number in left file (0-based), -1 if not present
	RightLine    int         // Line number in right file (0-based), -1 if not present
	IsFillerLine bool        // True for empty lines added for alignment
	WordRanges   []DiffRange // Character ranges that differ within the line (for DiffLineModified)
}

// DiffRange represents a character range within a line that differs
type DiffRange struct {
	Start int // Start character position (inclusive)
	End   int // End character position (exclusive)
}

// DiffResult contains the result of comparing two texts
type DiffResult struct {
	LeftLines  []DiffLine // Lines for left side display
	RightLines []DiffLine // Lines for right side display
}
