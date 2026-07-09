package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mocyuto/zgt/internal/agentstatus"
	"github.com/stretchr/testify/assert"
)

func withAgentTempHome(t *testing.T) string {
	t.Helper()
	tmp, err := os.MkdirTemp("", "zgt-agent-cmd-home-*")
	if err != nil {
		t.Fatalf("MkdirTemp: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(tmp) })
	orig := os.Getenv("HOME")
	t.Setenv("HOME", tmp)
	_ = orig
	return tmp
}

func TestRunAgentHookClaudeUserPromptSubmit(t *testing.T) {
	withAgentTempHome(t)
	dir, _ := os.MkdirTemp("", "zgt-wt-*")

	in := claudeHookInput{HookEventName: "UserPromptSubmit", Cwd: dir, SessionID: "s1"}
	data, _ := json.Marshal(in)
	if err := runAgentHook("claude", bytes.NewReader(data)); err != nil {
		t.Fatalf("runAgentHook: %v", err)
	}
	rec, ok := agentstatus.GetStatus(dir)
	if !ok {
		t.Fatal("expected record")
	}
	assert.Equal(t, agentstatus.AgentClaude, rec.Agent)
	assert.Equal(t, agentstatus.StatusWorking, rec.Status)
	assert.Equal(t, "s1", rec.SessionID)
}

func TestRunAgentHookClaudeStop(t *testing.T) {
	withAgentTempHome(t)
	dir, _ := os.MkdirTemp("", "zgt-wt-*")

	in := claudeHookInput{HookEventName: "Stop", Cwd: dir}
	data, _ := json.Marshal(in)
	if err := runAgentHook("claude", bytes.NewReader(data)); err != nil {
		t.Fatal(err)
	}
	rec, _ := agentstatus.GetStatus(dir)
	assert.Equal(t, agentstatus.StatusIdle, rec.Status)
}

func TestRunAgentHookClaudeNotification(t *testing.T) {
	withAgentTempHome(t)
	dir, _ := os.MkdirTemp("", "zgt-wt-*")

	in := claudeHookInput{HookEventName: "Notification", Cwd: dir}
	data, _ := json.Marshal(in)
	if err := runAgentHook("claude", bytes.NewReader(data)); err != nil {
		t.Fatal(err)
	}
	rec, _ := agentstatus.GetStatus(dir)
	assert.Equal(t, agentstatus.StatusAsk, rec.Status)
}

func TestRunAgentHookClaudeSessionEndClears(t *testing.T) {
	withAgentTempHome(t)
	dir, _ := os.MkdirTemp("", "zgt-wt-*")

	if err := agentstatus.SetStatus(dir, agentstatus.AgentClaude, agentstatus.StatusWorking, ""); err != nil {
		t.Fatal(err)
	}
	in := claudeHookInput{HookEventName: "SessionEnd", Cwd: dir}
	data, _ := json.Marshal(in)
	if err := runAgentHook("claude", bytes.NewReader(data)); err != nil {
		t.Fatal(err)
	}
	if _, ok := agentstatus.GetStatus(dir); ok {
		t.Fatal("expected record cleared after SessionEnd")
	}
}

func TestRunAgentHookUnknownEventIsNoop(t *testing.T) {
	withAgentTempHome(t)
	dir, _ := os.MkdirTemp("", "zgt-wt-*")
	in := claudeHookInput{HookEventName: "UnknownEvent", Cwd: dir}
	data, _ := json.Marshal(in)
	if err := runAgentHook("claude", bytes.NewReader(data)); err != nil {
		t.Fatalf("unknown event should be no-op, got %v", err)
	}
	if _, ok := agentstatus.GetStatus(dir); ok {
		t.Fatal("expected no record for unknown event")
	}
}

func TestRunAgentHookFallsBackToCwd(t *testing.T) {
	withAgentTempHome(t)
	origWd, _ := os.Getwd()
	defer os.Chdir(origWd)
	dir, _ := os.MkdirTemp("", "zgt-wt-*")
	os.Chdir(dir)

	// No cwd in input -> should use os.Getwd().
	in := claudeHookInput{HookEventName: "Stop"}
	data, _ := json.Marshal(in)
	if err := runAgentHook("claude", bytes.NewReader(data)); err != nil {
		t.Fatal(err)
	}
	if _, ok := agentstatus.GetStatus(dir); !ok {
		t.Fatal("expected record under current dir fallback")
	}
}

func TestAgentSetStatusCommand(t *testing.T) {
	withAgentTempHome(t)
	dir, _ := os.MkdirTemp("", "zgt-wt-*")

	t.Cleanup(func() {
		setStatusAgent = ""
		setStatusStatus = ""
		setStatusCwd = ""
		setStatusSID = ""
	})
	setStatusAgent = "opencode"
	setStatusStatus = "working"
	setStatusCwd = dir
	setStatusSID = "oc-1"
	if err := agentSetStatusCmd.RunE(agentSetStatusCmd, nil); err != nil {
		t.Fatalf("set-status: %v", err)
	}
	rec, ok := agentstatus.GetStatus(dir)
	if !ok {
		t.Fatal("expected record")
	}
	assert.Equal(t, agentstatus.AgentOpenCode, rec.Agent)
	assert.Equal(t, agentstatus.StatusWorking, rec.Status)
	assert.Equal(t, "oc-1", rec.SessionID)
}

func TestAgentSetStatusRejectsInvalidAgent(t *testing.T) {
	withAgentTempHome(t)
	t.Cleanup(func() {
		setStatusAgent = ""
		setStatusStatus = ""
		setStatusCwd = ""
	})
	setStatusAgent = "foobar"
	setStatusStatus = "working"
	setStatusCwd = "/tmp/whatever"
	err := agentSetStatusCmd.RunE(agentSetStatusCmd, nil)
	assert.Error(t, err)
}

func TestAgentSetStatusRejectsInvalidStatus(t *testing.T) {
	withAgentTempHome(t)
	t.Cleanup(func() {
		setStatusAgent = ""
		setStatusStatus = ""
		setStatusCwd = ""
	})
	setStatusAgent = "opencode"
	setStatusStatus = "bogus"
	setStatusCwd = "/tmp/whatever"
	err := agentSetStatusCmd.RunE(agentSetStatusCmd, nil)
	assert.Error(t, err)
}

func TestAgentClearStatusCommand(t *testing.T) {
	withAgentTempHome(t)
	dir, _ := os.MkdirTemp("", "zgt-wt-*")
	if err := agentstatus.SetStatus(dir, agentstatus.AgentOpenCode, agentstatus.StatusIdle, ""); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { clearStatusCwd = "" })
	clearStatusCwd = dir
	if err := agentClearStatusCmd.RunE(agentClearStatusCmd, nil); err != nil {
		t.Fatalf("clear-status: %v", err)
	}
	if _, ok := agentstatus.GetStatus(dir); ok {
		t.Fatal("expected record cleared")
	}
}

func TestWriteAgentStatusTableAlignsColumns(t *testing.T) {
	rows := []agentStatusRow{
		{Path: "/long/path/here", Branch: "main", Agent: "opencode", Status: "working", Age: "5s"},
		{Path: "/short", Branch: "feature-branch", Agent: "claude", Status: "ask", Age: "1m"},
		{Path: "/medium/path", Branch: "dev", Agent: "-", Status: "idle?", Age: "30s"},
	}
	var buf bytes.Buffer
	if err := writeAgentStatusTable(&buf, rows); err != nil {
		t.Fatalf("writeAgentStatusTable: %v", err)
	}
	out := buf.String()
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	assert.Len(t, lines, len(rows)+1) // header + data rows

	// Every line should have the same number of double-space column
	// separators, and the AGE column should start at the same visual
	// offset (i.e. the byte index after stripping ANSI codes is equal).
	stripANSI := func(s string) string {
		return strings.NewReplacer("\033[32m", "", "\033[36m", "", "\033[90m", "", "\033[0m", "").Replace(s)
	}
	ageOffset := -1
	for _, line := range lines {
		visible := stripANSI(line)
		idx := strings.LastIndex(visible, "  ")
		if idx < 0 {
			t.Fatalf("expected column separator in line: %q", visible)
		}
		off := idx + 2
		if ageOffset < 0 {
			ageOffset = off
		} else if off != ageOffset {
			t.Errorf("AGE column misaligned: got offset %d, want %d in line: %q", off, ageOffset, visible)
		}
	}
}

func TestWriteAgentStatusTableColorizesKnownStatuses(t *testing.T) {
	rows := []agentStatusRow{
		{Path: "/p", Branch: "main", Agent: "opencode", Status: "working", Age: "1s"},
		{Path: "/p2", Branch: "dev", Agent: "opencode", Status: "ask", Age: "2s"},
		{Path: "/p3", Branch: "feat", Agent: "opencode", Status: "idle", Age: "3s"},
	}
	var buf bytes.Buffer
	_ = writeAgentStatusTable(&buf, rows)
	out := buf.String()
	assert.Contains(t, out, "\033[32m") // green for working
	assert.Contains(t, out, "\033[36m") // cyan for ask
	assert.Contains(t, out, "\033[90m") // gray for idle
}

func TestWriteAgentStatusTableLeavesUnknownStatusPlain(t *testing.T) {
	rows := []agentStatusRow{
		{Path: "/p", Branch: "main", Agent: "-", Status: "idle?", Age: "1s"},
	}
	var buf bytes.Buffer
	_ = writeAgentStatusTable(&buf, rows)
	out := buf.String()
	// "idle?" should appear without any color escape codes wrapping it.
	assert.Contains(t, out, "idle?")
	assert.NotContains(t, out, "\033[32midle?")
	assert.NotContains(t, out, "\033[36midle?")
	assert.NotContains(t, out, "\033[90midle?")
}

func TestIsZgtClaudeHook(t *testing.T) {
	assert.True(t, isZgtClaudeHook("zgt agent hook claude"))
	assert.True(t, isZgtClaudeHook("zgt agent hook claude --foo"))
	assert.False(t, isZgtClaudeHook("echo hello"))
	assert.False(t, isZgtClaudeHook(""))
	assert.False(t, isZgtClaudeHook("zgt agent hook opencode"))
}

func TestAgentHookCmdRejectsInvalidAgent(t *testing.T) {
	withAgentTempHome(t)
	// cobra.OnlyValidArgs rejects anything not in ValidArgs. Validate the
	// Args validator directly (do NOT call RunE, which would read os.Stdin).
	err := agentHookCmd.Args(agentHookCmd, []string{"foobar"})
	assert.Error(t, err)
}

func TestAgentHookCmdAcceptsClaude(t *testing.T) {
	if err := agentHookCmd.Args(agentHookCmd, []string{"claude"}); err != nil {
		t.Fatalf("expected claude to be accepted, got %v", err)
	}
}

func TestInstallClaudeHooksCreatesAndMerges(t *testing.T) {
	withAgentTempHome(t)
	if err := installClaudeHooks(); err != nil {
		t.Fatalf("install: %v", err)
	}
	path, _ := claudeSettingsPath()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("settings.json not written: %v", err)
	}
	var settings map[string]any
	if err := json.Unmarshal(data, &settings); err != nil {
		t.Fatalf("invalid settings.json: %v", err)
	}
	hooks, ok := settings["hooks"].(map[string]any)
	if !ok {
		t.Fatal("expected hooks map")
	}
	for _, e := range claudeHookEvents {
		groups, ok := hooks[e.event].([]any)
		if !ok || len(groups) == 0 {
			t.Errorf("event %s missing zgt group", e.event)
			continue
		}
		found := false
		for _, g := range groups {
			gm, _ := g.(map[string]any)
			handlers, _ := gm["hooks"].([]any)
			for _, h := range handlers {
				hm, _ := h.(map[string]any)
				if hm["command"] == ZgtClaudeHookCommand {
					found = true
					if e.matcher != "" {
						if gm["matcher"] != e.matcher {
							t.Errorf("event %s matcher = %v, want %q", e.event, gm["matcher"], e.matcher)
						}
					}
				}
			}
		}
		if !found {
			t.Errorf("event %s has no zgt handler", e.event)
		}
	}
}

