package proxy

import (
	"bufio"
	"encoding/json"
	"net"
	"os/exec"
	"testing"
)

func TestGetFileForwardsLSPFlag(t *testing.T) {
	for _, startLSP := range []bool{false, true} {
		local, remote := net.Pipe()
		client := &Client{stdin: bufio.NewWriter(local), stdout: bufio.NewScanner(local)}
		pm := &ProxyManager{clients: map[string]*Client{"testhost": client}}
		requests := make(chan JSONRPCRequest, 1)
		go func() {
			var request JSONRPCRequest
			if err := json.NewDecoder(remote).Decode(&request); err != nil {
				return
			}
			requests <- request
			json.NewEncoder(remote).Encode(map[string]interface{}{
				"id": request.ID, "result": map[string]interface{}{"content": "file contents"},
			})
		}()
		content, err := pm.GetFile("testhost:/project/script.sh", startLSP)
		local.Close()
		remote.Close()
		if err != nil {
			t.Fatal(err)
		}
		request := <-requests
		if request.Method != "get_file" || request.Params["path"] != "/project/script.sh" || request.Params["start_lsp"] != startLSP {
			t.Fatalf("unexpected request: %+v", request)
		}
		if content.Content != "file contents" {
			t.Fatalf("unexpected content: %q", content.Content)
		}
	}
}

func TestGetFileLSPActivation(t *testing.T) {
	code := `
import ast, logging, subprocess, time
from pathlib import Path
from typing import Dict

logger = logging.getLogger('test')
tree = ast.parse(Path('veda-proxy.py').read_text())
method = next(m for c in tree.body if isinstance(c, ast.ClassDef)
              and c.name == 'VedaProxy' for m in c.body
              if isinstance(m, ast.FunctionDef) and m.name == 'get_file')
method.decorator_list = []
exec(compile(ast.Module(body=[method], type_ignores=[]), 'veda-proxy.py', 'exec'))

class LSP:
    def __init__(self, running):
        self.servers = {'bash': {}} if running else {}
        self.diagnostics = {}
        self.started = []
        self.opened = []

    def get_language_for_file(self, path):
        return 'bash'

    def start_server_async(self, language):
        self.started.append(language)

    def _open_file(self, language, path):
        self.opened.append((language, path))

class Proxy:
    get_file = get_file

    def _run_shell_command(self, cmd, cwd):
        return subprocess.CompletedProcess(cmd, 0, stdout='test-branch\n')

    def _get_file_git_changes(self, path):
        return {'added': [3], 'deleted': [], 'modified': [5]}

path = Path('veda-proxy.py').resolve()
for running in (False, True):
    proxy = Proxy()
    proxy.lsp = LSP(running)
    background = proxy.get_file(str(path), start_lsp=False)
    assert proxy.lsp.started == [], proxy.lsp.started
    assert proxy.lsp.opened == [], proxy.lsp.opened
    assert background['content'] == path.read_text()
    assert background['changes'] == {'added': [3], 'deleted': [], 'modified': [5]}
    assert background['branch'] == 'test-branch'

    opened = proxy.get_file(str(path))
    assert opened == background
    assert proxy.lsp.started == ([] if running else ['bash'])
    assert proxy.lsp.opened == [('bash', str(path))]
`
	cmd := exec.Command("python3", "-c", code)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("get_file LSP activation: %v\n%s", err, out)
	}
}
