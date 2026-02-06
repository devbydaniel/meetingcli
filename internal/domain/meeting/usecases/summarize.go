package usecases

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

// Summarize generates a meeting summary using Claude Haiku 4.5.
type Summarize struct {
	APIKey       string
	SystemPrompt string
}

// Execute generates a summary from the transcript and writes summary.md.
func (s *Summarize) Execute(transcript string, meetingDir string) (string, error) {
	if s.APIKey == "" {
		return "", fmt.Errorf("anthropic API key not set: set MEETINGCLI_ANTHROPIC_API_KEY or add anthropic_api_key to config")
	}

	reqBody := anthropicRequest{
		Model:     "claude-haiku-4-5",
		MaxTokens: 4096,
		System:    s.SystemPrompt,
		Messages: []anthropicMessage{
			{
				Role:    "user",
				Content: "Here is the meeting transcript to summarize:\n\n" + transcript,
			},
		},
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", "https://api.anthropic.com/v1/messages", bytes.NewReader(jsonBody))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", s.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("calling Anthropic API: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("anthropic API error (HTTP %d): %s", resp.StatusCode, string(respBody))
	}

	var apiResp anthropicResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return "", fmt.Errorf("parsing Anthropic response: %w", err)
	}

	// Extract text from response
	var summary string
	for _, block := range apiResp.Content {
		if block.Type == "text" {
			summary += block.Text
		}
	}

	if summary == "" {
		return "", fmt.Errorf("empty response from Anthropic API")
	}

	// Write summary.md
	summaryContent := "# Meeting Summary\n\n" + summary + "\n"
	summaryPath := filepath.Join(meetingDir, "summary.md")
	if err := os.WriteFile(summaryPath, []byte(summaryContent), 0o644); err != nil {
		return "", fmt.Errorf("writing summary: %w", err)
	}

	return summary, nil
}

type anthropicRequest struct {
	Model     string             `json:"model"`
	MaxTokens int                `json:"max_tokens"`
	System    string             `json:"system"`
	Messages  []anthropicMessage `json:"messages"`
}

type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicResponse struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
}
