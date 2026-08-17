// Package wizard sequences phases for `loom new` and `loom retrofit`. It is
// deliberately plain Go, not the LLM — the model only ever fills in
// question wording within a phase the wizard already decided to run.
package wizard

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Gudapati-sai/loom/internal/backend"
	"github.com/Gudapati-sai/loom/internal/detect"
	"github.com/Gudapati-sai/loom/internal/llm"
	"github.com/Gudapati-sai/loom/internal/phase"
	"github.com/Gudapati-sai/loom/internal/provenance"
	"github.com/Gudapati-sai/loom/internal/render"
	"github.com/Gudapati-sai/loom/internal/trace"
	"github.com/Gudapati-sai/loom/internal/tui"
)

// askQuestion tries the local model, retries once, then falls back to the
// static question set — the model is an enhancement the wizard can never
// get stuck waiting on.
func askQuestion(ctx context.Context, client *llm.Client, systemCtx, phaseKey string) llm.Question {
	if client != nil {
		if q, err := client.AskQuestion(ctx, systemCtx, phaseKey); err == nil {
			return q
		}
		// Brief pause before the retry: a freshly launched backend may still
		// be settling even after the /v1/models probe succeeds.
		time.Sleep(300 * time.Millisecond)
		if q, err := client.AskQuestion(ctx, systemCtx, phaseKey); err == nil {
			return q
		}
	}
	if q, ok := llm.Static[phaseKey]; ok {
		return q
	}
	// A phase key with no static entry must never yield a zero-value
	// Question (blank prompt, no options) — that would render as an empty
	// screen. Fall back to a plain free-text prompt instead.
	return llm.Question{Prompt: "Please answer:", FreeText: true}
}

func toTUIOptions(opts []llm.Option) []tui.Option {
	out := make([]tui.Option, len(opts))
	for i, o := range opts {
		out[i] = tui.Option{Label: o.Label, Explanation: o.Explanation}
	}
	return out
}

// ask runs the right screen for a Question — a select list if the model
// (or the static set) actually provided options, plain text otherwise.
// Earlier builds always called RunText here, so a model response with
// real options was silently discarded in favor of a generic text box.
func ask(q llm.Question) (value, explanation string, err error) {
	if !q.FreeText && len(q.Options) > 0 {
		return tui.RunSelect(q.Prompt, toTUIOptions(q.Options), q.AllowCustom)
	}
	v, err := tui.RunText(q.Prompt)
	return v, "", err
}

// contextSummary builds a "here's what we know so far" string from prior
// answers. Without this, every prompt sent to the model was generic and
// context-blind — it couldn't reference the project name, stack, or
// feature even when a real model was answering, which is a big part of
// why the flow felt fixed regardless of whether a model was connected.
func contextSummary(st *phase.State) string {
	var parts []string
	for _, k := range []struct{ key, label string }{
		{"name", "project name"},
		{"stack", "stack"},
		{"feature_name", "feature being built"},
		{"feature_problem", "problem it solves"},
	} {
		if v, ok := st.Answers[k.key]; ok && v != "" {
			parts = append(parts, k.label+": "+v)
		}
	}
	if len(parts) == 0 {
		return "(nothing answered yet)"
	}
	return strings.Join(parts, "; ")
}

// checkClient makes a backend available and reports the outcome through
// the tracer either way — silently falling back with no explanation is
// exactly the kind of thing that makes a working local model look broken.
// With a launcher configured this is attach-or-launch: attach to a running
// backend, otherwise spawn it and wait for it to answer. With no launcher
// it degrades to the old one-shot probe.
func checkClient(tr *trace.Tracer, client *llm.Client, l *backend.Launcher) *llm.Client {
	if client == nil {
		return nil
	}
	tr.Step("checking for a local model at %s", client.BaseURL)
	if l == nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		ok, reason := client.Available(ctx)
		if !ok {
			tr.Warn("no local model reachable (%s) — using the built-in question set", reason)
			return nil
		}
		tr.Done("local model available at %s (model: %s)", client.BaseURL, client.Model)
		return client
	}
	status, err := l.Ensure(context.Background(), tr.Step)
	if err != nil {
		tr.Warn("%s", err)
		return nil
	}
	tr.Done("%s (model: %s)", status, client.Model)
	return client
}

// recall returns a previously-saved answer for key, or asks for it and
// saves it. This is what makes `loom resume` skip already-answered
// questions instead of asking again.
func recall(st *phase.State, key string, fn func() (value, question, explanation string, err error)) (string, error) {
	if v, ok := st.Answers[key]; ok {
		return v, nil
	}
	v, q, e, err := fn()
	if err != nil {
		return "", err
	}
	st.Answers[key] = v
	st.Questions[key] = q
	st.Explanations[key] = e
	_ = st.Save()
	return v, nil
}