func TestInstallClaudeHooksPreservesUserHooks(t *testing.T) {
	withAgentTempHome(t)
	path, _ := claudeSettingsPath()
	_ = os.MkdirAll(filepath.Dir(path), 0755)
	existing := map[string]any{
		"hooks": map[string]any{
			"Stop": []any{
				map[string]any{
					"hooks": []any{
						map[string]any{"type": "command", "command": "echo user-hook"},
					},
				},
			},
		},
	}
	data, _ := json.Marshal(existing)
	_ = os.WriteFile(path, data, 0644)

	if err := installClaudeHooks(); err != nil {
		t.Fatalf("install: %v", err)
	}
	settings, _ := loadJSONMap(path)
	hooks := settings["hooks"].(map[string]any)
	groups := hooks["Stop"].([]any)

	var commands []string
	for _, g := range groups {
		gm := g.(map[string]any)
		for _, h := range gm["hooks"].([]any) {
			hm := h.(map[string]any)
			commands = append(commands, hm["command"].(string))
		}
	}
	assert.Contains(t, commands, "echo user-hook")
	assert.Contains(t, commands, ZgtClaudeHookCommand)
}

func TestInstallClaudeHooksIsIdempotent(t *testing.T) {
	withAgentTempHome(t)
	if err := installClaudeHooks(); err != nil {
		t.Fatal(err)
	}
	if err := installClaudeHooks(); err != nil {
		t.Fatal(err)
	}
	settings, _ := loadJSONMap(claudeSettingsPathMust(t))
	for _, e := range claudeHookEvents {
		groups := settings["hooks"].(map[string]any)[e.event].([]any)
		count := 0
		for _, g := range groups {
			gm := g.(map[string]any)
			for _, h := range gm["hooks"].([]any) {
				if hm, _ := h.(map[string]any); hm["command"] == ZgtClaudeHookCommand {
					count++
				}
			}
		}
		if count != 1 {
			t.Errorf("event %s has %d zgt handlers, want 1", e.event, count)
		}
	}
}

