package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/signal"
	"sort"
	"strings"
	"syscall"
	"text/tabwriter"
	"time"

	"github.com/mocyuto/zgt/internal/agentstatus"
	"github.com/mocyuto/zgt/internal/config"
	"github.com/mocyuto/zgt/internal/git"
	"github.com/mocyuto/zgt/internal/logger"
	"github.com/spf13/cobra"
)

var agentCmd = &cobra.Command{
	Use:   "agent",
	Short: "Show and manage AI agent (opencode / Claude Code) status for worktrees",
	Long: `Show and manage AI agent status for worktrees.

Reports whether opencode or Claude Code sessions running inside each
worktree are working, idle, or waiting. Status is written by hooks/plugins
installed via 'zgt agent install'.`,
}

var agentStatusWatch bool
var agentStatusAll bool

var agentStatusCmd = &cobra.Command{
	Use:     "status [worktree-name]",
	Aliases: []string{"ls"},
	Short:   "Show agent status for each worktree",
	Long:    `Show the idle/working/waiting status of opencode / Claude Code sessions for each worktree.`,
	Args:    cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if !config.AppConfig.Agent.Enabled {
			logger.Warn("agent integration is disabled in config (agent.enabled=false)")
		}
		if agentStatusWatch {
			return watchAgentStatus(args)
		}
		return printAgentStatus(args)
	},
}

// claudeHookInput is the subset of fields Claude Code sends to hooks on
// stdin. We only need the event name, cwd, and session id.
type claudeHookInput struct {
	HookEventName string `json:"hook_event_name"`
	Cwd           string `json:"cwd"`
	SessionID     string `json:"session_id"`
}

var agentHookCmd = &cobra.Command{
	Use:   "hook <claude>",
	Short: "Internal: update agent status from a Claude Code hook (stdin JSON)",
	Long: `Internal command invoked by the Claude Code hooks installed via
'zgt agent install'. Reads the hook event JSON from stdin and writes the
corresponding agent status. Not intended to be called directly.`,
	Args:      cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	ValidArgs: []string{"claude"},
	Hidden:    true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runAgentHook(args[0], os.Stdin)
	},
}

var (
	setStatusAgent  string
	setStatusStatus string
	setStatusCwd    string
	setStatusSID    string
)

var agentSetStatusCmd = &cobra.Command{
	Use:   "set-status",
	Short: "Write an agent status record for a worktree (used by the opencode plugin)",
	Long: `Write an agent status record for the given cwd. Intended to be
called by the opencode plugin installed via 'zgt agent install'.`,
	Hidden: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if setStatusCwd == "" {
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			setStatusCwd = cwd
		}
		agent := agentstatus.Agent(setStatusAgent)
		if !validAgent(agent) {
			return fmt.Errorf("invalid --agent %q; expected claude or opencode", setStatusAgent)
		}
		status := agentstatus.Status(setStatusStatus)
		if !validStatus(status) {
			return fmt.Errorf("invalid --status %q; expected working, idle, or waiting", setStatusStatus)
		}
		return agentstatus.SetStatus(setStatusCwd, agent, status, setStatusSID)
	},
}

func validAgent(a agentstatus.Agent) bool {
	return a == agentstatus.AgentClaude || a == agentstatus.AgentOpenCode
}

func validStatus(s agentstatus.Status) bool {
	return s == agentstatus.StatusWorking || s == agentstatus.StatusIdle || s == agentstatus.StatusWaiting
}

var clearStatusCwd string

var agentClearStatusCmd = &cobra.Command{
	Use:    "clear-status",
	Short:  "Remove an agent status record for a worktree (used by the opencode plugin)",
	Hidden: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if clearStatusCwd == "" {
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			clearStatusCwd = cwd
		}
		return agentstatus.ClearStatus(clearStatusCwd)
	},
}

func init() {
	rootCmd.AddCommand(agentCmd)
	agentCmd.AddCommand(agentStatusCmd)
	agentCmd.AddCommand(agentHookCmd)
	agentCmd.AddCommand(agentSetStatusCmd)
	agentCmd.AddCommand(agentClearStatusCmd)

	agentStatusCmd.Flags().BoolVarP(&agentStatusWatch, "watch", "w", false, "refresh status every 2 seconds until interrupted")
	agentStatusCmd.Flags().BoolVarP(&agentStatusAll, "all", "a", false, "include worktrees with no agent status")

	agentSetStatusCmd.Flags().StringVar(&setStatusAgent, "agent", "opencode", "agent name (claude|opencode)")
	agentSetStatusCmd.Flags().StringVar(&setStatusStatus, "status", "", "status (working|idle|waiting)")
	agentSetStatusCmd.Flags().StringVar(&setStatusCwd, "cwd", "", "worktree path (defaults to current directory)")
	agentSetStatusCmd.Flags().StringVar(&setStatusSID, "session-id", "", "optional session id")
	_ = agentSetStatusCmd.MarkFlagRequired("status")

	agentClearStatusCmd.Flags().StringVar(&clearStatusCwd, "cwd", "", "worktree path (defaults to current directory)")
}

