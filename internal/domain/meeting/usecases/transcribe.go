package usecases

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// Transcribe handles audio transcription via Mistral Voxtral API.
type Transcribe struct {
	APIKey string
}

// TranscriptSegment represents a diarized segment of the transcript.
type TranscriptSegment struct {
	Speaker string `json:"speaker"`
	Text    string `json:"text"`
}

// TranscriptResult holds the full transcription result.
type TranscriptResult struct {
	Text     string              `json:"text"`
	Segments []TranscriptSegment `json:"segments"`
}

// Execute transcribes the audio file and writes transcript.md to the meeting directory.
func (t *Transcribe) Execute(audioPath string, meetingDir string) (*TranscriptResult, error) {
	if t.APIKey == "" {
		return nil, fmt.Errorf("mistral API key not set: set MEETINGCLI_MISTRAL_API_KEY or add mistral_api_key to config")
	}

	// Build multipart request
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add model field
	if err := writer.WriteField("model", "voxtral-mini-latest"); err != nil {
		return nil, err
	}

	// Add diarize field
	if err := writer.WriteField("diarize", "true"); err != nil {
		return nil, err
	}

	// Add audio file
	file, err := os.Open(audioPath)
	if err != nil {
		return nil, fmt.Errorf("opening audio file: %w", err)
	}
	defer file.Close()

	part, err := writer.CreateFormFile("file", filepath.Base(audioPath))
	if err != nil {
		return nil, err
	}
	if _, err := io.Copy(part, file); err != nil {
		return nil, err
	}

	if err := writer.Close(); err != nil {
		return nil, err
	}

	// Make request
	req, err := http.NewRequest("POST", "https://api.mistral.ai/v1/audio/transcriptions", body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+t.APIKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("calling Mistral API: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("mistral API error (HTTP %d): %s", resp.StatusCode, string(respBody))
	}

	// Parse response
	var apiResp transcriptionAPIResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("parsing Mistral response: %w", err)
	}

	result := &TranscriptResult{
		Text: apiResp.Text,
	}

	// Extract diarized segments if available
	for _, seg := range apiResp.Segments {
		result.Segments = append(result.Segments, TranscriptSegment{
			Speaker: seg.Speaker,
			Text:    seg.Text,
		})
	}

	// Write transcript.md
	transcriptPath := filepath.Join(meetingDir, "transcript.md")
	content := formatTranscript(result)
	if err := os.WriteFile(transcriptPath, []byte(content), 0o644); err != nil {
		return nil, fmt.Errorf("writing transcript: %w", err)
	}

	return result, nil
}

func formatTranscript(result *TranscriptResult) string {
	var sb strings.Builder
	sb.WriteString("# Meeting Transcript\n\n")

	if len(result.Segments) > 0 {
		currentSpeaker := ""
		for _, seg := range result.Segments {
			if seg.Speaker != currentSpeaker {
				currentSpeaker = seg.Speaker
				speaker := currentSpeaker
				if speaker == "" {
					speaker = "Unknown"
				}
				sb.WriteString(fmt.Sprintf("\n**%s:**\n", speaker))
			}
			sb.WriteString(seg.Text + " ")
		}
	} else {
		// No diarization — just dump the full text
		sb.WriteString(result.Text)
	}

	sb.WriteString("\n")
	return sb.String()
}

// transcriptionAPIResponse matches the Mistral transcription API response.
type transcriptionAPIResponse struct {
	Text     string `json:"text"`
	Segments []struct {
		Speaker string `json:"speaker"`
		Text    string `json:"text"`
	} `json:"segments"`
}
