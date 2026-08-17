// Package trace prints Claude-Code-style step lines as the wizard works,
// and keeps a local, append-only record of the same entries in the
// target project. Nothing here is ever transmitted anywhere — it's a
// diagnostic trail for the person running loom, not analytics.
package trace

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/charmbracelet/lipgloss"
)

type Level string

const (
	LevelStep  Level = "step"
	LevelDone  Level = "done"
	LevelWarn  Level = "warn"
	LevelError Level = "error"
	LevelPlan  Level = "plan"
)

// Entry is one line of the trace, as written to .loom-trace.jsonl.
type Entry struct {
	Time    time.Time `json:"time"`
	Level   Level     `json:"level"`
	Phase   string    `json:"phase"`
	Message string    `json:"message"`
}

var (
	stepStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("212"))
	doneStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	warnStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	errStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	dimStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
)

// Tracer accumulates entries in memory and prints each one immediately;
// call Flush to append them to .loom-trace.jsonl in the target project.
type Tracer struct {
	targetDir string
	phase     string
	entries   []Entry
}

func New(targetDir string) *Tracer {
	return &Tracer{targetDir: targetDir}
}

// SetPhase tags every subsequent entry until the next SetPhase call, so
// the trace file can be filtered/grouped by phase later.
func (t *Tracer) SetPhase(phase string) { t.phase = phase }

func (t *Tracer) record(level Level, msg string) {
	t.entries = append(t.entries, Entry{Time: time.Now(), Level: level, Phase: t.phase, Message: msg})
}

// Step marks an action about to happen.
func (t *Tracer) Step(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	fmt.Println(stepStyle.Render("●") + " " + msg)
	t.record(LevelStep, msg)
}

// Done marks an action that completed successfully.
func (t *Tracer) Done(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	fmt.Println(doneStyle.Render("✓") + " " + msg)
	t.record(LevelDone, msg)
}

// Warn marks something worth noticing that isn't fatal.
func (t *Tracer) Warn(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	fmt.Println(warnStyle.Render("⚠") + " " + msg)
	t.record(LevelWarn, msg)
}

// Error marks a failed action.
func (t *Tracer) Error(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	fmt.Println(errStyle.Render("✗") + " " + msg)
	t.record(LevelError, msg)
}

// Plan marks a proposed action in plan mode — dimmed and prefixed
// differently from Step so it reads as "would happen," not "is happening."
func (t *Tracer) Plan(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	fmt.Println(dimStyle.Render("  · " + msg))
	t.record(LevelPlan, msg)
}

// Flush appends every recorded entry to .loom-trace.jsonl, one JSON
// object per line, then clears the in-memory buffer. Safe to call more
// than once per run.
func (t *Tracer) Flush() error {
	if len(t.entries) == 0 {
		return nil
	}
	if err := os.MkdirAll(t.targetDir, 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(filepath.Join(t.targetDir, ".loom-trace.jsonl"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	for _, e := range t.entries {
		if err := enc.Encode(e); err != nil {
			return err
		}
	}
	t.entries = nil
	return nil
}
