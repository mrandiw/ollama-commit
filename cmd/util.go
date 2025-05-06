package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Config holds the application configuration
type Config struct {
	OllamaAPIURL   string `json:"ollamaApiUrl"`
	DefaultModel   string `json:"defaultModel"`
	PromptTemplate string `json:"promptTemplate"`
}

// LoadConfig loads configuration from file or returns defaults
func LoadConfig() Config {
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
