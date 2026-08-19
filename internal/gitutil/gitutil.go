package gitutil

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// RepoRoot walks up from cwd (or start) until .git is found.
func RepoRoot(start string) (string, error) {
	dir := start
	if dir == "" {
		var err error
		dir, err = os.Getwd()
		if err != nil {
			return "", err
		}
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("not a git repository")
		}
		dir = parent
	}
}

func runGit(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("git %s: %s", strings.Join(args, " "), msg)
	}
	return strings.TrimSpace(stdout.String()), nil
}

// HeadSHA returns the current HEAD commit hash.
// On rev-parse failure (e.g. empty repo), returns ("", err) — never zero SHA with nil error.
func HeadSHA(repoRoot string) (string, error) {
	out, err := runGit(repoRoot, "rev-parse", "HEAD")
	if err != nil {
		return "", err
	}
	return out, nil
}

// IsDirty reports uncommitted changes (fail-safe true on error).
func IsDirty(repoRoot string) bool {
	out, err := runGit(repoRoot, "status", "--porcelain")
	if err != nil {
		return true
	}
	return out != ""
}

// NotesRef is the write ref for attestation capsules.
const NotesRef = "refs/notes/curbpack"

// NotesShow returns note body for commit from curbpack notes, else legacy cyberready notes.
func NotesShow(repoRoot, commit string) (string, error) {
	out, err := runGit(repoRoot, "notes", "--ref=curbpack", "show", commit)
	if err == nil {
		return out, nil
	}
	out, err2 := runGit(repoRoot, "notes", "--ref=cyberready", "show", commit)
	if err2 != nil {
		return "", err
	}
	return out, nil
}

// NotesAdd writes (force) a note on commit under refs/notes/curbpack only.
func NotesAdd(repoRoot, commit, message string) error {
	_, err := runGit(repoRoot, "notes", "--ref=curbpack", "add", "-f", "-m", message, commit)
	return err
}

// ChangedFiles returns paths changed vs HEAD (staged + unstaged + untracked).
// Paths are slash-normalized relative to repo root.
func ChangedFiles(repoRoot string) (map[string]struct{}, error) {
	out := map[string]struct{}{}
	addLines := func(s string) {
		for _, line := range strings.Split(s, "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			// status --porcelain: XY PATH or XY ORIG -> PATH
			if len(line) >= 4 {
				rest := strings.TrimSpace(line[3:])
				if i := strings.Index(rest, " -> "); i >= 0 {
					rest = rest[i+4:]
				}
				rest = strings.Trim(rest, `"`)
				out[filepath.ToSlash(rest)] = struct{}{}
			}
		}
	}
	porcelain, err := runGit(repoRoot, "status", "--porcelain")
	if err != nil {
		return nil, err
	}
	addLines(porcelain)
	// Also include files differing from merge-base / HEAD for committed-but-local? porcelain covers WD.
	// For --diff against last commit content changes already staged:
	diff, err := runGit(repoRoot, "diff", "--name-only", "HEAD")
	if err == nil {
		for _, line := range strings.Split(diff, "\n") {
			line = strings.TrimSpace(line)
			if line != "" {
				out[filepath.ToSlash(line)] = struct{}{}
			}
		}
	}
	return out, nil
}

// DiffNameOnly returns slash-normalized paths that differ between fromRev and toRev.
// When pathspecs is non-empty, they are passed after `--` (git diff --name-only A B -- paths).
// Empty pathspecs returns nil (does not dump the whole tree). Results are sorted.
func DiffNameOnly(repoRoot, fromRev, toRev string, pathspecs []string) ([]string, error) {
	if strings.TrimSpace(fromRev) == "" || strings.TrimSpace(toRev) == "" {
		return nil, fmt.Errorf("git diff --name-only: empty revision")
	}
	if len(pathspecs) == 0 {
		return nil, nil
	}
	args := make([]string, 0, 5+len(pathspecs))
	args = append(args, "diff", "--name-only", fromRev, toRev, "--")
	args = append(args, pathspecs...)
	out, err := runGit(repoRoot, args...)
	if err != nil {
		return nil, err
	}
	var names []string
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		names = append(names, filepath.ToSlash(line))
	}
	sort.Strings(names)
	return names, nil
}

