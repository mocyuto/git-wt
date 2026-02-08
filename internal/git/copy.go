package git

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

func CopyIgnoredFiles(sourceRoot, targetPath string, ignorePatterns []string, verbose bool) error {
	cmd := exec.Command("git", "ls-files", "--others", "--ignored", "--exclude-standard")
	cmd.Dir = sourceRoot
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	// Collect all files to copy first
	var filesToCopy []string
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		relPath := scanner.Text()

		// Filter by ignore patterns (gitignore style)
		ignored := false
		for _, pattern := range ignorePatterns {
			// Check if pattern matches the full path
			match, err := filepath.Match(pattern, relPath)
			if err == nil && match {
				ignored = true
				break
			}

			// Check if pattern matches any part of the path
			pathParts := strings.Split(relPath, string(os.PathSeparator))
			for i := range pathParts {
				match, err := filepath.Match(pattern, pathParts[i])
				if err == nil && match {
					ignored = true
					break
				}

				subPath := strings.Join(pathParts[i:], string(os.PathSeparator))
				match, err = filepath.Match(pattern, subPath)
				if err == nil && match {
					ignored = true
					break
				}
			}

			if ignored {
				break
			}
		}

		if ignored {
			if verbose {
				fmt.Printf("Skipping ignored file: %s\n", relPath)
			}
			continue
		}

		// Check if it's a file (not directory)
		src := filepath.Join(sourceRoot, relPath)
		info, err := os.Stat(src)
		if err != nil || info.IsDir() {
			continue
		}

		filesToCopy = append(filesToCopy, relPath)
	}

	if err := cmd.Wait(); err != nil {
		return err
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
					firstErr = fmt.Errorf("failed to create directory for %s: %v", dst, err)
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

func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return err
	}

	sourceInfo, err := os.Stat(src)
	if err != nil {
		return err
	}
	return os.Chmod(dst, sourceInfo.Mode())
}