// runAgentHook maps a Claude Code hook event to an agentstatus write/clear.
func runAgentHook(agent string, r io.Reader) error {
	data, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	var in claudeHookInput
	if err := json.Unmarshal(data, &in); err != nil {
		return fmt.Errorf("parse hook input: %v", err)
	}
	if in.Cwd == "" {
		cwd, _ := os.Getwd()
		in.Cwd = cwd
	}
	ag := agentstatus.Agent(agent)
	switch in.HookEventName {
	case "UserPromptSubmit":
		return agentstatus.SetStatus(in.Cwd, ag, agentstatus.StatusWorking, in.SessionID)
	case "Stop":
		return agentstatus.SetStatus(in.Cwd, ag, agentstatus.StatusIdle, in.SessionID)
	case "Notification":
		// permission_prompt / idle_prompt indicate the agent is blocked
		// waiting for the user; other notifications (auth_success etc.)
		// are still "waiting" in the sense that the session is paused.
		return agentstatus.SetStatus(in.Cwd, ag, agentstatus.StatusWaiting, in.SessionID)
	case "SessionEnd":
		return agentstatus.ClearStatus(in.Cwd)
	default:
		// Unknown event: no-op so we stay forward-compatible.
		return nil
	}
}

type agentStatusRow struct {
	Path   string
	Branch string
	Agent  string
	Status string
	Age    string
}

func collectAgentStatusRows(showAll bool) ([]agentStatusRow, error) {
	wts, err := git.GetWorktrees()
	if err != nil {
		return nil, err
	}
	staleAge := agentstatus.ParseStaleAge(config.AppConfig.Agent.StaleAfter)

	// Build a set of worktree paths for quick lookup, plus a map of any
	// status records that don't correspond to a known worktree (so --all
	// can surface orphaned records).
	known := make(map[string]bool, len(wts))
	for _, wt := range wts {
		known[agentstatus.Normalize(wt.Path)] = true
	}

	var rows []agentStatusRow
	for _, wt := range wts {
		branch := strings.TrimPrefix(wt.Branch, "refs/heads/")
		if branch == "" && wt.HEAD != "" {
			head := wt.HEAD
			if len(head) > 7 {
				head = head[:7]
			}
			branch = "(" + head + "...)"
		}
		rec, ok := agentstatus.FindByPath(wt.Path, staleAge)
		if !ok {
			if showAll {
				rows = append(rows, agentStatusRow{Path: wt.Path, Branch: branch, Agent: "-", Status: "idle?"})
			}
			continue
		}
		rows = append(rows, agentStatusRow{
			Path:   wt.Path,
			Branch: branch,
			Agent:  string(rec.Agent),
			Status: string(rec.Status),
			Age:    humanizeAge(rec.UpdatedAt),
		})
	}

	if showAll {
		// Include status records whose CWD isn't a registered worktree.
		// Skip stale ones so we don't display long-dead sessions.
		all, _ := agentstatus.ListStatuses()
		for _, rec := range all {
			if known[agentstatus.Normalize(rec.CWD)] {
				continue
			}
			if agentstatus.IsStale(rec, staleAge) {
				continue
			}
			rows = append(rows, agentStatusRow{
				Path:   rec.CWD,
				Branch: "(unknown)",
				Agent:  string(rec.Agent),
				Status: string(rec.Status),
				Age:    humanizeAge(rec.UpdatedAt),
			})
		}
	}

	sort.SliceStable(rows, func(i, j int) bool {
		return rows[i].Path < rows[j].Path
	})
	return rows, nil
}

func printAgentStatus(args []string) error {
	if config.AppConfig.Agent.Enabled {
		// Prune stale records before listing so dead sessions disappear.
		_, _ = agentstatus.PruneStale(agentstatus.ParseStaleAge(config.AppConfig.Agent.StaleAfter))
	}
	rows, err := collectAgentStatusRows(agentStatusAll)
	if err != nil {
		return err
	}

	// Optional filter by worktree name / path substring.
	if len(args) == 1 {
		filter := args[0]
		filtered := rows[:0]
		for _, r := range rows {
			if strings.Contains(r.Path, filter) || strings.Contains(r.Branch, filter) {
				filtered = append(filtered, r)
			}
		}
		rows = filtered
	}

	if len(rows) == 0 {
		fmt.Println("No agent status records found.")
		fmt.Println("Run `zgt agent install` to enable status reporting from opencode / Claude Code.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "PATH\tBRANCH\tAGENT\tSTATUS\tAGE")
	for _, r := range rows {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", r.Path, r.Branch, r.Agent, r.Status, r.Age)
	}
	return w.Flush()
}

func watchAgentStatus(args []string) error {
	// Hide cursor while watching; restore it on any exit path, including
	// Ctrl-C/SIGTERM (which would otherwise skip the deferred restore).
	fmt.Print("\033[?25l")
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	defer func() {
		signal.Stop(sigCh)
		fmt.Print("\033[?25h")
	}()
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	render := func() {
		fmt.Print("\033[2J\033[H")
		_ = printAgentStatus(args)
	}
	render()
	for {
		select {
		case <-ticker.C:
			render()
		case <-sigCh:
			return nil
		}
	}
}

func humanizeAge(updatedAt int64) string {
	d := time.Since(time.Unix(updatedAt, 0))
	if d < 0 {
		d = 0
	}
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	default:
		return fmt.Sprintf("%dh", int(d.Hours()))
	}
}
