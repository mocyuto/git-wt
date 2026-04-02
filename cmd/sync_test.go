package cmd

import (
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
		filter   string
		expected []string
	}{
		{
			name:     "No filter",
			filter:   "",
			expected: files,
		},
		{
			name:     "Filter by substring in the middle",
			filter:   "config",
			expected: []string{"config/settings.yaml"},
		},
		{
			name:     "Filter by extension",
			filter:   ".env",
			expected: []string{".env"},
		},
		{
			name:     "Filter by multiple matching files",
			filter:   "s",
			expected: []string{"config/settings.yaml", "internal/db/schema.sql", "logs/app.log", "secret.txt"},
		},
		{
			name:     "No match",
			filter:   "nonexistent",
			expected: nil,
		},
		{
			name:     "Match by directory name",
			filter:   "internal/",
			expected: []string{"internal/db/schema.sql"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := filterFiles(files, tt.filter)
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("filterFiles() = %v, want %v", got, tt.expected)
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
}
