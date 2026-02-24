package cmd

import (
	"fmt"
	"os"

	"github.com/gdamore/tcell/v2"
	"github.com/mocyuto/zgt/internal/git"
	"github.com/mocyuto/zgt/internal/gitroot"
	"github.com/mocyuto/zgt/internal/logger"
	"github.com/spf13/cobra"
)

var (
	syncAll bool
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync ignored files from current worktree to root project",
	Long:  `Sync files that are in .gitignore from the current worktree back to the main project root.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}

		// 1. Identify main project root
		mainRoot, err := gitroot.GetMainProjectRootFromPath(cwd)
		if err != nil {
			return fmt.Errorf("failed to identify main project root: %v", err)
		}

		if cwd == mainRoot {
			return fmt.Errorf("current directory is already the main project root")
		}

		// 2. Get ignored files
		files, err := git.GetIgnoredFiles(cwd)
		if err != nil {
			return err
		}

		if len(files) == 0 {
			logger.Success("No ignored files found to sync.")
			return nil
		}

		var selectedFiles []string
		if syncAll {
			selectedFiles = files
		} else {
			// 3. Custom TUI Selection
			selectedFiles, err = selectFilesCustomTUI(files)
			if err != nil {
				return err
			}
		}

		if len(selectedFiles) == 0 {
			logger.Info("No files selected. Sync cancelled.")
			return nil
		}

		// 4. Sync files
		err = git.SyncFiles(cwd, mainRoot, selectedFiles)
		if err != nil {
			return err
		}

		logger.Success("Successfully synced %d files to %s", len(selectedFiles), mainRoot)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(syncCmd)
	syncCmd.Flags().BoolVarP(&syncAll, "all", "a", false, "Sync all ignored files without TUI selection")
	// Also support --force as an alias for --all as requested
	syncCmd.Flags().BoolVar(&syncAll, "force", false, "Alias for --all")
}

func selectFilesCustomTUI(files []string) ([]string, error) {
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
		title := " Select files to sync "
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
		for i := 0; i < maxDisplay && i+offset < len(files); i++ {
			idx := i + offset
			f := files[idx]
			row := i + 3
			itemStyle := style
			if idx == cursor {
				itemStyle = selectedStyle
			}

			checkbox := "[ ]"
			if selected[idx] {
				checkbox = "[x]"
			}

			text := fmt.Sprintf("%s %s", checkbox, f)
			for j, r := range text {
				s.SetContent(j+2, row, r, nil, itemStyle)
			}
		}

		// Footer
		footer := " (j/k, p/n, Up/Down: Move | Space/Enter: Toggle | s: Sync | q/Esc: Cancel) "
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
				return nil, fmt.Errorf("sync cancelled")
			case tcell.KeyUp, tcell.KeyCtrlP:
				cursor--
				if cursor < 0 {
					cursor = len(files) - 1
				}
			case tcell.KeyDown, tcell.KeyCtrlN:
				cursor++
				if cursor >= len(files) {
					cursor = 0
				}
			case tcell.KeyEnter:
				selected[cursor] = !selected[cursor]
			case tcell.KeyRune:
				switch ev.Rune() {
				case 'q':
					return nil, fmt.Errorf("sync cancelled")
				case 'j':
					cursor++
					if cursor >= len(files) {
						cursor = 0
					}
				case 'k':
					cursor--
					if cursor < 0 {
						cursor = len(files) - 1
					}
				case ' ':
					selected[cursor] = !selected[cursor]
				case 's':
					var result []string
					for i, f := range files {
						if selected[i] {
							result = append(result, f)
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
