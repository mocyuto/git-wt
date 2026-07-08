package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mocyuto/zgt/internal/agentstatus"
	"github.com/mocyuto/zgt/internal/config"
	"github.com/stretchr/testify/assert"
)

func filepathEvalSymlinks(p string) (string, error) {
	return filepath.EvalSymlinks(p)
}

func TestAgentBadgeDisabledReturnsEmpty(t *testing.T) {
	withAgentTempHome(t)
	dir, _ := os.MkdirTemp("", "zgt-badge-*")
	if err := agentstatus.SetStatus(dir, agentstatus.AgentClaude, agentstatus.StatusWorking, ""); err != nil {
		t.Fatal(err)
	}
	config.AppConfig.Agent.Enabled = false
	defer func() { config.AppConfig.Agent.Enabled = true }()
	assert.Empty(t, agentBadgeFor(dir))
}

func TestAgentBadgeEmptyCWDReturnsEmpty(t *testing.T) {
	withAgentTempHome(t)
	config.AppConfig.Agent.Enabled = true
	assert.Empty(t, agentBadgeFor(""))
}

func TestAgentBadgeNoRecordReturnsEmpty(t *testing.T) {
	withAgentTempHome(t)
	config.AppConfig.Agent.Enabled = true
	dir, _ := os.MkdirTemp("", "zgt-norec-*")
	assert.Empty(t, agentBadgeFor(dir))
}

func TestAgentBadgeWorking(t *testing.T) {
	withAgentTempHome(t)
	config.AppConfig.Agent.Enabled = true
	dir, _ := os.MkdirTemp("", "zgt-working-*")
	if err := agentstatus.SetStatus(dir, agentstatus.AgentClaude, agentstatus.StatusWorking, "s"); err != nil {
		t.Fatal(err)
	}
	got := agentBadgeFor(dir)
	assert.Contains(t, got, "claude")
	assert.Contains(t, got, "working")
	assert.Contains(t, got, "\033[32m") // green
}

func TestAgentBadgeWaiting(t *testing.T) {
	withAgentTempHome(t)
	config.AppConfig.Agent.Enabled = true
	dir, _ := os.MkdirTemp("", "zgt-waiting-*")
	if err := agentstatus.SetStatus(dir, agentstatus.AgentOpenCode, agentstatus.StatusWaiting, ""); err != nil {
		t.Fatal(err)
	}
	got := agentBadgeFor(dir)
	assert.Contains(t, got, "opencode")
	assert.Contains(t, got, "waiting")
	assert.Contains(t, got, "\033[33m") // yellow
}

func TestAgentBadgeIdle(t *testing.T) {
	withAgentTempHome(t)
	config.AppConfig.Agent.Enabled = true
	dir, _ := os.MkdirTemp("", "zgt-idle-*")
	if err := agentstatus.SetStatus(dir, agentstatus.AgentClaude, agentstatus.StatusIdle, ""); err != nil {
		t.Fatal(err)
	}
	got := agentBadgeFor(dir)
	assert.Contains(t, got, "idle")
	assert.Contains(t, got, "\033[90m") // gray
}

func TestAgentBadgeSubdirectoryMatchesAncestor(t *testing.T) {
	withAgentTempHome(t)
	config.AppConfig.Agent.Enabled = true
	root, _ := os.MkdirTemp("", "zgt-ancestor-*")
	root, _ = filepathEvalSymlinks(root)
	sub := root + "/src/deep"
	if err := os.MkdirAll(sub, 0755); err != nil {
		t.Fatal(err)
	}
	if err := agentstatus.SetStatus(root, agentstatus.AgentClaude, agentstatus.StatusWorking, ""); err != nil {
		t.Fatal(err)
	}
	got := agentBadgeFor(sub)
	assert.Contains(t, got, "working")
}

func TestTmuxLsCommandRegistered(t *testing.T) {
	assert.Equal(t, "ls", tmuxLsCmd.Use)
	assert.NotEmpty(t, tmuxLsCmd.Short)
}

func TestListTmuxWindowsNoTmuxPrintsMessage(t *testing.T) {
	// No tmux running in CI; ListTmuxWindows should print the no-windows
	// message and return nil rather than crashing.
	orig := os.Getenv("TMUX")
	os.Unsetenv("TMUX")
	defer os.Setenv("TMUX", orig)
	err := ListTmuxWindows()
	assert.NoError(t, err)
}
