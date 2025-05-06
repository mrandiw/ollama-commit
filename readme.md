# Ollama Commit

A simple CLI tool that uses Ollama's API to automatically generate git commit messages based on your changes.

## Requirements

- Go 1.18+ installed
- Git installed
- Ollama running locally (default: http://localhost:11434)

## Installation

### From Source

1. Clone the repository:
   ```bash
   git clone https://github.com/mrandiw/ollama-commit.git
   cd ollama-commit
   ```

2. Build the executable:
   ```bash
   go build -o ollama-commit .
   ```

3. Move the executable to your PATH:
   ```bash
   sudo mv ollama-commit /usr/local/bin/
   # OR for a user-local installation
   mkdir -p ~/bin
   mv ollama-commit ~/bin/
   # Make sure ~/bin is in your PATH
   ```

### Using Go Install

You can also install directly using Go:

```bash
go install github.com/mrandiw/ollama-commit@latest
```

## Usage

Make sure Ollama is running and has your preferred model loaded.

Basic usage:
```bash
# Show the generated commit message without committing
ollama-commit
```

Automatically commit with the generated message:
```bash
ollama-commit -a
```

Specify a different Ollama model:
```bash
ollama-commit -model codellama
```

## Configuration

You can configure ollama-commit using a configuration file. The tool looks for configuration in the following locations:

1. `./ollama-commit.json` (current directory)
2. `~/.ollama-commit.json` (home directory)

You can create a configuration file manually or use the `-save-config` flag to save your current settings:

```bash
# Save your current settings to ~/.ollama-commit.json
ollama-commit -model codellama -url http://localhost:11434/api/generate -save-config
```

### Configuration File Format

The configuration file is in JSON format:

```json
{
  "ollamaApiUrl": "http://localhost:11434/api/generate",
  "defaultModel": "llama3",
  "promptTemplate": "Generate a concise and descriptive git commit message based on the following changes.\nFollow best practices for git commit messages: use imperative mood, keep it under 50 characters for the first line,\nand add more details in a body if necessary.\n\nRespond ONLY with the commit message, no other text, explanation, or quotes.\nJust the commit message that would be used with 'git commit -m'.\n\nChanges:\n%s"
}
```

Command-line flags will override the configuration file settings.

Available flags:
- `-a`: Automatically commit using the generated message
- `-model string`: Ollama model to use (default from config or "llama3")
- `-y`: Skip confirmation prompt (used with -a)
- `-url string`: Ollama API URL (default from config or "http://localhost:11434/api/generate")
- `-save-config`: Save current settings as your default configuration

## Example

```bash
# Show the generated commit message without committing
$ ollama-commit
Generated commit message:
------------------------
feat: add user authentication and password reset functionality
------------------------
Use -a flag to automatically commit with this message

# Commit with confirmation
$ ollama-commit -a
Generated commit message:
------------------------
feat: add user authentication and password reset functionality
------------------------
Are you sure you want to use this commit message? (y/n): y
Changes committed successfully!

# Commit without confirmation
$ ollama-commit -a -y
Generated commit message:
------------------------
feat: add user authentication and password reset functionality
------------------------
Changes committed successfully!
```

## License

MIT