// parentNoteCapsule is the minimal JSON shape read from git notes for Merkle chaining.
type parentNoteCapsule struct {
	StateHash string `json:"state_hash"`
}

// ParentNoteHash reads the previous commit's note state_hash for Merkle chaining.
// It intentionally ignores any existing note on the current commit so re-attest is reproducible.
func ParentNoteHash(repoRoot, commit string) string {
	prev, err := runGit(repoRoot, "rev-parse", commit+"^")
	if err != nil {
		return ""
	}
	body, err := NotesShow(repoRoot, prev)
	if err != nil || body == "" {
		return ""
	}
	var cap parentNoteCapsule
	if json.Unmarshal([]byte(body), &cap) == nil && cap.StateHash != "" {
		return cap.StateHash
	}
	// Legacy string hack for pre-JSON notes.
	const key = `"state_hash"`
	idx := strings.Index(body, key)
	if idx < 0 {
		return ""
	}
	rest := body[idx+len(key):]
	q1 := strings.Index(rest, `"`)
	if q1 < 0 {
		return ""
	}
	rest = rest[q1+1:]
	q2 := strings.Index(rest, `"`)
	if q2 < 0 {
		return ""
	}
	return rest[:q2]
}

// LatestNoteCommit returns the most recent commit (from HEAD ancestry) that has a
// curbpack or legacy cyberready note. Do not use `git log -1 --notes=… --format=%H`:
// that returns HEAD even when HEAD has no note.
func LatestNoteCommit(repoRoot string) (string, error) {
	noted := map[string]struct{}{}
	for _, ref := range []string{"curbpack", "cyberready"} {
		out, err := runGit(repoRoot, "notes", "--ref="+ref, "list")
		if err != nil || out == "" {
			continue
		}
		for _, line := range strings.Split(out, "\n") {
			fields := strings.Fields(line)
			// notes list: <note-object> <commit>
			if len(fields) >= 2 {
				noted[fields[len(fields)-1]] = struct{}{}
			}
		}
	}
	if len(noted) == 0 {
		return "", fmt.Errorf("no attest notes found")
	}
	revList, err := runGit(repoRoot, "rev-list", "HEAD")
	if err != nil {
		return "", err
	}
	for _, commit := range strings.Split(revList, "\n") {
		commit = strings.TrimSpace(commit)
		if commit == "" {
			continue
		}
		if _, ok := noted[commit]; ok {
			// Dual-read show confirms the note body is readable.
			if _, err := NotesShow(repoRoot, commit); err == nil {
				return commit, nil
			}
		}
	}
	return "", fmt.Errorf("no attest notes found")
}

// FileCommitMeta is the latest commit touching path (git log -1).
type FileCommitMeta struct {
	Hash  string
	Email string
	Name  string
	Time  time.Time
}

// FileLastCommit returns metadata for the most recent commit that touched rel path.
func FileLastCommit(repoRoot, rel string) (FileCommitMeta, error) {
	rel = filepath.ToSlash(strings.TrimSpace(rel))
	if rel == "" {
		return FileCommitMeta{}, fmt.Errorf("empty path")
	}
	out, err := runGit(repoRoot, "log", "-1", "--format="+`%H%x1f%ae%x1f%an%x1f%ct`, "--", rel)
	if err != nil {
		return FileCommitMeta{}, err
	}
	if out == "" {
		return FileCommitMeta{}, fmt.Errorf("no commits for %s", rel)
	}
	parts := strings.Split(out, "\x1f")
	if len(parts) < 4 {
		return FileCommitMeta{}, fmt.Errorf("unexpected git log format for %s", rel)
	}
	var sec int64
	fmt.Sscanf(parts[3], "%d", &sec)
	return FileCommitMeta{
		Hash:  parts[0],
		Email: parts[1],
		Name:  parts[2],
		Time:  time.Unix(sec, 0).UTC(),
	}, nil
}

// FileTouchedSinceRef reports whether rel has a commit after sinceRef (exclusive..HEAD].
func FileTouchedSinceRef(repoRoot, sinceRef, rel string) (bool, error) {
	sinceRef = strings.TrimSpace(sinceRef)
	if sinceRef == "" {
		return false, fmt.Errorf("empty since_ref")
	}
	out, err := runGit(repoRoot, "log", "-1", "--format=%H", sinceRef+"..HEAD", "--", rel)
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(out) != "", nil
}
