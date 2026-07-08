package agentstatus

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// ParseStaleAge parses a human-friendly duration string (e.g. "1h", "30m",
// "2h30m") into a time.Duration. Empty input falls back to defaultStaleAge.
// Invalid input also falls back to defaultStaleAge so a malformed config
// never breaks status lookups.
func ParseStaleAge(s string) time.Duration {
	if strings.TrimSpace(s) == "" {
		return defaultStaleAge
	}
	d, err := time.ParseDuration(s)
	if err != nil || d <= 0 {
		return defaultStaleAge
	}
	return d
}

// Agent identifies which coding agent wrote a status entry.
type Agent string

const (
	AgentClaude   Agent = "claude"
	AgentOpenCode Agent = "opencode"
)

// Status values describe what the agent is currently doing for a worktree.
type Status string

const (
	StatusWorking Status = "working" // agent is processing a prompt / running tools
	StatusIdle    Status = "idle"    // agent finished a turn and is awaiting input
	StatusWaiting Status = "waiting" // agent is blocked on a permission / question
)

// Record is the on-disk representation of an agent's status for one CWD.
type Record struct {
	Agent     Agent  `json:"agent"`
	Status    Status `json:"status"`
	CWD       string `json:"cwd"`
	SessionID string `json:"session_id,omitempty"`
	UpdatedAt int64  `json:"updated_at"` // unix seconds
}

// defaultStaleAge is used when no explicit age is provided to PruneStale /
// IsStale. Agent status files are updated on every turn, so anything older
// than an hour almost certainly reflects a crashed / killed session.
const defaultStaleAge = time.Hour

// statusDir returns the directory used to store per-CWD status files.
func statusDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "zgt", "agent-status"), nil
}

// keyForPath hashes a normalized absolute path into a stable filename.
// Filenames must be filesystem-safe; SHA-1 of the normalized path keeps the
// directory listing bounded regardless of path length or special chars.
func keyForPath(path string) string {
	h := sha1.Sum([]byte(path))
	return hex.EncodeToString(h[:])
}

func recordPath(path string) (string, error) {
	dir, err := statusDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, keyForPath(path)+".json"), nil
}

// SetStatus writes (atomically) a status record for the given CWD. The path is
// normalized before hashing so symlink-variant paths resolve to the same file.
func SetStatus(cwd string, agent Agent, status Status, sessionID string) error {
	norm, err := normalize(cwd)
	if err != nil {
		return err
	}
	rec := Record{
		Agent:     agent,
		Status:    status,
		CWD:       norm,
		SessionID: sessionID,
		UpdatedAt: time.Now().Unix(),
	}
	data, err := json.MarshalIndent(&rec, "", "  ")
	if err != nil {
		return err
	}

	dir, err := statusDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	final, err := recordPath(norm)
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".agent-status-*.json")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }() // no-op if rename succeeded

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpName, final); err != nil {
		return err
	}
	return nil
}

