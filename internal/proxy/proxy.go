package proxy

import (
	"bufio"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"
	"veda/internal/log"
)

var LL *log.Logger = log.NewLogger("PROXY")

// maxScanTokenSize defines the maximum size for scanner buffers (1 GiB)
const maxScanTokenSize = 1024 * 1024 * 1024 // 1 GiB

// ProxyManager manages multiple proxy instances for different hosts
type ProxyManager struct {
	clients map[string]*Client // key is hostname, empty string for local
	mu      sync.Mutex         // mutex to ensure only one proxy call at a time
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

// getClient returns the appropriate client for the given path
func (pm *ProxyManager) getClient(path string) (*Client, string, error) {
	host, actualPath, isRemote := parseRemotePath(path)
	
	LL.Debug("getClient: path=%s, host=%s, actualPath=%s, isRemote=%v", path, host, actualPath, isRemote)
	
	if !isRemote {
		return pm.clients[""], path, nil
	}
	
	// Check if we already have a client for this host
	if client, exists := pm.clients[host]; exists {
		return client, actualPath, nil
	}
	
	// Create a new remote proxy for this host
	client, err := pm.createRemoteClient(host)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create remote proxy for %s: %w", host, err)
	}
	
	pm.clients[host] = client
	return client, actualPath, nil
}

// createRemoteClient creates a new proxy client connected to a remote host
func (pm *ProxyManager) createRemoteClient(host string) (*Client, error) {
	startTime := time.Now()
	LL.Info("Creating remote proxy connection to %s", host)
	
	// First, copy the proxy script to the remote host
	if err := copyProxyToRemote(host); err != nil {
		return nil, fmt.Errorf("failed to copy proxy to remote: %w", err)
	}
	
	// Start SSH connection and run the proxy
	sshArgs := []string{
		host,
		"-o", "ControlMaster=no",
		"-o", "ControlPath=none",
		"python3", "/tmp/veda-proxy.py",
	}
	
	cmd := exec.Command("ssh", sshArgs...)
	
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
	
	client := &Client{
		cmd:       cmd,
		stdin:     stdin,
		stdout:    stdout,
		requestID: 0,
	}
	
	duration := time.Since(startTime)
	LL.Info("Created remote proxy connection to %s in %v", host, duration)
	
	return client, nil
}

// copyProxyToRemote copies the embedded proxy script to the remote host
func copyProxyToRemote(host string) error {
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
	
	// Use scp to copy the file to remote /tmp/
	scpCmd := exec.Command("scp", 
		"-o", "ControlMaster=no",
		"-o", "ControlPath=none",
		tmpFile.Name(), 
		fmt.Sprintf("%s:/tmp/veda-proxy.py", host))
	
	// Capture output for debugging
	output, err := scpCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("scp failed: %w, output: %s", err, output)
	}
	
	// Make the script executable on the remote host
	chmodCmd := exec.Command("ssh",
		host,
		"-o", "ControlMaster=no", 
		"-o", "ControlPath=none",
		"chmod", "+x", "/tmp/veda-proxy.py")
	
	output, err = chmodCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("chmod failed: %w, output: %s", err, output)
	}
	
	duration := time.Since(startTime)
	LL.Info("Copied proxy script to %s:/tmp/veda-proxy.py in %v", host, duration)
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
	pm.mu.Lock()
	defer pm.mu.Unlock()
	
	startTime := time.Now()
	
	client, actualPath, err := pm.getClient(path)
	if err != nil {
		return nil, err
	}
	
	LL.Debug("ProxyManager.GetFile: path=%s, actualPath=%s", path, actualPath)
	
	// Call the original GetFile with the actual path (without HOST: prefix)
	fileContent, err := client.GetFile(actualPath)
	if err == nil {
		duration := time.Since(startTime)
		LL.Info("Read file %s (%d bytes) in %v", path, len(fileContent.Content), duration)
	}
	
	return fileContent, err
}

// WriteFile writes content to a file (local or remote)
func (pm *ProxyManager) WriteFile(path string, content string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	
	startTime := time.Now()
	
	client, actualPath, err := pm.getClient(path)
	if err != nil {
		return err
	}
	
	LL.Debug("ProxyManager.WriteFile: path=%s, actualPath=%s", path, actualPath)
	
	// Call the client's WriteFile with the actual path (without HOST: prefix)
	err = client.WriteFile(actualPath, content)
	if err == nil {
		duration := time.Since(startTime)
		LL.Info("Wrote file %s in %v", path, duration)
	}
	return err
}

