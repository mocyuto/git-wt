package cmd

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/mocyuto/zgt/internal/config"
	"github.com/mocyuto/zgt/internal/git"
	"github.com/mocyuto/zgt/internal/gitroot"
	"github.com/mocyuto/zgt/internal/logger"
	"github.com/mocyuto/zgt/internal/state"
	"github.com/mocyuto/zgt/internal/tmux"
	"github.com/mocyuto/zgt/internal/zcontext"
	"github.com/spf13/cobra"
)

var tmuxOpenProfileFlag string

var tmuxOpenCmd = &cobra.Command{
	Use:   "open [worktree-name]",
	Short: "Open or activate tmux window for a worktree (interactive if no name given)",
	Args:  cobra.MaximumNArgs(1),
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) != 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		completions, err := git.GetWorktreeCompletions()
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}
		return completions, cobra.ShellCompDirectiveNoFileComp
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		// Validate profile flag up-front so both single and TUI modes fail
		// before any side effects.
		if tmuxOpenProfileFlag != "" && !config.ProfileExists(tmuxOpenProfileFlag) {
			return logger.Errorf("unknown profile %q; define it under the 'profiles' section in zgt.config.yml", tmuxOpenProfileFlag)
		}

		if len(args) == 0 {
			wts, err := git.GetWorktrees()
			if err != nil {
				return err
			}
			if len(wts) == 0 {
				fmt.Println("No worktrees found.")
				return nil
			}

			selectedWts, err := selectWorktreesTUI(wts)
			if err != nil {
				return err
			}

			if len(selectedWts) == 0 {
				fmt.Println("No worktrees selected.")
				return nil
			}

			profiles, err := resolveProfiles(selectedWts, tmuxOpenProfileFlag)
			if err != nil {
				return err
			}

			for _, wt := range selectedWts {
				branchName := strings.TrimPrefix(wt.Branch, "refs/heads/")
				profile := profiles[state.NormalizePath(wt.Path)]
				if err := openWorktree(wt.Path, branchName, profile); err != nil {
					fmt.Printf("Error opening %s: %v\n", branchName, err)
				}
			}
			return nil
		}

		search := args[0]
		path, branch, err := git.ResolveWorktreeInfo(search)
		if err != nil {
			return fmt.Errorf("failed to resolve worktree info for %s: %v", search, err)
		}

		profiles, err := resolveProfiles([]git.Worktree{{Path: path}}, tmuxOpenProfileFlag)
		if err != nil {
			return err
		}
		return openWorktree(path, branch, profiles[state.NormalizePath(path)])
	},
}

// resolveProfiles returns a map of normalized worktree path -> profile.
//
// When overrideProfile is non-empty it is validated and persisted to state
// (a single LoadState/SaveState cycle regardless of how many worktrees were
// passed), mirroring `zgt add --profile`. When empty, the existing profile
// stored for each worktree in state is returned without any state mutation.
func resolveProfiles(wts []git.Worktree, overrideProfile string) (map[string]string, error) {
	mainRoot, err := gitroot.GetMainProjectRoot()
	if err != nil {
		return nil, logger.Errorf("failed to get main project root: %v", err)
	}
	projectName := filepath.Base(mainRoot)
	_ = state.LoadState()

	result := make(map[string]string, len(wts))
	if overrideProfile == "" {
		for _, wt := range wts {
			absPath := state.NormalizePath(wt.Path)
			profile := ""
			if proj, ok := state.AppState.Projects[projectName]; ok {
				if w, ok := proj.Worktrees[absPath]; ok {
					profile = w.Profile
				}
			}
			result[absPath] = profile
		}
		return result, nil
	}

	for _, wt := range wts {
		absPath := state.NormalizePath(wt.Path)
		state.SetProfile(projectName, absPath, overrideProfile)
		result[absPath] = overrideProfile
	}
	if err := state.SaveState(); err != nil {
		return nil, logger.Errorf("failed to save state: %v", err)
	}
	return result, nil
}

func openWorktree(path, branch, profile string) error {
	ctx := zcontext.NewWithProfile(path, branch, profile)
	windowName := tmux.GetWindowName(ctx)

	windowID, exists, err := tmux.GetWindowIDByName(windowName)
	if err != nil {
		return err
	}

	if exists {
		fmt.Printf("Activating existing tmux window: %s\n", windowName)
		return tmux.ActivateWindow(windowID)
	}

	fmt.Printf("Creating new tmux window: %s\n", windowName)
	return tmux.Setup(ctx)
}

