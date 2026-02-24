package cmd

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/gdamore/tcell/v2"
	"github.com/mocyuto/zgt/internal/logger"
	"github.com/mocyuto/zgt/skills"
	"github.com/spf13/cobra"
)

var (
	skillInstallAll bool
)

var skillCmd = &cobra.Command{
	Use:   "skill",
	Short: "Manage agent skills",
	Long:  `Manage agent skills for the AI coding assistant.`,
}

var skillInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Install skills from the local skills directory",
	Long: `Install skills from the 'skills/' directory in the current repository
to various agent skills directories (local/global .claude and .agent).`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := runInstall(); err != nil {
			logger.Error("Failed to install skills: %v", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(skillCmd)
	skillCmd.AddCommand(skillInstallCmd)
	skillInstallCmd.Flags().BoolVarP(&skillInstallAll, "all", "a", false, "Install to all targets without selection")
}

type installTarget struct {
	name string
	path string
}

func getInstallTargets() ([]installTarget, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %v", err)
	}
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current directory: %v", err)
	}

	return []installTarget{
		{name: "local .claude", path: filepath.Join(cwd, ".claude", "skills")},
		{name: "local .agent", path: filepath.Join(cwd, ".agent", "skills")},
		{name: "global .claude", path: filepath.Join(home, ".claude", "skills")},
		{name: "global .agent", path: filepath.Join(home, ".agent", "skills")},
		{name: "global antigravity", path: filepath.Join(home, ".gemini", "antigravity", "skills")},
	}, nil
}

func runInstall() error {
	srcFS := skills.FS
	srcDir := "." // root of embedded FS

	targets, err := getInstallTargets()
	if err != nil {
		return err
	}

	var selectedPaths []string
	if skillInstallAll {
		for _, t := range targets {
			selectedPaths = append(selectedPaths, t.path)
		}
	} else {
		selectedPaths, err = selectTargetsCustomTUI(targets)
		if err != nil {
			return err
		}
	}

	if len(selectedPaths) == 0 {
		logger.Info("No targets selected. Installation cancelled.")
		return nil
	}

	entries, err := fs.ReadDir(srcFS, srcDir)
	if err != nil {
		return fmt.Errorf("failed to read embedded skills: %v", err)
	}

	for _, destDir := range selectedPaths {
		if err := os.MkdirAll(destDir, 0755); err != nil {
			return fmt.Errorf("failed to create destination directory %s: %v", destDir, err)
		}

		for _, entry := range entries {
			// Skip files in the root of embedded FS (like skills.go)
			if !entry.IsDir() {
				continue
			}

			skillName := entry.Name()
			srcSkillPath := skillName
			destSkillPath := filepath.Join(destDir, skillName)

			logger.Info("Installing %s to %s...", skillName, destDir)
			if err := copyEmbedDir(srcFS, srcSkillPath, destSkillPath); err != nil {
				return fmt.Errorf("failed to copy skill '%s' to '%s': %v", skillName, destDir, err)
			}
		}
	}

	logger.Success("Skills installed successfully to selected targets.")
	return nil
}

func selectTargetsCustomTUI(targets []installTarget) ([]string, error) {
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

	draw := func() {
		s.Clear()
		style := tcell.StyleDefault.Background(tcell.ColorReset).Foreground(tcell.ColorReset)
		selectedStyle := style.Background(tcell.ColorGray).Foreground(tcell.ColorWhite)

		title := " Select installation targets "
		for i, r := range title {
			s.SetContent(i+2, 1, r, nil, style.Bold(true))
		}

		for i, t := range targets {
			row := i + 3
			itemStyle := style
			if i == cursor {
				itemStyle = selectedStyle
			}

			checkbox := "[ ]"
			if selected[i] {
				checkbox = "[x]"
			}

			text := fmt.Sprintf("%s %s", checkbox, t.name)
			for j, r := range text {
				s.SetContent(j+2, row, r, nil, itemStyle)
			}
		}

		footer := " (j/k, p/n, Up/Down: Move | Space/Enter: Toggle | s: Install | q/Esc: Cancel) "
		for i, r := range footer {
			s.SetContent(i+2, len(targets)+5, r, nil, style.Italic(true))
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
				return nil, fmt.Errorf("installation cancelled")
			case tcell.KeyUp, tcell.KeyCtrlP:
				cursor--
				if cursor < 0 {
					cursor = len(targets) - 1
				}
			case tcell.KeyDown, tcell.KeyCtrlN:
				cursor++
				if cursor >= len(targets) {
					cursor = 0
				}
			case tcell.KeyEnter:
				selected[cursor] = !selected[cursor]
			case tcell.KeyRune:
				switch ev.Rune() {
				case 'q':
					return nil, fmt.Errorf("installation cancelled")
				case 'j':
					cursor++
					if cursor >= len(targets) {
						cursor = 0
					}
				case 'k':
					cursor--
					if cursor < 0 {
						cursor = len(targets) - 1
					}
				case ' ':
					selected[cursor] = !selected[cursor]
				case 's':
					var result []string
					for i, t := range targets {
						if selected[i] {
							result = append(result, t.path)
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

func copyEmbedDir(srcFS fs.FS, src, dest string) error {
	if err := os.RemoveAll(dest); err != nil {
		return err
	}

	return fs.WalkDir(srcFS, src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		destPath := filepath.Join(dest, relPath)

		if d.IsDir() {
			return os.MkdirAll(destPath, 0755)
		}

		data, err := fs.ReadFile(srcFS, path)
		if err != nil {
			return err
		}

		return os.WriteFile(destPath, data, d.Type().Perm())
	})
}
