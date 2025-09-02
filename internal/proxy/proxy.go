package proxy

import (
	"bufio"
	"context"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
	"veda/internal/log"
)

var LL *log.Logger = log.NewLogger("PROXY")

// maxScanTokenSize defines the maximum size for scanner buffers (1 GiB)
const maxScanTokenSize = 1024 * 1024 * 1024 // 1 GiB

// ProxyRequestTimeout defines the timeout for proxy requests
const ProxyRequestTimeout = 10 * time.Second

// ProxyVersion must match the version reported by the Python proxy script.
// IMPORTANT: Increment this every time you change either this Go file
// or the Python proxy implementation (veda-proxy.py). See CLAUDE.md.
const ProxyVersion = "1.0.17"

// StatusCallback is a function that receives status updates during connection
type StatusCallback func(status string)

// ProxyManager manages multiple proxy instances for different hosts
type ProxyManager struct {
	clients map[string]*Client // key is hostname, empty string for local
	// mu protects access to the clients map and related state.
	// Do not hold mu while performing network I/O or long-running operations.
	mu             sync.Mutex
	statusCallback StatusCallback // callback for status updates
}

// NewProxyManager creates a new proxy manager
func NewProxyManager() (*ProxyManager, error) {
	pm := &ProxyManager{
		clients: make(map[string]*Client),
	}

	// Start local proxy
	localClient, err := NewClient()
	if err != nil {
		return nil, fmt.Errorf("failed to start local proxy: %w", err)
	}
	pm.clients[""] = localClient

	return pm, nil
}

// SetStatusCallback sets the callback for status updates
func (pm *ProxyManager) SetStatusCallback(callback StatusCallback) {
	pm.statusCallback = callback
}

// reportStatus reports a status update via the callback if set
func (pm *ProxyManager) reportStatus(status string) {
	if pm.statusCallback != nil {
		pm.statusCallback(status)
	}
	// Also emit a final blank status to signal completion cases explicitly
}

// getClient returns the appropriate client for the given path
func (pm *ProxyManager) getClient(path string) (*Client, string, error) {
	host, actualPath, isRemote := parseRemotePath(path)

	LL.Debug("getClient: path=%s, host=%s, actualPath=%s, isRemote=%v", path, host, actualPath, isRemote)

	// Local client fast-path
	if !isRemote {
		pm.mu.Lock()
		c := pm.clients[""]
		pm.mu.Unlock()
		if c == nil {
			return nil, "", fmt.Errorf("local proxy client not initialized")
		}
		return c, path, nil
	}

	// Check if we already have a client for this host
	pm.mu.Lock()
	if client, exists := pm.clients[host]; exists {
		pm.mu.Unlock()
		return client, actualPath, nil
	}
	pm.mu.Unlock()

	// Create a new remote proxy for this host (no lock held during network I/O)
	client, err := pm.createRemoteClient(host)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create remote proxy: %w", err)
	}

	// Store the client (handle possible race with another creator)
	pm.mu.Lock()
	if existing, exists := pm.clients[host]; exists {
		pm.mu.Unlock()
		// Another goroutine won the race; close ours and return existing
		_ = client.Close()
		return existing, actualPath, nil
	}
	pm.clients[host] = client
	pm.mu.Unlock()
	return client, actualPath, nil
}

// startRemoteProxy starts a remote proxy instance and performs version handshake
func (pm *ProxyManager) startRemoteProxy(host string) (*Client, *exec.Cmd, error) {
	pm.reportStatus(fmt.Sprintf("Starting proxy on %s...", host))
	sshArgs := []string{
		"-T",
		// Connectivity + keepalive + non-interactive
		"-o", "ConnectTimeout=5",
		"-o", "ConnectionAttempts=1",
		"-o", "BatchMode=yes",
		"-o", "StrictHostKeyChecking=no",
		"-o", "ServerAliveInterval=15",
		"-o", "ServerAliveCountMax=3",
		"-o", "ControlMaster=no",
		"-o", "ControlPath=none",
		host,
		"python3", "~/.cache/veda/veda-proxy.py",
	}
	cmd := exec.Command("ssh", sshArgs...)

	stdinPipe, err := cmd.StdinPipe()
	if err != nil {
		return nil, nil, err
	}
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, nil, err
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return nil, nil, err
	}

	if err := cmd.Start(); err != nil {
		return nil, nil, err
	}

	// Log any stderr output from SSH/proxy
	go func() {
		scanner := bufio.NewScanner(stderrPipe)
		for scanner.Scan() {
			line := scanner.Text()
			LL.Debug("Remote proxy stderr [%s]: %s", host, line)
			fmt.Fprintf(os.Stderr, "REMOTE PROXY STDERR [%s]: %s\n", host, line)
		}
	}()

	stdin := bufio.NewWriter(stdinPipe)
	stdout := bufio.NewScanner(stdoutPipe)

	// Increase scanner buffer size to handle large responses
	buf := make([]byte, 0, maxScanTokenSize)
	stdout.Buffer(buf, maxScanTokenSize)

	// Wait for the hello/version line with a timeout
	version, err := readProxyHello(stdout, 5*time.Second)
	if err != nil {
		LL.Debug("Failed to read proxy hello on %s: %v", host, err)
		return nil, cmd, err
	}
	if version != ProxyVersion {
		LL.Info("Proxy version mismatch on %s (got %s, want %s)", host, version, ProxyVersion)
		return nil, cmd, fmt.Errorf("version mismatch: got=%s, want=%s", version, ProxyVersion)
	}

	client := &Client{
		cmd:       cmd,
		stdin:     stdin,
		stdout:    stdout,
		requestID: 0,
		pending:   make(map[int]chan map[string]interface{}),
		readerErr: make(chan error, 1),
	}
	client.readerOnce.Do(func() { go client.readerLoop() })
	return client, cmd, nil
}

// createRemoteClient creates a new proxy client connected to a remote host
func (pm *ProxyManager) createRemoteClient(host string) (*Client, error) {
	startTime := time.Now()
	LL.Info("Creating remote proxy connection to %s", host)

	// Step 1: connectivity check (fast-fail, no prompts)
	pm.reportStatus(fmt.Sprintf("Connecting to %s...", host))
	if err := pm.checkRemoteConnectivity(host, 5*time.Second); err != nil {
		return nil, fmt.Errorf("Failed to connect to %s: %s", host, err.Error())
	}

	// First try without copying (assume proxy exists on remote). If it fails or
	// version mismatches, copy and retry once.
	client, cmd, err := pm.startRemoteProxy(host)
	if err != nil {
		if cmd != nil && cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		// Re-verify connectivity before attempting to update/copy
		if errConn := pm.checkRemoteConnectivity(host, 5*time.Second); errConn != nil {
			return nil, fmt.Errorf("ssh connect failed: %w", errConn)
		}
		pm.reportStatus(fmt.Sprintf("Updating proxy on %s...", host))
		if errCopy := pm.copyProxyToRemote(host, 10*time.Second); errCopy != nil {
			return nil, fmt.Errorf("failed to copy proxy to remote: %w", errCopy)
		}
		client, cmd, err = pm.startRemoteProxy(host)
		if err != nil {
			if cmd != nil && cmd.Process != nil {
				_ = cmd.Process.Kill()
			}
			return nil, fmt.Errorf("failed to start proxy on %s after update: %w", host, err)
		}
	}

	duration := time.Since(startTime)
	LL.Info("Created remote proxy connection to %s in %v", host, duration)

	// Don't clear status - let the calling operation set its own status
	return client, nil
}

// copyProxyToRemote copies the embedded proxy script to the remote host (with timeout)
// This is called only when the proxy start fails, so we ensure all prerequisites are met
func (pm *ProxyManager) copyProxyToRemote(host string, timeout time.Duration) error {
	startTime := time.Now()

	// Create a temporary file with the proxy script
	tmpFile, err := os.CreateTemp("", "veda-proxy-*.py")
	if err != nil {
		return err
	}
	defer os.Remove(tmpFile.Name())

	// Write the embedded proxy script to the temp file
	if _, err := tmpFile.Write([]byte(embeddedProxy)); err != nil {
		tmpFile.Close()
		return err
	}
	tmpFile.Close()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Step 1: Ensure ~/.cache/veda directory exists on the remote host
	mkdirCmd := exec.CommandContext(ctx, "ssh",
		"-o", "ConnectTimeout=5",
		"-o", "ConnectionAttempts=1",
		"-o", "BatchMode=yes",
		"-o", "StrictHostKeyChecking=no",
		"-o", "ControlMaster=no",
		"-o", "ControlPath=none",
		host,
		"mkdir", "-p", "~/.cache/veda")

	output, err := mkdirCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("mkdir failed: %w, output: %s", err, output)
	}

	// Step 2: Copy the proxy script to remote ~/.cache/veda/ directory
	scpCmd := exec.CommandContext(ctx, "scp",
		"-o", "ConnectTimeout=5",
		"-o", "ConnectionAttempts=1",
		"-o", "BatchMode=yes",
		"-o", "StrictHostKeyChecking=no",
		"-o", "ControlMaster=no",
		"-o", "ControlPath=none",
		tmpFile.Name(),
		fmt.Sprintf("%s:~/.cache/veda/veda-proxy.py", host))

	output, err = scpCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("scp failed: %w, output: %s", err, output)
	}

	// Step 3: Make the script executable on the remote host
	pm.reportStatus(fmt.Sprintf("Setting up proxy on %s...", host))
	chmodCmd := exec.CommandContext(ctx, "ssh",
		"-o", "ConnectTimeout=5",
		"-o", "ConnectionAttempts=1",
		"-o", "BatchMode=yes",
		"-o", "StrictHostKeyChecking=no",
		"-o", "ControlMaster=no",
		"-o", "ControlPath=none",
		host,
		"chmod", "+x", "~/.cache/veda/veda-proxy.py")

	output, err = chmodCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("chmod failed: %w, output: %s", err, output)
	}

	duration := time.Since(startTime)
	LL.Info("Copied proxy script to %s:~/.cache/veda/veda-proxy.py in %v", host, duration)
	return nil
}

