package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/afelin/cyberready/internal/instrument"
)

// priorCacheSnapshot is the quiet accumulation whisper source (pre-overwrite).
type priorCacheSnapshot struct {
	OK             bool
	ReadinessScore int
	FailureCount   int
}

func loadPriorCache(root string) priorCacheSnapshot {
	path := filepath.Join(root, ".github", "cyberready", "cache", "latest_result.json")
	b, err := os.ReadFile(path)
	if err != nil {
		// Fall back to latest_failure.json (same payload shape).
		path = filepath.Join(root, ".github", "cyberready", "cache", "latest_failure.json")
		b, err = os.ReadFile(path)
		if err != nil {
			return priorCacheSnapshot{}
		}
	}
	var raw struct {
		ReadinessScore int               `json:"readiness_score"`
		Failures       []json.RawMessage `json:"failures"`
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		return priorCacheSnapshot{}
	}
	n := 0
	if raw.Failures != nil {
		n = len(raw.Failures)
	}
	return priorCacheSnapshot{
		OK:             true,
		ReadinessScore: raw.ReadinessScore,
		FailureCount:   n,
	}
}

// accumulationDeltaLine returns at most one quiet line when prior cache exists.
// Empty string when there is no prior evidence deposit to whisper about.
func accumulationDeltaLine(prior priorCacheSnapshot, nowScore int) string {
	if !prior.OK {
		return ""
	}
	if prior.ReadinessScore != nowScore {
		return fmt.Sprintf("Δ readiness %d→%d · evidence cache updated", prior.ReadinessScore, nowScore)
	}
	return "gates green · evidence cache updated"
}

// instrumentWhisperLines returns at most 3 dim instrument lines (readiness / deps / secrets).
// First run (no prior instrument): readiness line only when prior IR cache exists; deps/secrets quiet.
func instrumentWhisperLines(priorCache priorCacheSnapshot, priorInst instrument.Snapshot, priorOK bool, nowScore int, nowInst instrument.Snapshot) []string {
	var lines []string
	if line := accumulationDeltaLine(priorCache, nowScore); line != "" {
		lines = append(lines, line)
	}
	if !priorOK {
		return lines
	}
	if priorInst.DepsFP != nowInst.DepsFP {
		add, rem := instrument.DepDelta(priorInst, nowInst)
		lines = append(lines, fmt.Sprintf("Δ deps +%d/−%d", add, rem))
	}
	if priorInst.SecretHits != nowInst.SecretHits {
		lines = append(lines, fmt.Sprintf("Δ secret-hits %d→%d", priorInst.SecretHits, nowInst.SecretHits))
	}
	if len(lines) > 3 {
		lines = lines[:3]
	}
	return lines
}
