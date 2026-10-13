package proxy

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestProjectChangesIncludesNestedSubmodules(t *testing.T) {
	dir, err := os.MkdirTemp("../../artifacts", "project-changes-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	dir, err = filepath.Abs(dir)
	if err != nil {
		t.Fatal(err)
	}
	code := `
import ast, logging, os, re, subprocess, sys
from pathlib import Path
from typing import Dict, Any

logger = logging.getLogger('test')
tree = ast.parse(Path('veda-proxy.py').read_text())
methods = [m for c in tree.body if isinstance(c, ast.ClassDef)
           for m in c.body if isinstance(m, ast.FunctionDef)
           and m.name in ('get_project_changes', '_parse_git_diff')]
for method in methods:
    method.decorator_list = []
exec(compile(ast.Module(body=methods, type_ignores=[]), 'veda-proxy.py', 'exec'))

class Proxy:
    get_project_changes = get_project_changes
    _parse_git_diff = _parse_git_diff

    def _run_shell_command(self, cmd, cwd):
        return subprocess.run(cmd, cwd=cwd, capture_output=True, text=True, check=False)

fixture = Path(sys.argv[1])
os.environ['GIT_ALLOW_PROTOCOL'] = 'file'
os.environ['GIT_AUTHOR_NAME'] = os.environ['GIT_COMMITTER_NAME'] = 'Test User'
os.environ['GIT_AUTHOR_EMAIL'] = os.environ['GIT_COMMITTER_EMAIL'] = 'test@example.com'

def git(repo, *args):
    p = subprocess.run(['git', '-C', str(repo), *args], capture_output=True, text=True)
    assert p.returncode == 0, (args, p.stderr)
    return p.stdout.strip()

def commit(repo):
    git(repo, 'add', '.')
    git(repo, 'commit', '-qm', 'test fixture')

def create(name):
    repo = fixture / name
    repo.mkdir()
    git(repo, 'init', '-b', 'main')
    git(repo, 'config', 'core.hooksPath', '/dev/null')
    (repo / 'value.txt').write_text('old\n')
    (repo / 'deleted.txt').write_text('remove\n')
    commit(repo)
    return repo

leaf = create('leaf')
middle = create('middle')
git(middle, 'submodule', 'add', str(leaf), 'nested module')
commit(middle)
project = create('project')
git(project, 'submodule', 'add', str(middle), 'deps/middle module')
git(project, 'submodule', 'add', str(leaf), 'uninitialized')
git(project, 'submodule', 'add', str(leaf), 'clean')
commit(project)
git(project, 'submodule', 'update', '--init', '--recursive')
git(project, 'submodule', 'deinit', '-f', 'uninitialized')
sub = project / 'deps/middle module'
nested = sub / 'nested module'
# Traversal must not depend on the parent's submodule ignore setting.
git(project, 'config', 'submodule.deps/middle module.ignore', 'all')
git(sub, 'config', 'submodule.nested module.ignore', 'all')

proxy = Proxy()
result = proxy.get_project_changes(str(project))
assert result == {'success': True, 'changes': []}, result
(project / 'value.txt').write_text('old\nparent\n')
(sub / 'value.txt').write_text('old\nsubmodule\n')
git(sub, 'add', 'value.txt')
(sub / 'new.txt').write_text('untracked\n')
(nested / 'value.txt').write_text('old\nnested\n')
(nested / 'deleted.txt').unlink()
(nested / 'added.txt').write_text('staged addition\n')
git(nested, 'add', 'added.txt')

expected = {
    project / 'value.txt': ('M', False),
    sub / 'value.txt': ('M', True),
    sub / 'new.txt': ('A', False),
    nested / 'value.txt': ('M', False),
    nested / 'deleted.txt': ('D', False),
    nested / 'added.txt': ('A', True),
}
# Starting below the root must still include the entire project.
for cwd in (project, project / 'deps'):
    result = proxy.get_project_changes(str(cwd))
    assert result['success'], result
    changes = {Path(c['file']): c for c in result['changes']}
    assert len(result['changes']) == len(expected), result
    assert changes.keys() == expected.keys(), result
    for path, (status, staged) in expected.items():
        change = changes[path]
        assert change['relative_path'] == str(path.relative_to(project)), change
        assert (change['status'], change['staged']) == (status, staged), change
        if path.name == 'value.txt':
            assert change['added_lines'] == [2], change
            assert len(change['hunks']) == 1, change
assert not (project / 'uninitialized/.git').exists()
`
	cmd := exec.Command("python3", "-c", code, dir)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("project changes integration: %v\n%s", err, out)
	}
}