// checkRemoteConnectivity attempts a short ssh to verify host reachability
func (pm *ProxyManager) checkRemoteConnectivity(host string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "ssh",
		"-o", "ConnectTimeout=5",
		"-o", "ConnectionAttempts=1",
		"-o", "BatchMode=yes",
		"-o", "StrictHostKeyChecking=no",
		host,
		"echo", "ok")
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Map common failure modes to friendly messages
		if ctx.Err() != nil {
			return fmt.Errorf("connection timed out after %v", timeout)
		}
		out := strings.TrimSpace(string(output))
		lo := strings.ToLower(out)
		switch {
		case strings.Contains(lo, "permission denied"):
			return fmt.Errorf("permission denied")
		case strings.Contains(lo, "could not resolve hostname") || strings.Contains(lo, "name or service not known"):
			return fmt.Errorf("unknown host")
		case strings.Contains(lo, "no route to host"):
			return fmt.Errorf("no route to host")
		case strings.Contains(lo, "connection timed out"):
			return fmt.Errorf("connection timed out")
		case strings.Contains(lo, "host key verification failed"):
			return fmt.Errorf("host key verification failed")
		default:
			if out != "" {
				return fmt.Errorf("%s", out)
			}
			return fmt.Errorf("ssh error")
		}
	}
	return nil
}

// Close shuts down all proxy instances
func (pm *ProxyManager) Close() error {
	var lastErr error
	for host, client := range pm.clients {
		if err := client.Close(); err != nil {
			LL.Error("Failed to close proxy for host %s: %v", host, err)
			lastErr = err
		}
	}
	return lastErr
}

// GetFile reads the contents of a file (supports HOST:PATH format)
func (pm *ProxyManager) GetFile(path string) (*FileContent, error) {
	startTime := time.Now()

	client, actualPath, err := pm.getClient(path)
	if err != nil {
		return nil, err
	}

	LL.Debug("ProxyManager.GetFile: path=%s, actualPath=%s", path, actualPath)

	fileContent, err := client.GetFile(actualPath)
	if shouldReconnect(err) {
		host, _, _ := parseRemotePath(path)
		pm.reportStatus("Disconnected")
		if _, rerr := pm.reconnect(host); rerr == nil {
			// Retry once with a fresh client
			if client2, _, gerr := pm.getClient(path); gerr == nil {
				fileContent, err = client2.GetFile(actualPath)
			}
		}
	}
	if err == nil {
		duration := time.Since(startTime)
		LL.Info("Read file %s (%d bytes) in %v", path, len(fileContent.Content), duration)
	}
	return fileContent, err
}

// WriteFile writes content to a file (local or remote) and returns diagnostics
func (pm *ProxyManager) WriteFile(path string, content string) ([]Diagnostic, error) {
	startTime := time.Now()

	client, actualPath, err := pm.getClient(path)
	if err != nil {
		return nil, err
	}

	LL.Debug("ProxyManager.WriteFile: path=%s, actualPath=%s", path, actualPath)

	diagnostics, err := client.WriteFile(actualPath, content)
	if shouldReconnect(err) {
		host, _, _ := parseRemotePath(path)
		pm.reportStatus("Disconnected")
		if _, rerr := pm.reconnect(host); rerr == nil {
			if client2, _, gerr := pm.getClient(path); gerr == nil {
				diagnostics, err = client2.WriteFile(actualPath, content)
			}
		}
	}
	if err == nil {
		duration := time.Since(startTime)
		LL.Info("Wrote file %s in %v", path, duration)
	}
	return diagnostics, err
}

// GetFileAttributes gets attributes of a file or directory (supports HOST:PATH format)
func (pm *ProxyManager) GetFileAttributes(path string) (*FileAttributes, error) {
	startTime := time.Now()

	client, actualPath, err := pm.getClient(path)
	if err != nil {
		return nil, err
	}

	attrs, err := client.GetFileAttributes(actualPath)
	if shouldReconnect(err) {
		host, _, _ := parseRemotePath(path)
		pm.reportStatus("Disconnected")
		if _, rerr := pm.reconnect(host); rerr == nil {
			if client2, _, gerr := pm.getClient(path); gerr == nil {
				attrs, err = client2.GetFileAttributes(actualPath)
			}
		}
	}
	if err == nil {
		duration := time.Since(startTime)
		LL.Info("Got attributes for %s in %v", path, duration)
	}
	return attrs, err
}

// ListDirectoryRecursive lists directory contents recursively (supports HOST:PATH format)
func (pm *ProxyManager) ListDirectoryRecursive(path string) ([]FileInfo, error) {
	return pm.ListDirectoryRecursiveFiltered(path, "supported")
}

// ListDirectoryRecursiveFiltered lists directory contents recursively with filtering (supports HOST:PATH format)
func (pm *ProxyManager) ListDirectoryRecursiveFiltered(path string, filterMode string) ([]FileInfo, error) {
	startTime := time.Now()

	client, actualPath, err := pm.getClient(path)
	if err != nil {
		return nil, err
	}

	// Report status for the actual directory listing operation
	host, _, isRemote := parseRemotePath(path)
	if isRemote {
		pm.reportStatus(fmt.Sprintf("Listing files on %s...", host))
	} else {
		pm.reportStatus("Listing files...")
	}

	files, err := client.ListDirectoryRecursiveFiltered(actualPath, filterMode)
	if shouldReconnect(err) {
		host, _, _ := parseRemotePath(path)
		pm.reportStatus("Disconnected")
		if _, rerr := pm.reconnect(host); rerr == nil {
			if client2, _, gerr := pm.getClient(path); gerr == nil {
				files, err = client2.ListDirectoryRecursiveFiltered(actualPath, filterMode)
			}
		}
	}
	if err == nil {
		duration := time.Since(startTime)
		LL.Info("Listed directory recursively %s (%d files, filter=%s) in %v", path, len(files), filterMode, duration)
	}
	// Clear status after operation completes
	pm.reportStatus("")
	return files, err
}

// GoToDefinition gets the definition location for a symbol (supports HOST:PATH format)
func (pm *ProxyManager) GoToDefinition(filePath string, line int, character int) ([]Location, error) {
	client, actualPath, err := pm.getClient(filePath)
	if err != nil {
		return nil, err
	}
	locations, err := client.GoToDefinition(actualPath, line, character)
	if shouldReconnect(err) {
		host, _, _ := parseRemotePath(filePath)
		pm.reportStatus("Disconnected")
		if _, rerr := pm.reconnect(host); rerr == nil {
			if client2, actualPath2, gerr := pm.getClient(filePath); gerr == nil {
				locations, err = client2.GoToDefinition(actualPath2, line, character)
			}
		}
	}
	return locations, err
}

// FindReferences finds all references to a symbol (supports HOST:PATH format)
func (pm *ProxyManager) FindReferences(filePath string, line int, character int, includeDeclaration bool) ([]Location, error) {
	client, actualPath, err := pm.getClient(filePath)
	if err != nil {
		return nil, err
	}
	locations, err := client.FindReferences(actualPath, line, character, includeDeclaration)
	if shouldReconnect(err) {
		host, _, _ := parseRemotePath(filePath)
		pm.reportStatus("Disconnected")
		if _, rerr := pm.reconnect(host); rerr == nil {
			if client2, actualPath2, gerr := pm.getClient(filePath); gerr == nil {
				locations, err = client2.FindReferences(actualPath2, line, character, includeDeclaration)
			}
		}
	}
	return locations, err
}

// GetCompletions gets completion suggestions at the given position (supports HOST:PATH format)
func (pm *ProxyManager) GetCompletions(filePath string, line int, character int) (interface{}, error) {
	client, actualPath, err := pm.getClient(filePath)
	if err != nil {
		return nil, err
	}
	result, err := client.GetCompletions(actualPath, line, character)
	if shouldReconnect(err) {
		host, _, _ := parseRemotePath(filePath)
		pm.reportStatus("Disconnected")
		if _, rerr := pm.reconnect(host); rerr == nil {
			if client2, actualPath2, gerr := pm.getClient(filePath); gerr == nil {
				result, err = client2.GetCompletions(actualPath2, line, character)
			}
		}
	}
	return result, err
}

// GetDiagnostics gets all diagnostics from all open files
func (pm *ProxyManager) GetDiagnostics(path string) (map[string][]interface{}, error) {
	client, _, err := pm.getClient(path)
	if err != nil {
		return nil, err
	}
	res, err := client.GetDiagnostics()
	if shouldReconnect(err) {
		host, _, _ := parseRemotePath(path)
		pm.reportStatus("Disconnected")
		if _, rerr := pm.reconnect(host); rerr == nil {
			if client2, _, gerr := pm.getClient(path); gerr == nil {
				res, err = client2.GetDiagnostics()
			}
		}
	}
	return res, err
}

// GetDocumentSymbols gets document symbols for a file (supports HOST:PATH format)
func (pm *ProxyManager) GetDocumentSymbols(filePath string) ([]DocumentSymbol, error) {
	client, actualPath, err := pm.getClient(filePath)
	if err != nil {
		return nil, err
	}
	symbols, err := client.GetDocumentSymbols(actualPath)
	if shouldReconnect(err) {
		host, _, _ := parseRemotePath(filePath)
		pm.reportStatus("Disconnected")
		if _, rerr := pm.reconnect(host); rerr == nil {
			if client2, actualPath2, gerr := pm.getClient(filePath); gerr == nil {
				symbols, err = client2.GetDocumentSymbols(actualPath2)
			}
		}
	}
	return symbols, err
}

