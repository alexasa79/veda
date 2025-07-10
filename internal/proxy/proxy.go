package proxy

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os/exec"
	"syscall"
)

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

// Client represents a proxy client
type Client struct {
	cmd    *exec.Cmd
	stdin  *bufio.Writer
	stdout *bufio.Scanner
}

// NewClient creates a new proxy client and starts the proxy process
func NewClient() (*Client, error) {
	cmd := exec.Command("python3", "veda-proxy.py")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	stdinPipe, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	stdin := bufio.NewWriter(stdinPipe)
	stdout := bufio.NewScanner(stdoutPipe)
	
	// Increase scanner buffer size to handle large responses
	// Default is 64KB, we'll set it to 1MB
	const maxScanTokenSize = 1024 * 1024 // 1MB
	buf := make([]byte, 0, maxScanTokenSize)
	stdout.Buffer(buf, maxScanTokenSize)

	return &Client{
		cmd:    cmd,
		stdin:  stdin,
		stdout: stdout,
	}, nil
}

// ListDirectory sends a JSON RPC request to get directory listing
func (c *Client) ListDirectory(path string) ([]FileInfo, error) {
	request := JSONRPCRequest{
		Method: "list_directory",
		Params: map[string]interface{}{"path": path},
		ID:     1,
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

// Close terminates the proxy process
func (c *Client) Close() error {
	if c.cmd != nil && c.cmd.Process != nil {
		return c.cmd.Process.Kill()
	}
	return nil
}
