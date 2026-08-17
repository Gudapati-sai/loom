package provenance

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Gudapati-sai/loom/internal/phase"
)

// orderedKeys puts the standard flow's questions in a sensible reading
// order; anything else recorded in state (e.g. retrofit conflicts) is
// appended afterward, sorted for determinism.
var orderedKeys = []string{"name", "stack", "feature_name", "feature_problem", "feature_criteria"}

// WriteFromState renders the full answer-provenance transcript from a
// State's recorded answers — every question asked, what was chosen, and
// why. It's regenerated from state each time, so it survives `resume`
// without needing a separate append-only log to manage.
func WriteFromState(st *phase.State, targetDir string) error {
	header := fmt.Sprintf(
		"# Loom setup log\n\nGenerated %s for **%s**. Every question asked during setup, the answer given, "+
			"and the reasoning shown at the time — kept so the choices behind this project's shape aren't lost.\n\n---\n",
		time.Now().Format("2006-01-02 15:04"), st.ProjectName,
	)

	written := map[string]bool{}
	var entries []string
	appendEntry := func(k string) {
		q, ok := st.Questions[k]
		if !ok || written[k] {
			return
		}
		written[k] = true
		e := fmt.Sprintf("\n### %s\n\n**Answer:** %s\n", q, st.Answers[k])
		if exp := st.Explanations[k]; exp != "" {
			e += fmt.Sprintf("\n> %s\n", exp)
		}
		entries = append(entries, e)
	}

	for _, k := range orderedKeys {
		appendEntry(k)
	}
	var remaining []string
	for k := range st.Questions {
		if !written[k] {
			remaining = append(remaining, k)
		}
	}
	sort.Strings(remaining)
	for _, k := range remaining {
		appendEntry(k)
	}

	body := header + strings.Join(entries, "\n---\n")
	return os.WriteFile(filepath.Join(targetDir, ".loom-log.md"), []byte(body), 0o644)
}
