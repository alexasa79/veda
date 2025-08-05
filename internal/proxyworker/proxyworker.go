package proxyworker

import (
	"veda/internal/log"
	"veda/internal/proxy"
)

var LL *log.Logger = log.NewLogger("ProxyWorker")

// ProxyRequestType represents the type of proxy request
type ProxyRequestType int

const (
	RequestFoldingRanges ProxyRequestType = iota
	RequestSignatureHelp
	RequestProjectSymbols
	RequestDocumentSymbols
	RequestListDirectoryRecursive
	RequestResolvePath
	// Future: RequestGoToDefinition, etc.
)

// ProxyRequest represents a request to the proxy worker thread
type ProxyRequest struct {
	Type       ProxyRequestType
	FilePath   string
	Buffer     interface{} // Reference to the buffer that made the request
	Generation uint64      // Generation number for request validation

	// Signature help specific fields
	Line      int // Line number for signature help (0-based)
	Character int // Character position for signature help (0-based)

	// Project symbols specific fields
	Query      string      // Search query for symbols
	ResponseTo interface{} // Widget or component to send response to

	// Directory listing specific fields
	DirectoryPath string // Path to list recursively

	// Path resolution specific fields
	PathToResolve string // Path to check if it's a file or directory

	// Working directory for project-wide operations
	WorkingDirectory string // Current working directory (can include HOST: prefix)

	// Future: add other request-specific fields for different proxy operations
}
// ProxyResponse represents a response from the proxy worker thread
type ProxyResponse struct {
	Type          ProxyRequestType
	FilePath      string
	Buffer        interface{} // Reference to the buffer that made the request
	Generation    uint64      // Generation number from the original request
	FoldingRanges []proxy.FoldingRange
	SignatureHelp interface{} // Raw signature help response from proxy
	ProjectSymbols []proxy.ProjectSymbol // Project symbols response
	DocumentSymbols []proxy.DocumentSymbol // Document symbols response
	Query         string      // The query that was used for project/document symbols
	ResponseTo    interface{} // Widget or component to send response to
	Files         []proxy.FileInfo // Directory listing response
	Error         error
	
	// Path resolution response fields
	IsDirectory   bool   // Whether the path is a directory
	ResolvedPath  string // The resolved path
	WorkingDirectory string // Extracted working directory for files
	
	// Future: add other response fields for different proxy operations
}

