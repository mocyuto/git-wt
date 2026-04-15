package cmd

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestFilterFiles(t *testing.T) {
	files := []string{
		".env",
		"config/settings.yaml",
		"internal/db/schema.sql",
		"logs/app.log",
		"secret.txt",
	}

	tests := []struct {
		name     string
		include  string
		exclude  string
		expected []string
	}{
		{
			name:     "No filter",
			include:  "",
			exclude:  "",
			expected: files,
		},
		{
			name:     "Include only",
			include:  "config",
			exclude:  "",
			expected: []string{"config/settings.yaml"},
		},
		{
			name:     "Exclude only (single)",
			include:  "",
			exclude:  "config",
			expected: []string{".env", "internal/db/schema.sql", "logs/app.log", "secret.txt"},
		},
		{
			name:     "Exclude only (multiple with comma)",
			include:  "",
			exclude:  "config, logs",
			expected: []string{".env", "internal/db/schema.sql", "secret.txt"},
		},
		{
			name:     "Include and Exclude combined",
			include:  "s",
			exclude:  "log, config",
			expected: []string{"internal/db/schema.sql", "secret.txt"},
		},
		{
			name:     "Filter by extension",
			include:  ".env",
			exclude:  "",
			expected: []string{".env"},
		},
		{
			name:     "No match by include",
			include:  "nonexistent",
			exclude:  "",
			expected: nil,
		},
		{
			name:     "Match by directory name",
			include:  "internal/",
			exclude:  "",
			expected: []string{"internal/db/schema.sql"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := filterFiles(files, tt.include, tt.exclude)
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("%s: filterFiles() = %v, want %v", tt.name, got, tt.expected)
			}
		})
	}
}

func TestSyncCmdFlags(t *testing.T) {
	// Test that the path flag is registered
	flag := syncCmd.Flags().Lookup("path")
	if flag == nil {
		t.Fatal("path flag not found")
	}
	if flag.Shorthand != "p" {
		t.Errorf("expected shorthand 'p', got '%s'", flag.Shorthand)
	}

	// Test syncAll flag
	allFlag := syncCmd.Flags().Lookup("all")
	if allFlag == nil {
		t.Fatal("all flag not found")
	}
	if allFlag.Shorthand != "a" {
		t.Errorf("expected shorthand 'a', got '%s'", allFlag.Shorthand)
	}

	// Test ignore flag
	ignoreFlag := syncCmd.Flags().Lookup("ignore")
	if ignoreFlag == nil {
		t.Fatal("ignore flag not found")
	}
	if ignoreFlag.Shorthand != "i" {
		t.Errorf("expected shorthand 'i', got '%s'", ignoreFlag.Shorthand)
	}
}

func TestNormalizeSyncPaths(t *testing.T) {
	t.Run("run from worktree root", func(t *testing.T) {
		files, err := normalizeSyncPaths("/repo", "/repo", []string{".env", "config/local.yml"})
		if err != nil {
			t.Fatalf("normalizeSyncPaths returned error: %v", err)
		}

		expected := []string{".env", "config/local.yml"}
		if !reflect.DeepEqual(files, expected) {
			t.Fatalf("normalizeSyncPaths() = %v, want %v", files, expected)
		}
	})

	t.Run("run from subdirectory", func(t *testing.T) {
		files, err := normalizeSyncPaths("/repo", filepath.Join("/repo", "src"), []string{".env", "nested/local.yml"})
		if err != nil {
			t.Fatalf("normalizeSyncPaths returned error: %v", err)
		}

		expected := []string{
			filepath.Join("src", ".env"),
			filepath.Join("src", "nested", "local.yml"),
		}
		if !reflect.DeepEqual(files, expected) {
			t.Fatalf("normalizeSyncPaths() = %v, want %v", files, expected)
		}
	})
}