// GetLSPMessages gets LSP messages for a language
func (pm *ProxyManager) GetLSPMessages(path string, language string) ([]interface{}, error) {
	client, _, err := pm.getClient(path)
	if err != nil {
		return nil, err
	}
	msgs, err := client.GetLSPMessages(language)
	if shouldReconnect(err) {
		host, _, _ := parseRemotePath(path)
		pm.reportStatus("Disconnected")
		if _, rerr := pm.reconnect(host); rerr == nil {
			if client2, _, gerr := pm.getClient(path); gerr == nil {
				msgs, err = client2.GetLSPMessages(language)
			}
		}
	}
	return msgs, err
}

// GetFoldingRanges gets folding ranges for a file (supports HOST:PATH format)
func (pm *ProxyManager) GetFoldingRanges(filePath string) ([]FoldingRange, error) {
	client, actualPath, err := pm.getClient(filePath)
	if err != nil {
		return nil, err
	}
	ranges, err := client.GetFoldingRanges(actualPath)
	if shouldReconnect(err) {
		host, _, _ := parseRemotePath(filePath)
		pm.reportStatus("Disconnected")
		if _, rerr := pm.reconnect(host); rerr == nil {
			if client2, actualPath2, gerr := pm.getClient(filePath); gerr == nil {
				ranges, err = client2.GetFoldingRanges(actualPath2)
			}
		}
	}
	return ranges, err
}

// NotifyFileChange notifies the LSP server about file content changes (supports HOST:PATH format)
func (pm *ProxyManager) NotifyFileChange(filePath string, content string) error {
	client, actualPath, err := pm.getClient(filePath)
	if err != nil {
		return err
	}
	err = client.NotifyFileChange(actualPath, content)
	if shouldReconnect(err) {
		host, _, _ := parseRemotePath(filePath)
		pm.reportStatus("Disconnected")
		if _, rerr := pm.reconnect(host); rerr == nil {
			if client2, actualPath2, gerr := pm.getClient(filePath); gerr == nil {
				err = client2.NotifyFileChange(actualPath2, content)
			}
		}
	}
	return err
}

// GetSignatureHelp gets signature help for a function (supports HOST:PATH format)
func (pm *ProxyManager) GetSignatureHelp(filePath string, line int, character int) (interface{}, error) {
	client, actualPath, err := pm.getClient(filePath)
	if err != nil {
		return nil, err
	}
	resp, err := client.GetSignatureHelp(actualPath, line, character)
	if shouldReconnect(err) {
		host, _, _ := parseRemotePath(filePath)
		pm.reportStatus("Disconnected")
		if _, rerr := pm.reconnect(host); rerr == nil {
			if client2, actualPath2, gerr := pm.getClient(filePath); gerr == nil {
				resp, err = client2.GetSignatureHelp(actualPath2, line, character)
			}
		}
	}
	return resp, err
}

// ListProjectSymbols gets all symbols across the project
func (pm *ProxyManager) ListProjectSymbols(path string, query string) ([]ProjectSymbol, error) {
	client, _, err := pm.getClient(path)
	if err != nil {
		return nil, err
	}
	symbols, err := client.ListProjectSymbols(query)
	if shouldReconnect(err) {
		host, _, _ := parseRemotePath(path)
		pm.reportStatus("Disconnected")
		if _, rerr := pm.reconnect(host); rerr == nil {
			if client2, _, gerr := pm.getClient(path); gerr == nil {
				symbols, err = client2.ListProjectSymbols(query)
			}
		}
	}
	return symbols, err
}

// GetProjectChanges gets all changed files in the project
func (pm *ProxyManager) GetProjectChanges(path string) ([]ProjectChange, error) {
	client, actualPath, err := pm.getClient(path)
	if err != nil {
		return nil, err
	}
	workingDir := actualPath
	if actualPath == "" || actualPath == "." {
		workingDir = path
	}
	changes, err := client.GetProjectChanges(workingDir)
	if shouldReconnect(err) {
		host, _, _ := parseRemotePath(path)
		pm.reportStatus("Disconnected")
		if _, rerr := pm.reconnect(host); rerr == nil {
			if client2, actualPath2, gerr := pm.getClient(path); gerr == nil {
				wd := actualPath2
				if wd == "" || wd == "." {
					wd = path
				}
				changes, err = client2.GetProjectChanges(wd)
			}
		}
	}
	return changes, err
}

// GetGitFileVersion gets a specific git version of a file
func (pm *ProxyManager) GetGitFileVersion(path string, revision string) (string, error) {
	// Don't hold the lock while calling getClient, as it needs to acquire the lock itself
	client, actualPath, err := pm.getClient(path)
	if err != nil {
		return "", err
	}

	return client.GetGitFileVersion(actualPath, revision)
}

//go:embed veda-proxy.py
var embeddedProxy string

// proxyScriptPath is the expanded path to the proxy script for local execution
var proxyScriptPath string

// init extracts the embedded proxy script to ~/.cache/veda/ for local execution
func init() {
	// Get the home directory
	home, err := os.UserHomeDir()
	if err != nil {
		LL.Error("Failed to get home directory: %v", err)
		return
	}

	// Use ~/.cache/veda/ directory
	cacheDir := filepath.Join(home, ".cache", "veda")

	// Create the directory if it doesn't exist
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		LL.Error("Failed to create cache directory %s: %v", cacheDir, err)
		return
	}

	proxyScriptPath = filepath.Join(cacheDir, "veda-proxy.py")

	// Write the embedded proxy script to the fixed location
	if err := os.WriteFile(proxyScriptPath, []byte(embeddedProxy), 0755); err != nil {
		LL.Error("Failed to write proxy script to %s: %v", proxyScriptPath, err)
		return
	}

	LL.Debug("Extracted proxy script to: %s", proxyScriptPath)
}

// JSONRPCRequest represents a JSON RPC request
type JSONRPCRequest struct {
	Method string                 `json:"method"`
	Params map[string]interface{} `json:"params"`
	ID     int                    `json:"id"`
}

// JSONRPCResponse represents a JSON RPC response
type JSONRPCResponse struct {
	ID     int           `json:"id"`
	Result []interface{} `json:"result,omitempty"`
	Error  string        `json:"error,omitempty"`
}

// FileInfo represents file information from the proxy
type FileInfo struct {
	Name  string  `json:"name"`
	Type  string  `json:"type"`
	Size  int64   `json:"size"`
	Path  string  `json:"path"`
	Mtime float64 `json:"mtime"`
}

// FileAttributes represents detailed file/directory attributes
type FileAttributes struct {
	Name        string    `json:"name"`
	Path        string    `json:"path"`
	Type        string    `json:"type"`
	Size        int64     `json:"size"`
	Mode        string    `json:"mode"`
	Modified    time.Time `json:"modified"`
	Created     time.Time `json:"created"`
	Accessed    time.Time `json:"accessed"`
	IsDirectory bool      `json:"is_directory"`
	IsFile      bool      `json:"is_file"`
	IsSymlink   bool      `json:"is_symlink"`
	Exists      bool      `json:"exists"`
	Readable    bool      `json:"readable"`
	Writable    bool      `json:"writable"`
	Executable  bool      `json:"executable"`
}

// FileContent represents file content with change information
type FileContent struct {
	Content     string       `json:"content"`
	Changes     FileChanges  `json:"changes"`
	Branch      string       `json:"branch"`
	Diagnostics []Diagnostic `json:"diagnostics"`
}

// FileChanges represents git changes in a file
type FileChanges struct {
	Added    []int `json:"added"`
	Deleted  []int `json:"deleted"`
	Modified []int `json:"modified"`
}

// Diagnostic represents a single diagnostic item from LSP normalized for the editor
type Diagnostic struct {
	Line      int    `json:"line"`     // 1-based line number
	Severity  string `json:"severity"` // "error", "warning", "info", "hint"
	Message   string `json:"message"`
	StartChar int    `json:"start_char"` // 0-based start character in line
	EndChar   int    `json:"end_char"`   // 0-based end character (exclusive)
}

// ProjectChange represents a changed file in the project
type ProjectChange struct {
	File         string `json:"file"`
	Status       string `json:"status"`
	Staged       bool   `json:"staged"`
	AddedLines   []int  `json:"added_lines"`
	DeletedLines []int  `json:"deleted_lines"`
	Hunks        []Hunk `json:"hunks"`
}

// Hunk represents a git diff hunk
type Hunk struct {
	OldStart int    `json:"old_start"`
	NewStart int    `json:"new_start"`
	Header   string `json:"header"`
}

// Location represents a file location (used for LSP responses)
type Location struct {
	URI   string `json:"uri"`
	Range Range  `json:"range"`
}

// Range represents a text range in a file
type Range struct {
	Start Position `json:"start"`
	End   Position `json:"end"`
}

// Position represents a position in a text document
type Position struct {
	Line      int `json:"line"`
	Character int `json:"character"`
}

// DocumentSymbol represents a symbol in a document (LSP DocumentSymbol)
type DocumentSymbol struct {
	Name           string           `json:"name"`
	Detail         string           `json:"detail,omitempty"`
	Kind           int              `json:"kind"`
	Tags           []int            `json:"tags,omitempty"`
	Deprecated     bool             `json:"deprecated,omitempty"`
	Range          Range            `json:"range"`
	SelectionRange Range            `json:"selectionRange"`
	Children       []DocumentSymbol `json:"children,omitempty"`
	// For SymbolInformation format (older LSP)
	Location *Location `json:"location,omitempty"`
}