func TestUninstallClaudeHooksRemovesZgtOnly(t *testing.T) {
	withAgentTempHome(t)
	path, _ := claudeSettingsPath()
	_ = os.MkdirAll(filepath.Dir(path), 0755)
	existing := map[string]any{
		"hooks": map[string]any{
			"Stop": []any{
				map[string]any{"hooks": []any{
					map[string]any{"type": "command", "command": "echo user"},
					map[string]any{"type": "command", "command": ZgtClaudeHookCommand},
				}},
			},
		},
	}
	data, _ := json.Marshal(existing)
	_ = os.WriteFile(path, data, 0644)

	if err := uninstallClaudeHooks(); err != nil {
		t.Fatalf("uninstall: %v", err)
	}
	settings, _ := loadJSONMap(path)
	hooks, ok := settings["hooks"].(map[string]any)
	if !ok {
		return
	}
	groups := hooks["Stop"].([]any)
	for _, g := range groups {
		gm := g.(map[string]any)
		for _, h := range gm["hooks"].([]any) {
			hm := h.(map[string]any)
			if hm["command"] == ZgtClaudeHookCommand {
				t.Error("zgt hook should have been removed")
			}
		}
	}
	// User hook survives.
	commands := collectCommands(settings)
	assert.Contains(t, commands, "echo user")
}