// ListDirectory lists directory contents (supports HOST:PATH format)
func (pm *ProxyManager) ListDirectory(path string) ([]FileInfo, error) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	
	startTime := time.Now()
	
	client, actualPath, err := pm.getClient(path)
	if err != nil {
		return nil, err
	}
	
	files, err := client.ListDirectory(actualPath)
	if err == nil {
		duration := time.Since(startTime)
		LL.Info("Listed directory %s (%d files) in %v", path, len(files), duration)
	}
	
	return files, err
}

// ListDirectoryRecursive lists directory contents recursively (supports HOST:PATH format)
func (pm *ProxyManager) ListDirectoryRecursive(path string) ([]FileInfo, error) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	
	startTime := time.Now()
	
	client, actualPath, err := pm.getClient(path)
	if err != nil {
		return nil, err
	}
	
	files, err := client.ListDirectoryRecursive(actualPath)
	if err == nil {
		duration := time.Since(startTime)
		LL.Info("Listed directory recursively %s (%d files) in %v", path, len(files), duration)
	}
	
	return files, err
}

// GoToDefinition gets the definition location for a symbol (supports HOST:PATH format)
func (pm *ProxyManager) GoToDefinition(filePath string, line int, character int) ([]Location, error) {
	client, actualPath, err := pm.getClient(filePath)
	if err != nil {
		return nil, err
	}
	
	return client.GoToDefinition(actualPath, line, character)
}

// FindReferences finds all references to a symbol (supports HOST:PATH format)
func (pm *ProxyManager) FindReferences(filePath string, line int, character int, includeDeclaration bool) ([]Location, error) {
	client, actualPath, err := pm.getClient(filePath)
	if err != nil {
		return nil, err
	}
	
	return client.FindReferences(actualPath, line, character, includeDeclaration)
}

// GetCompletions gets completion suggestions at the given position (supports HOST:PATH format)
func (pm *ProxyManager) GetCompletions(filePath string, line int, character int) (interface{}, error) {
	client, actualPath, err := pm.getClient(filePath)
	if err != nil {
		return nil, err
	}
	
	return client.GetCompletions(actualPath, line, character)
}

// GetDiagnostics gets all diagnostics from all open files
func (pm *ProxyManager) GetDiagnostics() (map[string][]interface{}, error) {
	// Diagnostics are global, so we use the local proxy
	client := pm.clients[""]
	if client == nil {
		return nil, fmt.Errorf("local proxy not available")
	}
	
	return client.GetDiagnostics()
}

// GetDocumentSymbols gets document symbols for a file (supports HOST:PATH format)
func (pm *ProxyManager) GetDocumentSymbols(filePath string) ([]DocumentSymbol, error) {
	client, actualPath, err := pm.getClient(filePath)
	if err != nil {
		return nil, err
	}
	
	return client.GetDocumentSymbols(actualPath)
}

// GetLSPMessages gets LSP messages for a language
// Note: This doesn't use file paths, so it uses the local proxy
func (pm *ProxyManager) GetLSPMessages(language string) ([]interface{}, error) {
	client := pm.clients[""] // Always use local proxy for LSP messages
	if client == nil {
		return nil, fmt.Errorf("local proxy not initialized")
	}
	
	return client.GetLSPMessages(language)
}

// GetFoldingRanges gets folding ranges for a file (supports HOST:PATH format)
func (pm *ProxyManager) GetFoldingRanges(filePath string) ([]FoldingRange, error) {
	client, actualPath, err := pm.getClient(filePath)
	if err != nil {
		return nil, err
	}
	
	return client.GetFoldingRanges(actualPath)
}

// GetSignatureHelp gets signature help for a function (supports HOST:PATH format)
func (pm *ProxyManager) GetSignatureHelp(filePath string, line int, character int) (interface{}, error) {
	client, actualPath, err := pm.getClient(filePath)
	if err != nil {
		return nil, err
	}
	
	return client.GetSignatureHelp(actualPath, line, character)
}

// ListProjectSymbols gets all symbols across the project
// Note: This uses the local proxy for now, could be extended for remote projects
func (pm *ProxyManager) ListProjectSymbols(query string) ([]ProjectSymbol, error) {
	client := pm.clients[""] // Use local proxy for project symbols
	if client == nil {
		return nil, fmt.Errorf("local proxy not initialized")
	}
	
	return client.ListProjectSymbols(query)
}

