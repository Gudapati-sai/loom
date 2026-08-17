package phase

import (
	"testing"
	"time"
)

func TestStateSaveLoad(t *testing.T) {
	tmp := t.TempDir()
	st := NewState("new", "test-project", tmp)
	st.Answers["stack"] = "Go"
	st.Answers["feature_name"] = "photo-tool"

	if err := st.Save(); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	loaded, err := Load(tmp)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if loaded.ProjectName != "test-project" {
		t.Errorf("project name mismatch: got %q", loaded.ProjectName)
	}
	if loaded.Answers["stack"] != "Go" {
		t.Errorf("stack answer not persisted: got %q", loaded.Answers["stack"])
	}
	if loaded.Answers["feature_name"] != "photo-tool" {
		t.Errorf("feature_name not persisted: got %q", loaded.Answers["feature_name"])
	}
}

func TestStateResumeSkipsAnswered(t *testing.T) {
	tmp := t.TempDir()
	st := NewState("new", "resume-test", tmp)
	st.Answers["name"] = "already-answered"
	st.Answers["stack"] = "Python"
	_ = st.Save()

	loaded, err := Load(tmp)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	if loaded.Answers["name"] != "already-answered" {
		t.Errorf("resume should preserve answered question")
	}
	if loaded.Answers["stack"] != "Python" {
		t.Errorf("resume should preserve answered question")
	}
}

func TestNewStateTimestamps(t *testing.T) {
	before := time.Now()
	st := NewState("new", "ts-test", "/tmp")
	after := time.Now()
	if st.StartedAt.Before(before) || st.StartedAt.After(after) {
		t.Errorf("StartedAt not in expected range")
	}
	if !st.UpdatedAt.Equal(st.StartedAt) {
		t.Errorf("UpdatedAt should equal StartedAt on creation")
	}
}