func selectWorktreesTUI(wts []git.Worktree) ([]git.Worktree, error) {
	s, err := tcell.NewScreen()
	if err != nil {
		return nil, err
	}
	if err := s.Init(); err != nil {
		return nil, err
	}
	defer s.Fini()

	cursor := 0
	selected := make(map[int]bool)
	offset := 0

	draw := func() {
		s.Clear()
		style := tcell.StyleDefault.Background(tcell.ColorReset).Foreground(tcell.ColorReset)
		selectedStyle := style.Background(tcell.ColorGray).Foreground(tcell.ColorWhite)

		_, height := s.Size()
		maxDisplay := height - 6 // Reserved for title and footer
		if maxDisplay < 1 {
			maxDisplay = 1
		}

		// Title
		title := " Select worktrees to open in tmux "
		for i, r := range title {
			s.SetContent(i+2, 1, r, nil, style.Bold(true))
		}

		// Adjust offset if cursor is out of view
		if cursor < offset {
			offset = cursor
		} else if cursor >= offset+maxDisplay {
			offset = cursor - maxDisplay + 1
		}

		// Items
		for i := 0; i < maxDisplay && i+offset < len(wts); i++ {
			idx := i + offset
			wt := wts[idx]
			row := i + 3
			itemStyle := style
			if idx == cursor {
				itemStyle = selectedStyle
			}

			checkbox := "[ ]"
			if selected[idx] {
				checkbox = "[x]"
			}

			// Clean branch name
			branchName := strings.TrimPrefix(wt.Branch, "refs/heads/")
			if branchName == "" && wt.HEAD != "" {
				branchName = "(" + wt.HEAD[:7] + "...)"
			}

			text := fmt.Sprintf("%s %s (%s)", checkbox, branchName, wt.Path)
			for j, r := range text {
				s.SetContent(j+2, row, r, nil, itemStyle)
			}
		}

		// Footer
		footer := " (j/k, p/n, Up/Down: Move | Space/Enter: Toggle | o: Open | q/Esc: Cancel) "
		for i, r := range footer {
			s.SetContent(i+2, height-2, r, nil, style.Italic(true))
		}
		s.Show()
	}

	for {
		draw()
		ev := s.PollEvent()
		switch ev := ev.(type) {
		case *tcell.EventKey:
			switch ev.Key() {
			case tcell.KeyEscape, tcell.KeyCtrlC:
				return nil, fmt.Errorf("open cancelled")
			case tcell.KeyUp, tcell.KeyCtrlP:
				cursor--
				if cursor < 0 {
					cursor = len(wts) - 1
				}
			case tcell.KeyDown, tcell.KeyCtrlN:
				cursor++
				if cursor >= len(wts) {
					cursor = 0
				}
			case tcell.KeyEnter:
				selected[cursor] = !selected[cursor]
			case tcell.KeyRune:
				switch ev.Rune() {
				case 'q':
					return nil, fmt.Errorf("open cancelled")
				case 'j':
					cursor++
					if cursor >= len(wts) {
						cursor = 0
					}
				case 'k':
					cursor--
					if cursor < 0 {
						cursor = len(wts) - 1
					}
				case ' ':
					selected[cursor] = !selected[cursor]
				case 'o':
					var result []git.Worktree
					for i, wt := range wts {
						if selected[i] {
							result = append(result, wt)
						}
					}
					return result, nil
				}
			}
		case *tcell.EventResize:
			s.Sync()
		}
	}
}

func init() {
	tmuxCmd.AddCommand(tmuxOpenCmd)
	tmuxOpenCmd.Flags().StringVarP(&tmuxOpenProfileFlag, "profile", "", "", "select a profile from the 'profiles' section of zgt.config.yml")

	// Shell completion for --profile: list profile names defined in config.
	// config.InitConfig runs via cobra.OnInitialize before completion funcs,
	// so AppConfig.Profiles is populated by the time this is invoked.
	_ = tmuxOpenCmd.RegisterFlagCompletionFunc("profile", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return config.ProfileNames(), cobra.ShellCompDirectiveNoFileComp
	})
}