// GetProjectChanges gets all changed files in the project
func (pm *ProxyManager) GetProjectChanges() ([]ProjectChange, error) {
	client := pm.clients[""] // Use local proxy for project changes
	if client == nil {
		return nil, fmt.Errorf("local proxy not initialized")
	}
	
	return client.GetProjectChanges()
}

//go:embed veda-proxy.py
var embeddedProxy string

// proxyScriptPath is the fixed path to the proxy script
var proxyScriptPath = "/tmp/veda-proxy.py"

// init extracts the embedded proxy script to /tmp
func init() {
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
	Name string `json:"name"`
	Type string `json:"type"`
	Size int64  `json:"size"`
	Path string `json:"path"`
}

// FileContent represents file content with change information
type FileContent struct {
	Content string      `json:"content"`
	Changes FileChanges `json:"changes"`
}

// FileChanges represents git changes in a file
type FileChanges struct {
	Added    []int `json:"added"`
	Deleted  []int `json:"deleted"`
	Modified []int `json:"modified"`
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
	Name           string            `json:"name"`
	Detail         string            `json:"detail,omitempty"`
	Kind           int               `json:"kind"`
	Tags           []int             `json:"tags,omitempty"`
	Deprecated     bool              `json:"deprecated,omitempty"`
	Range          Range             `json:"range"`
	SelectionRange Range             `json:"selectionRange"`
	Children       []DocumentSymbol  `json:"children,omitempty"`
	// For SymbolInformation format (older LSP)
	Location       *Location         `json:"location,omitempty"`
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
	cmd       *exec.Cmd
	stdin     *bufio.Writer
	stdout    *bufio.Scanner
	requestID int
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

	return &Client{
		cmd:       cmd,
		stdin:     stdin,
		stdout:    stdout,
		requestID: 0,
	}, nil
}