// FoldingRange represents a folding range in a document
type FoldingRange struct {
	StartLine      int     `json:"startLine"`
	StartCharacter *int    `json:"startCharacter,omitempty"`
	EndLine        int     `json:"endLine"`
	EndCharacter   *int    `json:"endCharacter,omitempty"`
	Kind           *string `json:"kind,omitempty"` // "comment", "imports", "region"
}

// ProjectSymbol represents a symbol in the project (for fuzzy list)
type ProjectSymbol struct {
	Name      string `json:"name"`      // Display name with kind and file
	RawName   string `json:"raw_name"`  // Raw symbol name without decorations
	Path      string `json:"path"`      // File path
	Line      int    `json:"line"`      // Line number
	Character int    `json:"character"` // Character position
	Kind      int    `json:"kind"`      // LSP symbol kind
	Language  string `json:"language"`  // Language of the file
}

// Client represents a proxy client
type Client struct {
	cmd    *exec.Cmd
	stdin  *bufio.Writer
	stdout *bufio.Scanner

	// Request ID and writer access must be synchronized
	mu        sync.Mutex // Protects requestID and stdin writes
	requestID int

	// concurrent routing
	pendingMu  sync.Mutex
	pending    map[int]chan map[string]interface{}
	readerErr  chan error
	readerOnce sync.Once
}

// getNextRequestID safely increments and returns the next request ID
func (c *Client) getNextRequestID() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.requestID++
	return c.requestID
}

// shouldReconnect determines if an error suggests a dead proxy connection
func shouldReconnect(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	if strings.Contains(s, "proxy reader error") {
		return true
	}
	if strings.Contains(s, "proxy EOF") {
		return true
	}
	if strings.Contains(s, "failed to write request") || strings.Contains(s, "failed to flush request") {
		return true
	}
	return false
}

// reconnect drops and recreates a client for a host ("" for local)
func (pm *ProxyManager) reconnect(host string) (*Client, error) {
	pm.mu.Lock()
	old := pm.clients[host]
	delete(pm.clients, host)
	pm.mu.Unlock()
	if old != nil && old.cmd != nil && old.cmd.Process != nil {
		_ = old.cmd.Process.Kill()
	}
	if host == "" {
		pm.reportStatus("Restarting local proxy...")
	} else {
		pm.reportStatus(fmt.Sprintf("Reconnecting to %s...", host))
	}
	var (
		client *Client
		err    error
	)
	if host == "" {
		client, err = NewClient()
	} else {
		client, err = pm.createRemoteClient(host)
	}
	if err != nil {
		pm.reportStatus("Disconnected")
		// Emit a special failure status so the editor can react (exit with message)
		pm.reportStatus(fmt.Sprintf("__RECONNECT_FAILED__: %s", err.Error()))
		return nil, err
	}
	pm.mu.Lock()
	pm.clients[host] = client
	pm.mu.Unlock()
	if host == "" {
		pm.reportStatus("Connected")
	} else {
		pm.reportStatus(fmt.Sprintf("Connected to %s", host))
	}
	// Clear status so the spinner stops once connected
	pm.reportStatus("")
	return client, nil
}

// ---- Terminal (PTY) session RPCs ----

// StartTerminal starts a PTY-backed terminal session on the proxy host.
// workingDir decides locality via existing client selection.
func (pm *ProxyManager) StartTerminal(workingDir string, command []string, env map[string]string, cols, rows int) (string, int, error) {
	client, actualPath, err := pm.getClient(workingDir)
	if err != nil {
		return "", 0, err
	}
	sid, pid, err := client.startTerminal(actualPath, command, env, cols, rows)
	if shouldReconnect(err) {
		host, _, _ := parseRemotePath(workingDir)
		pm.reportStatus("Disconnected")
		if _, rerr := pm.reconnect(host); rerr == nil {
			if client2, actualPath2, gerr := pm.getClient(workingDir); gerr == nil {
				sid, pid, err = client2.startTerminal(actualPath2, command, env, cols, rows)
			}
		}
	}
	return sid, pid, err
}

func (pm *ProxyManager) WriteTerminal(workingDir, sessionID string, data []byte) error {
	client, _, err := pm.getClient(workingDir)
	if err != nil {
		return err
	}
	err = client.writeTerminal(sessionID, data)
	if shouldReconnect(err) {
		host, _, _ := parseRemotePath(workingDir)
		pm.reportStatus("Disconnected")
		if _, rerr := pm.reconnect(host); rerr == nil {
			if client2, _, gerr := pm.getClient(workingDir); gerr == nil {
				err = client2.writeTerminal(sessionID, data)
			}
		}
	}
	return err
}

func (pm *ProxyManager) ReadTerminal(workingDir, sessionID string, maxBytes int) ([]byte, bool, *int, error) {
	client, _, err := pm.getClient(workingDir)
	if err != nil {
		return nil, false, nil, err
	}
	data, closed, exitCode, err := client.readTerminal(sessionID, maxBytes)
	if shouldReconnect(err) {
		host, _, _ := parseRemotePath(workingDir)
		pm.reportStatus("Disconnected")
		if _, rerr := pm.reconnect(host); rerr == nil {
			if client2, _, gerr := pm.getClient(workingDir); gerr == nil {
				data, closed, exitCode, err = client2.readTerminal(sessionID, maxBytes)
			}
		}
	}
	return data, closed, exitCode, err
}

func (pm *ProxyManager) ResizeTerminal(workingDir, sessionID string, cols, rows int) error {
	client, _, err := pm.getClient(workingDir)
	if err != nil {
		return err
	}
	err = client.resizeTerminal(sessionID, cols, rows)
	if shouldReconnect(err) {
		host, _, _ := parseRemotePath(workingDir)
		pm.reportStatus("Disconnected")
		if _, rerr := pm.reconnect(host); rerr == nil {
			if client2, _, gerr := pm.getClient(workingDir); gerr == nil {
				err = client2.resizeTerminal(sessionID, cols, rows)
			}
		}
	}
	return err
}

func (pm *ProxyManager) CloseTerminal(workingDir, sessionID string) error {
	client, _, err := pm.getClient(workingDir)
	if err != nil {
		return err
	}
	err = client.closeTerminal(sessionID)
	if shouldReconnect(err) {
		host, _, _ := parseRemotePath(workingDir)
		pm.reportStatus("Disconnected")
		if _, rerr := pm.reconnect(host); rerr == nil {
			if client2, _, gerr := pm.getClient(workingDir); gerr == nil {
				err = client2.closeTerminal(sessionID)
			}
		}
	}
	return err
}

// Client-side request helpers
func (c *Client) startTerminal(workingDir string, command []string, env map[string]string, cols, rows int) (string, int, error) {
	id := c.getNextRequestID()
	params := map[string]interface{}{"working_dir": workingDir, "cols": cols, "rows": rows}
	if len(command) > 0 {
		params["command"] = command
	}
	if len(env) > 0 {
		params["env"] = env
	}
	req := JSONRPCRequest{Method: "start_terminal", Params: params, ID: id}
	resp, err := c.sendRequestWithTimeout(req)
	if err != nil {
		return "", 0, err
	}
	if errMsg, ok := resp["error"].(string); ok && errMsg != "" {
		return "", 0, fmt.Errorf("proxy error: %s", errMsg)
	}
	if result, ok := resp["result"].(map[string]interface{}); ok {
		sid, _ := result["session_id"].(string)
		pid := 0
		if p, ok := result["pid"].(float64); ok {
			pid = int(p)
		}
		if sid != "" {
			return sid, pid, nil
		}
	}
	return "", 0, fmt.Errorf("invalid start_terminal response")
}

func (c *Client) writeTerminal(sessionID string, data []byte) error {
	id := c.getNextRequestID()
	b64 := base64StdEncode(data)
	req := JSONRPCRequest{Method: "write_terminal", Params: map[string]interface{}{"session_id": sessionID, "data_b64": b64}, ID: id}
	resp, err := c.sendRequestWithTimeout(req)
	if err != nil {
		return err
	}
	if errMsg, ok := resp["error"].(string); ok && errMsg != "" {
		return fmt.Errorf("proxy error: %s", errMsg)
	}
	return nil
}

func (c *Client) readTerminal(sessionID string, maxBytes int) ([]byte, bool, *int, error) {
	id := c.getNextRequestID()
	if maxBytes <= 0 {
		maxBytes = 65536
	}
	req := JSONRPCRequest{Method: "read_terminal", Params: map[string]interface{}{"session_id": sessionID, "max_bytes": maxBytes}, ID: id}
	resp, err := c.sendRequestWithTimeout(req)
	if err != nil {
		return nil, false, nil, err
	}
	if errMsg, ok := resp["error"].(string); ok && errMsg != "" {
		return nil, false, nil, fmt.Errorf("proxy error: %s", errMsg)
	}
	result, ok := resp["result"].(map[string]interface{})
	if !ok {
		return nil, false, nil, fmt.Errorf("invalid read_terminal response")
	}
	dataB64, _ := result["data_b64"].(string)
	data, decErr := base64StdDecode(dataB64)
	if decErr != nil {
		return nil, false, nil, decErr
	}
	closed, _ := result["closed"].(bool)
	var exitCodePtr *int
	if ec, ok := result["exit_code"].(float64); ok {
		ecInt := int(ec)
		exitCodePtr = &ecInt
	}
	return data, closed, exitCodePtr, nil
}

