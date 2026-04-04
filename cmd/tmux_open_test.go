package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTmuxOpenCmd(t *testing.T) {
	// Test that the command is correctly initialized and has the right properties
	assert.Equal(t, "open [worktree-name]", tmuxOpenCmd.Use)
	assert.NotEmpty(t, tmuxOpenCmd.Short)
}