// ListDirectory sends a JSON RPC request to get directory listing
func (c *Client) ListDirectory(path string) ([]FileInfo, error) {
	c.requestID++
	request := JSONRPCRequest{
		Method: "list_directory",
		Params: map[string]interface{}{"path": path},
		ID:     c.requestID,
	}

	requestBytes, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	c.stdin.WriteString(string(requestBytes) + "\n")
	c.stdin.Flush()


	if c.stdout.Scan() {
		var response JSONRPCResponse
		if err := json.Unmarshal(c.stdout.Bytes(), &response); err != nil {
			return nil, fmt.Errorf("failed to unmarshal response: %v", err)
		}

		if response.Error != "" {
			return nil, fmt.Errorf("proxy error: %s", response.Error)
		}

		var files []FileInfo
		for _, item := range response.Result {
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
		return files, nil
	}

	// Check for scanner error
	if err := c.stdout.Err(); err != nil {
		return nil, fmt.Errorf("scanner error: %v", err)
	}

	return nil, fmt.Errorf("failed to read proxy response: no data received")
}

// ListDirectoryRecursive sends a JSON RPC request to get recursive directory listing
func (c *Client) ListDirectoryRecursive(path string) ([]FileInfo, error) {
	request := JSONRPCRequest{
		Method: "list_directory_recursive",
		Params: map[string]interface{}{"path": path},
		ID:     2,
	}

	requestBytes, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	c.stdin.WriteString(string(requestBytes) + "\n")
	c.stdin.Flush()


	if c.stdout.Scan() {
		var response JSONRPCResponse
		if err := json.Unmarshal(c.stdout.Bytes(), &response); err != nil {
			return nil, fmt.Errorf("failed to unmarshal response: %v", err)
		}

		if response.Error != "" {
			return nil, fmt.Errorf("proxy error: %s", response.Error)
		}

		var files []FileInfo
		for _, item := range response.Result {
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
		return files, nil
	}

	// Check for scanner error
	if err := c.stdout.Err(); err != nil {
		return nil, fmt.Errorf("scanner error: %v", err)
	}

	return nil, fmt.Errorf("failed to read proxy response: no data received")
}

// parseRemotePath parses a path and returns (host, path, isRemote)
// Examples: "host:path/to/file" -> ("host", "path/to/file", true)
//           "/local/path" -> ("", "/local/path", false)
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
	
	request := JSONRPCRequest{
		Method: "get_file",
		Params: map[string]interface{}{"path": path},
		ID:     3,
	}

	requestBytes, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	c.stdin.WriteString(string(requestBytes) + "\n")
	c.stdin.Flush()


	if c.stdout.Scan() {
		var response struct {
			ID     int                    `json:"id"`
			Result map[string]interface{} `json:"result,omitempty"`
			Error  string                 `json:"error,omitempty"`
		}
		if err := json.Unmarshal(c.stdout.Bytes(), &response); err != nil {
			return nil, err
		}

		if response.Error != "" {
			return nil, fmt.Errorf("proxy error: %s", response.Error)
		}

		fileContent := &FileContent{
			Changes: FileChanges{
				Added:    []int{},
				Deleted:  []int{},
				Modified: []int{},
			},
		}

		if content, ok := response.Result["content"].(string); ok {
			fileContent.Content = content
		} else {
			return nil, fmt.Errorf("missing content in response")
		}

		// Parse changes if present
		if changes, ok := response.Result["changes"].(map[string]interface{}); ok {
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

		return fileContent, nil
	}

	return nil, fmt.Errorf("failed to read proxy response")
}

// StartLSPServer starts an LSP server for the specified language
func (c *Client) StartLSPServer(language string) error {
	request := JSONRPCRequest{
		Method: "start_lsp_server",
		Params: map[string]interface{}{"language": language},
		ID:     4,
	}

	requestBytes, err := json.Marshal(request)
	if err != nil {
		return err
	}

	c.stdin.WriteString(string(requestBytes) + "\n")
	c.stdin.Flush()


	if c.stdout.Scan() {
		var response struct {
			ID     int                    `json:"id"`
			Result map[string]interface{} `json:"result,omitempty"`
			Error  string                 `json:"error,omitempty"`
		}
		if err := json.Unmarshal(c.stdout.Bytes(), &response); err != nil {
			return fmt.Errorf("failed to unmarshal response: %v", err)
		}

		if response.Error != "" {
			return fmt.Errorf("proxy error: %s", response.Error)
		}

		if success, ok := response.Result["success"].(bool); ok && success {
			return nil
		}

		return fmt.Errorf("failed to start LSP server for %s", language)
	}

	return fmt.Errorf("failed to read proxy response")
}

// StopLSPServer stops the LSP server for the specified language
func (c *Client) StopLSPServer(language string) error {
	request := JSONRPCRequest{
		Method: "stop_lsp_server",
		Params: map[string]interface{}{"language": language},
		ID:     5,
	}

	requestBytes, err := json.Marshal(request)
	if err != nil {
		return err
	}

	c.stdin.WriteString(string(requestBytes) + "\n")
	c.stdin.Flush()


	if c.stdout.Scan() {
		var response struct {
			ID     int                    `json:"id"`
			Result map[string]interface{} `json:"result,omitempty"`
			Error  string                 `json:"error,omitempty"`
		}
		if err := json.Unmarshal(c.stdout.Bytes(), &response); err != nil {
			return fmt.Errorf("failed to unmarshal response: %v", err)
		}

		if response.Error != "" {
			return fmt.Errorf("proxy error: %s", response.Error)
		}

		if success, ok := response.Result["success"].(bool); ok && success {
			return nil
		}

		return fmt.Errorf("failed to stop LSP server for %s", language)
	}

	return fmt.Errorf("failed to read proxy response")
}

// GoToDefinition gets the definition location for a symbol at the given position
func (c *Client) GoToDefinition(filePath string, line int, character int) ([]Location, error) {
	c.requestID++
	request := JSONRPCRequest{
		Method: "go_to_definition",
		Params: map[string]interface{}{
			"file_path": filePath,
			"line":      line,
			"character": character,
		},
		ID: c.requestID,
	}

	requestBytes, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	// Log the request for debugging
	LL.Trace("GoToDefinition request: %s", string(requestBytes))

	c.stdin.WriteString(string(requestBytes) + "\n")
	c.stdin.Flush()


	if c.stdout.Scan() {
		var response struct {
			ID     int                    `json:"id"`
			Result map[string]interface{} `json:"result,omitempty"`
			Error  string                 `json:"error,omitempty"`
		}
		if err := json.Unmarshal(c.stdout.Bytes(), &response); err != nil {
			return nil, fmt.Errorf("failed to unmarshal response: %v", err)
		}

		if response.Error != "" {
			return nil, fmt.Errorf("proxy error: %s", response.Error)
		}

		if success, ok := response.Result["success"].(bool); ok && !success {
			if errMsg, ok := response.Result["error"].(string); ok {
				return nil, fmt.Errorf("LSP error: %s", errMsg)
			}
			return nil, fmt.Errorf("failed to get definition")
		}

		// Parse the LSP result
		var locations []Location
		if result, ok := response.Result["result"].([]interface{}); ok {
			for _, item := range result {
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
		} else if locMap, ok := response.Result["result"].(map[string]interface{}); ok {
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

	if err := c.stdout.Err(); err != nil {
		return nil, fmt.Errorf("scanner error: %v", err)
	}

	return nil, fmt.Errorf("failed to read proxy response")
}

// FindReferences finds all references to a symbol at the given position
func (c *Client) FindReferences(filePath string, line int, character int, includeDeclaration bool) ([]Location, error) {
	c.requestID++
	request := JSONRPCRequest{
		Method: "find_references",
		Params: map[string]interface{}{
			"file_path":           filePath,
			"line":                line,
			"character":           character,
			"include_declaration": includeDeclaration,
		},
		ID: c.requestID,
	}

	requestBytes, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	// Log the request for debugging
	LL.Trace("FindReferences request: %s", string(requestBytes))

	c.stdin.WriteString(string(requestBytes) + "\n")
	c.stdin.Flush()

	if c.stdout.Scan() {
		var response struct {
			ID     int                    `json:"id"`
			Result map[string]interface{} `json:"result,omitempty"`
			Error  string                 `json:"error,omitempty"`
		}
		if err := json.Unmarshal(c.stdout.Bytes(), &response); err != nil {
			return nil, fmt.Errorf("failed to unmarshal response: %v", err)
		}

		if response.Error != "" {
			return nil, fmt.Errorf("proxy error: %s", response.Error)
		}

		if success, ok := response.Result["success"].(bool); ok && !success {
			if errMsg, ok := response.Result["error"].(string); ok {
				return nil, fmt.Errorf("LSP error: %s", errMsg)
			}
			return nil, fmt.Errorf("failed to find references")
		}

		// Parse the LSP result
		var locations []Location
		if result, ok := response.Result["result"].([]interface{}); ok {
			for _, item := range result {
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

	if err := c.stdout.Err(); err != nil {
		return nil, fmt.Errorf("scanner error: %v", err)
	}

	return nil, fmt.Errorf("failed to read proxy response")
}

// GetLSPMessages gets all LSP messages for a specific language
func (c *Client) GetLSPMessages(language string) ([]interface{}, error) {
	c.requestID++
	request := JSONRPCRequest{
		Method: "get_lsp_messages",
		Params: map[string]interface{}{"language": language},
		ID:     c.requestID,
	}

	requestBytes, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	c.stdin.WriteString(string(requestBytes) + "\n")
	c.stdin.Flush()


	if c.stdout.Scan() {
		var response struct {
			ID     int                    `json:"id"`
			Result map[string]interface{} `json:"result,omitempty"`
			Error  string                 `json:"error,omitempty"`
		}
		if err := json.Unmarshal(c.stdout.Bytes(), &response); err != nil {
			return nil, fmt.Errorf("failed to unmarshal response: %v", err)
		}

		if response.Error != "" {
			return nil, fmt.Errorf("proxy error: %s", response.Error)
		}

		if success, ok := response.Result["success"].(bool); ok && success {
			if messages, ok := response.Result["messages"].([]interface{}); ok {
				return messages, nil
			}
		}

		return nil, fmt.Errorf("failed to get LSP messages")
	}

	return nil, fmt.Errorf("failed to read proxy response")
}

// GetLSPNotifications gets pending showMessage notifications for a language
func (c *Client) GetLSPNotifications(language string) ([]map[string]interface{}, error) {
	c.requestID++
	request := JSONRPCRequest{
		Method: "get_lsp_notifications",
		Params: map[string]interface{}{"language": language},
		ID:     c.requestID,
	}

	requestBytes, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	c.stdin.WriteString(string(requestBytes) + "\n")
	c.stdin.Flush()


	if c.stdout.Scan() {
		var response struct {
			ID     int                    `json:"id"`
			Result map[string]interface{} `json:"result,omitempty"`
			Error  string                 `json:"error,omitempty"`
		}
		if err := json.Unmarshal(c.stdout.Bytes(), &response); err != nil {
			return nil, fmt.Errorf("failed to unmarshal response: %v", err)
		}

		if response.Error != "" {
			return nil, fmt.Errorf("proxy error: %s", response.Error)
		}

		if success, ok := response.Result["success"].(bool); ok && success {
			if notifications, ok := response.Result["notifications"].([]interface{}); ok {
				var result []map[string]interface{}
				for _, n := range notifications {
					if notif, ok := n.(map[string]interface{}); ok {
						result = append(result, notif)
					}
				}
				return result, nil
			}
		}

		return nil, fmt.Errorf("failed to get LSP notifications")
	}

	return nil, fmt.Errorf("failed to read proxy response")
}

// GetDocumentSymbols gets document symbols (functions, classes, etc.) for a file
func (c *Client) GetDocumentSymbols(filePath string) ([]DocumentSymbol, error) {
	c.requestID++
	request := JSONRPCRequest{
		Method: "get_document_symbols",
		Params: map[string]interface{}{
			"file_path": filePath,
		},
		ID: c.requestID,
	}

	requestBytes, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	// Log the request for debugging
	LL.Trace("GetDocumentSymbols request: %s", string(requestBytes))

	c.stdin.WriteString(string(requestBytes) + "\n")
	if err := c.stdin.Flush(); err != nil {
		return nil, fmt.Errorf("failed to flush request: %w", err)
	}


	if c.stdout.Scan() {
		var response struct {
			ID     int                    `json:"id"`
			Result map[string]interface{} `json:"result,omitempty"`
			Error  string                 `json:"error,omitempty"`
		}

		responseText := c.stdout.Text()
		LL.Trace("GetDocumentSymbols response: %s", responseText)

		if err := json.Unmarshal([]byte(responseText), &response); err != nil {
			return nil, err
		}

		if response.Error != "" {
			return nil, fmt.Errorf("proxy error: %s", response.Error)
		}

		if success, ok := response.Result["success"].(bool); ok && success {
			if result, ok := response.Result["result"].([]interface{}); ok {
				// Convert the result to DocumentSymbol objects
				symbols := make([]DocumentSymbol, 0, len(result))
				for _, item := range result {
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

		if errorMsg, ok := response.Result["error"].(string); ok {
			return nil, fmt.Errorf("LSP error: %s", errorMsg)
		}

		return nil, fmt.Errorf("failed to get document symbols")
	}

	if err := c.stdout.Err(); err != nil {
		return nil, fmt.Errorf("scanner error: %w", err)
	}

	return nil, fmt.Errorf("failed to read proxy response")
}

// GetFoldingRanges gets folding ranges for a file
func (c *Client) GetFoldingRanges(filePath string) ([]FoldingRange, error) {
	c.requestID++
	request := JSONRPCRequest{
		Method: "get_folding_ranges",
		Params: map[string]interface{}{
			"file_path": filePath,
		},
		ID: c.requestID,
	}

	requestBytes, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	// Log the request for debugging
	LL.Trace("GetFoldingRanges request: %s", string(requestBytes))

	c.stdin.WriteString(string(requestBytes) + "\n")
	if err := c.stdin.Flush(); err != nil {
		return nil, fmt.Errorf("failed to flush request: %w", err)
	}


	if c.stdout.Scan() {
		var response struct {
			ID     int                    `json:"id"`
			Result map[string]interface{} `json:"result,omitempty"`
			Error  string                 `json:"error,omitempty"`
		}

		responseText := c.stdout.Text()
		LL.Trace("GetFoldingRanges response: %s", responseText)

		if err := json.Unmarshal([]byte(responseText), &response); err != nil {
			return nil, err
		}

		if response.Error != "" {
			return nil, fmt.Errorf("proxy error: %s", response.Error)
		}

		if success, ok := response.Result["success"].(bool); ok && success {
			if result, ok := response.Result["result"].([]interface{}); ok {
				// Convert the result to FoldingRange objects
				ranges := make([]FoldingRange, 0, len(result))
				for _, item := range result {
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

		if errorMsg, ok := response.Result["error"].(string); ok {
			return nil, fmt.Errorf("LSP error: %s", errorMsg)
		}

		return nil, fmt.Errorf("failed to get folding ranges")
	}

	if err := c.stdout.Err(); err != nil {
		return nil, fmt.Errorf("scanner error: %w", err)
	}

	return nil, fmt.Errorf("failed to read proxy response")
}

// GetSignatureHelp gets signature help for a function at the given position
func (c *Client) GetSignatureHelp(filePath string, line int, character int) (interface{}, error) {
	c.requestID++
	request := JSONRPCRequest{
		Method: "get_signature_help",
		Params: map[string]interface{}{
			"file_path": filePath,
			"line":      line,
			"character": character,
		},
		ID: c.requestID,
	}

	requestBytes, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	// Log the request for debugging
	LL.Trace("GetSignatureHelp request: %s", string(requestBytes))

	c.stdin.WriteString(string(requestBytes) + "\n")
	c.stdin.Flush()


	if c.stdout.Scan() {
		responseText := c.stdout.Text()
		LL.Trace("GetSignatureHelp response: %s", responseText)

		var response map[string]interface{}
		if err := json.Unmarshal([]byte(responseText), &response); err != nil {
			return nil, fmt.Errorf("failed to unmarshal response: %v", err)
		}

		if response["error"] != nil {
			return nil, fmt.Errorf("proxy error: %v", response["error"])
		}

		if result, ok := response["result"].(map[string]interface{}); ok {
			if result["success"] == true && result["result"] != nil {
				return result["result"], nil
			}
			if errorMsg, ok := result["error"].(string); ok {
				return nil, fmt.Errorf("%s", errorMsg)
			}
		}
	}

	return nil, fmt.Errorf("failed to get signature help")
}

// ListProjectSymbols gets all symbols across the entire project
func (c *Client) ListProjectSymbols(query string) ([]ProjectSymbol, error) {
	c.requestID++
	request := JSONRPCRequest{
		Method: "list_project_symbols",
		Params: map[string]interface{}{
			"query": query,
		},
		ID: c.requestID,
	}

	requestBytes, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	// Log the request for debugging
	LL.Trace("ListProjectSymbols request: %s", string(requestBytes))

	c.stdin.WriteString(string(requestBytes) + "\n")
	c.stdin.Flush()


	if c.stdout.Scan() {
		var response struct {
			ID     int                    `json:"id"`
			Result map[string]interface{} `json:"result,omitempty"`
			Error  string                 `json:"error,omitempty"`
		}
		
		responseText := c.stdout.Text()
		LL.Trace("ListProjectSymbols response: %s", responseText)
		
		if err := json.Unmarshal([]byte(responseText), &response); err != nil {
			return nil, fmt.Errorf("failed to unmarshal response: %v", err)
		}

		if response.Error != "" {
			return nil, fmt.Errorf("proxy error: %s", response.Error)
		}

		if success, ok := response.Result["success"].(bool); ok && success {
			if symbolsData, ok := response.Result["symbols"].([]interface{}); ok {
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

		if errorMsg, ok := response.Result["error"].(string); ok {
			return nil, fmt.Errorf("LSP error: %s", errorMsg)
		}

		return nil, fmt.Errorf("no symbols found in project")
	}

	if err := c.stdout.Err(); err != nil {
		return nil, fmt.Errorf("scanner error: %w", err)
	}

	return nil, fmt.Errorf("failed to read proxy response")
}

// Close terminates the proxy process
// WriteFile writes content to a file
func (c *Client) WriteFile(path string, content string) error {
	request := JSONRPCRequest{
		Method: "write_file",
		Params: map[string]interface{}{
			"path":    path,
			"content": content,
		},
		ID: c.requestID,
	}
	c.requestID++
	
	requestBytes, err := json.Marshal(request)
	if err != nil {
		return err
	}
	
	c.stdin.WriteString(string(requestBytes) + "\n")
	c.stdin.Flush()
	
	if c.stdout.Scan() {
		var response struct {
			ID     int                    `json:"id"`
			Result map[string]interface{} `json:"result,omitempty"`
			Error  string                 `json:"error,omitempty"`
		}
		if err := json.Unmarshal(c.stdout.Bytes(), &response); err != nil {
			return err
		}
		
		if response.Error != "" {
			return fmt.Errorf("proxy error: %s", response.Error)
		}
		
		if result, ok := response.Result["success"].(bool); ok && result {
			return nil
		}
		
		return fmt.Errorf("failed to write file")
	}
	
	return fmt.Errorf("failed to read proxy response")
}

// GetCompletions gets completion suggestions at the given position
func (c *Client) GetCompletions(filePath string, line int, character int) (interface{}, error) {
	c.requestID++
	request := JSONRPCRequest{
		Method: "get_completions",
		Params: map[string]interface{}{
			"file_path": filePath,
			"line":      line,
			"character": character,
		},
		ID: c.requestID,
	}
	
	requestBytes, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}
	
	// Log the request for debugging
	LL.Trace("GetCompletions request: %s", string(requestBytes))
	
	c.stdin.WriteString(string(requestBytes) + "\n")
	c.stdin.Flush()
	
	if c.stdout.Scan() {
		var response struct {
			ID     int                    `json:"id"`
			Result map[string]interface{} `json:"result,omitempty"`
			Error  string                 `json:"error,omitempty"`
		}
		if err := json.Unmarshal(c.stdout.Bytes(), &response); err != nil {
			return nil, fmt.Errorf("failed to unmarshal response: %v", err)
		}
		
		if response.Error != "" {
			return nil, fmt.Errorf("proxy error: %s", response.Error)
		}
		
		if success, ok := response.Result["success"].(bool); ok && success {
			if result, ok := response.Result["result"]; ok {
				return result, nil
			}
		}
		
		return nil, fmt.Errorf("no completions found")
	}
	
	return nil, fmt.Errorf("failed to read proxy response")
}

// GetDiagnostics retrieves all diagnostics from all open files
func (c *Client) GetDiagnostics() (map[string][]interface{}, error) {
	c.requestID++
	request := JSONRPCRequest{
		Method: "get_diagnostics",
		Params: map[string]interface{}{},
		ID:     c.requestID,
	}
	
	requestBytes, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}
	
	// Log the request for debugging
	LL.Trace("GetDiagnostics request: %s", string(requestBytes))
	
	c.stdin.WriteString(string(requestBytes) + "\n")
	c.stdin.Flush()
	
	// Read response
	if c.stdout.Scan() {
		line := c.stdout.Text()
		LL.Trace("GetDiagnostics response: %s", line)
		
		var response struct {
			ID     int                    `json:"id"`
			Result map[string]interface{} `json:"result,omitempty"`
			Error  string                 `json:"error,omitempty"`
		}
		if err := json.Unmarshal([]byte(line), &response); err != nil {
			return nil, err
		}
		
		if response.Error != "" {
			return nil, fmt.Errorf("proxy error: %s", response.Error)
		}
		
		if success, ok := response.Result["success"].(bool); ok && success {
			if diagnostics, ok := response.Result["diagnostics"].(map[string]interface{}); ok {
				// Convert map[string]interface{} to map[string][]interface{}
				result := make(map[string][]interface{})
				for uri, diags := range diagnostics {
					if diagList, ok := diags.([]interface{}); ok {
						result[uri] = diagList
					}
				}
				return result, nil
			}
		}
		
		return nil, fmt.Errorf("no diagnostics found")
	}
	
	return nil, fmt.Errorf("failed to read proxy response")
}

// GetProjectChanges retrieves all changes in the project
func (c *Client) GetProjectChanges() ([]ProjectChange, error) {
	c.requestID++
	request := JSONRPCRequest{
		Method: "get_project_changes",
		Params: map[string]interface{}{},
		ID:     c.requestID,
	}
	
	requestBytes, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}
	
	// Log the request for debugging
	LL.Trace("GetProjectChanges request: %s", string(requestBytes))
	
	c.stdin.WriteString(string(requestBytes) + "\n")
	c.stdin.Flush()
	
	// Read response
	if c.stdout.Scan() {
		line := c.stdout.Text()
		LL.Trace("GetProjectChanges response: %s", line)
		
		var response struct {
			ID     int                    `json:"id"`
			Result map[string]interface{} `json:"result,omitempty"`
			Error  string                 `json:"error,omitempty"`
		}
		if err := json.Unmarshal([]byte(line), &response); err != nil {
			return nil, err
		}
		
		if response.Error != "" {
			return nil, fmt.Errorf("proxy error: %s", response.Error)
		}
		
		if success, ok := response.Result["success"].(bool); ok && success {
			if changes, ok := response.Result["changes"].([]interface{}); ok {
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
		
		return nil, fmt.Errorf("no changes found")
	}
	
	return nil, fmt.Errorf("failed to read proxy response")
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
