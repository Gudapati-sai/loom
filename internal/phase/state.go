package phase

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// State tracks wizard progress for a single project so a session can be
// paused and resumed with `loom resume` — already-answered questions are
// never asked again.
type State struct {
	ProjectName  string            `json:"project_name"`
	TargetDir    string            `json:"target_dir"`
	Mode         string            `json:"mode"` // "new" or "retrofit"
	Phase        string            `json:"phase"`
	Done         bool              `json:"done"`
	Answers      map[string]string `json:"answers"`
	Questions    map[string]string `json:"questions"`
	Explanations map[string]string `json:"explanations"`
	StartedAt    time.Time         `json:"started_at"`
	UpdatedAt    time.Time         `json:"updated_at"`
}

const stateFileName = ".loom-state.json"

func statePath(dir string) string {
	return filepath.Join(dir, stateFileName)
}

func NewState(mode, projectName, targetDir string) *State {
	now := time.Now()
	return &State{
		ProjectName:  projectName,
		TargetDir:    targetDir,
		Mode:         mode,
		Answers:      map[string]string{},
		Questions:    map[string]string{},
		Explanations: map[string]string{},
		StartedAt:    now,
		UpdatedAt:    now,
	}
}

func Load(dir string) (*State, error) {
	data, err := os.ReadFile(statePath(dir))
	if err != nil {
		return nil, err
	}
	var s State
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

func (s *State) Save() error {
	s.UpdatedAt = time.Now()
	if err := os.MkdirAll(s.TargetDir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(statePath(s.TargetDir), data, 0o644)
}