// confirmPlan shows the files a bulk write is about to touch and asks
// before anything is written. Only used when planMode is on.
func confirmPlan(tr *trace.Tracer, toWrite, existing []string) (bool, error) {
	tr.Plan("would write %d fixed files", len(toWrite))
	for _, f := range toWrite {
		tr.Plan("+ %s", f)
	}
	if len(existing) > 0 {
		tr.Plan("%d file(s) already exist and would be left alone: %v", len(existing), existing)
	}
	choice, _, err := tui.RunSelect("Proceed with this plan?", []tui.Option{
		{Label: "Yes, write these files", Explanation: "Apply the plan now."},
		{Label: "No, stop here", Explanation: "Exit without writing anything."},
	}, false)
	if err != nil {
		return false, err
	}
	return choice == "Yes, write these files", nil
}

// RunNew scaffolds a brand-new project, or continues one already in
// progress in targetDir. The file plan is always shown in the TUI and
// confirmed before anything touches disk.
func RunNew(targetDir string, client *llm.Client, launcher *backend.Launcher) error {
	ctx := context.Background()
	tr := trace.New(targetDir)
	defer func() {
		if err := tr.Flush(); err != nil {
			fmt.Fprintln(os.Stderr, "warning: could not write trace file:", err)
		}
	}()

	client = checkClient(tr, client, launcher)

	st, err := phase.Load(targetDir)
	if err != nil {
		st = phase.NewState("new", "", targetDir)
	} else {
		tr.Step("resuming from phase: %s", st.Phase)
	}

	data := render.TemplateData{Date: time.Now().Format("2006-01-02")}

	tr.SetPhase("welcome")
	name, err := recall(st, "name", func() (string, string, string, error) {
		q := llm.Static["welcome_name"].Prompt
		v, err := tui.RunText(q)
		return v, q, "", err
	})
	if err != nil {
		return err
	}
	st.ProjectName = name
	data.ProjectName = name
	st.Phase = "stack"
	_ = st.Save()

	tr.SetPhase("stack")
	stack, err := recall(st, "stack", func() (string, string, string, error) {
		q := askQuestion(ctx, client, fmt.Sprintf(
			"Context so far: %s. Ask what primary language/stack this project should use, "+
				"with a genuinely useful trade-off explanation for each option — not generic filler.",
			contextSummary(st),
		), "stack")
		v, e, err := ask(q)
		return v, q.Prompt, e, err
	})
	if err != nil {
		return err
	}
	data.Stack = stack
	st.Phase = "scaffold"
	_ = st.Save()

	tr.SetPhase("scaffold")
	if _, ok := st.Answers["scaffolded"]; !ok {
		toWrite, existing := render.PlanFixed(targetDir)
		proceed, err := confirmPlan(tr, toWrite, existing)
		if err != nil {
			return err
		}
		if !proceed {
			tr.Warn("stopped before scaffolding — nothing was written")
			return nil
		}

		tr.Step("writing fixed files to %s", targetDir)
		written, _, err := render.WriteFixed(targetDir, false)
		if err != nil {
			tr.Error("scaffold failed: %v", err)
			return err
		}
		if err := render.RenderReadme(targetDir, data); err != nil {
			return err
		}
		tr.Done("wrote %d fixed files", len(written))

		gate := phase.CheckScaffoldGate(targetDir, render.FixedDestinations())
		if !gate.Passed {
			tr.Error("scaffold gate failed: %s", gate.Reason)
			return fmt.Errorf("scaffold gate failed: %s", gate.Reason)
		}
		tr.Done("gate passed: %s", gate.Reason)
		st.Answers["scaffolded"] = "true"
		_ = st.Save()
	}
	st.Phase = "feature"
	_ = st.Save()

	tr.SetPhase("feature")
	featureName, err := recall(st, "feature_name", func() (string, string, string, error) {
		q := askQuestion(ctx, client, fmt.Sprintf(
			"Context so far: %s. Ask for the name of the first feature to build.",
			contextSummary(st),
		), "feature_name")
		v, e, err := ask(q)
		return v, q.Prompt, e, err
	})
	if err != nil {
		return err
	}
	data.FeatureName = featureName

	problem, err := recall(st, "feature_problem", func() (string, string, string, error) {
		q := askQuestion(ctx, client, fmt.Sprintf(
			"Context so far: %s. Ask what problem this feature solves, in 1-2 sentences.",
			contextSummary(st),
		), "feature_problem")
		v, e, err := ask(q)
		return v, q.Prompt, e, err
	})
	if err != nil {
		return err
	}
	data.Problem = problem

	criterion, err := recall(st, "feature_criteria", func() (string, string, string, error) {
		q := askQuestion(ctx, client, fmt.Sprintf(
			"Context so far: %s. Propose 3-4 concrete, testable acceptance criteria as selectable "+
				"options, each genuinely grounded in the stated problem — not generic placeholders — "+
				"with allow_custom true so the user can type their own instead. Each option's explanation "+
				"should say why that criterion would prove the feature actually works.",
			contextSummary(st),
		), "feature_criteria")
		v, e, err := ask(q)
		return v, q.Prompt, e, err
	})
	if err != nil {
		return err
	}
	data.Criterion = criterion

	tr.SetPhase("gate")
	st.Phase = "gate"
	_ = st.Save()

	prdPath, err := render.RenderPRD(targetDir, data)
	if err != nil {
		return err
	}

	fgate := phase.CheckFeatureGate(filepath.Join(targetDir, prdPath))
	if !fgate.Passed {
		tr.Warn("feature gate not fully satisfied — %s", fgate.Reason)
	} else {
		tr.Done("gate passed: %s", fgate.Reason)
	}

	if err := provenance.WriteFromState(st, targetDir); err != nil {
		return err
	}

	st.Phase = "done"
	st.Done = true
	if err := st.Save(); err != nil {
		return err
	}

	tr.Done("%s is ready at %s", name, targetDir)
	fmt.Println("  next: open AGENTS.md, then start the TDD loop on", prdPath)
	return nil
}

