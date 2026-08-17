package phase

import (
	"os"
	"path/filepath"
	"strings"
)

// GateResult reports whether a phase's automated check passed. Nothing
// advances past a phase on the agent's say-so alone — this is what makes
// the check real instead of narrated.
type GateResult struct {
	Passed bool
	Reason string
}

// CheckFeatureGate verifies the drafted PRD has real content before the
// wizard treats the feature phase as complete.
func CheckFeatureGate(prdPath string) GateResult {
	data, err := os.ReadFile(prdPath)
	if err != nil {
		return GateResult{false, "PRD file not found: " + err.Error()}
	}
	content := string(data)

	problem := sectionBody(content, "## Problem", "## Constraints")
	if strings.TrimSpace(problem) == "" {
		return GateResult{false, "Problem statement is empty"}
	}

	criteria := sectionBody(content, "## Acceptance criteria", "## Open questions")
	trimmed := strings.TrimSpace(criteria)
	if !strings.HasPrefix(trimmed, "1.") {
		return GateResult{false, "No acceptance criteria found"}
	}

	return GateResult{true, "PRD has a problem statement and at least one acceptance criterion"}
}

func sectionBody(content, start, end string) string {
	si := strings.Index(content, start)
	if si == -1 {
		return ""
	}
	rest := content[si+len(start):]
	if ei := strings.Index(rest, end); ei != -1 {
		return rest[:ei]
	}
	return rest
}

// CheckScaffoldGate verifies every fixed file was actually written to disk.
func CheckScaffoldGate(targetDir string, files []string) GateResult {
	for _, f := range files {
		if _, err := os.Stat(filepath.Join(targetDir, f)); err != nil {
			return GateResult{false, "missing: " + f}
		}
	}
	return GateResult{true, "all fixed files present"}
}
