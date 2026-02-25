package unit

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestContractJSONArtifactsAreValid(t *testing.T) {
	root, err := findRepoRoot()
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}

	patterns := []string{
		"contracts/api/v1/*.json",
		"contracts/events/v1/*.json",
		"contracts/schemas/v1/*.json",
	}

	found := 0
	for _, pattern := range patterns {
		matches, err := filepath.Glob(filepath.Join(root, pattern))
		if err != nil {
			t.Fatalf("invalid glob pattern %s: %v", pattern, err)
		}
		for _, path := range matches {
			found++
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read %s: %v", path, err)
			}
			var payload any
			if err := json.Unmarshal(data, &payload); err != nil {
				t.Fatalf("invalid json contract file %s: %v", path, err)
			}
		}
	}

	if found == 0 {
		t.Fatalf("no contract json artifacts found")
	}
}

func findRepoRoot() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	current := wd
	for {
		if _, err := os.Stat(filepath.Join(current, "go.mod")); err == nil {
			return current, nil
		}
		parent := filepath.Dir(current)
		if parent == current {
			return "", fmt.Errorf("go.mod not found from %s", wd)
		}
		current = parent
	}
}