func (c *Client) resizeTerminal(sessionID string, cols, rows int) error {
	id := c.getNextRequestID()
	req := JSONRPCRequest{Method: "resize_terminal", Params: map[string]interface{}{"session_id": sessionID, "cols": cols, "rows": rows}, ID: id}
	resp, err := c.sendRequestWithTimeout(req)
	if err != nil {
		return err
	}
	if errMsg, ok := resp["error"].(string); ok && errMsg != "" {
		return fmt.Errorf("proxy error: %s", errMsg)
	}
	return nil
}

func (c *Client) closeTerminal(sessionID string) error {
	id := c.getNextRequestID()
	req := JSONRPCRequest{Method: "close_terminal", Params: map[string]interface{}{"session_id": sessionID}, ID: id}
	resp, err := c.sendRequestWithTimeout(req)
	if err != nil {
		return err
	}
	if errMsg, ok := resp["error"].(string); ok && errMsg != "" {
		return fmt.Errorf("proxy error: %s", errMsg)
	}
	return nil
}

// base64 helpers isolated for clarity; stdlib functions are used under the hood
func base64StdEncode(b []byte) string          { return base64.StdEncoding.EncodeToString(b) }
func base64StdDecode(s string) ([]byte, error) { return base64.StdEncoding.DecodeString(s) }

// sendRequestWithTimeout sends a JSON RPC request and waits for response with timeout
func (c *Client) sendRequestWithTimeout(request JSONRPCRequest) (map[string]interface{}, error) {
	// Ensure reader is running exactly once
	c.readerOnce.Do(func() {
		if c.pending == nil {
			c.pending = make(map[int]chan map[string]interface{})
		}
		if c.readerErr == nil {
			c.readerErr = make(chan error, 1)
		}
		go c.readerLoop()
	})

	requestBytes, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Register waiter
	respCh := make(chan map[string]interface{}, 1)
	c.pendingMu.Lock()
	c.pending[request.ID] = respCh
	c.pendingMu.Unlock()

	// Send the request (protect stdin writes with mutex)
	c.mu.Lock()
	_, writeErr := c.stdin.WriteString(string(requestBytes) + "\n")
	if writeErr == nil {
		writeErr = c.stdin.Flush()
	}
	c.mu.Unlock()

	if writeErr != nil {
		c.pendingMu.Lock()
		delete(c.pending, request.ID)
		c.pendingMu.Unlock()
		return nil, fmt.Errorf("failed to send request: %w", writeErr)
	}

	timer := time.NewTimer(ProxyRequestTimeout)
	defer timer.Stop()

	select {
	case resp := <-respCh:
		c.pendingMu.Lock()
		delete(c.pending, request.ID)
		c.pendingMu.Unlock()
		return resp, nil
	case err := <-c.readerErr:
		c.pendingMu.Lock()
		delete(c.pending, request.ID)
		c.pendingMu.Unlock()
		return nil, fmt.Errorf("proxy reader error: %v", err)
	case <-timer.C:
		c.pendingMu.Lock()
		delete(c.pending, request.ID)
		c.pendingMu.Unlock()
		return nil, fmt.Errorf("proxy request timeout after %v", ProxyRequestTimeout)
	}
}

func (c *Client) readerLoop() {
	for {
		if !c.stdout.Scan() {
			if err := c.stdout.Err(); err != nil {
				LL.Debug("Proxy reader scanner error: %v", err)
				select {
				case c.readerErr <- err:
				default:
				}
			} else {
				LL.Debug("Proxy reader: no data received (EOF)")
				select {
				case c.readerErr <- fmt.Errorf("proxy EOF"):
				default:
				}
			}
			return
		}

		raw := c.stdout.Text()
		trimmed := strings.TrimLeft(raw, "\x00\r\n \t")
		if idx := strings.IndexByte(trimmed, '{'); idx > 0 {
			trimmed = trimmed[idx:]
		}
		if trimmed == "" {
			continue
		}

		logText := trimmed
		if len(logText) > 500 {
			logText = logText[:500] + "... (truncated)"
		}
		LL.Trace("Proxy response: %s", logText)

		var msg map[string]interface{}
		if err := json.Unmarshal([]byte(trimmed), &msg); err != nil {
			LL.Debug("Failed to unmarshal proxy response: %v, response length: %d", err, len(trimmed))
			if len(trimmed) > 200 {
				LL.Debug("Raw response (first 200 chars): %q", trimmed[:200])
			} else {
				LL.Debug("Raw response: %q", trimmed)
			}
			continue
		}

		idVal, ok := msg["id"]
		if !ok {
			LL.Debug("Dropping proxy message without id")
			continue
		}

		var id int
		switch v := idVal.(type) {
		case float64:
			id = int(v)
		case int:
			id = v
		case string:
			var x int
			if _, err := fmt.Sscanf(strings.TrimSpace(v), "%d", &x); err != nil {
				LL.Debug("Dropping proxy message with non-integer id: %q", v)
				continue
			}
			id = x
		default:
			LL.Debug("Dropping proxy message with unsupported id type %T", v)
			continue
		}

		c.pendingMu.Lock()
		ch, exists := c.pending[id]
		c.pendingMu.Unlock()
		if !exists {
			LL.Debug("Dropping out-of-band proxy response for id=%d (no waiter)", id)
			continue
		}

		select {
		case ch <- msg:
		default:
			LL.Debug("Response channel full for id=%d; dropping", id)
		}
	}
}

// NewClient creates a new proxy client and starts the proxy process
func NewClient() (*Client, error) {
	cmd := exec.Command("python3", proxyScriptPath)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	stdinPipe, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	// Log any stderr output from the proxy
	go func() {
		scanner := bufio.NewScanner(stderrPipe)
		for scanner.Scan() {
			line := scanner.Text()
			LL.Debug("Proxy stderr: %s", line)
			// Also print to stdout for test visibility
			fmt.Fprintf(os.Stderr, "PROXY STDERR: %s\n", line)
		}
	}()

	stdin := bufio.NewWriter(stdinPipe)
	stdout := bufio.NewScanner(stdoutPipe)

	// Increase scanner buffer size to handle large responses
	buf := make([]byte, 0, maxScanTokenSize)
	stdout.Buffer(buf, maxScanTokenSize)

	// Perform version handshake
	if version, err := readProxyHello(stdout, 5*time.Second); err != nil {
		LL.Debug("Failed to read local proxy hello: %v", err)
	} else if version != ProxyVersion {
		LL.Error("Local proxy version mismatch: got %s, want %s", version, ProxyVersion)
	}

	client := &Client{
		cmd:       cmd,
		stdin:     stdin,
		stdout:    stdout,
		requestID: 0,
		pending:   make(map[int]chan map[string]interface{}),
		readerErr: make(chan error, 1),
	}
	client.readerOnce.Do(func() { go client.readerLoop() })
	return client, nil
}

// GetFileAttributes sends a JSON RPC request to get file/directory attributes
func (c *Client) GetFileAttributes(path string) (*FileAttributes, error) {
	id := c.getNextRequestID()
	request := JSONRPCRequest{
		Method: "get_file_attributes",
		Params: map[string]interface{}{"path": path},
		ID:     id,
	}

	response, err := c.sendRequestWithTimeout(request)
	if err != nil {
		return nil, err
	}

	if errorMsg, ok := response["error"].(string); ok && errorMsg != "" {
		return nil, fmt.Errorf("proxy error: %s", errorMsg)
	}

	attrs := &FileAttributes{}
	if result, ok := response["result"].(map[string]interface{}); ok {
		// Safely extract fields with type checking
		if name, ok := result["name"].(string); ok {
			attrs.Name = name
		}
		if path, ok := result["path"].(string); ok {
			attrs.Path = path
		}
		if fileType, ok := result["type"].(string); ok {
			attrs.Type = fileType
		}
		if size, ok := result["size"].(float64); ok {
			attrs.Size = int64(size)
		}
		if isDir, ok := result["is_directory"].(bool); ok {
			attrs.IsDirectory = isDir
		}
		if isFile, ok := result["is_file"].(bool); ok {
			attrs.IsFile = isFile
		}
		if isSymlink, ok := result["is_symlink"].(bool); ok {
			attrs.IsSymlink = isSymlink
		}
		if exists, ok := result["exists"].(bool); ok {
			attrs.Exists = exists
		}
		if readable, ok := result["readable"].(bool); ok {
			attrs.Readable = readable
		}
		if writable, ok := result["writable"].(bool); ok {
			attrs.Writable = writable
		}
		if executable, ok := result["executable"].(bool); ok {
			attrs.Executable = executable
		}
		if mode, ok := result["mode"].(string); ok {
			attrs.Mode = mode
		}
		if modified, ok := result["modified"].(float64); ok {
			attrs.Modified = time.Unix(int64(modified), 0)
		}
		if created, ok := result["created"].(float64); ok {
			attrs.Created = time.Unix(int64(created), 0)
		}
		if accessed, ok := result["accessed"].(float64); ok {
			attrs.Accessed = time.Unix(int64(accessed), 0)
		}
	}
	return attrs, nil
}

// ListDirectoryRecursive sends a JSON RPC request to get recursive directory listing
func (c *Client) ListDirectoryRecursive(path string) ([]FileInfo, error) {
	return c.ListDirectoryRecursiveFiltered(path, "supported")
}

