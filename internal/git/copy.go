package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"

	"github.com/mocyuto/git-wt/internal/logger"
)

func CopyIgnoredFiles(sourceRoot, targetPath string, ignorePatterns []string, verbose bool) error {
	allFiles, err := getIgnoredFilesBase(sourceRoot)
	if err != nil {
		return err
	}

	var filesToCopy []string
	for _, relPath := range allFiles {
		if isPathIgnoredByPatterns(relPath, ignorePatterns) {
			if verbose {
				fmt.Printf("Skipping ignored file: %s\n", relPath)
			}
			continue
		}
		filesToCopy = append(filesToCopy, relPath)
	}

	// Warning for uncommitted/untracked config files NOT in .gitignore
	configFiles := []string{"git-wt.config.yml", "git-wt.config.yaml"}
	for _, cfg := range configFiles {
		src := filepath.Join(sourceRoot, cfg)
		if info, err := os.Stat(src); err == nil && !info.IsDir() {
			// Check if it's ignored by git
			checkCmd := exec.Command("git", "check-ignore", "-q", cfg)
			checkCmd.Dir = sourceRoot
			err := checkCmd.Run()
			if err != nil {
				// Exit code is non-zero if not ignored
				logger.Warn("%s exists but is not in .gitignore. It will not be copied to the new worktree.", cfg)
			}
		}
	}

	// Use worker pool pattern to limit concurrency
	const maxWorkers = 20
	sem := make(chan struct{}, maxWorkers)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var firstErr error
	var completed int
	totalFiles := len(filesToCopy)

	// Show initial progress
	if totalFiles > 0 {
		fmt.Printf("\rProgress: 0/%d (0%%)", totalFiles)
	}

	for _, relPath := range filesToCopy {
		wg.Add(1)
		sem <- struct{}{} // Acquire semaphore

		go func(relPath string) {
			defer wg.Done()
			defer func() { <-sem }() // Release semaphore

			src := filepath.Join(sourceRoot, relPath)
			dst := filepath.Join(targetPath, relPath)

			if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
				mu.Lock()
				if firstErr == nil {
					firstErr = logger.Errorf("failed to create directory for %s: %v", dst, err)
				}
				mu.Unlock()
				return
			}

			if err := copyFile(src, dst); err != nil {
				if verbose {
					fmt.Printf("\nFailed to copy %s: %v\n", relPath, err)
				}
			} else {
				mu.Lock()
				completed++
				percentage := int(float64(completed) / float64(totalFiles) * 100)
				if verbose {
					fmt.Printf("\rProgress: %d/%d (%d%%) - Copied: %s\n", completed, totalFiles, percentage, relPath)
				} else {
					fmt.Printf("\rProgress: %d/%d (%d%%)", completed, totalFiles, percentage)
				}
				mu.Unlock()
			}
		}(relPath)
	}

	wg.Wait()

	// Print newline after progress display
	if totalFiles > 0 {
		fmt.Println()
	}

	return firstErr
}
