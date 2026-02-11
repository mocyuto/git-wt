package git

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// copyFile copies a single file from src to dst.
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

// getIgnoredFilesBase returns the raw list of ignored files from git.
func getIgnoredFilesBase(dir string) ([]string, error) {
	cmd := exec.Command("git", "ls-files", "--others", "--ignored", "--exclude-standard")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list ignored files: %v", err)
	}

	var files []string
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		relPath := scanner.Text()
		if relPath == "" {
			continue
		}

		// Explicitly skip git-wt configuration files
		if relPath == "git-wt.config.yml" || relPath == "git-wt.config.yaml" {
			continue
		}

		// Check if it's a file (not directory)
		src := filepath.Join(dir, relPath)
		info, err := os.Stat(src)
		if err != nil || info.IsDir() {
			continue
		}

		files = append(files, relPath)
	}

	return files, nil
}

// isPathIgnoredByPatterns checks if a path matches any of the given ignore patterns.
func isPathIgnoredByPatterns(relPath string, ignorePatterns []string) bool {
	for _, pattern := range ignorePatterns {
		// Check if pattern matches the full path
		match, err := filepath.Match(pattern, relPath)
		if err == nil && match {
			return true
		}

		// Check if pattern matches any part of the path
		pathParts := strings.Split(relPath, string(os.PathSeparator))
		for i := range pathParts {
			match, err := filepath.Match(pattern, pathParts[i])
			if err == nil && match {
				return true
			}

			subPath := strings.Join(pathParts[i:], string(os.PathSeparator))
			match, err = filepath.Match(pattern, subPath)
			if err == nil && match {
				return true
			}
		}
	}
	return false
}
