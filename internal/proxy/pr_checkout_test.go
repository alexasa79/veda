package proxy

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestPRCheckoutWithNestedSubmodules(t *testing.T) {
	// All repositories and generated files belong to this isolated test fixture.
	dir, err := os.MkdirTemp("../../artifacts", "pr-checkout-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	dir, err = filepath.Abs(dir)
	if err != nil {
		t.Fatal(err)
	}
	code := `
import ast, json, os, re, subprocess, sys, time
from pathlib import Path
from types import SimpleNamespace
from typing import Dict, List, Any
from urllib.parse import urlparse

tree = ast.parse(Path('veda-proxy.py').read_text())
method = next(m for c in tree.body if isinstance(c, ast.ClassDef)
              for m in c.body if isinstance(m, ast.FunctionDef) and m.name == 'checkout_pr')
method.decorator_list = []
exec(compile(ast.Module(body=[method], type_ignores=[]), 'veda-proxy.py', 'exec'))

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
    return git(repo, 'rev-parse', 'HEAD')

def create(name):
    repo = fixture / name
    repo.mkdir()
    git(repo, 'init', '-b', 'main')
    git(repo, 'config', 'core.hooksPath', '/dev/null')
    (repo / 'value.txt').write_text('old\n')
    commit(repo)
    return repo

leaf = create('leaf')
leaf_old = git(leaf, 'rev-parse', 'HEAD')
(leaf / 'value.txt').write_text('leaf PR\n')
leaf_new = commit(leaf)

middle = create('middle')
git(middle, 'submodule', 'add', str(leaf), 'deps/leaf')
git(middle / 'deps/leaf', 'checkout', leaf_old)
middle_old = commit(middle)
git(middle / 'deps/leaf', 'checkout', leaf_new)
middle_new = commit(middle)

project = create('project')
git(project, 'submodule', 'add', str(middle), 'deps/middle')
git(project / 'deps/middle', 'checkout', middle_old)
git(project, 'config', '-f', '.gitmodules', 'submodule.deps/middle.ignore', 'dirty')
(project / '.gitignore').write_text('build/\n')
base = commit(project)
git(project, 'checkout', '-b', 'review-feature')
(project / 'value.txt').write_text('PR source\n')
git(project / 'deps/middle', 'checkout', middle_new)
head = commit(project)

def restore():
    git(project, 'checkout', 'main')
    git(project, 'submodule', 'update', '--init', '--recursive')
restore()
# The leaf is deliberately uninitialized: checkout must initialize it recursively.
git(project / 'deps/middle', 'submodule', 'deinit', '-f', 'deps/leaf')
(project / 'build').mkdir()
database = project / 'build/compile_commands.json'
database.write_text('preserve existing build metadata')

class FakeLSP:
    def __init__(self):
        self.stopped = False
        self.pending_file_opens = {'cpp': {'old'}}
        self.file_versions = {'old': 1}
        self.diagnostics = {'old': []}
        self.lsp_errors = {'old': 'error'}
    def stop_all_servers(self): self.stopped = True

class Fake:
    def __init__(self):
        self.lsp = FakeLSP()
        self.calls = []
        self.fail_update = False
        self.repo_url = 'https://github.com/acme/project'
    def _run_shell_command(self, cmd, cwd=None, timeout=None):
        self.calls.append(cmd)
        if cmd[:3] == ['gh', 'repo', 'view']:
            return SimpleNamespace(returncode=0, stdout=json.dumps({'url': self.repo_url}), stderr='')
        if cmd[:3] == ['gh', 'pr', 'checkout']:
            assert cmd == ['gh', 'pr', 'checkout', self.repo_url + '/pull/12']
            cmd = ['git', 'checkout', 'review-feature']
        if self.fail_update and cmd[:3] == ['git', 'submodule', 'update']:
            return SimpleNamespace(returncode=1, stdout='', stderr='submodule fetch failed')
        return subprocess.run(cmd, cwd=cwd, capture_output=True, text=True, timeout=timeout)

reference = 'https://github.com/acme/project/pull/12'
source = str(project / 'value.txt')
f = Fake()
r = checkout_pr(f, str(project), reference, head, [source])
assert r['success'], r
assert git(project, 'branch', '--show-current') == 'review-feature'
assert git(project / 'deps/middle', 'rev-parse', 'HEAD') == middle_new
assert git(project / 'deps/middle/deps/leaf', 'rev-parse', 'HEAD') == leaf_new
assert r['files'] == [{'path': source, 'content': 'PR source\n', 'error': ''}], r
assert f.lsp.stopped and not f.lsp.file_versions and not f.lsp.diagnostics
assert database.read_text() == 'preserve existing build metadata'
assert not any('--force' in c or '--remote' in c or 'stash' in c for c in f.calls)

# Local submodule changes must stop checkout before any branch mutation.
restore()
git(project, 'config', 'submodule.deps/middle.ignore', 'none')
nested_file = project / 'deps/middle/deps/leaf/value.txt'
nested_file.write_text('local changes\n')
f = Fake()
r = checkout_pr(f, str(project), reference, head, [source])
assert not r['success'] and 'local changes' in r['error'], r
assert not f.lsp.stopped and git(project, 'branch', '--show-current') == 'main'
assert nested_file.read_text() == 'local changes\n'
git(nested_file.parent, 'restore', 'value.txt')
git(project, 'config', '--unset', 'submodule.deps/middle.ignore')

# Respect .gitmodules ignore=dirty for build files and nonconflicting edits.
local_file = project / 'deps/middle/value.txt'
local_file.write_text('local configuration\n')
build_file = project / 'deps/middle/build-output.txt'
build_file.write_text('existing build output\n')
assert not git(project, 'status', '--porcelain')
f = Fake()
r = checkout_pr(f, str(project), reference, head, [source])
assert r['success'], r
assert local_file.read_text() == 'local configuration\n'
assert build_file.read_text() == 'existing build output\n'
assert git(nested_file.parent, 'rev-parse', 'HEAD') == leaf_new
git(local_file.parent, 'restore', 'value.txt')
build_file.unlink()
restore()

# Ignored dirt must still be protected by Git when updating a submodule conflicts.
nested_file.write_text('keep this local edit\n')
f = Fake()
r = checkout_pr(f, str(project), reference, head, [source])
assert not r['success'], r
assert nested_file.read_text() == 'keep this local edit\n'
assert git(nested_file.parent, 'rev-parse', 'HEAD') == leaf_old
git(nested_file.parent, 'restore', 'value.txt')
restore()

# A submodule failure after switching must report and reload the new working files.
f = Fake()
f.fail_update = True
r = checkout_pr(f, str(project), reference, head, [source])
assert not r['success'] and 'submodule fetch failed' in r['error'], r
assert r['root'] == str(project) and r['files'][0]['content'] == 'PR source\n', r
assert git(project, 'branch', '--show-current') == 'review-feature'
restore()

# Refuse a PR from another repository without checking it out.
f = Fake()
f.repo_url = 'https://github.com/other/project'
r = checkout_pr(f, str(project), reference, head, [source])
assert not r['success'] and 'different repository' in r['error'], r
assert not f.lsp.stopped

# A moving PR head cannot silently attach navigation to the wrong version.
f = Fake()
r = checkout_pr(f, str(project), reference, 'f'*40, [source])
assert not r['success'] and 'changed during checkout' in r['error'], r
assert r['root'] == str(project), r
`
	cmd := exec.Command("python3", "-c", code, dir)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("checkout integration: %v\n%s", err, out)
	}
}