// ListDirectoryRecursiveFiltered sends a JSON RPC request to get recursive directory listing with filtering
func (c *Client) ListDirectoryRecursiveFiltered(path string, filterMode string) ([]FileInfo, error) {
	id := c.getNextRequestID()
	request := JSONRPCRequest{
		Method: "list_directory_recursive",
		Params: map[string]interface{}{"path": path, "filter_mode": filterMode},
		ID:     id,
	}

	response, err := c.sendRequestWithTimeout(request)
	if err != nil {
		return nil, err
	}

	if errorMsg, ok := response["error"].(string); ok && errorMsg != "" {
		return nil, fmt.Errorf("proxy error: %s", errorMsg)
	}

	var files []FileInfo
	if result, ok := response["result"].([]interface{}); ok {
		for _, item := range result {
			if fileMap, ok := item.(map[string]interface{}); ok {
				file := FileInfo{}

				// Safely extract fields with type checking
				if name, ok := fileMap["name"].(string); ok {
					file.Name = name
				}
				if fileType, ok := fileMap["type"].(string); ok {
					file.Type = fileType
				}
				if path, ok := fileMap["path"].(string); ok {
					file.Path = path
				}
				if size, ok := fileMap["size"].(float64); ok {
					file.Size = int64(size)
				}

				// Skip entries with missing required fields
				if file.Name != "" && file.Type != "" {
					files = append(files, file)
				}
			}
		}
	}
	return files, nil
}

// parseRemotePath parses a path and returns (host, path, isRemote)
// Examples: "host:path/to/file" -> ("host", "path/to/file", true)
//
//	"/local/path" -> ("", "/local/path", false)
func parseRemotePath(path string) (string, string, bool) {
	// Check if path contains a colon that's not part of a Windows drive (C:)
	colonIdx := strings.Index(path, ":")
	if colonIdx > 0 {
		// Check if it's not a Windows drive letter
		if colonIdx != 1 || (colonIdx == 1 && len(path) > 2 && path[2] != '/' && path[2] != '\\') {
			host := path[:colonIdx]
			remotePath := path[colonIdx+1:]
			// Basic validation - host should not contain slashes
			if !strings.Contains(host, "/") && !strings.Contains(host, "\\") {
				return host, remotePath, true
			}
		}
	}
	return "", path, false
}

// GetFile reads the contents of a file and returns git change information
func (c *Client) GetFile(path string) (*FileContent, error) {
	LL.Debug("Client.GetFile: path=%s", path)

	id := c.getNextRequestID()
	request := JSONRPCRequest{
		Method: "get_file",
		Params: map[string]interface{}{"path": path},
		ID:     id,
	}

	response, err := c.sendRequestWithTimeout(request)
	if err != nil {
		return nil, err
	}

	if errorMsg, ok := response["error"].(string); ok && errorMsg != "" {
		return nil, fmt.Errorf("proxy error: %s", errorMsg)
	}

	fileContent := &FileContent{
		Changes: FileChanges{
			Added:    []int{},
			Deleted:  []int{},
			Modified: []int{},
		},
		Diagnostics: []Diagnostic{},
	}

	// Parse result
	if result, ok := response["result"].(map[string]interface{}); ok {
		if content, ok := result["content"].(string); ok {
			fileContent.Content = content
		} else {
			return nil, fmt.Errorf("missing content in response")
		}

		// Parse branch if present
		if branch, ok := result["branch"].(string); ok {
			fileContent.Branch = branch
		} else {
			fileContent.Branch = "main" // Default
		}

		// Parse changes if present
		if changes, ok := result["changes"].(map[string]interface{}); ok {
			if added, ok := changes["added"].([]interface{}); ok {
				for _, v := range added {
					if line, ok := v.(float64); ok {
						fileContent.Changes.Added = append(fileContent.Changes.Added, int(line))
					}
				}
			}
			if deleted, ok := changes["deleted"].([]interface{}); ok {
				for _, v := range deleted {
					if line, ok := v.(float64); ok {
						fileContent.Changes.Deleted = append(fileContent.Changes.Deleted, int(line))
					}
				}
			}
			if modified, ok := changes["modified"].([]interface{}); ok {
				for _, v := range modified {
					if line, ok := v.(float64); ok {
						fileContent.Changes.Modified = append(fileContent.Changes.Modified, int(line))
					}
				}
			}
		}

		// Parse diagnostics if present
		if diags, ok := result["diagnostics"].([]interface{}); ok {
			for _, item := range diags {
				if dm, ok := item.(map[string]interface{}); ok {
					d := Diagnostic{}
					if ln, ok := dm["line"].(float64); ok {
						d.Line = int(ln)
					}
					if sev, ok := dm["severity"].(string); ok {
						d.Severity = sev
					}
					if msg, ok := dm["message"].(string); ok {
						d.Message = msg
					}
					if sc, ok := dm["start_char"].(float64); ok {
						d.StartChar = int(sc)
					}
					if ec, ok := dm["end_char"].(float64); ok {
						d.EndChar = int(ec)
					}
					if d.Line > 0 && d.Severity != "" {
						fileContent.Diagnostics = append(fileContent.Diagnostics, d)
					}
				}
			}
		}
	}

	return fileContent, nil
}

// StartLSPServer starts an LSP server for the specified language
func (c *Client) StartLSPServer(language string) error {
	id := c.getNextRequestID()
	request := JSONRPCRequest{
		Method: "start_lsp_server",
		Params: map[string]interface{}{"language": language},
		ID:     id,
	}

	response, err := c.sendRequestWithTimeout(request)
	if err != nil {
		return err
	}

	if errorMsg, ok := response["error"].(string); ok && errorMsg != "" {
		return fmt.Errorf("proxy error: %s", errorMsg)
	}

	if result, ok := response["result"].(map[string]interface{}); ok {
		if success, ok := result["success"].(bool); ok && success {
			return nil
		}
	}

	return fmt.Errorf("failed to start LSP server for %s", language)
}

// StopLSPServer stops the LSP server for the specified language
func (c *Client) StopLSPServer(language string) error {
	id := c.getNextRequestID()
	request := JSONRPCRequest{
		Method: "stop_lsp_server",
		Params: map[string]interface{}{"language": language},
		ID:     id,
	}

	response, err := c.sendRequestWithTimeout(request)
	if err != nil {
		return err
	}

	if errorMsg, ok := response["error"].(string); ok && errorMsg != "" {
		return fmt.Errorf("proxy error: %s", errorMsg)
	}

	if result, ok := response["result"].(map[string]interface{}); ok {
		if success, ok := result["success"].(bool); ok && success {
			return nil
		}
	}

	return fmt.Errorf("failed to stop LSP server for %s", language)
}

// GoToDefinition gets the definition location for a symbol at the given position
func (c *Client) GoToDefinition(filePath string, line int, character int) ([]Location, error) {
	id := c.getNextRequestID()
	request := JSONRPCRequest{
		Method: "go_to_definition",
		Params: map[string]interface{}{
			"file_path": filePath,
			"line":      line,
			"character": character,
			// propagate timeout to proxy (milliseconds)
			"timeout_ms": int(ProxyRequestTimeout / time.Millisecond),
		},
		ID: id,
	}

	response, err := c.sendRequestWithTimeout(request)
	if err != nil {
		return nil, err
	}

	if errorMsg, ok := response["error"].(string); ok && errorMsg != "" {
		return nil, fmt.Errorf("proxy error: %s", errorMsg)
	}

	if result, ok := response["result"].(map[string]interface{}); ok {
		if success, ok := result["success"].(bool); ok && !success {
			if errMsg, ok := result["error"].(string); ok {
				return nil, fmt.Errorf("LSP error: %s", errMsg)
			}
			return nil, fmt.Errorf("failed to get definition")
		}

		// Parse the LSP result
		var locations []Location
		if lspResult, ok := result["result"].([]interface{}); ok {
			for _, item := range lspResult {
				if locMap, ok := item.(map[string]interface{}); ok {
					loc := Location{}
					if uri, ok := locMap["uri"].(string); ok {
						loc.URI = uri
					}
					if rangeMap, ok := locMap["range"].(map[string]interface{}); ok {
						if startMap, ok := rangeMap["start"].(map[string]interface{}); ok {
							if line, ok := startMap["line"].(float64); ok {
								loc.Range.Start.Line = int(line)
							}
							if char, ok := startMap["character"].(float64); ok {
								loc.Range.Start.Character = int(char)
							}
						}
						if endMap, ok := rangeMap["end"].(map[string]interface{}); ok {
							if line, ok := endMap["line"].(float64); ok {
								loc.Range.End.Line = int(line)
							}
							if char, ok := endMap["character"].(float64); ok {
								loc.Range.End.Character = int(char)
							}
						}
					}
					locations = append(locations, loc)
				}
			}
		} else if locMap, ok := result["result"].(map[string]interface{}); ok {
			// Single location response
			loc := Location{}
			if uri, ok := locMap["uri"].(string); ok {
				loc.URI = uri
			}
			if rangeMap, ok := locMap["range"].(map[string]interface{}); ok {
				if startMap, ok := rangeMap["start"].(map[string]interface{}); ok {
					if line, ok := startMap["line"].(float64); ok {
						loc.Range.Start.Line = int(line)
					}
					if char, ok := startMap["character"].(float64); ok {
						loc.Range.Start.Character = int(char)
					}
				}
				if endMap, ok := rangeMap["end"].(map[string]interface{}); ok {
					if line, ok := endMap["line"].(float64); ok {
						loc.Range.End.Line = int(line)
					}
					if char, ok := endMap["character"].(float64); ok {
						loc.Range.End.Character = int(char)
					}
				}
			}
			if loc.URI != "" {
				locations = append(locations, loc)
			}
		}

		return locations, nil
	}

	return nil, fmt.Errorf("failed to get definition")
}