// ClearStatus removes the status record for the given CWD, if any. Missing
// files are not an error.
func ClearStatus(cwd string) error {
	norm, err := normalize(cwd)
	if err != nil {
		return err
	}
	p, err := recordPath(norm)
	if err != nil {
		return err
	}
	if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// GetStatus reads the status record for the exact (normalized) CWD. Returns
// (nil, false) when no record exists. The caller is responsible for stale
// checks via IsStale.
func GetStatus(cwd string) (*Record, bool) {
	norm, err := normalize(cwd)
	if err != nil {
		return nil, false
	}
	p, err := recordPath(norm)
	if err != nil {
		return nil, false
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return nil, false
	}
	var rec Record
	if err := json.Unmarshal(data, &rec); err != nil {
		return nil, false
	}
	return &rec, true
}

// FindByPath returns the most specific status record whose CWD contains the
// given path. A pane CWD is usually a subdirectory of a worktree root, so we
// match the longest stored CWD that is an ancestor of (or equal to) the
// target. Returns (nil, false) when nothing matches or the best match is
// stale. Pass an optional age to override the default stale threshold
// (e.g. from config); otherwise defaultStaleAge is used.
func FindByPath(path string, ages ...time.Duration) (*Record, bool) {
	target, err := normalize(path)
	if err != nil {
		return nil, false
	}
	all, err := ListStatuses()
	if err != nil {
		return nil, false
	}

	var matches []*Record
	for _, r := range all {
		rel, err := filepath.Rel(r.CWD, target)
		if err != nil {
			continue
		}
		// rel == "." means target == r.CWD; otherwise rel must not escape
		// r.CWD (no ".." segment), which makes target a descendant.
		if rel != "." && pathEscapes(rel) {
			continue
		}
		matches = append(matches, r)
	}
	if len(matches) == 0 {
		return nil, false
	}

	// Prefer the longest CWD (most specific ancestor). On a tie, prefer the
	// newest UpdatedAt. If the best match is stale, treat as not found so the
	// UI doesn't show a stale "working" badge forever.
	sort.SliceStable(matches, func(i, j int) bool {
		if len(matches[i].CWD) != len(matches[j].CWD) {
			return len(matches[i].CWD) > len(matches[j].CWD)
		}
		return matches[i].UpdatedAt > matches[j].UpdatedAt
	})
	best := matches[0]
	if IsStale(best, ages...) {
		return nil, false
	}
	return best, true
}

// pathEscapes returns true if rel contains a ".." segment, which would mean
// the target path is not actually underneath the candidate CWD.
func pathEscapes(rel string) bool {
	parts := strings.Split(rel, string(filepath.Separator))
	for _, p := range parts {
		if p == ".." {
			return true
		}
	}
	return false
}

// ListStatuses reads every status record in the status directory. Returns an
// empty slice when the directory is missing or unreadable.
func ListStatuses() ([]*Record, error) {
	dir, err := statusDir()
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []*Record
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		var rec Record
		if err := json.Unmarshal(data, &rec); err != nil {
			continue
		}
		out = append(out, &rec)
	}
	return out, nil
}

// IsStale reports whether a record is older than the given age. A zero age
// falls back to defaultStaleAge.
func IsStale(rec *Record, ages ...time.Duration) bool {
	if rec == nil {
		return true
	}
	age := defaultStaleAge
	if len(ages) > 0 && ages[0] > 0 {
		age = ages[0]
	}
	return time.Since(time.Unix(rec.UpdatedAt, 0)) > age
}

// PruneStale removes status files whose UpdatedAt is older than maxAge. A
// zero maxAge falls back to defaultStaleAge. Returns the count of removed
// files and any error encountered.
func PruneStale(maxAge time.Duration) (int, error) {
	if maxAge <= 0 {
		maxAge = defaultStaleAge
	}
	dir, err := statusDir()
	if err != nil {
		return 0, err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}
	removed := 0
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		var rec Record
		if err := json.Unmarshal(data, &rec); err != nil {
			// Corrupt file: remove it so it doesn't poison listings.
			if err := os.Remove(filepath.Join(dir, e.Name())); err == nil {
				removed++
			}
			continue
		}
		if time.Since(time.Unix(rec.UpdatedAt, 0)) > maxAge {
			if err := os.Remove(filepath.Join(dir, e.Name())); err == nil {
				removed++
			}
		}
	}
	return removed, nil
}

// normalize returns an absolute, symlink-evaluated path. It mirrors
// state.NormalizePath so that status files line up with state entries.
func normalize(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve abs path: %v", err)
	}
	eval, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return abs, nil // symlink eval failure is non-fatal
	}
	return eval, nil
}

// Normalize is the exported form of normalize for callers outside the
// package (e.g. cmd) that need to match the same path key used for storage.
func Normalize(path string) string {
	n, err := normalize(path)
	if err != nil {
		return path
	}
	return n
}
