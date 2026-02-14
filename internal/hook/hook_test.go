package hook

import (
	"os"
	"strings"
	"testing"

	"github.com/mocyuto/zgt/internal/config"
	"github.com/mocyuto/zgt/internal/template"
)

func TestRunHooks_Env(t *testing.T) {
	// Setup config
	config.AppConfig.Env = map[string]string{
		"TEST_BRANCH": "{{.Branch}}",
		"TEST_STATIC": "hello",
	}
	config.AppConfig.Hooks.Add = []string{
		"echo $TEST_BRANCH > branch_out.txt",
		"echo $TEST_STATIC > static_out.txt",
	}

	tempDir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(origDir)

	ctx := template.Context{
		Path:   "test-path",
		Branch: "feature-xyz",
		Repo:   "test-repo",
	}

	RunHooks("add", ctx)

	// Verify outputs
	branchOut, err := os.ReadFile("branch_out.txt")
	if err != nil {
		t.Fatalf("Failed to read branch_out.txt: %v", err)
	}
	if strings.TrimSpace(string(branchOut)) != "feature-xyz" {
		t.Errorf("Expected branch_out.txt to be 'feature-xyz', got '%s'", strings.TrimSpace(string(branchOut)))
	}

	staticOut, err := os.ReadFile("static_out.txt")
	if err != nil {
		t.Fatalf("Failed to read static_out.txt: %v", err)
	}
	if strings.TrimSpace(string(staticOut)) != "hello" {
		t.Errorf("Expected static_out.txt to be 'hello', got '%s'", strings.TrimSpace(string(staticOut)))
	}
}
