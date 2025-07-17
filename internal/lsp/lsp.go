package lsp

import (
	"veda/internal/log"
	"veda/internal/proxy"
)

var LL *log.Logger = log.NewLogger("LSP")

// LSPRequestType represents the type of LSP request
type LSPRequestType int

const (
	RequestFoldingRanges LSPRequestType = iota
	// Future: RequestGoToDefinition, RequestSignatureHelp, etc.
)

// LSPRequest represents a request to the LSP worker thread
type LSPRequest struct {
	Type     LSPRequestType
	FilePath string
	Buffer   interface{} // Reference to the buffer that made the request
	// Future: add other request-specific fields for different LSP operations
}

// LSPResponse represents a response from the LSP worker thread
type LSPResponse struct {
	Type          LSPRequestType
	FilePath      string
	Buffer        interface{} // Reference to the buffer that made the request
	FoldingRanges []proxy.FoldingRange
	Error         error
	// Future: add other response fields for different LSP operations
}

// lspWorker is the main function that runs in the LSP worker thread
func lspWorker(proxyClient *proxy.Client, requestChan chan LSPRequest, responseChan chan LSPResponse) {
	LL.Debug("LSP worker thread started")
	for request := range requestChan {
		LL.Trace("Processing LSP request: type=%d, filePath=%s", request.Type, request.FilePath)

		switch request.Type {
		case RequestFoldingRanges:
			ranges, err := proxyClient.GetFoldingRanges(request.FilePath)
			response := LSPResponse{
				Type:          RequestFoldingRanges,
				FilePath:      request.FilePath,
				Buffer:        request.Buffer,
				FoldingRanges: ranges,
				Error:         err,
			}

			if err != nil {
				LL.Debug("GetFoldingRanges failed for %s: %v", request.FilePath, err)
			} else {
				LL.Trace("GetFoldingRanges succeeded for %s: got %d ranges", request.FilePath, len(ranges))
			}

			responseChan <- response

		// Future: handle other request types
		default:
			LL.Error("Unknown LSP request type: %d", request.Type)
		}
	}
	LL.Debug("LSP worker thread stopped")
}

// StartLSPWorker starts the LSP worker thread that handles all LSP communication
func StartLSPWorker(proxyClient *proxy.Client, requestChan chan LSPRequest, responseChan chan LSPResponse) {
	go lspWorker(proxyClient, requestChan, responseChan)
}
