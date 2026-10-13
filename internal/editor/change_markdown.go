package editor

import (
	"fmt"
	"strings"
)

type changeRow struct {
	Path, DiffLink, FileLink, Status string
	Added, Deleted, Modified         int
	Staged                           bool
}

// Keep externally supplied titles and paths on one Markdown line, outside link syntax.
func markdownLineLabel(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	return strings.NewReplacer("[", "［", "]", "］", "*", "∗", "`", "'", "\\", "/").Replace(s)
}

// Both local changes and PR snapshots use the same file row layout.
func writeChangeRow(b *strings.Builder, row changeRow) {
	status := row.Status
	switch status {
	case "modified":
		status = "M"
	case "added":
		status = "A"
	case "removed", "deleted":
		status = "D"
	case "renamed":
		status = "R"
	case "copied":
		status = "C"
	case "":
		status = "?"
	}
	if row.Staged {
		status += "●"
	}
	fmt.Fprintf(b, "[%s](%s)", markdownLineLabel(row.Path), row.DiffLink)
	if row.FileLink != "" {
		fmt.Fprintf(b, " [File](%s)", row.FileLink)
	}
	fmt.Fprintf(b, " +%d -%d", row.Added, row.Deleted)
	if row.Modified > 0 {
		fmt.Fprintf(b, ", %d modified", row.Modified)
	}
	fmt.Fprintf(b, " [%s]\n", status)
}
