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
	ID          string
	Name        string
	SessionName string
	Panes       []PaneStatus
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

	windowName := GetWindowName(ctx)
	paneIDMap := make(map[string]string)

	// Create first pane
	firstPane := cfg.Panes[0]

	cmd := exec.Command("tmux", "new-window", "-P", "-F", "#{pane_id}", "-n", windowName, "-c", ctx.Path)
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to create tmux window: %v", err)
	}
	currentPaneID := strings.TrimSpace(string(output))
	if firstPane.Id != "" {
		paneIDMap[firstPane.Id] = currentPaneID
	}

	// Send commands to first pane
	sendKeys(currentPaneID, firstPane.Commands, ctx)

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

		// Send commands to the new pane
		sendKeys(newPaneID, p.Commands, ctx)
	}

	return nil
}

// sendKeys sends each command in commands to the specified tmux pane.
func sendKeys(paneID string, commands []string, ctx template.Context) {
	for _, c := range commands {
		if c == "" {
			continue
		}
		// Replace placeholders in command
		substitutedCmd := template.Replace(c, ctx)
		sendCmd := exec.Command("tmux", "send-keys", "-t", paneID, substitutedCmd, "Enter")
		if err := sendCmd.Run(); err != nil {
			fmt.Printf("Warning: failed to send command to pane %s: %v\n", paneID, err)
		}
	}
}

// GetWindowName returns the configured window name or a default one based on context.
func GetWindowName(ctx template.Context) string {
	cfg := config.AppConfig.Tmux
	if cfg.WindowName != "" {
		return template.Replace(cfg.WindowName, ctx)
	}
	return fmt.Sprintf("[%s]%s", ctx.Repo, ctx.Branch)
}

// ActivateWindow switches to the specified tmux window.
func ActivateWindow(windowID string) error {
	if !isTmuxAvailable() || !isTmuxRunning() {
		return nil
	}
	cmd := exec.Command("tmux", "select-window", "-t", windowID)
	return cmd.Run()
}

// GetWindowIDByName returns the window ID if a window with the given name exists.
func GetWindowIDByName(name string) (string, bool, error) {
	windows, err := ListWindows()
	if err != nil {
		return "", false, err
	}
	for _, w := range windows {
		if w.Name == name {
			return w.ID, true, nil
		}
	}
	return "", false, nil
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

	cmd := exec.Command("tmux", "list-windows", "-a", "-F", "#{session_name} #{window_id} #{window_name}")
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
	cmd := exec.Command("tmux", "list-windows", "-F", "#{session_name} #{window_id} #{window_name}")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	return parseWindows(string(output))
}

// GetCurrentSessionName returns the name of the current tmux session if available.
func GetCurrentSessionName() (string, error) {
	if !isTmuxAvailable() || !isTmuxRunning() {
		return "", nil
	}
	cmd := exec.Command("tmux", "display-message", "-p", "#S")
	output, err := cmd.Output()
	if err != nil {
		// If fails (e.g. not inside tmux), return empty
		return "", nil
	}
	return strings.TrimSpace(string(output)), nil
}

func parseWindows(output string) ([]WindowStatus, error) {
	var windows []WindowStatus
	lines := strings.Split(strings.TrimSpace(output), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, " ", 3)
		if len(parts) < 3 {
			continue
		}
		windows = append(windows, WindowStatus{
			SessionName: parts[0],
			ID:          parts[1],
			Name:        parts[2],
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
		SessionName: firstLineParts[0],
		ID:          firstLineParts[1],
		Name:        firstLineParts[2],
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
