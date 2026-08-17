package brand

import (
	"testing"
)

func TestBannerContainsVersion(t *testing.T) {
	b := Banner()
	if b == "" {
		t.Fatal("Banner() returned empty string")
	}
	// banner should contain the version string
	if !contains(b, Version) {
		t.Errorf("Banner() missing version %q", Version)
	}
	// should contain the LOOM logo
	if !contains(b, "LOOM") && !contains(b, "██") {
		t.Errorf("Banner() missing logo art")
	}
}

func TestSmallContainsVersion(t *testing.T) {
	s := Small()
	if !contains(s, Version) {
		t.Errorf("Small() missing version %q", Version)
	}
	if !contains(s, "loom") {
		t.Errorf("Small() missing name")
	}
}

func contains(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}