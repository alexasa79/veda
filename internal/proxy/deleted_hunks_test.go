package proxy

import (
	"encoding/json"
	"os/exec"
	"reflect"
	"testing"
)

func TestGitDeletionAnchorsAndText(t *testing.T) {
	code := `
import ast, re
from pathlib import Path
from typing import Dict, Any

tree = ast.parse(Path('veda-proxy.py').read_text())
method = next(m for c in tree.body if isinstance(c, ast.ClassDef)
              for m in c.body if isinstance(m, ast.FunctionDef)
              and m.name == '_parse_git_diff')
exec(compile(ast.Module(body=[method], type_ignores=[]), 'veda-proxy.py', 'exec'))

cases = [
    ('@@ -254 +278,0 @@\n-removed\n', [279], [{'line': 279, 'lines': ['removed']}]),
    ('@@ -20,5 +30,3 @@\n context\n-old one\n-old two\n next\n tail\n',
     [31, 31], [{'line': 31, 'lines': ['old one', 'old two']}]),
    ('@@ -1,5 +1,3 @@\n first\n-old one\n middle\n-old two\n last\n',
     [2, 3], [{'line': 2, 'lines': ['old one']}, {'line': 3, 'lines': ['old two']}]),
    ('@@ -1 +0,0 @@\n-removed\n', [1], [{'line': 1, 'lines': ['removed']}]),
    ('@@ -1,2 +1 @@\n first\n-removed\n', [2], [{'line': 2, 'lines': ['removed']}]),
    ('@@ -1 +1 @@\n-old\n+new\n', [], []),
    ('--- a/file\n+++ b/file\n@@ -1 +0,0 @@\n---counter;\n',
     [1], [{'line': 1, 'lines': ['--counter;']}]),
]
for diff, deleted, hunks in cases:
    result = _parse_git_diff(None, diff)
    assert result['deleted'] == deleted, result
    assert result['deleted_hunks'] == hunks, result
`
	if out, err := exec.Command("python3", "-c", code).CombinedOutput(); err != nil {
		t.Fatalf("parse deletion hunks: %v\n%s", err, out)
	}
}

func TestParseFileChangesDeletedHunks(t *testing.T) {
	var data interface{}
	if err := json.Unmarshal([]byte(`{"deleted":[279,279],"deleted_hunks":[{"line":279,"lines":["old one","old two"]}]}`), &data); err != nil {
		t.Fatal(err)
	}
	changes := parseFileChanges(data)
	want := []DeletedHunk{{Line: 279, Lines: []string{"old one", "old two"}}}
	if changes == nil || !reflect.DeepEqual(changes.DeletedHunks, want) {
		t.Fatalf("deleted hunks = %+v, want %+v", changes, want)
	}
}
