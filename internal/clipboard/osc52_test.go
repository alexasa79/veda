package clipboard

import (
	"bytes"
	"encoding/base64"
	"io"
	"testing"
	"time"
)

// mockTerminalIO simulates terminal I/O for testing
type mockTerminalIO struct {
	writeBuffer bytes.Buffer
	readBuffer  bytes.Buffer
	readErr     error
}

func (m *mockTerminalIO) Write(p []byte) (n int, err error) {
	return m.writeBuffer.Write(p)
}

func (m *mockTerminalIO) Read(p []byte) (n int, err error) {
	if m.readErr != nil {
		return 0, m.readErr
	}
	return m.readBuffer.Read(p)
}

func (m *mockTerminalIO) SetReadDeadline(t time.Time) error {
	return nil
}

func TestBuildOSC52CopySequence(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple text",
			input:    "hello",
			expected: "\033]52;c;" + base64.StdEncoding.EncodeToString([]byte("hello")) + "\007",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "\033]52;c;\007",
		},
		{
			name:     "multiline text",
			input:    "line1\nline2\nline3",
			expected: "\033]52;c;" + base64.StdEncoding.EncodeToString([]byte("line1\nline2\nline3")) + "\007",
		},
		{
			name:     "unicode text",
			input:    "héllo wörld 你好",
			expected: "\033]52;c;" + base64.StdEncoding.EncodeToString([]byte("héllo wörld 你好")) + "\007",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BuildOSC52CopySequence(tt.input)
			if string(result) != tt.expected {
				t.Errorf("BuildOSC52CopySequence(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestBuildOSC52Response(t *testing.T) {
	text := "test clipboard content"
	response := BuildOSC52Response(text)

	expected := "\033]52;c;" + base64.StdEncoding.EncodeToString([]byte(text)) + "\007"
	if string(response) != expected {
		t.Errorf("BuildOSC52Response(%q) = %q, want %q", text, response, expected)
	}
}

func TestParseOSC52Payload(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
		wantErr bool
	}{
		{
			name:    "valid payload",
			content: "52;c;" + base64.StdEncoding.EncodeToString([]byte("hello")),
			want:    "hello",
			wantErr: false,
		},
		{
			name:    "empty payload",
			content: "52;c;",
			want:    "",
			wantErr: false,
		},
		{
			name:    "multiline payload",
			content: "52;c;" + base64.StdEncoding.EncodeToString([]byte("line1\nline2")),
			want:    "line1\nline2",
			wantErr: false,
		},
		{
			name:    "unicode payload",
			content: "52;c;" + base64.StdEncoding.EncodeToString([]byte("日本語")),
			want:    "日本語",
			wantErr: false,
		},
		{
			name:    "wrong prefix",
			content: "51;c;aGVsbG8=",
			want:    "",
			wantErr: true,
		},
		{
			name:    "too short",
			content: "52;",
			want:    "",
			wantErr: true,
		},
		{
			name:    "invalid base64",
			content: "52;c;not-valid-base64!!!",
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseOSC52Payload(tt.content)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseOSC52Payload() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("parseOSC52Payload() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestOSC52Copy(t *testing.T) {
	mock := &mockTerminalIO{}
	SetTerminalIO(mock)
	defer ResetTerminalIO()

	text := "test copy text"
	err := OSC52Copy(text)
	if err != nil {
		t.Fatalf("OSC52Copy() error = %v", err)
	}

	expected := BuildOSC52CopySequence(text)
	if !bytes.Equal(mock.writeBuffer.Bytes(), expected) {
		t.Errorf("OSC52Copy() wrote %q, want %q", mock.writeBuffer.Bytes(), expected)
	}
}

func TestOSC52Paste(t *testing.T) {
	tests := []struct {
		name     string
		response []byte
		want     string
		wantErr  bool
	}{
		{
			name:     "valid response with BEL terminator",
			response: BuildOSC52Response("clipboard content"),
			want:     "clipboard content",
			wantErr:  false,
		},
		{
			name:     "valid response with ESC \\ terminator",
			response: []byte("\033]52;c;" + base64.StdEncoding.EncodeToString([]byte("test")) + "\033\\"),
			want:     "test",
			wantErr:  false,
		},
		{
			name:     "empty clipboard",
			response: []byte("\033]52;c;\007"),
			want:     "",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockTerminalIO{}
			mock.readBuffer.Write(tt.response)
			SetTerminalIO(mock)
			defer ResetTerminalIO()

			got, err := OSC52Paste(100 * time.Millisecond)
			if (err != nil) != tt.wantErr {
				t.Errorf("OSC52Paste() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("OSC52Paste() = %q, want %q", got, tt.want)
			}

			// Verify the query was sent
			expectedQuery := []byte(osc52Query)
			if !bytes.Equal(mock.writeBuffer.Bytes(), expectedQuery) {
				t.Errorf("OSC52Paste() sent %q, want %q", mock.writeBuffer.Bytes(), expectedQuery)
			}
		})
	}
}

func TestOSC52PasteTimeout(t *testing.T) {
	mock := &mockTerminalIO{}
	// Don't write any response - should timeout
	mock.readErr = io.EOF
	SetTerminalIO(mock)
	defer ResetTerminalIO()

	_, err := OSC52Paste(50 * time.Millisecond)
	if err == nil {
		t.Error("OSC52Paste() expected timeout error, got nil")
	}
}