func TestUninstallClaudeHooksMissingFileIsNoop(t *testing.T) {
	withAgentTempHome(t)
	if err := uninstallClaudeHooks(); err != nil {
		t.Fatalf("uninstall on missing file should be nil, got %v", err)
	}
}

func TestInstallOpenCodePluginWritesFile(t *testing.T) {
	withAgentTempHome(t)
	if err := installOpenCodePlugin(); err != nil {
		t.Fatalf("install: %v", err)
	}
	path, _ := openCodePluginPath()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("plugin not written: %v", err)
	}
	assert.Contains(t, string(data), "ZgtStatusPlugin")
	assert.Contains(t, string(data), "zgt agent set-status")
}

func TestInstallOpenCodePluginIdempotent(t *testing.T) {
	withAgentTempHome(t)
	if err := installOpenCodePlugin(); err != nil {
		t.Fatal(err)
	}
	if err := installOpenCodePlugin(); err != nil {
		t.Fatal(err)
	}
	path, _ := openCodePluginPath()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	assert.True(t, info.Size() > 0)
}

func TestUninstallOpenCodePlugin(t *testing.T) {
	withAgentTempHome(t)
	if err := installOpenCodePlugin(); err != nil {
		t.Fatal(err)
	}
	if err := uninstallOpenCodePlugin(); err != nil {
		t.Fatalf("uninstall: %v", err)
	}
	if _, err := os.Stat(openCodePluginPathMust(t)); !os.IsNotExist(err) {
		t.Fatalf("expected plugin removed, got %v", err)
	}
}

