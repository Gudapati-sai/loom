package render

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSlug(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"feature name", "feature-name"},
		{"Feature Name", "feature-name"},
		{"Feature_Name", "feature-name"},
		{"café", "caf"},
		{"日本語", "feature"},     // all non-ASCII -> fallback
		{"  spaced  ", "spaced"},
		{"---dashes---", "dashes"},
		{"UPPER", "upper"},
		{"123numbers", "123numbers"},
	}
	for _, tc := range tests {
		if got := slug(tc.input); got != tc.expected {
			t.Errorf("slug(%q) = %q, want %q", tc.input, got, tc.expected)
		}
	}
}

func TestFixedDestinations(t *testing.T) {
	dests := FixedDestinations()
	if len(dests) == 0 {
		t.Fatal("FixedDestinations() returned empty list")
	}
	expected := []string{
		"AGENTS.md",
		".agent/mcp.json.example",
		".agent/skills/documentation-driven-development/SKILL.md",
		".agent/skills/tdd-loop/SKILL.md",
		".agent/skills/anti-hallucination/SKILL.md",
		".agent/skills/prompt-loops/SKILL.md",
		".agent/skills/security-checklist/SKILL.md",
		".agent/skills/pr-review/SKILL.md",
		".ci/lint.yml",
		".ci/security-scan.yml",
		".ci/test.yml",
		"docs/prd/TEMPLATE.md",
		"docs/adr/TEMPLATE.md",
	}
	for _, exp := range expected {
		found := false
		for _, d := range dests {
			if d == exp {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected destination %q not found", exp)
		}
	}
}

func TestWriteFixed_SkipExisting(t *testing.T) {
	tmp := t.TempDir()
	// Pre-create a file that should be skipped
	existing := filepath.Join(tmp, "AGENTS.md")
	if err := os.WriteFile(existing, []byte("custom"), 0o644); err != nil {
		t.Fatal(err)
	}
	written, skipped, err := WriteFixed(tmp, false)
	if err != nil {
		t.Fatalf("WriteFixed returned error: %v", err)
	}
	// AGENTS.md should be skipped, not written
	var wroteAgents bool
	for _, w := range written {
		if w == "AGENTS.md" {
			wroteAgents = true
		}
	}
	if wroteAgents {
		t.Errorf("AGENTS.md was written but should have been skipped")
	}
	found := false
	for _, s := range skipped {
		if s == "AGENTS.md" {
			found = true
		}
	}
	if !found {
		t.Errorf("AGENTS.md not in skipped list")
	}
}