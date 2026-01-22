package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/0mykull/gitty/internal/config"
)

const (
	OpenAIURL    = "https://api.openai.com/v1/chat/completions"
	AnthropicURL = "https://api.anthropic.com/v1/messages"
)

// OpenAI types
type openAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIRequest struct {
	Model       string          `json:"model"`
	Messages    []openAIMessage `json:"messages"`
	Temperature float64         `json:"temperature"`
}

type openAIResponse struct {
	Choices []struct {
		Message openAIMessage `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// Anthropic types
type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicRequest struct {
	Model       string             `json:"model"`
	MaxTokens   int                `json:"max_tokens"`
	System      string             `json:"system,omitempty"`
	Messages    []anthropicMessage `json:"messages"`
	Temperature float64            `json:"temperature,omitempty"`
}

type anthropicResponse struct {
	Content []struct {
		Text string `json:"text"`
		Type string `json:"type"`
	} `json:"content"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// GenerateCommitMessage generates a commit message from a diff using AI
func GenerateCommitMessage(diff string, cfg *config.Config) (string, error) {
	if cfg.AI.APIKey == "" {
		return "", fmt.Errorf("API key not configured. Set it in ~/.config/gitty/config.yaml or OPENAI_API_KEY env var")
	}

	// Truncate diff if too long
	if len(diff) > cfg.AI.MaxDiffSize {
		diff = diff[:cfg.AI.MaxDiffSize] + "\n...(truncated)"
	}

	systemPrompt := `You are a skilled developer writing git commit messages.
Format the message strictly as follows:
1. A single concise subject line (max 50 chars) that describes WHAT changed.
2. A blank line.
3. A detailed bulleted list of changes explaining WHY and HOW.

Use conventional commit prefixes when appropriate:
- feat: new feature
- fix: bug fix
- refactor: code refactoring
- docs: documentation changes
- style: formatting changes
- test: adding tests
- chore: maintenance tasks

IMPORTANT: Return raw text only. Do NOT wrap in markdown code blocks.`

	userPrompt := fmt.Sprintf("Generate a commit message for this diff:\n\n%s", diff)

	switch cfg.AI.Provider {
	case "anthropic":
		return generateAnthropicCommit(systemPrompt, userPrompt, cfg)
	default:
		return generateOpenAICommit(systemPrompt, userPrompt, cfg)
	}
}

func generateOpenAICommit(systemPrompt, userPrompt string, cfg *config.Config) (string, error) {
	reqBody := openAIRequest{
		Model: cfg.AI.Model,
		Messages: []openAIMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		Temperature: cfg.AI.Temperature,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", OpenAIURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+cfg.AI.APIKey)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("API call failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	var apiResp openAIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if apiResp.Error != nil {
		return "", fmt.Errorf("OpenAI error: %s", apiResp.Error.Message)
	}

	if len(apiResp.Choices) == 0 {
		return "", fmt.Errorf("no response from OpenAI")
	}

	content := strings.TrimSpace(apiResp.Choices[0].Message.Content)
	content = cleanMarkdown(content)

	return content, nil
}

func generateAnthropicCommit(systemPrompt, userPrompt string, cfg *config.Config) (string, error) {
	model := cfg.AI.Model
	if !strings.HasPrefix(model, "claude") {
		model = "claude-3-5-sonnet-20241022"
	}

	reqBody := anthropicRequest{
		Model:     model,
		MaxTokens: 1024,
		System:    systemPrompt,
		Messages: []anthropicMessage{
			{Role: "user", Content: userPrompt},
		},
		Temperature: cfg.AI.Temperature,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", AnthropicURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", cfg.AI.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("API call failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	var apiResp anthropicResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if apiResp.Error != nil {
		return "", fmt.Errorf("Anthropic error: %s", apiResp.Error.Message)
	}

	if len(apiResp.Content) == 0 {
		return "", fmt.Errorf("no response from Anthropic")
	}

	content := strings.TrimSpace(apiResp.Content[0].Text)
	content = cleanMarkdown(content)

	return content, nil
}

func cleanMarkdown(content string) string {
	// Remove markdown code blocks
	content = strings.ReplaceAll(content, "```markdown", "")
	content = strings.ReplaceAll(content, "```", "")
	return strings.TrimSpace(content)
}
