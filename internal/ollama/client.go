package ollama

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	DefaultEndpoint = "http://localhost:11434"
	DefaultModel    = "mistral:7b"
)

// Client handles communication with the Ollama API
type Client struct {
	endpoint   string
	model      string
	httpClient *http.Client
}

// NewClient creates a new Ollama client with default settings
func NewClient() *Client {
	return &Client{
		endpoint: DefaultEndpoint,
		model:    DefaultModel,
		httpClient: &http.Client{
			Timeout: 120 * time.Second, // LLM responses can take a while
		},
	}
}

// NewClientWithOptions creates a new Ollama client with custom settings
func NewClientWithOptions(endpoint, model string) *Client {
	if endpoint == "" {
		endpoint = DefaultEndpoint
	}
	if model == "" {
		model = DefaultModel
	}
	return &Client{
		endpoint: endpoint,
		model:    model,
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

// GenerateOptions contains parameters to control LLM behavior
type GenerateOptions struct {
	Temperature   float64 `json:"temperature,omitempty"`    // 0.0-1.0, lower = more deterministic
	TopP          float64 `json:"top_p,omitempty"`          // Nucleus sampling threshold
	TopK          int     `json:"top_k,omitempty"`          // Limits vocabulary choices
	RepeatPenalty float64 `json:"repeat_penalty,omitempty"` // Penalizes repetition
	Seed          int     `json:"seed,omitempty"`           // For reproducibility (0 = random)
}

// GenerateRequest represents a request to the Ollama generate API
type GenerateRequest struct {
	Model   string           `json:"model"`
	Prompt  string           `json:"prompt"`
	Stream  bool             `json:"stream"`
	Options *GenerateOptions `json:"options,omitempty"`
}

// GenerateResponse represents a response from the Ollama generate API
type GenerateResponse struct {
	Model     string `json:"model"`
	Response  string `json:"response"`
	Done      bool   `json:"done"`
	Error     string `json:"error,omitempty"`
	CreatedAt string `json:"created_at,omitempty"`
}

// Generate sends a prompt to Ollama and returns the response
func (c *Client) Generate(prompt string) (string, error) {
	reqBody := GenerateRequest{
		Model:  c.model,
		Prompt: prompt,
		Stream: false,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	url := c.endpoint + "/api/generate"
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request to Ollama: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("Ollama returned status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	var result GenerateResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if result.Error != "" {
		return "", fmt.Errorf("Ollama error: %s", result.Error)
	}

	return result.Response, nil
}

// SpellCheckAndFix corrects spelling and punctuation errors in the given text
// Returns the corrected text (or original if no errors found)
func (c *Client) SpellCheckAndFix(text string) (string, error) {
	prompt := fmt.Sprintf(`Fix spelling and punctuation errors in the text below.

Rules:
- Return ONLY the corrected text, nothing else
- Do NOT add or remove line breaks
- Do NOT split sentences into multiple lines
- Do NOT add paragraph breaks
- Do NOT change line lengths
- Do NOT add any extra text, commentary, or explanation
- Keep the exact same structure, just fix typos and punctuation
- If no errors, return the text unchanged

Text:
%s`, text)

	return c.Generate(prompt)
}

// GenerateWithOptions sends a prompt to Ollama with custom options
func (c *Client) GenerateWithOptions(prompt string, options *GenerateOptions) (string, error) {
	reqBody := GenerateRequest{
		Model:   c.model,
		Prompt:  prompt,
		Stream:  false,
		Options: options,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	url := c.endpoint + "/api/generate"
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request to Ollama: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("Ollama returned status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	var result GenerateResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if result.Error != "" {
		return "", fmt.Errorf("Ollama error: %s", result.Error)
	}

	return result.Response, nil
}

// SetModel changes the model used by the client
func (c *Client) SetModel(model string) {
	if model != "" {
		c.model = model
	}
}

// SetEndpoint changes the endpoint used by the client
func (c *Client) SetEndpoint(endpoint string) {
	if endpoint != "" {
		c.endpoint = strings.TrimSuffix(endpoint, "/")
	}
}

// GetModel returns the current model
func (c *Client) GetModel() string {
	return c.model
}

// GetEndpoint returns the current endpoint
func (c *Client) GetEndpoint() string {
	return c.endpoint
}
