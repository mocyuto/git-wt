package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mocyuto/zgt/internal/agentstatus"
	"github.com/mocyuto/zgt/internal/logger"
	"github.com/spf13/cobra"
)

// ZgtClaudeHookCommand is the command string zgt installs into Claude Code
// hook entries. It doubles as the marker used to identify zgt-managed
// entries on uninstall / reinstall.
const ZgtClaudeHookCommand = "zgt agent hook claude"

// claude hook events zgt manages. Notification uses a matcher so only
// permission/idle prompts flip the status to waiting.
var claudeHookEvents = []struct {
	event   string
	matcher string
}{
	{event: "UserPromptSubmit"},
	{event: "Stop"},
	{event: "Notification", matcher: "permission_prompt|idle_prompt"},
	{event: "SessionEnd"},
}

var agentInstallCmd = &cobra.Command{
	Use:   "install [claude|opencode|all]",
	Short: "Install hooks/plugins so opencode / Claude Code report status to zgt",
	Long: `Install the hooks (Claude Code) and plugin (opencode) that report
session status to zgt. Run once per machine. Re-running is safe and
updates to the latest zgt-managed configuration.

Without an argument, installs for both claude and opencode. Hooks are
written to ~/.claude/settings.json and the plugin to
~/.config/opencode/plugins/.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		target := "all"
		if len(args) == 1 {
			target = args[0]
		}
		return runAgentInstall(target)
	},
}

var agentUninstallCmd = &cobra.Command{
	Use:   "uninstall [claude|opencode|all]",
	Short: "Remove zgt agent status hooks/plugins",
	Long: `Remove the hooks (Claude Code) and plugin (opencode) installed by
'zgt agent install'. Without an argument, removes both.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		target := "all"
		if len(args) == 1 {
			target = args[0]
		}
		return runAgentUninstall(target)
	},
}

func init() {
	agentCmd.AddCommand(agentInstallCmd)
	agentCmd.AddCommand(agentUninstallCmd)
}

func runAgentInstall(target string) error {
	switch target {
	case "all":
		if err := installClaudeHooks(); err != nil {
			return err
		}
		if err := installOpenCodePlugin(); err != nil {
			return err
		}
		logger.Success("Installed agent status hooks for claude and opencode.")
	case "claude":
		if err := installClaudeHooks(); err != nil {
			return err
		}
		logger.Success("Installed agent status hooks for claude.")
	case "opencode":
		if err := installOpenCodePlugin(); err != nil {
			return err
		}
		logger.Success("Installed opencode status plugin.")
	default:
		return fmt.Errorf("unknown target %q; expected claude, opencode, or all", target)
	}
	return nil
}

func runAgentUninstall(target string) error {
	switch target {
	case "all":
		if err := uninstallClaudeHooks(); err != nil {
			return err
		}
		if err := uninstallOpenCodePlugin(); err != nil {
			return err
		}
		logger.Success("Uninstalled agent status hooks for claude and opencode.")
	case "claude":
		if err := uninstallClaudeHooks(); err != nil {
			return err
		}
		logger.Success("Uninstalled agent status hooks for claude.")
	case "opencode":
		if err := uninstallOpenCodePlugin(); err != nil {
			return err
		}
		logger.Success("Uninstalled opencode status plugin.")
	default:
		return fmt.Errorf("unknown target %q; expected claude, opencode, or all", target)
	}
	return nil
}

// ---- Claude Code ----

func claudeSettingsPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".claude", "settings.json"), nil
}

// installClaudeHooks merges zgt-managed hook entries into ~/.claude/settings.json.
// Existing user hooks are preserved; any prior zgt entries are replaced.
func installClaudeHooks() error {
	path, err := claudeSettingsPath()
	if err != nil {
		return err
	}
	settings, err := loadJSONMap(path)
	if err != nil {
		return err
	}

	hooks := ensureMap(settings, "hooks")
	// Remove prior zgt entries so a re-install is clean and idempotent.
	removeZgtClaudeHooks(hooks)
	for _, e := range claudeHookEvents {
		groups := ensureSlice(hooks, e.event)
		group := map[string]any{
			"hooks": []any{map[string]any{
				"type":    "command",
				"command": ZgtClaudeHookCommand,
			}},
		}
		if e.matcher != "" {
			group["matcher"] = e.matcher
		}
		hooks[e.event] = append(groups, group)
	}

	return writeJSONMap(path, settings)
}

