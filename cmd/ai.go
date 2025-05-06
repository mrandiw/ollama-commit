package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// OllamaRequest represents a request to the Ollama API
type OllamaRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

// OllamaResponse represents a response from the Ollama API
// The Ollama API might return the response in different formats
// We'll handle multiple possible response structures
type OllamaResponse struct {
	Response string `json:"response"`
	Content  string `json:"content"` // Some versions use content instead of response
}

// GenerateCommitMessage generates a commit message using the Ollama API
func GenerateCommitMessage(gitDiff, model, apiURL, promptTemplate string) (string, error) {
	// Prepare prompt for Ollama
	prompt := fmt.Sprintf(promptTemplate, gitDiff)

	// Prepare request to Ollama API
	ollamaReq := OllamaRequest{
		Model:  model,
		Prompt: prompt,
		Stream: false, // We want the complete response, not streamed
	}

	reqBody, err := json.Marshal(ollamaReq)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %v", err)
	}

	// Send request to Ollama API
	resp, err := http.Post(apiURL, "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		return "", fmt.Errorf("failed to call Ollama API: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("Ollama API returned non-OK status: %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	// Read the full response body
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %v", err)
	}

	// For debugging
	// fmt.Printf("Raw API Response: %s\n", string(bodyBytes))

	// Parse response
	var ollamaResp OllamaResponse
	if err := json.Unmarshal(bodyBytes, &ollamaResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %v", err)
	}

	// Check which field has the content
	var commitMsg string
	if ollamaResp.Response != "" {
		commitMsg = strings.TrimSpace(ollamaResp.Response)
	} else if ollamaResp.Content != "" {
		commitMsg = strings.TrimSpace(ollamaResp.Content)
	} else {
		// Try to find any relevant text in the response
		if strings.Contains(string(bodyBytes), "response") || strings.Contains(string(bodyBytes), "content") {
			// Try to extract the value manually
			for _, line := range strings.Split(string(bodyBytes), ",") {
				if strings.Contains(line, "response") || strings.Contains(line, "content") {
					parts := strings.SplitN(line, ":", 2)
					if len(parts) > 1 {
						commitMsg = strings.TrimSpace(parts[1])
						// Remove quotes
						commitMsg = strings.Trim(commitMsg, "\"' ")
						break
					}
				}
			}
		}

		// If still empty, use the entire response as a fallback
		if commitMsg == "" {
			commitMsg = strings.TrimSpace(string(bodyBytes))
		}
	}

	// Remove quotes if they're wrapping the message
	if (strings.HasPrefix(commitMsg, "\"") && strings.HasSuffix(commitMsg, "\"")) ||
		(strings.HasPrefix(commitMsg, "'") && strings.HasSuffix(commitMsg, "'")) {
		commitMsg = commitMsg[1 : len(commitMsg)-1]
	}

	return commitMsg, nil
}
