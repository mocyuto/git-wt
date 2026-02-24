package tmux

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/mocyuto/zgt/internal/config"
	"github.com/mocyuto/zgt/internal/template"
)

type PaneStatus struct {
	ID        string
	Title     string
	Command   string
	Running   string
	IsRunning bool
}

type WindowStatus struct {
	ID    string
	Name  string
	Panes []PaneStatus
}

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

// ListWindows returns a list of all tmux windows in all sessions.
func ListWindows() ([]WindowStatus, error) {
	if !isTmuxAvailable() || !isTmuxRunning() {
		return nil, nil
	}

	cmd := exec.Command("tmux", "list-windows", "-a", "-F", "#{window_id} #{window_name}")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	return parseWindows(string(output))
}

// ListSessionWindows returns a list of all tmux windows in the current session.
func ListSessionWindows() ([]WindowStatus, error) {
	if !isTmuxAvailable() || !isTmuxRunning() {
		return nil, nil
	}

	// Use current session
	cmd := exec.Command("tmux", "list-windows", "-F", "#{window_id} #{window_name}")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	return parseWindows(string(output))
}

func parseWindows(output string) ([]WindowStatus, error) {
	var windows []WindowStatus
	lines := strings.Split(strings.TrimSpace(output), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, " ", 2)
		if len(parts) < 2 {
			continue
		}
		windows = append(windows, WindowStatus{
			ID:   parts[0],
			Name: parts[1],
		})
	}
	return windows, nil
}

// GetWindowStatus returns the status of all panes in the given window.
func GetWindowStatus(windowID string) (*WindowStatus, error) {
	// window_id, window_name, pane_id, pane_current_command, pane_pid, pane_title
	format := "#{window_id} #{window_name} #{pane_id} #{pane_current_command} #{pane_pid} #{pane_title}"
	cmd := exec.Command("tmux", "list-panes", "-t", windowID, "-F", format)
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) == 0 || lines[0] == "" {
		return nil, fmt.Errorf("no panes found for window %s", windowID)
	}

	firstLineParts := strings.SplitN(lines[0], " ", 6)
	status := &WindowStatus{
		ID:   firstLineParts[0],
		Name: firstLineParts[1],
	}

	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, " ", 6)
		if len(parts) < 6 {
			continue
		}

		id := parts[2]
		currentCmd := parts[3]
		pid := parts[4]
		title := parts[5]

		runningProcess := ""
		isRunning := false

		// Try to find the real running process under the shell
		if currentCmd == "zsh" || currentCmd == "bash" || currentCmd == "sh" {
			// Find child processes of the shell PID
			childCmd := exec.Command("pgrep", "-P", pid)
			childOutput, err := childCmd.Output()
			if err == nil {
				childPIDs := strings.Fields(strings.TrimSpace(string(childOutput)))
				if len(childPIDs) > 0 {
					// Get the command name of the first child process
					psCmd := exec.Command("ps", "-o", "comm=", "-p", childPIDs[0])
					psOutput, err := psCmd.Output()
					if err == nil {
						runningProcess = strings.TrimSpace(string(psOutput))
						isRunning = true
					}
				}
			}
		} else {
			// If it's not a shell, the current command itself is the running process
			runningProcess = currentCmd
			isRunning = true
		}

		if runningProcess == "" {
			runningProcess = "Waiting/Idle"
		}

		status.Panes = append(status.Panes, PaneStatus{
			ID:        id,
			Title:     title,
			Command:   currentCmd,
			Running:   runningProcess,
			IsRunning: isRunning,
		})
	}

	return status, nil
}