// RunRetrofit scans an existing repo, adds any missing fixed files, and
// asks before touching anything that's already there. The file plan is
// always shown in the TUI and confirmed before anything is written.
func RunRetrofit(targetDir string, client *llm.Client, launcher *backend.Launcher) error {
	tr := trace.New(targetDir)
	defer func() {
		if err := tr.Flush(); err != nil {
			fmt.Fprintln(os.Stderr, "warning: could not write trace file:", err)
		}
	}()
	_ = checkClient(tr, client, launcher) // retrofit asks no model-backed questions, but attach-or-launch still ensures/surfaces backend connectivity

	tr.SetPhase("scan")
	tr.Step("scanning %s", targetDir)
	report, err := detect.Scan(targetDir, render.FixedDestinations())
	if err != nil {
		return err
	}
	tr.Done("detected stack: %s | git: %v | README: %v", orUnknown(report.Stack), report.HasGit, report.HasReadme)

	st, err := phase.Load(targetDir)
	if err != nil {
		st = phase.NewState("retrofit", filepath.Base(targetDir), targetDir)
	}
	st.Phase = "fixedfiles"
	_ = st.Save()

	tr.SetPhase("fixedfiles")
	toWrite, existing := render.PlanFixed(targetDir)
	proceed, err := confirmPlan(tr, toWrite, existing)
	if err != nil {
		return err
	}
	if !proceed {
		tr.Warn("stopped before writing — nothing was changed")
		return nil
	}

	written, skipped, err := render.WriteFixed(targetDir, false)
	if err != nil {
		return err
	}
	tr.Done("wrote %d new fixed files, %d already exist", len(written), len(skipped))

	for _, dst := range skipped {
		key := "conflict:" + dst
		if _, ok := st.Answers[key]; ok {
			continue
		}
		label, explanation, err := tui.RunSelect(
			fmt.Sprintf("%s already exists — keep it, or replace with the kit's version?", dst),
			[]tui.Option{
				{Label: "Keep existing", Explanation: "Leave the file exactly as it is."},
				{Label: "Replace with template", Explanation: "Overwrite with the kit's current version."},
			}, false,
		)
		if err != nil {
			return err
		}
		st.Answers[key] = label
		st.Questions[key] = fmt.Sprintf("%s already exists — keep or replace?", dst)
		st.Explanations[key] = explanation
		_ = st.Save()
		if label == "Replace with template" {
			if err := render.WriteOne(targetDir, dst); err != nil {
				return err
			}
			tr.Done("replaced %s", dst)
		} else {
			tr.Step("kept existing %s", dst)
		}
	}

	if err := provenance.WriteFromState(st, targetDir); err != nil {
		return err
	}

	st.Phase = "done"
	st.Done = true
	if err := st.Save(); err != nil {
		return err
	}
	tr.Done("retrofit complete")
	return nil
}

func orUnknown(s string) string {
	if s == "" {
		return "unknown"
	}
	return s
}