func TestUninstallOpenCodePluginMissingIsNoop(t *testing.T) {
	withAgentTempHome(t)
	if err := uninstallOpenCodePlugin(); err != nil {
		t.Fatalf("uninstall missing plugin should be nil, got %v", err)
	}
}

func TestRunAgentInstallAll(t *testing.T) {
	withAgentTempHome(t)
	if err := runAgentInstall("all"); err != nil {
		t.Fatalf("install all: %v", err)
	}
	if _, err := os.Stat(claudeSettingsPathMust(t)); err != nil {
		t.Errorf("claude settings not created: %v", err)
	}
	if _, err := os.Stat(openCodePluginPathMust(t)); err != nil {
		t.Errorf("opencode plugin not created: %v", err)
	}
}

func TestRunAgentInstallUnknownTargetErrors(t *testing.T) {
	withAgentTempHome(t)
	err := runAgentInstall("foobar")
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "unknown target"))
}

func TestRunAgentUninstallAll(t *testing.T) {
	withAgentTempHome(t)
	if err := runAgentInstall("all"); err != nil {
		t.Fatal(err)
	}
	if err := runAgentUninstall("all"); err != nil {
		t.Fatalf("uninstall all: %v", err)
	}
	if _, err := os.Stat(openCodePluginPathMust(t)); !os.IsNotExist(err) {
		t.Errorf("opencode plugin should be removed")
	}
}

// helpers

func claudeSettingsPathMust(t *testing.T) string {
	t.Helper()
	p, err := claudeSettingsPath()
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func openCodePluginPathMust(t *testing.T) string {
	t.Helper()
	p, err := openCodePluginPath()
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func collectCommands(settings map[string]any) []string {
	var out []string
	hooks, ok := settings["hooks"].(map[string]any)
	if !ok {
		return out
	}
	for _, raw := range hooks {
		groups, ok := raw.([]any)
		if !ok {
			continue
		}
		for _, g := range groups {
			gm, _ := g.(map[string]any)
			handlers, _ := gm["hooks"].([]any)
			for _, h := range handlers {
				hm, _ := h.(map[string]any)
				if cmd, ok := hm["command"].(string); ok {
					out = append(out, cmd)
				}
			}
		}
	}
	return out
}
