package cmd

import (
	"fmt"

	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/mocyuto/zgt/internal/git"
	"github.com/mocyuto/zgt/internal/tmux"
	"github.com/mocyuto/zgt/internal/zcontext"
	"github.com/spf13/cobra"
)

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

			for _, wt := range selectedWts {
				branchName := strings.TrimPrefix(wt.Branch, "refs/heads/")
				if err := openWorktree(wt.Path, branchName); err != nil {
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

		return openWorktree(path, branch)
	},
}

func openWorktree(path, branch string) error {
	ctx := zcontext.New(path, branch)
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
}
