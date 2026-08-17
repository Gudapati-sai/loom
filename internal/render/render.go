package render

import (
	"embed"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

//go:embed templates
var files embed.FS

// TemplateData is the full set of variables any fixed or dynamic template
// might reference.
type TemplateData struct {
	ProjectName string
	Stack       string
	Date        string
	FeatureName string
	Problem     string
	Criterion   string
}

type fileMapping struct{ src, dst string }

// fixedManifest maps embedded source paths to their destination inside a
// target project. Source paths deliberately avoid dot-directories (Go's
// embed skips them); the dot-prefixed destination is applied at write time.
var fixedManifest = []fileMapping{
	{"templates/AGENTS.md", "AGENTS.md"},
	{"templates/mcp.json.example", ".agent/mcp.json.example"},
	{"templates/skills/documentation-driven-development.md", ".agent/skills/documentation-driven-development/SKILL.md"},
	{"templates/skills/tdd-loop.md", ".agent/skills/tdd-loop/SKILL.md"},
	{"templates/skills/anti-hallucination.md", ".agent/skills/anti-hallucination/SKILL.md"},
	{"templates/skills/prompt-loops.md", ".agent/skills/prompt-loops/SKILL.md"},
	{"templates/skills/security-checklist.md", ".agent/skills/security-checklist/SKILL.md"},
	{"templates/skills/pr-review.md", ".agent/skills/pr-review/SKILL.md"},
	{"templates/ci/lint.yml", ".ci/lint.yml"},
	{"templates/ci/security-scan.yml", ".ci/security-scan.yml"},
	{"templates/ci/test.yml", ".ci/test.yml"},
	{"templates/prd-template.md", "docs/prd/TEMPLATE.md"},
	{"templates/adr-template.md", "docs/adr/TEMPLATE.md"},
}

// FixedDestinations returns every destination path the fixed-file set
// writes, so gate checks and retrofit's conflict scan can use the same
// list the renderer does.
func FixedDestinations() []string {
	out := make([]string, len(fixedManifest))
	for i, m := range fixedManifest {
		out[i] = m.dst
	}
	return out
}

// PlanFixed reports, without writing anything, which fixed files would be
// newly written and which already exist and would be left alone — used by
// plan mode to show what's about to happen before it happens.
func PlanFixed(targetDir string) (toWrite, existing []string) {
	for _, m := range fixedManifest {
		if _, err := os.Stat(filepath.Join(targetDir, m.dst)); err == nil {
			existing = append(existing, m.dst)
		} else {
			toWrite = append(toWrite, m.dst)
		}
	}
	return toWrite, existing
}

// WriteFixed copies every fixed-file template verbatim into targetDir.
// Existing files are skipped (not overwritten) unless overwrite is true —
// this is what lets retrofit mode ask before clobbering anything.
func WriteFixed(targetDir string, overwrite bool) (written, skipped []string, err error) {
	for _, m := range fixedManifest {
		data, rerr := files.ReadFile(m.src)
		if rerr != nil {
			return written, skipped, rerr
		}
		dst := filepath.Join(targetDir, m.dst)
		if !overwrite {
			if _, statErr := os.Stat(dst); statErr == nil {
				skipped = append(skipped, m.dst)
				continue
			}
		}
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return written, skipped, err
		}
		if err := os.WriteFile(dst, data, 0o644); err != nil {
			return written, skipped, err
		}
		written = append(written, m.dst)
	}
	return written, skipped, nil
}

// WriteOne writes a single fixed file by destination path, used by
// retrofit's per-file keep/replace flow.
func WriteOne(targetDir, dst string) error {
	for _, m := range fixedManifest {
		if m.dst != dst {
			continue
		}
		data, err := files.ReadFile(m.src)
		if err != nil {
			return err
		}
		full := filepath.Join(targetDir, dst)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			return err
		}
		return os.WriteFile(full, data, 0o644)
	}
	return nil
}

// RenderReadme renders the one fixed file that genuinely varies per
// project.
func RenderReadme(targetDir string, data TemplateData) error {
	return renderTemplate(targetDir, "templates/readme.md.tmpl", "README.md", data)
}

// RenderPRD drafts the first feature's PRD from the wizard's answers and
// returns its path relative to targetDir.
func RenderPRD(targetDir string, data TemplateData) (string, error) {
	dst := filepath.Join("docs", "prd", data.Date+"-"+slug(data.FeatureName)+".md")
	if err := renderTemplate(targetDir, "templates/prd-stub.md.tmpl", dst, data); err != nil {
		return "", err
	}
	return dst, nil
}

func renderTemplate(targetDir, src, dstRel string, data TemplateData) error {
	raw, err := files.ReadFile(src)
	if err != nil {
		return err
	}
	tmpl, err := template.New(dstRel).Parse(string(raw))
	if err != nil {
		return err
	}
	dst := filepath.Join(targetDir, dstRel)
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	f, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer f.Close()
	return tmpl.Execute(f, data)
}

func slug(s string) string {
	out := make([]rune, 0, len(s))
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			out = append(out, r)
		case r >= 'A' && r <= 'Z':
			out = append(out, r+32)
		case r == ' ' || r == '_' || r == '-':
			out = append(out, '-')
		}
	}
	// Trim stray dashes and guard the degenerate case: a feature name with
	// no ASCII characters ("café", "日本語") must not produce a filename
	// like 2026-08-17-.md or a bare date.
	res := strings.Trim(string(out), "-")
	if res == "" {
		return "feature"
	}
	return res
}
