package agentstatus

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// withTempHome redirects HOME to a temp dir and returns it so status files
// are isolated per test. cleanup restores the original HOME.
func withTempHome(t *testing.T) string {
	t.Helper()
	tmp, err := os.MkdirTemp("", "zgt-agent-status-home-*")
	if err != nil {
		t.Fatalf("MkdirTemp: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(tmp) })
	orig := os.Getenv("HOME")
	t.Setenv("HOME", tmp)
	_ = orig
	return tmp
}

func TestSetAndGetStatus(t *testing.T) {
	withTempHome(t)

	dir, err := os.MkdirTemp("", "zgt-wt-*")
	if err != nil {
		t.Fatal(err)
	}

	if err := SetStatus(dir, AgentClaude, StatusWorking, "sess-1"); err != nil {
		t.Fatalf("SetStatus: %v", err)
	}

	rec, ok := GetStatus(dir)
	if !ok {
		t.Fatal("expected record to be found")
	}
	assert.Equal(t, AgentClaude, rec.Agent)
	assert.Equal(t, StatusWorking, rec.Status)
	assert.Equal(t, "sess-1", rec.SessionID)
	assert.NotZero(t, rec.UpdatedAt)
}

func TestSetStatusAtomicallyWritesFile(t *testing.T) {
	withTempHome(t)
	dir, _ := os.MkdirTemp("", "zgt-wt-*")

	if err := SetStatus(dir, AgentOpenCode, StatusIdle, ""); err != nil {
		t.Fatalf("SetStatus: %v", err)
	}

	// No leftover temp files in the status dir.
	sd, _ := statusDir()
	entries, _ := os.ReadDir(sd)
	for _, e := range entries {
		assert.False(t, len(e.Name()) > 0 && e.Name()[0] == '.',
			"temp file %q should not remain after atomic write", e.Name())
	}
}

func TestClearStatus(t *testing.T) {
	withTempHome(t)
	dir, _ := os.MkdirTemp("", "zgt-wt-*")

	if err := SetStatus(dir, AgentClaude, StatusWorking, "s"); err != nil {
		t.Fatal(err)
	}
	if err := ClearStatus(dir); err != nil {
		t.Fatalf("ClearStatus: %v", err)
	}
	if _, ok := GetStatus(dir); ok {
		t.Fatal("expected record to be removed")
	}
}

func TestClearStatusMissingIsNotError(t *testing.T) {
	withTempHome(t)
	dir, _ := os.MkdirTemp("", "zgt-wt-*")
	if err := ClearStatus(dir); err != nil {
		t.Fatalf("ClearStatus on missing path should be nil, got %v", err)
	}
}

func TestSetStatusNormalizesPath(t *testing.T) {
	withTempHome(t)
	dir, _ := os.MkdirTemp("", "zgt-wt-*")
	abs, _ := filepath.Abs(dir)
	eval, _ := filepath.EvalSymlinks(abs)

	if err := SetStatus(dir, AgentClaude, StatusIdle, ""); err != nil {
		t.Fatal(err)
	}
	// Reading via a different relative form should still hit the same file
	// because both resolve to the same normalized absolute path.
	rec, ok := GetStatus(eval)
	if !ok {
		t.Fatal("expected record under normalized path")
	}
	assert.Equal(t, rec.CWD, eval)
}

func TestFindByPathExactAndSubdirectory(t *testing.T) {
	withTempHome(t)
	root, _ := os.MkdirTemp("", "zgt-root-*")
	root, _ = filepath.EvalSymlinks(root)
	sub := filepath.Join(root, "src", "deep")
	if err := os.MkdirAll(sub, 0755); err != nil {
		t.Fatal(err)
	}

	if err := SetStatus(root, AgentClaude, StatusWorking, ""); err != nil {
		t.Fatal(err)
	}

	// Exact match
	rec, ok := FindByPath(root)
	if !ok {
		t.Fatal("expected exact match")
	}
	assert.Equal(t, StatusWorking, rec.Status)

	// Subdirectory should resolve to the ancestor's status
	rec, ok = FindByPath(sub)
	if !ok {
		t.Fatal("expected ancestor match for subdirectory")
	}
	assert.Equal(t, root, rec.CWD)
}

func TestFindByPathPrefersMostSpecificAncestor(t *testing.T) {
	withTempHome(t)
	root, _ := os.MkdirTemp("", "zgt-root-*")
	root, _ = filepath.EvalSymlinks(root)
	child := filepath.Join(root, "child")
	if err := os.MkdirAll(child, 0755); err != nil {
		t.Fatal(err)
	}

	if err := SetStatus(root, AgentClaude, StatusIdle, ""); err != nil {
		t.Fatal(err)
	}
	if err := SetStatus(child, AgentOpenCode, StatusWorking, ""); err != nil {
		t.Fatal(err)
	}

	// A path inside `child` should match `child` (longer CWD), not `root`.
	deep := filepath.Join(child, "file.go")
	rec, ok := FindByPath(deep)
	if !ok {
		t.Fatal("expected match")
	}
	assert.Equal(t, child, rec.CWD)
	assert.Equal(t, AgentOpenCode, rec.Agent)
}

func TestFindByPathReturnsFalseWhenStale(t *testing.T) {
	withTempHome(t)
	root, _ := os.MkdirTemp("", "zgt-root-*")
	root, _ = filepath.EvalSymlinks(root)

	if err := SetStatus(root, AgentClaude, StatusWorking, ""); err != nil {
		t.Fatal(err)
	}
	// Manually backdate the record beyond defaultStaleAge.
	rec, ok := GetStatus(root)
	if !ok {
		t.Fatal("expected record")
	}
	rec.UpdatedAt = time.Now().Add(-2 * time.Hour).Unix()
	data, _ := json.MarshalIndent(rec, "", "  ")
	p, _ := recordPath(root)
	if err := os.WriteFile(p, data, 0644); err != nil {
		t.Fatal(err)
	}

	if _, ok := FindByPath(root); ok {
		t.Fatal("expected stale record to be ignored")
	}
}

func TestFindByPathNoMatch(t *testing.T) {
	withTempHome(t)
	dir, _ := os.MkdirTemp("", "zgt-other-*")
	if _, ok := FindByPath(dir); ok {
		t.Fatal("expected no match in empty status dir")
	}
}

func TestListStatuses(t *testing.T) {
	withTempHome(t)
	a, _ := os.MkdirTemp("", "zgt-a-*")
	b, _ := os.MkdirTemp("", "zgt-b-*")

	if err := SetStatus(a, AgentClaude, StatusWorking, "1"); err != nil {
		t.Fatal(err)
	}
	if err := SetStatus(b, AgentOpenCode, StatusIdle, "2"); err != nil {
		t.Fatal(err)
	}

	all, err := ListStatuses()
	if err != nil {
		t.Fatal(err)
	}
	assert.Len(t, all, 2)

	agents := map[Agent]bool{}
	for _, r := range all {
		agents[r.Agent] = true
	}
	assert.True(t, agents[AgentClaude])
	assert.True(t, agents[AgentOpenCode])
}

func TestPruneStaleRemovesOldRecords(t *testing.T) {
	withTempHome(t)
	dir, _ := os.MkdirTemp("", "zgt-wt-*")
	if err := SetStatus(dir, AgentClaude, StatusWorking, ""); err != nil {
		t.Fatal(err)
	}
	// Backdate
	rec, _ := GetStatus(dir)
	rec.UpdatedAt = time.Now().Add(-3 * time.Hour).Unix()
	data, _ := json.MarshalIndent(rec, "", "  ")
	p, _ := recordPath(dir)
	_ = os.WriteFile(p, data, 0644)

	removed, err := PruneStale(time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	assert.Greater(t, removed, 0)
	if _, ok := GetStatus(dir); ok {
		t.Fatal("expected stale record to be pruned")
	}
}

func TestPruneStaleLeavesFreshRecords(t *testing.T) {
	withTempHome(t)
	dir, _ := os.MkdirTemp("", "zgt-wt-*")
	if err := SetStatus(dir, AgentClaude, StatusWorking, ""); err != nil {
		t.Fatal(err)
	}
	removed, err := PruneStale(time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 0, removed)
	if _, ok := GetStatus(dir); !ok {
		t.Fatal("fresh record should survive prune")
	}
}

func TestIsStale(t *testing.T) {
	stale := &Record{UpdatedAt: time.Now().Add(-2 * time.Hour).Unix()}
	fresh := &Record{UpdatedAt: time.Now().Unix()}
	assert.True(t, IsStale(stale))
	assert.False(t, IsStale(fresh))
	assert.True(t, IsStale(nil))
}

func TestKeyForPathStable(t *testing.T) {
	a := keyForPath("/foo/bar")
	b := keyForPath("/foo/bar")
	c := keyForPath("/foo/baz")
	assert.Equal(t, a, b)
	assert.NotEqual(t, a, c)
}

func TestParseStaleAge(t *testing.T) {
	assert.Equal(t, time.Hour, ParseStaleAge("1h"))
	assert.Equal(t, 30*time.Minute, ParseStaleAge("30m"))
	assert.Equal(t, 2*time.Hour+30*time.Minute, ParseStaleAge("2h30m"))
	// Empty -> default.
	assert.Equal(t, defaultStaleAge, ParseStaleAge(""))
	// Invalid -> default (never zero, never negative).
	assert.Equal(t, defaultStaleAge, ParseStaleAge("garbage"))
	assert.Equal(t, defaultStaleAge, ParseStaleAge("-5m"))
	assert.Equal(t, defaultStaleAge, ParseStaleAge("0s"))
}

func TestNormalize(t *testing.T) {
	abs := Normalize("/tmp")
	assert.True(t, strings.HasPrefix(abs, "/"))
	// A relative path resolves to an absolute one under the cwd.
	rel := Normalize(".")
	assert.True(t, strings.HasPrefix(rel, "/"))
}

func TestFindByPathRespectsCustomStaleAge(t *testing.T) {
	withTempHome(t)
	root, _ := os.MkdirTemp("", "zgt-staleage-*")
	root, _ = filepath.EvalSymlinks(root)
	if err := SetStatus(root, AgentClaude, StatusWorking, ""); err != nil {
		t.Fatal(err)
	}
	// Backdate by 10 minutes.
	rec, _ := GetStatus(root)
	rec.UpdatedAt = time.Now().Add(-10 * time.Minute).Unix()
	data, _ := json.MarshalIndent(rec, "", "  ")
	p, _ := recordPath(root)
	_ = os.WriteFile(p, data, 0644)

	// Default age (1h) -> not stale -> found.
	if _, ok := FindByPath(root); !ok {
		t.Fatal("expected record within default 1h age")
	}
	// Custom age of 5m -> record is stale -> not found.
	if _, ok := FindByPath(root, 5*time.Minute); ok {
		t.Fatal("expected record to be stale under 5m age")
	}
}

func TestRecordRoundTripJSON(t *testing.T) {
	rec := Record{
		Agent:     AgentClaude,
		Status:    StatusWaiting,
		CWD:       "/x",
		SessionID: "abc",
		UpdatedAt: 12345,
	}
	data, err := json.Marshal(&rec)
	if err != nil {
		t.Fatal(err)
	}
	var got Record
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, rec, got)
}
