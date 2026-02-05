package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

// RunHooks executes hooks for a given action (e.g., "add", "rm")
func RunHooks(action string, targetPath string, branch string) {
	key := "hooks." + action
	val := viper.Get(key)
	if val == nil {
		return
	}

	var commands []string
	switch v := val.(type) {
	case string:
		commands = []string{v}
	case []any:
		for _, item := range v {
			if s, ok := item.(string); ok {
				commands = append(commands, s)
			}
		}
	case []string:
		commands = v
	}

	if len(commands) == 0 {
		return
	}

	absPath, err := filepath.Abs(targetPath)
	if err != nil {
		fmt.Printf("Warning: failed to get absolute path for hooks: %v\n", err)
		absPath = targetPath
	}

	for _, cmdStr := range commands {
		executeCommand(cmdStr, absPath, branch)
	}
}

func executeCommand(cmdStr string, absPath string, branch string) {
	fmt.Printf("--- Executing hook: %s ---\n", cmdStr)

	// Replace placeholders
	replacedCmd := strings.ReplaceAll(cmdStr, "{{.Path}}", absPath)
	replacedCmd = strings.ReplaceAll(replacedCmd, "{{.Branch}}", branch)

	// Execute via shell to support pipes, redirects, etc.
	// We use /bin/sh -c because users expect shell behavior
	cmd := exec.Command("/bin/sh", "-c", replacedCmd)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fmt.Printf("Warning: hook failed: %v\n", err)
	}
}