// proxyWorker is the main function that runs in the proxy worker thread
func proxyWorker(proxyManager *proxy.ProxyManager, requestChan chan ProxyRequest, responseChan chan ProxyResponse) {
	LL.Debug("Proxy worker thread started")
	for request := range requestChan {
		LL.Trace("Processing proxy request: type=%d, filePath=%s", request.Type, request.FilePath)

		switch request.Type {
		case RequestFoldingRanges:
			ranges, err := proxyManager.GetFoldingRanges(request.FilePath)
			response := ProxyResponse{
				Type:          RequestFoldingRanges,
				FilePath:      request.FilePath,
				Buffer:        request.Buffer,
				Generation:    request.Generation,
				FoldingRanges: ranges,
				Error:         err,
			}

			if err != nil {
				LL.Debug("GetFoldingRanges failed for %s: %v", request.FilePath, err)
			} else {
				LL.Trace("GetFoldingRanges succeeded for %s: got %d ranges", request.FilePath, len(ranges))
			}

			responseChan <- response

		case RequestSignatureHelp:
			sigHelp, err := proxyManager.GetSignatureHelp(request.FilePath, request.Line, request.Character)
			response := ProxyResponse{
				Type:          RequestSignatureHelp,
				FilePath:      request.FilePath,
				Buffer:        request.Buffer,
				Generation:    request.Generation,
				SignatureHelp: sigHelp,
				Error:         err,
			}

			if err != nil {
				LL.Debug("GetSignatureHelp failed for %s at %d:%d: %v", request.FilePath, request.Line, request.Character, err)
			} else {
				LL.Trace("GetSignatureHelp succeeded for %s at %d:%d", request.FilePath, request.Line, request.Character)
			}

			responseChan <- response

		case RequestProjectSymbols:
			// Use the working directory from the request, or empty string as fallback
			workingDir := request.WorkingDirectory
			if workingDir == "" {
				workingDir = "." // Default to current directory
			}
			symbols, err := proxyManager.ListProjectSymbols(workingDir, request.Query)
			response := ProxyResponse{
				Type:           RequestProjectSymbols,
				FilePath:       request.FilePath,
				Buffer:         request.Buffer,
				Generation:     request.Generation,
				ProjectSymbols: symbols,
				Query:          request.Query,
				ResponseTo:     request.ResponseTo,
				Error:          err,
			}

			if err != nil {
				LL.Debug("ListProjectSymbols failed for query '%s': %v", request.Query, err)
			} else {
				LL.Trace("ListProjectSymbols succeeded for query '%s': got %d symbols", request.Query, len(symbols))
			}

			// Send to specific channel if provided, otherwise to general response channel
			if request.ResponseTo != nil {
				if ch, ok := request.ResponseTo.(chan ProxyResponse); ok {
					select {
					case ch <- response:
						LL.Trace("Sent project symbols response to specific channel")
					default:
						LL.Debug("Specific response channel full, sending to general channel")
						responseChan <- response
					}
				} else {
					responseChan <- response
				}
			} else {
				responseChan <- response
			}

		case RequestDocumentSymbols:
			symbols, err := proxyManager.GetDocumentSymbols(request.FilePath)
			response := ProxyResponse{
				Type:            RequestDocumentSymbols,
				FilePath:        request.FilePath,
				Buffer:          request.Buffer,
				Generation:      request.Generation,
				DocumentSymbols: symbols,
				ResponseTo:      request.ResponseTo,
				Error:           err,
			}

			if err != nil {
				LL.Debug("GetDocumentSymbols failed for %s: %v", request.FilePath, err)
			} else {
				LL.Trace("GetDocumentSymbols succeeded for %s: got %d symbols", request.FilePath, len(symbols))
			}

			// Send to specific channel if provided, otherwise to general response channel
			if request.ResponseTo != nil {
				if ch, ok := request.ResponseTo.(chan ProxyResponse); ok {
					select {
					case ch <- response:
						LL.Trace("Sent document symbols response to specific channel")
					default:
						LL.Debug("Specific response channel full, sending to general channel")
						responseChan <- response
					}
				} else {
					responseChan <- response
				}
			} else {
				responseChan <- response
			}

		case RequestListDirectoryRecursive:
			files, err := proxyManager.ListDirectoryRecursive(request.DirectoryPath)
			response := ProxyResponse{
				Type:       RequestListDirectoryRecursive,
				FilePath:   request.DirectoryPath,
				ResponseTo: request.ResponseTo,
				Files:      files,
				Error:      err,
			}

			if err != nil {
				LL.Debug("ListDirectoryRecursive failed for %s: %v", request.DirectoryPath, err)
			} else {
				LL.Debug("ListDirectoryRecursive succeeded for %s: got %d files", request.DirectoryPath, len(files))
			}

			// Send to specific channel if provided, otherwise to general response channel
			if request.ResponseTo != nil {
				if ch, ok := request.ResponseTo.(chan ProxyResponse); ok {
					select {
					case ch <- response:
						LL.Trace("Sent directory listing response to specific channel")
					default:
						LL.Debug("Specific response channel full, sending to general channel")
						responseChan <- response
					}
				} else {
					responseChan <- response
				}
			} else {
				responseChan <- response
			}

		default:
			LL.Error("Unknown proxy request type: %d", request.Type)
		}
	}
	LL.Debug("Proxy worker thread stopped")
}

// StartProxyWorker starts the proxy worker thread that handles all proxy communication
func StartProxyWorker(proxyManager *proxy.ProxyManager, requestChan chan ProxyRequest, responseChan chan ProxyResponse) {
	go proxyWorker(proxyManager, requestChan, responseChan)
}
