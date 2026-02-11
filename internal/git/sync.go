package git

import (
	"fmt"
	"os"
	"path/filepath"
)

// GetIgnoredFiles returns a list of files that are ignored by git in the given directory.
func GetIgnoredFiles(dir string) ([]string, error) {
	return getIgnoredFilesBase(dir)
}

// SyncFiles copies the specified files from the source directory to the target directory.
func SyncFiles(sourceRoot, targetRoot string, files []string) error {
	for _, relPath := range files {
		src := filepath.Join(sourceRoot, relPath)
		dst := filepath.Join(targetRoot, relPath)

		if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
			return fmt.Errorf("failed to create directory for %s: %v", dst, err)
		}

		if err := copyFile(src, dst); err != nil {
			return fmt.Errorf("failed to copy %s: %v", relPath, err)
		}
	}
	return nil
}