func uninstallClaudeHooks() error {
	path, err := claudeSettingsPath()
	if err != nil {
		return err
	}
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			logger.Info("No Claude Code settings file at %s; nothing to uninstall.", path)
			return nil
		}
		return err
	}
	settings, err := loadJSONMap(path)
	if err != nil {
		return err
	}
	if hooks, ok := settings["hooks"].(map[string]any); ok {
		removeZgtClaudeHooks(hooks)
		if len(hooks) == 0 {
			delete(settings, "hooks")
		}
	}
	return writeJSONMap(path, settings)
}

// removeZgtClaudeHooks strips every zgt-managed handler (and now-empty
// matcher groups) from all hook events in place.
func removeZgtClaudeHooks(hooks map[string]any) {
	for event, raw := range hooks {
		groups, ok := raw.([]any)
		if !ok {
			continue
		}
		kept := make([]any, 0, len(groups))
		for _, g := range groups {
			group, ok := g.(map[string]any)
			if !ok {
				kept = append(kept, g)
				continue
			}
			handlers, _ := group["hooks"].([]any)
			filtered := make([]any, 0, len(handlers))
			for _, h := range handlers {
				handler, ok := h.(map[string]any)
				if !ok {
					filtered = append(filtered, h)
					continue
				}
				if cmd, _ := handler["command"].(string); isZgtClaudeHook(cmd) {
					continue
				}
				filtered = append(filtered, h)
			}
			if len(filtered) == 0 {
				continue // drop empty group
			}
			group["hooks"] = filtered
			kept = append(kept, group)
		}
		if len(kept) == 0 {
			delete(hooks, event)
		} else {
			hooks[event] = kept
		}
	}
}

// isZgtClaudeHook returns true for command strings that invoke the zgt
// claude hook (exact match or with leading args).
func isZgtClaudeHook(cmd string) bool {
	return cmd == ZgtClaudeHookCommand || strings.HasPrefix(cmd, ZgtClaudeHookCommand+" ")
}

// ---- OpenCode ----

func openCodePluginPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "opencode", "plugins", agentstatus.OpenCodePluginName), nil
}

func installOpenCodePlugin() error {
	path, err := openCodePluginPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("create opencode plugins dir: %v", err)
	}
	src, err := agentstatus.OpenCodePluginSource()
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, src, 0644); err != nil {
		return fmt.Errorf("write opencode plugin: %v", err)
	}
	return nil
}

func uninstallOpenCodePlugin() error {
	path, err := openCodePluginPath()
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// ---- JSON helpers (preserve unknown keys / ordering as much as possible) ----

func loadJSONMap(path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]any{}, nil
		}
		return nil, err
	}
	// Tolerate trailing whitespace / empty files.
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" {
		return map[string]any{}, nil
	}
	out := map[string]any{}
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, fmt.Errorf("parse %s: %v", path, err)
	}
	return out, nil
}

func writeJSONMap(path string, m map[string]any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	// Atomic write (temp + rename) so a crash mid-write cannot corrupt a
	// critical user config file like ~/.claude/settings.json.
	tmp, err := os.CreateTemp(filepath.Dir(path), ".zgt-settings-*.json")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }() // no-op if rename succeeded
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}

func ensureMap(m map[string]any, key string) map[string]any {
	if v, ok := m[key].(map[string]any); ok {
		return v
	}
	v := map[string]any{}
	m[key] = v
	return v
}

func ensureSlice(m map[string]any, key string) []any {
	if v, ok := m[key].([]any); ok {
		return v
	}
	return []any{}
}
