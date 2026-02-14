package cmd

import (
	"fmt"
	"os"

	"github.com/gdamore/tcell/v2"
	"github.com/mocyuto/zgt/internal/git"
	"github.com/mocyuto/zgt/internal/logger"
	"github.com/rivo/tview"
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
		mainRoot, err := git.GetMainProjectRootFromPath(cwd)
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
			// 3. TUI Selection
			selectedFiles, err = selectFilesTUI(files)
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

func selectFilesTUI(files []string) ([]string, error) {
	app := tview.NewApplication()

	list := tview.NewList()
	list.SetTitle(" Select files to sync (Press Enter to toggle, 's' to finish, 'q' to cancel) ").
		SetBorder(true)

	selected := make(map[string]bool)
	for _, f := range files {
		selected[f] = false
		updateListItem(list, f, false)
	}

	var result []string
	var finalErr error

	list.SetSelectedFunc(func(index int, mainText string, secondaryText string, shortcut rune) {
		filename := files[index]
		selected[filename] = !selected[filename]

		// Update the list item text to show selection status
		list.RemoveItem(index)
		list.InsertItem(index, getListItemText(filename, selected[filename]), "", 0, nil)
		list.SetCurrentItem(index)
	})

	list.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Rune() == 's' {
			for f, isSelected := range selected {
				if isSelected {
					result = append(result, f)
				}
			}
			app.Stop()
			return nil
		}
		if event.Rune() == 'q' || event.Key() == tcell.KeyEscape {
			finalErr = fmt.Errorf("sync cancelled by user")
			app.Stop()
			return nil
		}
		return event
	})

	if err := app.SetRoot(list, true).Run(); err != nil {
		return nil, err
	}

	return result, finalErr
}

func getListItemText(filename string, isSelected bool) string {
	if isSelected {
		return "[green][x][white] " + filename
	}
	return "[ ] " + filename
}

func updateListItem(list *tview.List, filename string, isSelected bool) {
	list.AddItem(getListItemText(filename, isSelected), "", 0, nil)
}
