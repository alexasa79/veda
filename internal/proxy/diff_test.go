package proxy

import (
	"go/format"
	"strings"
	"testing"
)

func TestComputeTextDiffAlignsInsertedFieldAfterGofmt(t *testing.T) {
	before := formatDiffFixture(t, `package sample

var config = Config{
	Name:           "sample",
	Range:          5,
	Risk:           0.10,
	Leverage:       1.0,
	TopN:           21,
	MinimumVolume:  35_000_000,
	MinimumPercent: 0.025,
}
`)
	after := formatDiffFixture(t, `package sample

var config = Config{
	Name:               "sample",
	Range:              5,
	Risk:               0.10,
	MaximumRiskPerItem: 0.01,
	Leverage:           1.0,
	TopN:               21,
	MinimumVolume:      35_000_000,
	MinimumPercent:     0.025,
}
`)

	pm, err := NewProxyManager()
	if err != nil {
		t.Fatalf("create proxy manager: %v", err)
	}
	defer pm.Close()

	aligned, err := pm.ComputeTextDiff(before, after)
	if err != nil {
		t.Fatalf("compute text diff: %v", err)
	}

	var inserted []AlignedLine
	for _, line := range aligned {
		if line.Tag == "insert" {
			inserted = append(inserted, line)
		}
	}
	if len(inserted) != 1 {
		t.Fatalf("expected one inserted line, got %d: %#v", len(inserted), inserted)
	}
	if !strings.Contains(inserted[0].Right, "MaximumRiskPerItem:") {
		t.Errorf("expected inserted field to be MaximumRiskPerItem, got %q", inserted[0].Right)
	}

	for _, line := range aligned {
		if strings.Contains(line.Right, "MinimumPercent:") &&
			!strings.Contains(line.Left, "MinimumPercent:") {
			t.Errorf("expected MinimumPercent fields to align, got left %q and right %q", line.Left, line.Right)
		}
	}
}

func formatDiffFixture(t *testing.T, source string) string {
	t.Helper()
	formatted, err := format.Source([]byte(source))
	if err != nil {
		t.Fatalf("format fixture: %v", err)
	}
	return string(formatted)
}
