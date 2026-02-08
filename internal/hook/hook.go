package hook

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/mocyuto/git-wt/internal/config"
)

// HookContext holds information for hook placeholders
type HookContext struct {
	Path   string
	Branch string
	Repo   string
}

// RunHooks executes hooks for a given action (e.g., "add", "rm")
func RunHooks(action string, ctx HookContext) {
	var commands []string
	switch action {
	case "add":
		commands = config.AppConfig.Hooks.Add
	case "rm":
		commands = config.AppConfig.Hooks.RM
	}

	if len(commands) == 0 {
		return
	}

	absPath, err := filepath.Abs(ctx.Path)
	if err != nil {
		fmt.Printf("Warning: failed to get absolute path for hooks: %v\n", err)
		absPath = ctx.Path
	}

	for _, cmdStr := range commands {
		executeCommand(cmdStr, absPath, ctx)
	}
}

func executeCommand(cmdStr string, absPath string, ctx HookContext) {
	fmt.Printf("--- Executing hook: %s ---\n", cmdStr)

	// Replace placeholders
	replacedCmd := strings.ReplaceAll(cmdStr, "{{.Path}}", absPath)
	replacedCmd = strings.ReplaceAll(replacedCmd, "{{.Branch}}", ctx.Branch)
	replacedCmd = strings.ReplaceAll(replacedCmd, "{{.Repo}}", ctx.Repo)

	// Execute via shell to support pipes, redirects, etc.
	// We use /bin/sh -c because users expect shell behavior
	cmd := exec.Command("/bin/sh", "-c", replacedCmd)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fmt.Printf("Warning: hook failed: %v\n", err)
	}
}
