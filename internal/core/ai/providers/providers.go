// Package providers implements AI provider clients.
// This package provides implementations for OpenAI, DeepSeek, and Ollama.
package providers

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Message represents a chat message.
type Message struct {
	Role    string `json:"role"`    // system, user, assistant
	Content string `json:"content"`
}

// ChatRequest represents a chat request.
type ChatRequest struct {
	Messages    []Message `json:"messages"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
	Temperature float64   `json:"temperature,omitempty"`
}

// ChatResponse represents a chat response.
type ChatResponse struct {
	Content      string `json:"content"`
	TokensUsed   int    `json:"tokens_used"`
	FinishReason string `json:"finish_reason"`
	Model        string `json:"model"`
}

// StreamChunk represents a streaming response chunk.
type StreamChunk struct {
	Content string `json:"content"`
	Done    bool   `json:"done"`
	Error   error  `json:"error,omitempty"`
}

// Provider defines the interface for AI providers.
type Provider interface {
	// Chat sends a message and receives a response.
	Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error)

	// StreamChat sends a message and receives a streaming response.
	StreamChat(ctx context.Context, req *ChatRequest) (<-chan StreamChunk, error)

	// CountTokens estimates the number of tokens in the text.
	CountTokens(text string) int

	// Name returns the provider name.
	Name() string
}

// OpenAIProvider implements the Provider interface for OpenAI.
type OpenAIProvider struct {
	apiKey     string
	model      string
	baseURL    string
	httpClient *http.Client
}

// NewOpenAIProvider creates a new OpenAI provider.
func NewOpenAIProvider(apiKey, model, baseURL string) *OpenAIProvider {
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	if model == "" {
		model = "gpt-4o"
	}
	
	// Create transport that ignores proxy
	transport := &http.Transport{
		Proxy: nil, // Disable proxy
	}
	
	return &OpenAIProvider{
		apiKey:  apiKey,
		model:   model,
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout:   120 * time.Second,
			Transport: transport,
		},
	}
}

// openAIRequest represents an OpenAI API request.
type openAIRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
	Temperature float64   `json:"temperature,omitempty"`
	Stream      bool      `json:"stream,omitempty"`
}

// openAIResponse represents an OpenAI API response.
type openAIResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index        int     `json:"index"`
		Message      Message `json:"message"`
		FinishReason string  `json:"finish_reason"`
		Delta        *struct {
			Content string `json:"content"`
		} `json:"delta,omitempty"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error,omitempty"`
}

