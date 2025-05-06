package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type Config struct {
	OllamaAPIURL   string `json:"ollamaApiUrl"`
	DefaultModel   string `json:"defaultModel"`
	PromptTemplate string `json:"promptTemplate"`
}

func loadConfig() Config {
	// Default configuration
	defaultConfig := Config{
		OllamaAPIURL: "http://localhost:11434/api/generate",
		DefaultModel: "gemma3:1b",
		PromptTemplate: `Generate a concise and descriptive git commit message based on the following changes.
Follow best practices for git commit messages: use imperative mood, keep it under 50 characters for the first line,
and add more details in a body if necessary. 

Respond ONLY with the commit message, no other text, explanation, or quotes. 
Just the commit message that would be used with 'git commit -m'.

Changes:
%s`,
	}

	// Look for config file in current directory
	configFile := "ollama-commit.json"
	data, err := os.ReadFile(configFile)

	// If no config in current directory, check home directory
	if err != nil {
		homeDir, homeDirErr := os.UserHomeDir()
		if homeDirErr == nil {
			homeConfig := filepath.Join(homeDir, ".ollama-commit.json")
			data, err = os.ReadFile(homeConfig)
		}
	}

	// If config found, unmarshal it
	if err == nil {
		var config Config
		if err := json.Unmarshal(data, &config); err == nil {
			// Merge with defaults (only set values that are not empty)
			if config.OllamaAPIURL != "" {
				defaultConfig.OllamaAPIURL = config.OllamaAPIURL
			}
			if config.DefaultModel != "" {
				defaultConfig.DefaultModel = config.DefaultModel
			}
			if config.PromptTemplate != "" {
				defaultConfig.PromptTemplate = config.PromptTemplate
			}
		}
	}

	return defaultConfig
}

type OllamaRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

// The Ollama API might return the response in different formats
// We'll handle multiple possible response structures
type OllamaResponse struct {
	Response string `json:"response"`
	Content  string `json:"content"` // Some versions use content instead of response
}

func main() {
	// Load configuration
	config := loadConfig()

	// Define flags with defaults from config
	autoCommit := flag.Bool("a", false, "Automatically commit using the generated message")
	model := flag.String("model", config.DefaultModel, "Ollama model to use")
	noConfirm := flag.Bool("y", false, "Skip confirmation prompt")
	saveConfig := flag.Bool("save-config", false, "Save current settings to config file")
	ollamaURL := flag.String("url", config.OllamaAPIURL, "Ollama API URL")
	flag.Parse()

	// Save configuration if requested
	if *saveConfig {
		config.DefaultModel = *model
		config.OllamaAPIURL = *ollamaURL

		// Convert config to JSON
		configJSON, err := json.MarshalIndent(config, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating config JSON: %v\n", err)
			os.Exit(1)
		}

		// Write to home directory
		homeDir, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting home directory: %v\n", err)
			os.Exit(1)
		}

		configPath := filepath.Join(homeDir, ".ollama-commit.json")
		if err := os.WriteFile(configPath, configJSON, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing config file: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Configuration saved to %s\n", configPath)
		os.Exit(0)
	}

	// Get git diff
	gitDiff, err := getGitDiff()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting git diff: %v\n", err)
		os.Exit(1)
	}

	if gitDiff == "" {
		fmt.Println("No changes to commit")
		os.Exit(0)
	}

	// Generate commit message using Ollama
	commitMsg, err := generateCommitMessage(gitDiff, *model, *ollamaURL, config.PromptTemplate)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating commit message: %v\n", err)
		os.Exit(1)
	}

	// Print the generated commit message
	fmt.Println("Generated commit message:")
	fmt.Println("------------------------")
	fmt.Println(commitMsg)
	fmt.Println("------------------------")

	// If auto-commit flag is set
	if *autoCommit {
		// Skip confirmation if -y flag is provided
		if !*noConfirm {
			confirmed := confirmCommit(commitMsg)
			if !confirmed {
				fmt.Println("Commit aborted.")
				os.Exit(0)
			}
		}

		err = executeGitCommit(commitMsg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error executing git commit: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Changes committed successfully!")
	} else {
		fmt.Println("Use -a flag to automatically commit with this message")
	}
}

func getGitDiff() (string, error) {
	// Check if in a git repository
	cmdStatus := exec.Command("git", "status")
	if err := cmdStatus.Run(); err != nil {
		return "", fmt.Errorf("not in a git repository or git is not installed")
	}

	// Get staged changes
	cmdDiff := exec.Command("git", "diff", "--staged")
	diffOutput, err := cmdDiff.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get git diff: %v", err)
	}

	// If no staged changes, try to get unstaged changes
	if len(diffOutput) == 0 {
		cmdDiff = exec.Command("git", "diff")
		diffOutput, err = cmdDiff.Output()
		if err != nil {
			return "", fmt.Errorf("failed to get git diff: %v", err)
		}
	}

	return string(diffOutput), nil
}

func generateCommitMessage(gitDiff, model, apiURL, promptTemplate string) (string, error) {
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

func confirmCommit(message string) bool {
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Are you sure you want to use this commit message? (y/n): ")
	input, err := reader.ReadString('\n')
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
		return false
	}

	input = strings.TrimSpace(strings.ToLower(input))
	return input == "y" || input == "yes"
}

func executeGitCommit(message string) error {
	cmd := exec.Command("git", "commit", "-m", message)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
