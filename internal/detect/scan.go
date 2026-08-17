package detect

import (
	"os"
	"path/filepath"
)

// Report summarizes what already exists in a repo being retrofitted, so
// the wizard only asks about what it genuinely can't infer.
type Report struct {
	Stack         string
	HasGit        bool
	HasReadme     bool
	ExistingFixed []string
}

var stackMarkers = []struct{ Marker, Stack string }{
	{"go.mod", "Go"},
	{"package.json", "TypeScript/Node"},
	{"pyproject.toml", "Python"},
	{"requirements.txt", "Python"},
	{"Cargo.toml", "Rust"},
	{"pubspec.yaml", "Dart/Flutter"},
}

func Scan(dir string, fixedDestinations []string) (Report, error) {
	var r Report
	if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
		r.HasGit = true
	}
	if _, err := os.Stat(filepath.Join(dir, "README.md")); err == nil {
		r.HasReadme = true
	}
	for _, sm := range stackMarkers {
		if _, err := os.Stat(filepath.Join(dir, sm.Marker)); err == nil {
			r.Stack = sm.Stack
			break
		}
	}
	for _, dst := range fixedDestinations {
		if _, err := os.Stat(filepath.Join(dir, dst)); err == nil {
			r.ExistingFixed = append(r.ExistingFixed, dst)
		}
	}
	return r, nil
}
