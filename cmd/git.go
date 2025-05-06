package cmd

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// GetGitDiff retrieves git diff from the repository
func GetGitDiff() (string, error) {
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

// ConfirmCommit asks the user to confirm the commit message
func ConfirmCommit(message string) bool {
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

// ExecuteGitCommit performs the git commit with the given message
func ExecuteGitCommit(message string) error {
	cmd := exec.Command("git", "commit", "-m", message)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
