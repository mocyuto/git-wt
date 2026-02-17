package tmux

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/mocyuto/zgt/internal/config"
	"github.com/mocyuto/zgt/internal/template"
)

// Setup creates a new tmux window and splits it into panes as configured.
func Setup(ctx template.Context) error {
	cfg := config.AppConfig.Tmux
	if !cfg.Enabled || len(cfg.Panes) == 0 {
		return nil
	}

	if !isTmuxAvailable() {
		return fmt.Errorf("tmux command not found")
	}

	if !isTmuxRunning() {
		return fmt.Errorf("no tmux session running")
	}

	windowName := cfg.WindowName
	if windowName == "" {
		windowName = fmt.Sprintf("[%s]%s", ctx.Repo, ctx.Branch)
	}
	paneIDMap := make(map[string]string)

	// Create first pane
	firstPane := cfg.Panes[0]
	firstCmd := strings.Join(firstPane.Commands, "; ")
	if firstCmd != "" {
		firstCmd = fmt.Sprintf("%s; exec $SHELL", firstCmd)
	}

	cmd := exec.Command("tmux", "new-window", "-P", "-F", "#{pane_id}", "-n", windowName, "-c", ctx.Path, firstCmd)
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to create tmux window: %v", err)
	}
	currentPaneID := strings.TrimSpace(string(output))
	if firstPane.Id != "" {
		paneIDMap[firstPane.Id] = currentPaneID
	}

	// Create additional panes
	for i := 1; i < len(cfg.Panes); i++ {
		p := cfg.Panes[i]
		targetPaneID := currentPaneID
		if p.Target != "" {
			if id, ok := paneIDMap[p.Target]; ok {
				targetPaneID = id
			} else {
				fmt.Printf("Warning: target pane ID '%s' not found, using last created pane\n", p.Target)
			}
		}

		args := []string{"split-window", "-P", "-F", "#{pane_id}", "-t", targetPaneID, "-c", ctx.Path}
		switch p.Split {
		case "horizontal", "h":
			args = append(args, "-h")
		case "vertical", "v":
			args = append(args, "-v")
		}

		if p.Size != "" {
			if before, ok := strings.CutSuffix(p.Size, "%"); ok {
				args = append(args, "-p", before)
			} else {
				args = append(args, "-l", p.Size)
			}
		}

		paneCmd := strings.Join(p.Commands, "; ")
		if paneCmd != "" {
			paneCmd = fmt.Sprintf("%s; exec $SHELL", paneCmd)
		}
		args = append(args, paneCmd)

		cmd := exec.Command("tmux", args...)
		output, err := cmd.Output()
		if err != nil {
			fmt.Printf("Warning: failed to create tmux pane %d: %v\n", i, err)
			continue
		}
		newPaneID := strings.TrimSpace(string(output))
		currentPaneID = newPaneID
		if p.Id != "" {
			paneIDMap[p.Id] = newPaneID
		}
	}

	return nil
}

func isTmuxAvailable() bool {
	_, err := exec.LookPath("tmux")
	return err == nil
}

func isTmuxRunning() bool {
	// TMUX environment variable is usually set if inside a session
	if os.Getenv("TMUX") != "" {
		return true
	}
	// Also check if server is running
	cmd := exec.Command("tmux", "ls")
	return cmd.Run() == nil
}