// FindReferences finds all references to a symbol at the given position
func (c *Client) FindReferences(filePath string, line int, character int, includeDeclaration bool) ([]Location, error) {
	id := c.getNextRequestID()
	request := JSONRPCRequest{
		Method: "find_references",
		Params: map[string]interface{}{
			"file_path":           filePath,
			"line":                line,
			"character":           character,
			"include_declaration": includeDeclaration,
			"timeout_ms":          int(ProxyRequestTimeout / time.Millisecond),
		},
		ID: id,
	}

	response, err := c.sendRequestWithTimeout(request)
	if err != nil {
		return nil, err
	}

	if errorMsg, ok := response["error"].(string); ok && errorMsg != "" {
		return nil, fmt.Errorf("proxy error: %s", errorMsg)
	}

	if result, ok := response["result"].(map[string]interface{}); ok {
		if success, ok := result["success"].(bool); ok && !success {
			if errMsg, ok := result["error"].(string); ok {
				return nil, fmt.Errorf("LSP error: %s", errMsg)
			}
			return nil, fmt.Errorf("failed to find references")
		}

		// Parse the LSP result
		var locations []Location
		if lspResult, ok := result["result"].([]interface{}); ok {
			for _, item := range lspResult {
				if locMap, ok := item.(map[string]interface{}); ok {
					loc := Location{}
					if uri, ok := locMap["uri"].(string); ok {
						loc.URI = uri
					}
					if rangeMap, ok := locMap["range"].(map[string]interface{}); ok {
						if startMap, ok := rangeMap["start"].(map[string]interface{}); ok {
							if line, ok := startMap["line"].(float64); ok {
								loc.Range.Start.Line = int(line)
							}
							if char, ok := startMap["character"].(float64); ok {
								loc.Range.Start.Character = int(char)
							}
						}
						if endMap, ok := rangeMap["end"].(map[string]interface{}); ok {
							if line, ok := endMap["line"].(float64); ok {
								loc.Range.End.Line = int(line)
							}
							if char, ok := endMap["character"].(float64); ok {
								loc.Range.End.Character = int(char)
							}
						}
					}
					locations = append(locations, loc)
				}
			}
		}

		return locations, nil
	}

	return nil, fmt.Errorf("failed to find references")
}

// GetLSPMessages gets all LSP messages for a specific language
func (c *Client) GetLSPMessages(language string) ([]interface{}, error) {
	id := c.getNextRequestID()
	request := JSONRPCRequest{
		Method: "get_lsp_messages",
		Params: map[string]interface{}{"language": language},
		ID:     id,
	}

	response, err := c.sendRequestWithTimeout(request)
	if err != nil {
		return nil, err
	}

	if errorMsg, ok := response["error"].(string); ok && errorMsg != "" {
		return nil, fmt.Errorf("proxy error: %s", errorMsg)
	}

	if result, ok := response["result"].(map[string]interface{}); ok {
		if success, ok := result["success"].(bool); ok && success {
			if messages, ok := result["messages"].([]interface{}); ok {
				return messages, nil
			}
		}
	}

	return nil, fmt.Errorf("failed to get LSP messages")
}

// GetLSPNotifications gets pending showMessage notifications for a language
func (c *Client) GetLSPNotifications(language string) ([]map[string]interface{}, error) {
	id := c.getNextRequestID()
	request := JSONRPCRequest{
		Method: "get_lsp_notifications",
		Params: map[string]interface{}{"language": language},
		ID:     id,
	}

	response, err := c.sendRequestWithTimeout(request)
	if err != nil {
		return nil, err
	}

	if errorMsg, ok := response["error"].(string); ok && errorMsg != "" {
		return nil, fmt.Errorf("proxy error: %s", errorMsg)
	}

	if result, ok := response["result"].(map[string]interface{}); ok {
		if success, ok := result["success"].(bool); ok && success {
			if notifications, ok := result["notifications"].([]interface{}); ok {
				var notifResult []map[string]interface{}
				for _, n := range notifications {
					if notif, ok := n.(map[string]interface{}); ok {
						notifResult = append(notifResult, notif)
					}
				}
				return notifResult, nil
			}
		}
	}

	return nil, fmt.Errorf("failed to get LSP notifications")
}

// GetDocumentSymbols gets document symbols (functions, classes, etc.) for a file
func (c *Client) GetDocumentSymbols(filePath string) ([]DocumentSymbol, error) {
	id := c.getNextRequestID()
	request := JSONRPCRequest{
		Method: "get_document_symbols",
		Params: map[string]interface{}{
			"file_path":  filePath,
			"timeout_ms": int(ProxyRequestTimeout / time.Millisecond),
		},
		ID: id,
	}

	response, err := c.sendRequestWithTimeout(request)
	if err != nil {
		return nil, err
	}

	if errorMsg, ok := response["error"].(string); ok && errorMsg != "" {
		return nil, fmt.Errorf("proxy error: %s", errorMsg)
	}

	if result, ok := response["result"].(map[string]interface{}); ok {
		if success, ok := result["success"].(bool); ok && success {
			if lspResult, ok := result["result"].([]interface{}); ok {
				// Convert the result to DocumentSymbol objects
				symbols := make([]DocumentSymbol, 0, len(lspResult))
				for _, item := range lspResult {
					symbolData, err := json.Marshal(item)
					if err != nil {
						continue
					}
					var symbol DocumentSymbol
					if err := json.Unmarshal(symbolData, &symbol); err != nil {
						continue
					}

					// If this is SymbolInformation format (has location), convert it
					if symbol.Location != nil && symbol.Range.Start.Line == 0 && symbol.Range.End.Line == 0 {
						symbol.Range = symbol.Location.Range
					}

					symbols = append(symbols, symbol)
				}
				return symbols, nil
			}
		}

		if errorMsg, ok := result["error"].(string); ok {
			return nil, fmt.Errorf("LSP error: %s", errorMsg)
		}
	}

	return nil, fmt.Errorf("failed to get document symbols")
}

// GetFoldingRanges gets folding ranges for a file
func (c *Client) GetFoldingRanges(filePath string) ([]FoldingRange, error) {
	id := c.getNextRequestID()
	request := JSONRPCRequest{
		Method: "get_folding_ranges",
		Params: map[string]interface{}{
			"file_path":  filePath,
			"timeout_ms": int(ProxyRequestTimeout / time.Millisecond),
		},
		ID: id,
	}

	response, err := c.sendRequestWithTimeout(request)
	if err != nil {
		return nil, err
	}

	if errorMsg, ok := response["error"].(string); ok && errorMsg != "" {
		return nil, fmt.Errorf("proxy error: %s", errorMsg)
	}

	if result, ok := response["result"].(map[string]interface{}); ok {
		if success, ok := result["success"].(bool); ok && success {
			if lspResult, ok := result["result"].([]interface{}); ok {
				// Convert the result to FoldingRange objects
				ranges := make([]FoldingRange, 0, len(lspResult))
				for _, item := range lspResult {
					rangeData, err := json.Marshal(item)
					if err != nil {
						continue
					}
					var foldingRange FoldingRange
					if err := json.Unmarshal(rangeData, &foldingRange); err != nil {
						continue
					}
					ranges = append(ranges, foldingRange)
				}
				return ranges, nil
			}
		}

		if errorMsg, ok := result["error"].(string); ok {
			return nil, fmt.Errorf("LSP error: %s", errorMsg)
		}
	}

	return nil, fmt.Errorf("failed to get folding ranges")
}

// NotifyFileChange notifies the LSP server about file content changes
func (c *Client) NotifyFileChange(filePath string, content string) error {
	id := c.getNextRequestID()
	request := JSONRPCRequest{
		Method: "notify_file_change",
		Params: map[string]interface{}{
			"file_path": filePath,
			"content":   content,
		},
		ID: id,
	}

	response, err := c.sendRequestWithTimeout(request)
	if err != nil {
		return err
	}

	if errorMsg, ok := response["error"]; ok && errorMsg != nil {
		return fmt.Errorf("proxy error: %v", errorMsg)
	}

	if result, ok := response["result"].(map[string]interface{}); ok {
		if status, ok := result["status"].(string); ok {
			if status == "success" {
				LL.Trace("Successfully sent didChange notification for %s", filePath)
				return nil
			} else if status == "skipped" {
				// Not an error, just no LSP for this file
				return nil
			}
		}
		if errorMsg, ok := result["error"].(string); ok {
			return fmt.Errorf("%s", errorMsg)
		}
	}

	return nil
}

// GetSignatureHelp gets signature help for a function at the given position
func (c *Client) GetSignatureHelp(filePath string, line int, character int) (interface{}, error) {
	id := c.getNextRequestID()
	request := JSONRPCRequest{
		Method: "get_signature_help",
		Params: map[string]interface{}{
			"file_path":  filePath,
			"line":       line,
			"character":  character,
			"timeout_ms": int(ProxyRequestTimeout / time.Millisecond),
		},
		ID: id,
	}

	response, err := c.sendRequestWithTimeout(request)
	if err != nil {
		return nil, err
	}

	if errorMsg, ok := response["error"]; ok && errorMsg != nil {
		return nil, fmt.Errorf("proxy error: %v", errorMsg)
	}

	if result, ok := response["result"].(map[string]interface{}); ok {
		if result["success"] == true && result["result"] != nil {
			return result["result"], nil
		}
		if errorMsg, ok := result["error"].(string); ok {
			return nil, fmt.Errorf("%s", errorMsg)
		}
	}

	return nil, fmt.Errorf("failed to get signature help")
}

