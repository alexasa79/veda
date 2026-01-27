package clipboard

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"time"
)

// OSC 52 escape sequences for clipboard access
// Format: ESC ] 52 ; c ; <base64-data> BEL
// Query:  ESC ] 52 ; c ; ? BEL
// Response: ESC ] 52 ; c ; <base64-data> BEL (or ESC \)

const (
	osc52Prefix = "\033]52;c;"
	osc52Query  = "\033]52;c;?\007"
	bel         = '\007'
	esc         = '\033'
)

// TerminalClipboard provides OSC 52 clipboard operations through a terminal.
// This is implemented by the terminal package.
type TerminalClipboard interface {
	CopyToClipboard(text string) error
	QueryClipboard(timeout time.Duration) (string, error)
}

var terminalClipboard TerminalClipboard

// SetTerminalClipboard sets the terminal clipboard interface.
// This should be called by the editor after the terminal is initialized.
func SetTerminalClipboard(tc TerminalClipboard) {
	terminalClipboard = tc
}

// TerminalIO provides access to terminal stdin/stdout for OSC 52 operations.
// This interface allows mocking in tests.
type TerminalIO interface {
	Write(p []byte) (n int, err error)
	Read(p []byte) (n int, err error)
	SetReadDeadline(t time.Time) error
}

// defaultTerminalIO wraps os.Stdin/os.Stdout
type defaultTerminalIO struct{}

func (d *defaultTerminalIO) Write(p []byte) (n int, err error) {
	return os.Stdout.Write(p)
}

func (d *defaultTerminalIO) Read(p []byte) (n int, err error) {
	return os.Stdin.Read(p)
}

func (d *defaultTerminalIO) SetReadDeadline(t time.Time) error {
	// os.Stdin doesn't support deadlines directly
	// We'll handle timeout differently
	return nil
}

var termIO TerminalIO = &defaultTerminalIO{}

// SetTerminalIO sets the terminal I/O interface (for testing)
func SetTerminalIO(t TerminalIO) {
	termIO = t
}

// ResetTerminalIO resets to default terminal I/O
func ResetTerminalIO() {
	termIO = &defaultTerminalIO{}
}

// OSC52Copy copies text to clipboard using OSC 52 escape sequence
func OSC52Copy(text string) error {
	// Use terminal clipboard if available (preferred for proper coordination)
	if terminalClipboard != nil {
		return terminalClipboard.CopyToClipboard(text)
	}

	// Fallback to direct write (may not work if terminal input loop is active)
	encoded := base64.StdEncoding.EncodeToString([]byte(text))
	seq := fmt.Sprintf("%s%s\007", osc52Prefix, encoded)
	_, err := termIO.Write([]byte(seq))
	return err
}

// OSC52Paste reads clipboard using OSC 52 escape sequence.
// This requires terminal support and may not work in all environments.
// timeout specifies how long to wait for the terminal response.
func OSC52Paste(timeout time.Duration) (string, error) {
	// Use terminal clipboard if available (required for proper coordination with input loop)
	if terminalClipboard != nil {
		return terminalClipboard.QueryClipboard(timeout)
	}

	// Fallback to direct read (may not work if terminal input loop is active)
	// Send the query
	_, err := termIO.Write([]byte(osc52Query))
	if err != nil {
		return "", fmt.Errorf("failed to send OSC 52 query: %w", err)
	}

	// Read response with timeout
	response, err := readOSC52Response(timeout)
	if err != nil {
		return "", err
	}

	return response, nil
}

// readOSC52Response reads and parses an OSC 52 response from the terminal.
// Expected format: ESC ] 52 ; c ; <base64-data> BEL (or ESC \)
func readOSC52Response(timeout time.Duration) (string, error) {
	buf := make([]byte, 1)
	var response bytes.Buffer
	deadline := time.Now().Add(timeout)

	// State machine for parsing OSC 52 response
	// States: 0=waiting for ESC, 1=got ESC waiting for ], 2=reading until terminator
	state := 0
	inPayload := false

	for time.Now().Before(deadline) {
		// Use a channel with timeout for reading
		readCh := make(chan struct {
			n   int
			err error
		}, 1)

		go func() {
			n, err := termIO.Read(buf)
			readCh <- struct {
				n   int
				err error
			}{n, err}
		}()

		select {
		case result := <-readCh:
			if result.err != nil {
				if result.err == io.EOF {
					return "", fmt.Errorf("unexpected EOF while reading OSC 52 response")
				}
				return "", result.err
			}
			if result.n == 0 {
				continue
			}

			b := buf[0]

			switch state {
			case 0: // Waiting for ESC
				if b == esc {
					state = 1
				}
			case 1: // Got ESC, waiting for ]
				if b == ']' {
					state = 2
					response.Reset()
				} else {
					state = 0 // Reset, not an OSC sequence
				}
			case 2: // Reading OSC content
				if b == bel {
					// End of OSC sequence
					return parseOSC52Payload(response.String())
				} else if b == esc {
					// Could be ESC \ terminator, peek next
					state = 3
				} else {
					response.WriteByte(b)
					// Check if we've seen "52;c;" prefix
					if !inPayload && response.Len() >= 5 {
						if bytes.HasPrefix(response.Bytes(), []byte("52;c;")) {
							inPayload = true
						}
					}
				}
			case 3: // Got ESC in OSC, check for \
				if b == '\\' {
					// ESC \ terminator
					return parseOSC52Payload(response.String())
				} else {
					// Not a terminator, add ESC to response and continue
					response.WriteByte(esc)
					response.WriteByte(b)
					state = 2
				}
			}

		case <-time.After(time.Until(deadline)):
			return "", fmt.Errorf("timeout waiting for OSC 52 response")
		}
	}

	return "", fmt.Errorf("timeout waiting for OSC 52 response")
}

// parseOSC52Payload extracts and decodes the base64 payload from OSC 52 response
func parseOSC52Payload(content string) (string, error) {
	// Expected format: "52;c;<base64-data>"
	const prefix = "52;c;"
	if len(content) < len(prefix) {
		return "", fmt.Errorf("invalid OSC 52 response: too short")
	}
	if content[:len(prefix)] != prefix {
		return "", fmt.Errorf("invalid OSC 52 response: wrong prefix")
	}

	encoded := content[len(prefix):]
	if encoded == "" {
		return "", nil // Empty clipboard
	}

	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("failed to decode OSC 52 payload: %w", err)
	}

	return string(decoded), nil
}

// BuildOSC52CopySequence builds the OSC 52 escape sequence for copying text.
// This is useful for testing or when you need the raw sequence.
func BuildOSC52CopySequence(text string) []byte {
	encoded := base64.StdEncoding.EncodeToString([]byte(text))
	return []byte(fmt.Sprintf("%s%s\007", osc52Prefix, encoded))
}

// BuildOSC52Response builds a mock OSC 52 response for testing.
func BuildOSC52Response(text string) []byte {
	encoded := base64.StdEncoding.EncodeToString([]byte(text))
	return []byte(fmt.Sprintf("\033]52;c;%s\007", encoded))
}