// Chat sends a chat request to OpenAI.
func (p *OpenAIProvider) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	body := openAIRequest{
		Model:       p.model,
		Messages:    req.Messages,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Ensure baseURL ends correctly
	apiURL := strings.TrimSpace(p.baseURL)
	
	// If baseURL already ends with /chat/completions, use it directly
	if strings.HasSuffix(apiURL, "/chat/completions") {
		// Use as-is
	} else if strings.HasSuffix(apiURL, "/v1") || strings.HasSuffix(apiURL, "/v2") {
		// If baseURL ends with /v1 or /v2, append /chat/completions
		apiURL = apiURL + "/chat/completions"
	} else if !strings.HasSuffix(apiURL, "/") {
		// If baseURL doesn't end with /, add it then append chat/completions
		apiURL = apiURL + "/chat/completions"
	} else {
		// baseURL ends with /, just append chat/completions
		apiURL = apiURL + "chat/completions"
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response body for debugging
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check if response is not OK
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var openAIResp openAIResponse
	if err := json.Unmarshal(respBody, &openAIResp); err != nil {
		preview := string(respBody)
		if len(preview) > 200 {
			preview = preview[:200]
		}
		return nil, fmt.Errorf("failed to decode response: %w (body: %s)", err, preview)
	}

	if openAIResp.Error != nil {
		return nil, fmt.Errorf("OpenAI API error: %s", openAIResp.Error.Message)
	}

	if len(openAIResp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	return &ChatResponse{
		Content:      openAIResp.Choices[0].Message.Content,
		TokensUsed:   openAIResp.Usage.TotalTokens,
		FinishReason: openAIResp.Choices[0].FinishReason,
		Model:        openAIResp.Model,
	}, nil
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// StreamChat sends a streaming chat request to OpenAI.
func (p *OpenAIProvider) StreamChat(ctx context.Context, req *ChatRequest) (<-chan StreamChunk, error) {
	body := openAIRequest{
		Model:       p.model,
		Messages:    req.Messages,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
		Stream:      true,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/chat/completions", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	ch := make(chan StreamChunk, 100)

	go func() {
		defer close(ch)
		defer resp.Body.Close()

		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Text()
			if !strings.HasPrefix(line, "data: ") {
				continue
			}

			data := strings.TrimPrefix(line, "data: ")
			if data == "[DONE]" {
				ch <- StreamChunk{Done: true}
				return
			}

			var streamResp openAIResponse
			if err := json.Unmarshal([]byte(data), &streamResp); err != nil {
				ch <- StreamChunk{Error: err}
				return
			}

			if streamResp.Error != nil {
				ch <- StreamChunk{Error: fmt.Errorf(streamResp.Error.Message)}
				return
			}

			if len(streamResp.Choices) > 0 && streamResp.Choices[0].Delta != nil {
				ch <- StreamChunk{
					Content: streamResp.Choices[0].Delta.Content,
				}
			}
		}

		if err := scanner.Err(); err != nil {
			ch <- StreamChunk{Error: err}
		}
	}()

	return ch, nil
}

// CountTokens estimates token count for OpenAI models.
func (p *OpenAIProvider) CountTokens(text string) int {
	charCount := len(text)
	chineseRatio := float64(countChineseChars(text)) / float64(charCount+1)
	if chineseRatio > 0.5 {
		return charCount / 2
	}
	return charCount / 4
}

// Name returns the provider name.
func (p *OpenAIProvider) Name() string {
	return "openai"
}

// DeepSeekProvider implements the Provider interface for DeepSeek.
type DeepSeekProvider struct {
	apiKey     string
	model      string
	baseURL    string
	httpClient *http.Client
}

// NewDeepSeekProvider creates a new DeepSeek provider.
func NewDeepSeekProvider(apiKey, model, baseURL string) *DeepSeekProvider {
	if baseURL == "" {
		baseURL = "https://api.deepseek.com/v1"
	}
	if model == "" {
		model = "deepseek-chat"
	}
	return &DeepSeekProvider{
		apiKey:  apiKey,
		model:   model,
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

// Chat sends a chat request to DeepSeek.
func (p *DeepSeekProvider) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	body := openAIRequest{
		Model:       p.model,
		Messages:    req.Messages,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/chat/completions", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	var deepseekResp openAIResponse
	if err := json.NewDecoder(resp.Body).Decode(&deepseekResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if deepseekResp.Error != nil {
		return nil, fmt.Errorf("DeepSeek API error: %s", deepseekResp.Error.Message)
	}

	if len(deepseekResp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	return &ChatResponse{
		Content:      deepseekResp.Choices[0].Message.Content,
		TokensUsed:   deepseekResp.Usage.TotalTokens,
		FinishReason: deepseekResp.Choices[0].FinishReason,
		Model:        deepseekResp.Model,
	}, nil
}

// StreamChat sends a streaming chat request to DeepSeek.
func (p *DeepSeekProvider) StreamChat(ctx context.Context, req *ChatRequest) (<-chan StreamChunk, error) {
	body := openAIRequest{
		Model:       p.model,
		Messages:    req.Messages,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
		Stream:      true,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/chat/completions", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	ch := make(chan StreamChunk, 100)

	go func() {
		defer close(ch)
		defer resp.Body.Close()

		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Text()
			if !strings.HasPrefix(line, "data: ") {
				continue
			}

			data := strings.TrimPrefix(line, "data: ")
			if data == "[DONE]" {
				ch <- StreamChunk{Done: true}
				return
			}

			var streamResp openAIResponse
			if err := json.Unmarshal([]byte(data), &streamResp); err != nil {
				ch <- StreamChunk{Error: err}
				return
			}

			if streamResp.Error != nil {
				ch <- StreamChunk{Error: fmt.Errorf(streamResp.Error.Message)}
				return
			}

			if len(streamResp.Choices) > 0 && streamResp.Choices[0].Delta != nil {
				ch <- StreamChunk{
					Content: streamResp.Choices[0].Delta.Content,
				}
			}
		}

		if err := scanner.Err(); err != nil {
			ch <- StreamChunk{Error: err}
		}
	}()

	return ch, nil
}

// CountTokens estimates token count for DeepSeek models.
func (p *DeepSeekProvider) CountTokens(text string) int {
	return countChineseChars(text)/2 + (len(text)-countChineseChars(text))/4
}

// Name returns the provider name.
func (p *DeepSeekProvider) Name() string {
	return "deepseek"
}

// OllamaProvider implements the Provider interface for Ollama.
type OllamaProvider struct {
	baseURL    string
	model      string
	httpClient *http.Client
}

// NewOllamaProvider creates a new Ollama provider.
func NewOllamaProvider(baseURL, model string) *OllamaProvider {
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}
	if model == "" {
		model = "llama3"
	}
	return &OllamaProvider{
		baseURL: baseURL,
		model:   model,
		httpClient: &http.Client{
			Timeout: 300 * time.Second,
		},
	}
}

// ollamaRequest represents an Ollama API request.
type ollamaRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
	Stream   bool      `json:"stream"`
	Options  struct {
		Temperature float64 `json:"temperature,omitempty"`
		NumPredict  int     `json:"num_predict,omitempty"`
	} `json:"options,omitempty"`
}

// ollamaResponse represents an Ollama API response.
type ollamaResponse struct {
	Model     string  `json:"model"`
	CreatedAt string  `json:"created_at"`
	Message   Message `json:"message"`
	Done      bool    `json:"done"`
	Error     string  `json:"error,omitempty"`
}

// Chat sends a chat request to Ollama.
func (p *OllamaProvider) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	body := ollamaRequest{
		Model:    p.model,
		Messages: req.Messages,
		Stream:   false,
	}
	body.Options.Temperature = req.Temperature
	body.Options.NumPredict = req.MaxTokens

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/api/chat", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(bodyBytes)), "\n")
	var lastResp ollamaResponse
	for _, line := range lines {
		if line == "" {
			continue
		}
		var ollamaResp ollamaResponse
		if err := json.Unmarshal([]byte(line), &ollamaResp); err != nil {
			continue
		}
		lastResp = ollamaResp
	}

	if lastResp.Error != "" {
		return nil, fmt.Errorf("Ollama error: %s", lastResp.Error)
	}

	return &ChatResponse{
		Content:      lastResp.Message.Content,
		TokensUsed:   0,
		FinishReason: "stop",
		Model:        lastResp.Model,
	}, nil
}

// StreamChat sends a streaming chat request to Ollama.
func (p *OllamaProvider) StreamChat(ctx context.Context, req *ChatRequest) (<-chan StreamChunk, error) {
	body := ollamaRequest{
		Model:    p.model,
		Messages: req.Messages,
		Stream:   true,
	}
	body.Options.Temperature = req.Temperature
	body.Options.NumPredict = req.MaxTokens

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/api/chat", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	ch := make(chan StreamChunk, 100)

	go func() {
		defer close(ch)
		defer resp.Body.Close()

		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Text()
			if line == "" {
				continue
			}

			var ollamaResp ollamaResponse
			if err := json.Unmarshal([]byte(line), &ollamaResp); err != nil {
				ch <- StreamChunk{Error: err}
				return
			}

			if ollamaResp.Error != "" {
				ch <- StreamChunk{Error: fmt.Errorf(ollamaResp.Error)}
				return
			}

			ch <- StreamChunk{
				Content: ollamaResp.Message.Content,
				Done:    ollamaResp.Done,
			}

			if ollamaResp.Done {
				return
			}
		}

		if err := scanner.Err(); err != nil {
			ch <- StreamChunk{Error: err}
		}
	}()

	return ch, nil
}

// CountTokens estimates token count for Ollama models.
func (p *OllamaProvider) CountTokens(text string) int {
	return len(text) / 4
}

// Name returns the provider name.
func (p *OllamaProvider) Name() string {
	return "ollama"
}

// countChineseChars counts Chinese characters in text.
func countChineseChars(text string) int {
	count := 0
	for _, r := range text {
		if r >= 0x4e00 && r <= 0x9fff {
			count++
		}
	}
	return count
}
