package hook

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/mocyuto/zgt/internal/config"
	"github.com/mocyuto/zgt/internal/state"
	"github.com/mocyuto/zgt/internal/template"
)

// RunHooks executes hooks for a given action (e.g., "add", "rm").
// The profile (stored in ctx.Profile) selects profile-specific hook overrides
// from the config. Top-level hooks are always run first; profile-specific
// hooks are appended afterwards.
func RunHooks(action string, ctx template.Context) {
	commands := config.ProfileHooks(action, ctx.Profile)

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

func executeCommand(cmdStr string, absPath string, ctx template.Context) {
	fmt.Printf("--- Executing hook: %s ---\n", cmdStr)

	// Replace placeholders
	ctx.Path = absPath
	replacedCmd := template.Replace(cmdStr, ctx)

	// Execute via shell to support pipes, redirects, etc.
	// We use /bin/sh -c because users expect shell behavior
	cmd := exec.Command("/bin/sh", "-c", replacedCmd)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Merge system env with config env (profile-aware).
	env := os.Environ()
	profileEnv := config.ProfileEnv(ctx.Profile)
	if len(profileEnv) > 0 {
		replacedEnv := template.ReplaceMap(profileEnv, ctx)
		for k, v := range replacedEnv {
			env = append(env, fmt.Sprintf("%s=%s", k, v))
		}
	}

	// Inject per-worktree port vars (NAME_PORT=basePort+idx) so compose-style
	// hooks that interpolate ${API_PORT} / ${ENVOY_PORT} pick up the worktree's
	// assigned port without needing a prior `eval "$(zgt env)"`.
	if len(config.AppConfig.Ports) > 0 {
		if ports, ok := state.GetWorktreePortsByPath(absPath); ok {
			for name, basePort := range config.AppConfig.Ports {
				idx, ok := ports[name]
				if !ok {
					continue
				}
				envName := strings.ToUpper(name) + "_PORT"
				env = append(env, fmt.Sprintf("%s=%d", envName, basePort+idx))
			}
		}
	}

	cmd.Env = env

	if err := cmd.Run(); err != nil {
		fmt.Printf("Warning: hook failed: %v\n", err)
	}
}
