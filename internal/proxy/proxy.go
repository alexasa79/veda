package proxy

import (
	"bufio"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"time"
	"veda/internal/log"
)

var LL *log.Logger = log.NewLogger("PROXY")

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
	// Default is 64KB, we'll set it to 1MB
	const maxScanTokenSize = 1024 * 1024 // 1MB
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
	
	// Simulate remote proxy delay
	time.Sleep(500 * time.Millisecond)

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
	
	// Simulate remote proxy delay
	time.Sleep(500 * time.Millisecond)

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

// GetFile reads the contents of a file
func (c *Client) GetFile(path string) (string, error) {
	request := JSONRPCRequest{
		Method: "get_file",
		Params: map[string]interface{}{"path": path},
		ID:     3,
	}

	requestBytes, err := json.Marshal(request)
	if err != nil {
		return "", err
	}

	c.stdin.WriteString(string(requestBytes) + "\n")
	c.stdin.Flush()
	
	// Simulate remote proxy delay
	time.Sleep(500 * time.Millisecond)

	if c.stdout.Scan() {
		var response struct {
			ID     int                    `json:"id"`
			Result map[string]interface{} `json:"result,omitempty"`
			Error  string                 `json:"error,omitempty"`
		}
		if err := json.Unmarshal(c.stdout.Bytes(), &response); err != nil {
			return "", err
		}

		if response.Error != "" {
			return "", fmt.Errorf("proxy error: %s", response.Error)
		}

		if content, ok := response.Result["content"].(string); ok {
			return content, nil
		}

		return "", fmt.Errorf("unexpected response format")
	}

	return "", fmt.Errorf("failed to read proxy response")
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
	
	// Simulate remote proxy delay
	time.Sleep(500 * time.Millisecond)

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
	
	// Simulate remote proxy delay
	time.Sleep(500 * time.Millisecond)

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
	
	// Simulate remote proxy delay
	time.Sleep(500 * time.Millisecond)

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
	
	// Simulate remote proxy delay
	time.Sleep(500 * time.Millisecond)

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
	
	// Simulate remote proxy delay
	time.Sleep(500 * time.Millisecond)

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
	
	// Simulate remote proxy delay
	time.Sleep(500 * time.Millisecond)

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
	
	// Simulate remote proxy delay
	time.Sleep(500 * time.Millisecond)

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
	
	// Simulate remote proxy delay
	time.Sleep(500 * time.Millisecond)

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
				return nil, fmt.Errorf(errorMsg)
			}
		}
	}

	return nil, fmt.Errorf("failed to get signature help")
}

// Close terminates the proxy process
func (c *Client) Close() error {
	if c.cmd != nil && c.cmd.Process != nil {
		return c.cmd.Process.Kill()
	}
	return nil
}
