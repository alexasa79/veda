package proxy

import (
	"os/exec"
	"testing"
)

// Exercise the actual Python method with a fake gh runner, without credentials,
// network access, or a proxy process. Keep all test code in this Go test file.
func TestPRReviewProxy(t *testing.T) {
	code := `
import ast, base64, json, re
from pathlib import Path
from types import SimpleNamespace
from typing import Dict, Any
from urllib.parse import urlparse, quote

tree = ast.parse(Path('veda-proxy.py').read_text())
method = next(m for c in tree.body if isinstance(c, ast.ClassDef)
              for m in c.body if isinstance(m, ast.FunctionDef) and m.name == 'review_pr')
method.decorator_list = []
exec(compile(ast.Module(body=[method], type_ignores=[]), 'veda-proxy.py', 'exec'))

class WorkingPath:
    data = b'new\n'
    def __init__(self, path): self.path = path
    def resolve(self): return self
    def __truediv__(self, path): return WorkingPath(self.path + '/' + path)
    def is_relative_to(self, root): return self.path.startswith(root.path + '/')
    def is_file(self): return True
    def read_bytes(self): return self.data
    def __str__(self): return self.path
Path = WorkingPath

base, head, ancestor = 'a'*40, 'b'*40, 'c'*40
pr = {'number': 12, 'title': 'Fix', 'body': None, 'html_url': 'https://github.com/acme/repo/pull/12',
      'user': {'login': 'alice'}, 'base': {'sha': base}, 'head': {'sha': head}, 'changed_files': 2}
files = [{'filename': 'new name.go', 'previous_filename': 'old.go', 'status': 'renamed'},
         {'filename': 'gone.go', 'status': 'removed'}]

class Fake:
    def __init__(self):
        self.calls = []
        self.moving = False
        self.binary = False
        self.failure = False
        self.page = False
    def _run_shell_command(self, cmd, cwd=None, timeout=None):
        assert cmd[0] in ('gh', 'git') and cwd == '/project' and timeout == 60
        if cmd[0] == 'git':
            value = '/project' if cmd[-1] == '--show-toplevel' else head
            return SimpleNamespace(returncode=0, stderr='', stdout=value)
        self.calls.append(cmd)
        if self.failure:
            return SimpleNamespace(returncode=1, stderr='not authenticated', stdout='')
        if cmd[1:3] == ['repo', 'view']:
            value = {'url': 'https://github.com/acme/repo'}
        elif cmd[1:3] == ['pr', 'list']:
            assert '--json' in cmd and '--limit' in cmd
            value = [{'number': 12, 'title': 'Fix', 'url': pr['html_url'], 'author': pr['user']}]
        else:
            assert cmd[1:4] == ['api', '--hostname', 'github.com']
            path = cmd[4]
            if path.endswith('/pulls/12'):
                value = json.loads(json.dumps(pr))
                if self.moving and sum(c[-1].endswith('/pulls/12') for c in self.calls) > 1:
                    value['head']['sha'] = 'd'*40
                if self.page:
                    value['changed_files'] = 101
            elif '/compare/' in path:
                assert base+'...'+head in path
                value = {'merge_base_commit': {'sha': ancestor}}
            elif '/files?' in path:
                if self.page:
                    value = [files[0]] * (100 if path.endswith('page=1') else 1)
                else:
                    value = files
            elif '/contents/' in path:
                assert '?ref='+ancestor in path or '?ref='+head in path
                content = b'\x00binary' if self.binary else b'old\n' if '?ref='+ancestor in path else b'new\n'
                value = {'type': 'file', 'encoding': 'base64', 'content': base64.b64encode(content).decode()}
            else:
                raise AssertionError(path)
        return SimpleNamespace(returncode=0, stderr='', stdout=json.dumps(value))

f = Fake()
r = review_pr(f, '/project', {'action': 'list'})
assert r['success'] and len(r['requested']) == 1 and len(r['assigned']) == 1, r
assert any('review-requested:@me' in c for c in f.calls)
assert any('--assignee' in c and '@me' in c for c in f.calls)

f = Fake()
r = review_pr(f, '/project', {'action': 'view', 'reference': '12'})
assert r['success'] and r['base'] == ancestor and r['head'] == head and r['files'] == files, r

f = Fake()
f.page = True
r = review_pr(f, '/project', {'action': 'view', 'reference': pr['html_url']})
assert r['success'] and len(r['files']) == 101, r

f = Fake()
f.moving = True
r = review_pr(f, '/project', {'action': 'view', 'reference': '12'})
assert not r['success'] and 'changed while loading' in r['error'], r

query = {'action': 'file', 'url': pr['html_url'], 'base': ancestor, 'head': head,
         'path': 'new name(#L2).go', 'old_path': 'old.go', 'status': 'renamed'}
f = Fake()
r = review_pr(f, '/project', query)
assert r['success'] and r['left'] == 'old\n' and r['right'] == 'new\n', r
assert '/contents/old.go?' in f.calls[0][-1]
assert '/contents/new%20name%28%23L2%29.go?' in f.calls[1][-1]

for status, empty in [('added', 'left'), ('removed', 'right')]:
    f = Fake()
    r = review_pr(f, '/project', dict(query, status=status))
    assert r['success'] and r[empty] == '' and len(f.calls) == 1, r

f = Fake()
f.binary = True
WorkingPath.data = b'\x00binary'
r = review_pr(f, '/project', query)
assert r['success'] and r['binary'], r
WorkingPath.data = b'new\n'

f = Fake()
f.failure = True
r = review_pr(f, '/project', {'action': 'list'})
assert not r['success'] and 'not authenticated' in r['error'], r

for reference in ['-1', '--help', 'file:///etc/passwd', 'https://github.com/a/b/issues/12']:
    f = Fake()
    r = review_pr(f, '/project', {'action': 'view', 'reference': reference})
    assert not r['success'] and not f.calls, r
f = Fake()
r = review_pr(f, '/project', dict(query, head='--help'))
assert not r['success'] and not f.calls, r
`
	cmd := exec.Command("python3", "-c", code)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("PR proxy tests: %v\n%s", err, output)
	}
}
