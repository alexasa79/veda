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
	RequestSignatureHelp
	RequestProjectSymbols
	// Future: RequestGoToDefinition, etc.
)

// LSPRequest represents a request to the LSP worker thread
type LSPRequest struct {
	Type       LSPRequestType
	FilePath   string
	Buffer     interface{} // Reference to the buffer that made the request
	Generation uint64      // Generation number for request validation

	// Signature help specific fields
	Line      int // Line number for signature help (0-based)
	Character int // Character position for signature help (0-based)

	// Project symbols specific fields
	Query      string      // Search query for symbols
	ResponseTo interface{} // Widget or component to send response to

	// Future: add other request-specific fields for different LSP operations
}
// LSPResponse represents a response from the LSP worker thread
type LSPResponse struct {
	Type          LSPRequestType
	FilePath      string
	Buffer        interface{} // Reference to the buffer that made the request
	Generation    uint64      // Generation number from the original request
	FoldingRanges []proxy.FoldingRange
	SignatureHelp interface{} // Raw signature help response from LSP
	ProjectSymbols []proxy.ProjectSymbol // Project symbols response
	Query         string      // The query that was used for project symbols
	ResponseTo    interface{} // Widget or component to send response to
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
			sigHelp, err := proxyClient.GetSignatureHelp(request.FilePath, request.Line, request.Character)
			response := LSPResponse{
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
			symbols, err := proxyClient.ListProjectSymbols(request.Query)
			response := LSPResponse{
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
				if ch, ok := request.ResponseTo.(chan LSPResponse); ok {
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