// ListProjectSymbols gets all symbols across the entire project
func (c *Client) ListProjectSymbols(query string) ([]ProjectSymbol, error) {
	id := c.getNextRequestID()
	request := JSONRPCRequest{
		Method: "list_project_symbols",
		Params: map[string]interface{}{
			"query":      query,
			"timeout_ms": int(ProxyRequestTimeout / time.Millisecond),
		},
		ID: id,
	}

	response, err := c.sendRequestWithTimeout(request)
	if err != nil {
		return nil, err
	}

	if errorMsg, ok := response["error"].(string); ok && errorMsg != "" {
		return nil, fmt.Errorf("proxy error: %s", errorMsg)
	}

	if result, ok := response["result"].(map[string]interface{}); ok {
		if success, ok := result["success"].(bool); ok && success {
			if symbolsData, ok := result["symbols"].([]interface{}); ok {
				// Convert the result to ProjectSymbol objects
				symbols := make([]ProjectSymbol, 0, len(symbolsData))
				for _, item := range symbolsData {
					symbolBytes, err := json.Marshal(item)
					if err != nil {
						continue
					}
					var symbol ProjectSymbol
					if err := json.Unmarshal(symbolBytes, &symbol); err != nil {
						continue
					}
					symbols = append(symbols, symbol)
				}
				return symbols, nil
			}
		}

		if errorMsg, ok := result["error"].(string); ok {
			return nil, fmt.Errorf("LSP error: %s", errorMsg)
		}
	}

	return nil, fmt.Errorf("no symbols found in project")
}

// WriteFile writes content to a file and returns diagnostics
func (c *Client) WriteFile(path string, content string) ([]Diagnostic, error) {
	id := c.getNextRequestID()
	request := JSONRPCRequest{
		Method: "write_file",
		Params: map[string]interface{}{
			"path":    path,
			"content": content,
		},
		ID: id,
	}

	response, err := c.sendRequestWithTimeout(request)
	if err != nil {
		return nil, err
	}

	if errorMsg, ok := response["error"].(string); ok && errorMsg != "" {
		return nil, fmt.Errorf("proxy error: %s", errorMsg)
	}

	var diagnostics []Diagnostic

	if result, ok := response["result"].(map[string]interface{}); ok {
		if success, ok := result["success"].(bool); ok && success {
			// Parse diagnostics if present
			if diags, ok := result["diagnostics"].([]interface{}); ok {
				for _, item := range diags {
					if dm, ok := item.(map[string]interface{}); ok {
						d := Diagnostic{}
						if ln, ok := dm["line"].(float64); ok {
							d.Line = int(ln)
						}
						if sev, ok := dm["severity"].(string); ok {
							d.Severity = sev
						}
						if msg, ok := dm["message"].(string); ok {
							d.Message = msg
						}
						if sc, ok := dm["start_char"].(float64); ok {
							d.StartChar = int(sc)
						}
						if ec, ok := dm["end_char"].(float64); ok {
							d.EndChar = int(ec)
						}
						if d.Line > 0 && d.Severity != "" {
							diagnostics = append(diagnostics, d)
						}
					}
				}
			}
			return diagnostics, nil
		}
	}

	return nil, fmt.Errorf("failed to write file")
}

// GetCompletions gets completion suggestions at the given position
func (c *Client) GetCompletions(filePath string, line int, character int) (interface{}, error) {
	id := c.getNextRequestID()
	request := JSONRPCRequest{
		Method: "get_completions",
		Params: map[string]interface{}{
			"file_path":  filePath,
			"line":       line,
			"character":  character,
			"timeout_ms": int(ProxyRequestTimeout / time.Millisecond),
		},
		ID: id,
	}

	response, err := c.sendRequestWithTimeout(request)
	if err != nil {
		return nil, err
	}

	if errorMsg, ok := response["error"].(string); ok && errorMsg != "" {
		return nil, fmt.Errorf("proxy error: %s", errorMsg)
	}

	if result, ok := response["result"].(map[string]interface{}); ok {
		if success, ok := result["success"].(bool); ok && success {
			if lspResult, ok := result["result"]; ok {
				return lspResult, nil
			}
		}
	}

	return nil, fmt.Errorf("no completions found")
}

// GetDiagnostics retrieves all diagnostics from all open files
func (c *Client) GetDiagnostics() (map[string][]interface{}, error) {
	id := c.getNextRequestID()
	request := JSONRPCRequest{
		Method: "get_diagnostics",
		Params: map[string]interface{}{},
		ID:     id,
	}

	response, err := c.sendRequestWithTimeout(request)
	if err != nil {
		return nil, err
	}

	if errorMsg, ok := response["error"].(string); ok && errorMsg != "" {
		return nil, fmt.Errorf("proxy error: %s", errorMsg)
	}

	if result, ok := response["result"].(map[string]interface{}); ok {
		if success, ok := result["success"].(bool); ok && success {
			if diagnostics, ok := result["diagnostics"].(map[string]interface{}); ok {
				// Convert map[string]interface{} to map[string][]interface{}
				diagResult := make(map[string][]interface{})
				for uri, diags := range diagnostics {
					if diagList, ok := diags.([]interface{}); ok {
						diagResult[uri] = diagList
					}
				}
				return diagResult, nil
			}
		}
	}

	return nil, fmt.Errorf("no diagnostics found")
}

// GetProjectChanges retrieves all changes in the project
func (c *Client) GetProjectChanges(workingDir string) ([]ProjectChange, error) {
	id := c.getNextRequestID()
	request := JSONRPCRequest{
		Method: "get_project_changes",
		Params: map[string]interface{}{
			"working_dir": workingDir,
		},
		ID: id,
	}

	response, err := c.sendRequestWithTimeout(request)
	if err != nil {
		return nil, err
	}

	if errorMsg, ok := response["error"].(string); ok && errorMsg != "" {
		return nil, fmt.Errorf("proxy error: %s", errorMsg)
	}

	if result, ok := response["result"].(map[string]interface{}); ok {
		if success, ok := result["success"].(bool); ok && success {
			if changes, ok := result["changes"].([]interface{}); ok {
				var projectChanges []ProjectChange
				for _, change := range changes {
					if changeMap, ok := change.(map[string]interface{}); ok {
						pc := ProjectChange{
							File:         getString(changeMap, "file"),
							Status:       getString(changeMap, "status"),
							Staged:       getBool(changeMap, "staged"),
							AddedLines:   getIntSlice(changeMap, "added_lines"),
							DeletedLines: getIntSlice(changeMap, "deleted_lines"),
						}
						// Parse hunks
						if hunks, ok := changeMap["hunks"].([]interface{}); ok {
							for _, hunk := range hunks {
								if hunkMap, ok := hunk.(map[string]interface{}); ok {
									h := Hunk{
										OldStart: getInt(hunkMap, "old_start"),
										NewStart: getInt(hunkMap, "new_start"),
										Header:   getString(hunkMap, "header"),
									}
									pc.Hunks = append(pc.Hunks, h)
								}
							}
						}
						projectChanges = append(projectChanges, pc)
					}
				}
				return projectChanges, nil
			}
		}
	}

	return nil, fmt.Errorf("no changes found")
}

// GetGitFileVersion gets a specific git version of a file
func (c *Client) GetGitFileVersion(path string, revision string) (string, error) {
	c.requestID++
	request := JSONRPCRequest{
		Method: "get_git_file_version",
		Params: map[string]interface{}{
			"path":     path,
			"revision": revision,
		},
		ID: c.requestID,
	}

	response, err := c.sendRequestWithTimeout(request)
	if err != nil {
		return "", err
	}

	if errorMsg, ok := response["error"].(string); ok && errorMsg != "" {
		return "", fmt.Errorf("proxy error: %s", errorMsg)
	}

	if result, ok := response["result"].(map[string]interface{}); ok {
		if success, ok := result["success"].(bool); ok && success {
			if content, ok := result["content"].(string); ok {
				return content, nil
			}
		}
		if errMsg, ok := result["error"].(string); ok {
			return "", fmt.Errorf("git error: %s", errMsg)
		}
	}

	return "", fmt.Errorf("failed to get git file version")
}

func (c *Client) Close() error {
	if c.cmd != nil && c.cmd.Process != nil {
		return c.cmd.Process.Kill()
	}
	return nil
}

// Helper functions for parsing JSON
func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func getBool(m map[string]interface{}, key string) bool {
	if v, ok := m[key].(bool); ok {
		return v
	}
	return false
}

func getInt(m map[string]interface{}, key string) int {
	if v, ok := m[key].(float64); ok {
		return int(v)
	}
	return 0
}

func getIntSlice(m map[string]interface{}, key string) []int {
	var result []int
	if v, ok := m[key].([]interface{}); ok {
		for _, item := range v {
			if num, ok := item.(float64); ok {
				result = append(result, int(num))
			}
		}
	}
	return result
}

// readProxyHello waits for an initial JSON line with a version field from the proxy.
// The proxy prints this as the very first line on startup.
func readProxyHello(scanner *bufio.Scanner, timeout time.Duration) (string, error) {
	type helloMsg struct {
		Version string `json:"version"`
	}

	lineCh := make(chan string, 1)
	errCh := make(chan error, 1)

	go func() {
		if scanner.Scan() {
			lineCh <- scanner.Text()
			return
		}
		if err := scanner.Err(); err != nil {
			errCh <- err
		} else {
			errCh <- fmt.Errorf("no data received")
		}
	}()

	select {
	case line := <-lineCh:
		// sanitize leading NULs/whitespace and start from first '{'
		line = strings.TrimLeft(line, "\x00\r\n \t")
		if idx := strings.IndexByte(line, '{'); idx > 0 {
			line = line[idx:]
		}
		var h helloMsg
		if err := json.Unmarshal([]byte(line), &h); err != nil {
			return "", fmt.Errorf("invalid hello: %w", err)
		}
		if h.Version == "" {
			return "", fmt.Errorf("hello missing version")
		}
		return h.Version, nil
	case err := <-errCh:
		return "", err
	case <-time.After(timeout):
		return "", fmt.Errorf("hello timeout after %v", timeout)
	}
}
